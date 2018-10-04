package xl

import (
	"archive/zip"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"github.com/jhoonb/archivex"
	"github.com/pkg/errors"
	"github.com/xebialabs/yaml"
)

type Document struct {
	unmarshalleddocument
	Line     int
	Column   int
	ApplyZip string
}

type DocumentReader struct {
	decoder *yaml.Decoder
}

type unmarshalleddocument struct {
	Kind       string
	ApiVersion string                        `yaml:"apiVersion"`
	Metadata   map[interface{}]interface{}   `yaml:"metadata,omitempty"`
	Spec	   interface{}					 `yaml:"spec,omitempty"`
}

type processingContext struct {
	context      *Context
	artifactsDir string
	zipfile      *os.File
	zipwriter    *zip.Writer
	seenFiles    map[string]bool
}

func NewDocumentReader(reader io.Reader) *DocumentReader {
	decoder := yaml.NewDecoder(reader)
	decoder.SetStrict(true)
	return &DocumentReader{
		decoder: decoder,
	}
}

func (reader *DocumentReader) ReadNextYamlDocument() (*Document, error) {
	pdoc := unmarshalleddocument{}
	line, column, err := reader.decoder.DecodeWithPosition(&pdoc)

	if line == 0 {
		line++ // the YAML parser counts from 0, but people count from 1
	} else {
		line += 2 // the YAML parser returns the line number of the document separator before the document _and_ count from 0
	}

	if err != nil {
		return &Document{unmarshalleddocument{}, line, column, ""}, err
	}

	doc := Document{pdoc,  line, column, ""}
	if doc.Metadata == nil {
		doc.Metadata = map[interface{}]interface{}{}
	}

	return &doc, nil
}

func ParseYamlDocument(yamlDoc string) (*Document, error) {
	return NewDocumentReader(strings.NewReader(yamlDoc)).ReadNextYamlDocument()
}

func (doc *Document) Preprocess(context *Context, artifactsDir string) error {
	c := processingContext{context, artifactsDir, nil, nil, make(map[string]bool)}

	if c.context != nil {
		if c.context.XLDeploy != nil && c.context.XLDeploy.AcceptsDoc(doc) {
			c.context.XLDeploy.PreprocessDoc(doc)
		}

		if c.context.XLRelease != nil && c.context.XLRelease.AcceptsDoc(doc) {
			c.context.XLRelease.PreprocessDoc(doc)
		}
	}
	spec := TransformToMap(doc.Spec)

	err := doc.processListOfMaps(spec, &c)

	if c.zipfile != nil {
		defer func() {
			if c.zipwriter != nil {
				c.zipwriter.Close()
			}
			c.zipfile.Close()
			if doc.ApplyZip == "" {
				os.Remove(c.zipfile.Name())
			}
		}()
	}

	if err != nil {
		if c.zipfile != nil {
			os.Remove(c.zipfile.Name())
		}
		return err
	}

	if c.zipwriter != nil {
		yamlwriter, err := c.zipwriter.Create("index.yaml")
		if err != nil {
			return err
		}

		docBytes, err := doc.RenderYamlDocument()
		if err != nil {
			return err
		}

		_, err = yamlwriter.Write(docBytes)
		if err != nil {
			return err
		}

		doc.ApplyZip = c.zipfile.Name()
	}

	return err
}

func (doc *Document) processListOfMaps(l []map[interface{}]interface{}, c *processingContext) error {
	for _, v := range l {
		err := doc.processMap(v, c)
		if err != nil {
			return err
		}
	}
	return nil
}

func (doc *Document) processList(l []interface{}, c *processingContext) error {
	for i, v := range l {
		newV, err := doc.processValue(v, c)
		if err != nil {
			return err
		}
		l[i] = newV
	}
	return nil
}

func (doc *Document) processMap(m map[interface{}]interface{}, c *processingContext) (error) {
	for k, v := range m {
		newV, err := doc.processValue(v, c)
		if err != nil {
			return err
		}
		m[k] = newV
	}
	return nil
}

func (doc *Document) processValue(v interface{}, c *processingContext) (interface{}, error) {
	switch tv := v.(type) {
	case []interface{}:
		err := doc.processList(tv, c)
		if err != nil {
			return nil, err
		}
	case map[interface{}]interface{}:
		err := doc.processMap(tv, c)
		if err != nil {
			return nil, err
		}
	case []map[interface{}]interface{}:
		err := doc.processListOfMaps(tv, c)
		if err != nil {
			return nil, err
		}
	case yaml.CustomTag:
		newV, err := doc.processCustomTag(&tv, c)
		if err != nil {
			return nil, err
		}
		return newV, nil
	}
	return v, nil
}

func (doc *Document) processCustomTag(tag *yaml.CustomTag, c *processingContext) (interface{}, error) {
	switch tag.Tag {
	case "!file":
		return doc.processFileTag(tag, c)
	default:
		return nil, errors.New(fmt.Sprintf("unknown tag %s %s", tag.Tag, tag.Value))
	}
}

func (doc *Document) processFileTag(tag *yaml.CustomTag, c *processingContext) (interface{}, error) {
	doc.normalizeFileTag(tag, c)

	err := doc.validateFileTag(tag, c)
	if err != nil {
		return nil, err
	}

	if c.zipwriter == nil {
		zipfile, err := ioutil.TempFile("", "yaml")
		if err != nil {
			return nil, err
		}
		Verbose("...... first !file tag found, creating temporary ZIP file `%s`\n", zipfile.Name())
		c.zipfile = zipfile
		c.zipwriter = zip.NewWriter(c.zipfile)
	}

	filename := tag.Value

	if _, found := c.seenFiles[filename]; found {
		Verbose("...... file `%s` has already been added to the ZIP file. Skipping it\n", filename)
		return tag, nil
	}

	fileTag, writeError := doc.writeFileOrDir(tag, filename, c)
	if writeError != nil {
		return nil, writeError
	} else {
		c.seenFiles[filename] = true
		return fileTag, nil
	}
}

func (doc *Document) normalizeFileTag(tag *yaml.CustomTag, c *processingContext)  {
	tag.Value = filepath.Clean(tag.Value)
}

func (doc *Document) validateFileTag(tag *yaml.CustomTag, c *processingContext) error {
	if c.artifactsDir == "" {
		return errors.New("cannot process !file tags if artifactsDir has not been set")
	}
	filename := tag.Value
	if filepath.IsAbs(filename) {
		return errors.New(fmt.Sprintf("absolute path is not allowed in !file tag: %s", filename))
	}
	if isRelativePath(filename) {
		return errors.New(fmt.Sprintf("relative path with .. is not allowed in !file tag: %s", filename))
	}
	return nil
}

func isRelativePath(filename string) bool {
	for _, p := range strings.Split(filename, string(os.PathSeparator)) {
		if p == ".." {
			return true
		}
	}
	return false
}

func (doc *Document) writeFileOrDir(tag *yaml.CustomTag, filename string, c *processingContext) (interface{}, error) {
	fullFilename := filepath.Join(c.artifactsDir, filename)

	fi, err := os.Stat(fullFilename)
	if err != nil {
		return nil, err
	}
	switch mode := fi.Mode(); {
	case mode.IsDir():
		return doc.writeDirectory(tag, filename, fullFilename, c)
	}
	return tag, doc.writeFile(filename, fullFilename, c)
}

func (doc *Document) writeDirectory(tag *yaml.CustomTag, filename string, fullFilename string, c *processingContext) (interface{}, error) {
	Verbose("...... adding directory `%s` to ZIP file\n", filename)

	w, err := c.zipwriter.Create(filename)
	if err != nil {
		return nil, err
	}

	z := &archivex.ZipFile{}
	z.CreateWriter(filename, w)
	defer z.Close()
	return tag, z.AddAll(fullFilename, false)
}

func (doc *Document) writeFile(filename string, fullFilename string, c *processingContext) error {
	Verbose("...... adding file `%s` to ZIP file\n", filename)

	r, err := os.Open(fullFilename)
	if err != nil {
		return err
	}
	defer r.Close()

	w, err := c.zipwriter.Create(filename)
	if err != nil {
		return err
	}

	io.Copy(w, r)
	return nil
}

func (doc *Document) RenderYamlDocument() ([]byte, error) {
	rendered, err := yaml.Marshal(doc.unmarshalleddocument)
	if err != nil {
		return nil, err
	}
	return rendered, nil
}

func (doc *Document) Cleanup() {
	if doc.ApplyZip != "" {
		Verbose("...... deleting temporary file `%s`\n", doc.ApplyZip)
		os.Remove(doc.ApplyZip)
		doc.ApplyZip = ""
	}
}

func TransformToMap(spec interface {}) [] map[interface{}] interface{} {
	var convertedMap [] map[interface{}] interface{}

	switch typeVal := spec.(type) {
	case []interface {}:
		list := make([]map[interface{}]interface{}, 0)
		for _, v := range typeVal {
			list = append(list, v.(map[interface{}]interface{}))
		}
		temporaryList := make([]map[interface{}]interface{}, len(list))

		for i, v := range list {
			temporaryList[i] = v
		}

		convertedMap = temporaryList
	case []map[interface{}]interface{}:
		convertedMap = typeVal
	case map[interface{}]interface{}:
		list := [...]map[interface{}]interface{}{typeVal}
		temporaryList := make([]map[interface{}]interface{}, 1)

		for i, v := range list {
			temporaryList[i] = v
		}

		convertedMap = temporaryList
	}

	return convertedMap
}
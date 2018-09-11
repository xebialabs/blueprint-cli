package xl

import (
	"archive/zip"
	"fmt"
	"io"
	"os"
	"io/ioutil"
	"path/filepath"
	"strings"
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
	ApiVersion string `yaml:"apiVersion"`
	Metadata   map[interface{}]interface{} `yaml:"metadata,omitempty"`
	Spec       []map[interface{}]interface{} `yaml:"spec,omitempty"`
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
	if doc.Spec == nil {
		doc.Spec = []map[interface{}]interface{}{}
	}

	return &doc, nil
}

func ParseYamlDocument(yamlDoc string) (*Document, error) {
	return NewDocumentReader(strings.NewReader(yamlDoc)).ReadNextYamlDocument()
}

func (doc *Document) Preprocess(context *Context, artifactsDir string) (error) {
	c := processingContext{context, artifactsDir, nil, nil, make(map[string]bool)}

	if c.context != nil {
		if c.context.XLDeploy != nil && c.context.XLDeploy.AcceptsDoc(doc) {
			c.context.XLDeploy.PreprocessDoc(doc)
		}

		if c.context.XLRelease != nil && c.context.XLRelease.AcceptsDoc(doc) {
			c.context.XLRelease.PreprocessDoc(doc)
		}
	}

	err := doc.processListOfMaps(doc.Spec, &c)

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

func (doc *Document) processListOfMaps(l []map[interface{}]interface{}, c *processingContext) (error) {
	for _, v := range l {
		err := doc.processMap(v, c)
		if err != nil {
			return err
		}
	}
	return nil
}

func (doc *Document) processList(l []interface{}, c *processingContext) (error) {
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
		newV, err := doc.processCustomTag(tv, c)
		if err != nil {
			return nil, err
		}
		return newV, nil
	}
	return v, nil
}

func (doc *Document) processCustomTag(tag yaml.CustomTag, c *processingContext) (interface{}, error) {
	switch tag.Tag {
	case "!file":
		return doc.processFileTag(tag, c)
	default:
		return nil, errors.New(fmt.Sprintf("unknown tag %s %s", tag.Tag, tag.Value))
	}
}

func (doc *Document) processFileTag(tag yaml.CustomTag, c *processingContext) (interface{}, error) {
	if c.artifactsDir == "" {
		return nil, errors.New("cannot process !file tags if artifactsDir has not been set")
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
		Verbose("...... file `%s` is already in the ZIP file, skipping it\n", filename)
		return tag, nil
	}

	if filepath.IsAbs(filename) {
		return nil, errors.New(fmt.Sprintf("absolute path `%s` is not allowed in !file tag", filename))
	}
	if isRelativePath(filename) {
		return nil, errors.New(fmt.Sprintf("relative path with .. `%s` is not allowed in !file tag", filename))
	}

	fullFilename := filepath.Join(c.artifactsDir, filename)
	Verbose("...... !file tag `%s` in XL YAML document was resolved to full path `%s`\n", filename, fullFilename)

	fi, err := os.Stat(fullFilename)
	if err != nil {
		return nil, err
	}
	switch mode := fi.Mode(); {
	case mode.IsDir():
		return nil, errors.New(fmt.Sprintf("directories are not supported in !file tag: %s", filename))
	}

	r, err := os.Open(fullFilename)
	if err != nil {
		return nil, err
	}
	defer r.Close()

	w, err := c.zipwriter.Create(filename)
	if err != nil {
		return nil, err
	}
	Verbose("...... adding file `%s` to ZIP file\n", filename)
	io.Copy(w, r)

	c.seenFiles[filename] = true

	return tag, nil
}

func isRelativePath(filename string) bool {
	for _, p := range strings.Split(filename, string(os.PathSeparator)) {
		if p == ".." {
			return true
		}
	}
	return false
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

package xl

import (
	"archive/zip"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	"strconv"

	"github.com/jhoonb/archivex"
	"github.com/xebialabs/xl-cli/pkg/util"
	"github.com/xebialabs/yaml"
)

type Document struct {
	unmarshalleddocument
	Line   int
	Column int
	Zip    string
}

type DocumentReader struct {
	decoder *yaml.Decoder
}

type unmarshalleddocument struct {
	Kind       string
	ApiVersion string                      `yaml:"apiVersion"`
	Metadata   map[interface{}]interface{} `yaml:"metadata,omitempty"`
	Spec       interface{}                 `yaml:"spec,omitempty"`
}

type processingContext struct {
	context      *Context
	artifactsDir string
	zipfile      *os.File
	zipwriter    *zip.Writer
	seenFiles    map[string]bool
	counter      uint64
}

func (context *processingContext) IncrementCounter() string {
	context.counter++
	return strconv.FormatUint(context.counter, 10)
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

	doc := Document{pdoc, line, column, ""}
	if doc.Metadata == nil {
		doc.Metadata = map[interface{}]interface{}{}
	}

	return &doc, nil
}

func ParseYamlDocument(yamlDoc string) (*Document, error) {
	return NewDocumentReader(strings.NewReader(yamlDoc)).ReadNextYamlDocument()
}

func (doc *Document) Preprocess(context *Context, artifactsDir string) error {
	c := processingContext{context, artifactsDir, nil, nil, make(map[string]bool), 0}

	if c.context != nil {
		if c.context.XLDeploy != nil && c.context.XLDeploy.AcceptsDoc(doc) {
			c.context.XLDeploy.PreprocessDoc(doc)
		}

		if c.context.XLRelease != nil && c.context.XLRelease.AcceptsDoc(doc) {
			c.context.XLRelease.PreprocessDoc(doc)
		}
	}
	spec := util.TransformToMap(doc.Spec)

	err := doc.processListOfMaps(spec, &c)

	if c.zipfile != nil {
		defer func() {
			if c.zipwriter != nil {
				c.zipwriter.Close()
			}
			c.zipfile.Close()
			if doc.Zip == "" {
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

		doc.Zip = c.zipfile.Name()
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

func (doc *Document) processMap(m map[interface{}]interface{}, c *processingContext) error {
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
	case "!value":
		return doc.processValueTag(tag, c)
	case "!file":
		return doc.processFileTag(tag, c)
	case "!format":
		return doc.processFormatTag(tag, c)
	default:
		return nil, fmt.Errorf("unknown tag %s %s", tag.Tag, tag.Value)
	}
}

func (doc *Document) processFormatTag(tag *yaml.CustomTag, c *processingContext) (interface{}, error) {
	return Substitute(tag.Value, c.context.values)
}

func (doc *Document) processValueTag(tag *yaml.CustomTag, c *processingContext) (interface{}, error) {
	value, exists := c.context.values[tag.Value]
	if !exists {
		return nil, fmt.Errorf("No value found for !value %s", tag.Value)
	}

	util.Verbose("\tSubstituting value for [%s]\n", tag.Value)

	return value, nil
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
		util.Verbose("\tfirst !file tag found, creating temporary ZIP file %s\n", zipfile.Name())
		c.zipfile = zipfile
		c.zipwriter = zip.NewWriter(c.zipfile)
	}

	relativeFilename := tag.Value

	if _, found := c.seenFiles[relativeFilename]; found {
		util.Verbose("\tfile %s has already been added to the ZIP file. Skipping it\n", relativeFilename)
		return tag, nil
	}

	fileTag, writeError := doc.writeFileOrDir(tag, relativeFilename, c)
	if writeError != nil {
		return nil, writeError
	}
	c.seenFiles[relativeFilename] = true
	return fileTag, nil
}

func (doc *Document) normalizeFileTag(tag *yaml.CustomTag, c *processingContext) {
	tag.Value = filepath.Clean(tag.Value)
}

func (doc *Document) validateFileTag(tag *yaml.CustomTag, c *processingContext) error {
	if c.artifactsDir == "" {
		return fmt.Errorf("cannot process !file tags if artifactsDir has not been set")
	}
	filename := tag.Value
	return util.ValidateFilePath(filename, "!file tag")
}

func (doc *Document) writeFileOrDir(tag *yaml.CustomTag, relativeFilename string, c *processingContext) (interface{}, error) {
	absoluteFilename := filepath.Join(c.artifactsDir, relativeFilename)

	fi, err := os.Stat(absoluteFilename)
	if err != nil {
		return nil, err
	}

	counter := c.IncrementCounter()
	zipEntryFilename := filepath.Join(counter, filepath.Base(relativeFilename))
	tag.Value = zipEntryFilename

	mode := fi.Mode()
	if mode.IsDir() {
		util.Verbose("\tadding directory for !file %s\n", relativeFilename)
		return tag, doc.writeDirectory(zipEntryFilename, absoluteFilename, c)
	} else {
		util.Verbose("\tadding file for !file %s\n", relativeFilename)
		return tag, doc.writeFile(zipEntryFilename, absoluteFilename, c)
	}
}

func (doc *Document) writeDirectory(zipEntryFilename string, fullFilename string, c *processingContext) error {
	w, err := c.zipwriter.Create(zipEntryFilename)
	if err != nil {
		return err
	}

	z := &archivex.ZipFile{}
	_ = z.CreateWriter(zipEntryFilename, w)
	defer z.Close()
	return z.AddAll(fullFilename, false)
}

func (doc *Document) writeFile(zipEntryFilename string, fullFilename string, c *processingContext) error {
	r, err := os.Open(fullFilename)
	if err != nil {
		return err
	}
	defer r.Close()

	w, err := c.zipwriter.Create(zipEntryFilename)
	if err != nil {
		return err
	}

	_, _ = io.Copy(w, r)
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
	if doc.Zip != "" {
		util.Verbose("\tdeleting temporary file %s\n", doc.Zip)
		_ = os.Remove(doc.Zip)
		doc.Zip = ""
	}
}

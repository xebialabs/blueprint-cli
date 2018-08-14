package lib

import (
	"gopkg.in/yaml.v2"
	"io"
)

type Document struct {
	Kind       string
	ApiVersion string `yaml:"apiVersion"`
	Metadata   map[interface{}]interface{}
	Spec       []map[interface{}]interface{}
}

type DocumentReader struct {
	decoder *yaml.Decoder
}

func NewDocumentReader(reader io.Reader) *DocumentReader {
	decoder := yaml.NewDecoder(reader)
	decoder.SetStrict(true)
	return &DocumentReader{
		decoder: decoder,
	}
}

func (reader *DocumentReader) ReadNextYamlDocument() (*Document, error) {
	doc := Document{}
	err := reader.decoder.Decode(&doc)
	if err != nil {
		return nil, err
	}
	return &doc, nil
}

func ParseYamlDocument(yamlDoc string) (*Document, error) {
	doc := Document{}
	err := yaml.Unmarshal([]byte(yamlDoc), &doc)
	if err != nil {
		return nil, err
	}

	return &doc, nil
}

func (doc *Document) RenderYamlDocument() ([]byte, error) {
	d, err := yaml.Marshal(&doc)
	if err != nil {
		return nil, err
	}
	return d, nil
}
package lib

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

type DummyXLServer struct {
	docs []Document
}

func (dummy *DummyXLServer) SendDoc(doc *Document) error {
	dummy.docs = append(dummy.docs, *doc)
	return nil
}

func TestContext(t *testing.T) {
	t.Run("sendDocumentToCorrectServer", func(t *testing.T) {
		xld := DummyXLServer{}
		xlr := DummyXLServer{}
		context := Context{
			XLDeploy:  &xld,
			XLRelease: &xlr,
		}

		yamlDoc := `
apiVersion: xl-deploy/v1alpha1
kind: Applications
spec:
- name: Applications/AWS
  type: core.Directory
`
		doc, err := ParseYamlDocument(yamlDoc)
		assert.NotNil(t, doc)
		assert.Nil(t, err)

		err2 := context.ProcessSingleDocument(doc)
		assert.Nil(t, err2)
		assert.Equal(t, 1, len(xld.docs))
		assert.Equal(t, *doc, xld.docs[0])
		assert.Equal(t, 0, len(xlr.docs))
	})

	t.Run("errorMessageOnWrongApiVersion", func(t *testing.T) {
		xld := DummyXLServer{}
		xlr := DummyXLServer{}
		context := Context{
			XLDeploy:  &xld,
			XLRelease: &xlr,
		}

		yamlDoc := "apiVersion: xxxx"
		doc, err := ParseYamlDocument(yamlDoc)
		assert.NotNil(t, doc)
		assert.Nil(t, err)

		err2 := context.ProcessSingleDocument(doc)
		assert.NotNil(t, err2)
		assert.Equal(t, "unknown apiVersion: xxxx", err2.Error())
	})

	t.Run("errorMessageOnWrongApiVersion", func(t *testing.T) {
		xld := DummyXLServer{}
		xlr := DummyXLServer{}
		context := Context{
			XLDeploy:  &xld,
			XLRelease: &xlr,
		}

		yamlDoc := ""
		doc, err := ParseYamlDocument(yamlDoc)
		assert.NotNil(t, doc)
		assert.Nil(t, err)

		err2 := context.ProcessSingleDocument(doc)
		assert.NotNil(t, err2)
		assert.Equal(t, "apiVersion missing", err2.Error())
	})
}

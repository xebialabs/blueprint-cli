package xl

import (
	"fmt"
	"github.com/stretchr/testify/assert"
	"testing"
)

type DummyXLServer struct {
	accepts           []string
	preprocessInvoked int
	docs              []Document
}

func (dummy *DummyXLServer) GenerateDoc(filename string, path string, override bool, globalPermissions bool, users bool, roles bool) error {
	return nil
}

func (dummy *DummyXLServer) AcceptsDoc(doc *Document) bool {
	for _, accept := range dummy.accepts {
		if accept == doc.ApiVersion {
			return true
		}
	}
	return false
}

func (dummy *DummyXLServer) PreprocessDoc(doc *Document) {
	dummy.preprocessInvoked++
}

func (dummy *DummyXLServer) SendDoc(doc *Document) (*Changes, error) {
	dummy.docs = append(dummy.docs, *doc)
	return nil, nil
}

func (server *DummyXLServer) GetTaskStatus(taskId string) (*TaskState, error) {
	return nil, nil
}

func (server *DummyXLServer) GetSchema() ([]byte, error) {
	return nil, nil
}

func TestContext(t *testing.T) {
	t.Run("send YAML document with xl-deploy apiVersion to XL Deploy server", func(t *testing.T) {
		xld := DummyXLServer{accepts: []string{XldApiVersion}}
		xlr := DummyXLServer{accepts: []string{XlrApiVersion}}
		context := Context{XLDeploy: &xld, XLRelease: &xlr}

		yamlDoc := fmt.Sprintf(`apiVersion: %s
kind: Applications
spec:
- name: Applications/AWS
  type: core.Directory`, XldApiVersion)

		doc, err := ParseYamlDocument(yamlDoc)

		assert.NotNil(t, doc)
		assert.Nil(t, err)

		_, err2 := context.ProcessSingleDocument(doc, "")

		assert.Nil(t, err2)
		assert.Equal(t, 1, xld.preprocessInvoked)
		assert.Equal(t, 1, len(xld.docs))
		assert.Equal(t, *doc, xld.docs[0])
		assert.Equal(t, 0, len(xlr.docs))
	})

	t.Run("report error when YAML document contains unknown apiVersion", func(t *testing.T) {
		xld := DummyXLServer{accepts: []string{XldApiVersion}}
		xlr := DummyXLServer{accepts: []string{XlrApiVersion}}
		context := Context{XLDeploy: &xld, XLRelease: &xlr}

		yamlDoc := "apiVersion: xxxx"

		doc, err := ParseYamlDocument(yamlDoc)

		assert.NotNil(t, doc)
		assert.Nil(t, err)

		_, err2 := context.ProcessSingleDocument(doc, "")

		assert.NotNil(t, err2)
		assert.Equal(t, "unknown apiVersion: xxxx", err2.Error())
	})

	t.Run("report error when YAML document does not contain an apiVersion", func(t *testing.T) {
		xld := DummyXLServer{accepts: []string{XldApiVersion}}
		xlr := DummyXLServer{accepts: []string{XlrApiVersion}}
		context := Context{XLDeploy: &xld, XLRelease: &xlr}

		yamlDoc := "kind: Version"

		doc, err := ParseYamlDocument(yamlDoc)

		assert.NotNil(t, doc)
		assert.Nil(t, err)

		_, err2 := context.ProcessSingleDocument(doc, "")

		assert.NotNil(t, err2)
		assert.Equal(t, "apiVersion missing", err2.Error())
	})
}

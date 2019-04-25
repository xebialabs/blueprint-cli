package xl

import (
	"fmt"
	"github.com/stretchr/testify/assert"
	"github.com/xebialabs/xl-cli/pkg/models"
	"testing"
)

type DummyXLServer struct {
	accepts           []string
	preprocessInvoked int
	applyDocs         []Document
	previewDocs       []Document
}

func (dummy *DummyXLServer) GenerateDoc(filename string, path string, override bool, globalPermissions bool, users bool, roles bool, environments bool, applications bool, includePasswords bool) error {
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
	dummy.applyDocs = append(dummy.applyDocs, *doc)
	return nil, nil
}

func (server *DummyXLServer) GetTaskStatus(taskId string) (*TaskState, error) {
	return nil, nil
}

func (server *DummyXLServer) GetSchema() ([]byte, error) {
	return nil, nil
}

func (server *DummyXLServer) PreviewDoc(doc *Document) (*models.PreviewResponse, error) {
	server.previewDocs = append(server.previewDocs, *doc)
	return nil, nil
}

func TestContext(t *testing.T) {
	t.Run("send YAML document with xl-deploy apiVersion to XL Deploy server when apply was called", func(t *testing.T) {
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
		assert.Equal(t, 1, len(xld.applyDocs))
		assert.Equal(t, *doc, xld.applyDocs[0])
		assert.Equal(t, 0, len(xlr.applyDocs))
	})

	t.Run("send YAML document with xl-deploy apiVersion to XL Deploy server when preview was called", func(t *testing.T) {
		xld := DummyXLServer{accepts: []string{XldApiVersion}}
		xlr := DummyXLServer{accepts: []string{XlrApiVersion}}
		context := Context{XLDeploy: &xld, XLRelease: &xlr}

		yamlDoc := fmt.Sprintf(`apiVersion: %s
kind: Deployment
spec:
  package: Applications/PetPortal/1.0
  environment: Environments/AWS Environment
  undeployDependencies: true
  orchestrators:
    - parallel-by-deployed`, XldApiVersion)
		doc, err := ParseYamlDocument(yamlDoc)

		assert.NotNil(t, doc)
		assert.Nil(t, err)

		_, err2 := context.PreviewSingleDocument(doc, "")

		assert.Nil(t, err2)
		assert.Equal(t, 1, xld.preprocessInvoked)
		assert.Equal(t, 1, len(xld.previewDocs))
		assert.Equal(t, *doc, xld.previewDocs[0])
		assert.Equal(t, 0, len(xlr.previewDocs))
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

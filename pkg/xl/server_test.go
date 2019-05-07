package xl

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/xebialabs/xl-cli/pkg/models"
)

type DummyHTTPServer struct {
	capturedPath         string
	capturedBytes        []byte
	capturedFilename     string
	mockTaskInfoResponse string
}

func (d *DummyHTTPServer) GenerateYamlDoc(path string, generateFilename string, override bool) error {
	return nil
}

func (d *DummyHTTPServer) ApplyYamlDoc(path string, yamlDocBytes []byte) (*Changes, error) {
	d.capturedPath = path
	d.capturedBytes = yamlDocBytes
	d.capturedFilename = ""
	return nil, nil
}

func (d *DummyHTTPServer) ApplyYamlZip(path string, zipfilename string) (*Changes, error) {
	d.capturedPath = path
	d.capturedBytes = nil
	d.capturedFilename = zipfilename
	return nil, nil
}

func (d *DummyHTTPServer) PreviewYamlDoc(path string, yamlDocBytes []byte) (*models.PreviewResponse, error) {
	d.capturedPath = path
	d.capturedBytes = yamlDocBytes
	d.capturedFilename = ""
	return nil, nil
}

func (d *DummyHTTPServer) TaskInfo(resource string) (map[string]interface{}, error) {
	d.capturedPath = resource
	d.capturedBytes = nil
	d.capturedFilename = ""

	var response map[string]interface{}
	err := json.Unmarshal([]byte(d.mockTaskInfoResponse), &response)
	if err != nil {
		return nil, err
	}
	return response, nil
}

func (server *DummyHTTPServer) DownloadSchema(resource string) ([]byte, error) {
	return nil, nil
}

func TestServer(t *testing.T) {
	t.Run(fmt.Sprintf("XL Deploy should accept %s documents", XldApiVersion), func(t *testing.T) {
		doc := Document{unmarshalleddocument{"Applications", XldApiVersion, nil, nil, nil}, 0, 0, "", ToProcess{true, true, true}}
		xlDeployServer := XLDeployServer{&DummyHTTPServer{}, "", "", "", ""}

		assert.True(t, xlDeployServer.AcceptsDoc(&doc))
	})

	t.Run(fmt.Sprintf("XL Deploy should not accept %s documents", XlrApiVersion), func(t *testing.T) {
		doc := Document{unmarshalleddocument{"Applications", XlrApiVersion, nil, nil, nil}, 0, 0, "", ToProcess{true, true, true}}
		xlDeployServer := XLDeployServer{&DummyHTTPServer{}, "", "", "", ""}

		assert.False(t, xlDeployServer.AcceptsDoc(&doc))
	})

	t.Run(fmt.Sprintf("XL Release should accept %s documents", XlrApiVersion), func(t *testing.T) {
		doc := Document{unmarshalleddocument{"Applications", XlrApiVersion, nil, nil, nil}, 0, 0, "", ToProcess{true, true, true}}
		xlReleaseServer := XLReleaseServer{&DummyHTTPServer{}, ""}

		assert.True(t, xlReleaseServer.AcceptsDoc(&doc))
	})

	t.Run(fmt.Sprintf("XL Release should not accept %s documents", XldApiVersion), func(t *testing.T) {
		doc := Document{unmarshalleddocument{"Applications", XldApiVersion, nil, nil, nil}, 0, 0, "", ToProcess{true, true, true}}
		xlReleaseServer := XLReleaseServer{&DummyHTTPServer{}, ""}

		assert.False(t, xlReleaseServer.AcceptsDoc(&doc))
	})

	t.Run("should send ZIP if generated", func(t *testing.T) {
		artifactsDir, err := ioutil.TempDir("", "should_send_ZIP_if_generated")
		if err != nil {
			assert.FailNow(t, "cannot open temporary directory", err)
		}
		defer os.RemoveAll(artifactsDir)
		artifactContents := "cats=5\ndogs=8\n"
		_ = ioutil.WriteFile(filepath.Join(artifactsDir, "petclinic.properties"), []byte(artifactContents), 0644)

		yamlDoc := fmt.Sprintf(`apiVersion: %s
kind: Applications
spec:
- name: PetClinic
  type: udm.Application
  children:
  - name: '1.0'
    type: udm.DeploymentPackage
    children:
    - name: conf
      type: file.File
      file: !file petclinic.properties`, XldApiVersion)

		doc, err := ParseYamlDocument(yamlDoc)

		assert.Nil(t, err)
		assert.NotNil(t, doc)

		dummyServer := DummyHTTPServer{}
		xlDeployServer := XLDeployServer{Server: &dummyServer}
		context := &Context{XLDeploy: &XLDeployServer{Server: &dummyServer}, XLRelease: &XLReleaseServer{Server: &DummyHTTPServer{}}}

		_ = doc.Preprocess(context, artifactsDir)
		defer doc.Cleanup()

		_, err = xlDeployServer.SendDoc(doc)

		assert.Nil(t, err)
		assert.Equal(t, "deployit/devops-as-code/apply", dummyServer.capturedPath)
		assert.Nil(t, dummyServer.capturedBytes)
		assert.Equal(t, doc.Zip, dummyServer.capturedFilename)
	})

	t.Run("XLD should properly parse task status", func(t *testing.T) {
		dummyServer := DummyHTTPServer{
			mockTaskInfoResponse: `{
    "id": "12345",
	"state": "EXECUTING",
    "activeBlocks": [
        "0_1_1"
    ],
    "block": {
        "blocks": [
            {
                "id": "0_1",
                "state": "EXECUTING",
                "description": "Deploy",
				"phase": "true",
                "block": {
                    "id": "0_1_1",
                    "state": "EXECUTING",
                    "description": "Update on Localhost"
                }
            },
            {
                "id": "0_2",
                "state": "PENDING",
                "description": "",
				"phase": "true",
                "block": {
                    "id": "0_2_1",
                    "state": "PENDING",
                    "description": "Register changes for PetPortal"
                }
            }
        ]
    }
}`,
		}
		xlDeployServer := XLDeployServer{Server: &dummyServer}
		state, _ := xlDeployServer.GetTaskStatus("12345")
		assert.NotNil(t, state)
		assert.Equal(t, "EXECUTING", state.State)
		assert.Len(t, state.CurrentSteps, 1)
		step := state.CurrentSteps[0]
		assert.Equal(t, "Update on Localhost", step.Name)
		assert.Equal(t, "EXECUTING", step.State)
		assert.Equal(t, true, step.Automated)
	})

	t.Run("XLR should properly parse task status", func(t *testing.T) {
		dummyServer := DummyHTTPServer{
			mockTaskInfoResponse: `{
    "id": "12345",
    "type": "xlrelease.Release",
    "status": "FAILING",
    "currentSimpleTasks": [
        {
            "title": "Parallel / Create Environment",
            "type": "xlrelease.CustomScriptTask",
            "status": "FAILED",
            "automated": true
        },
        {
            "title": "Parallel / Do Some manual task",
            "type": "xlrelease.Task",
            "status": "IN_PROGRESS",
            "automated": false
        }
    ]
}`,
		}
		xlReleaseServer := XLReleaseServer{Server: &dummyServer}
		state, _ := xlReleaseServer.GetTaskStatus("12345")

		assert.NotNil(t, state)
		assert.Equal(t, "FAILING", state.State)
		assert.Len(t, state.CurrentSteps, 2)
		step1 := state.CurrentSteps[0]
		assert.Equal(t, "Parallel / Create Environment", step1.Name)
		assert.Equal(t, "FAILED", step1.State)
		assert.Equal(t, true, step1.Automated)

		step2 := state.CurrentSteps[1]
		assert.Equal(t, "Parallel / Do Some manual task", step2.Name)
		assert.Equal(t, "IN_PROGRESS", step2.State)
		assert.Equal(t, false, step2.Automated)
	})
}

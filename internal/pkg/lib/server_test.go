package lib

import (
	"github.com/stretchr/testify/assert"
	"testing"
	"io/ioutil"
	"os"
	"path/filepath"
)

type DummyHTTPServer struct {
	capturedPath     string
	capturedBytes    []byte
	capturedFilename string
}

func (d *DummyHTTPServer) PostYamlDoc(path string, yamlDocBytes []byte) error {
	d.capturedPath = path
	d.capturedBytes = yamlDocBytes
	d.capturedFilename = ""
	return nil
}

func (d *DummyHTTPServer) PostYamlZip(path string, zipfilename string) error {
	d.capturedPath = path
	d.capturedBytes = nil
	d.capturedFilename = zipfilename
	return nil
}

func TestServer(t *testing.T) {
	t.Run("XL Deploy should accept xl-deploy/v1alpha1 documents", func(t *testing.T) {
		doc := Document{unmarshalleddocument{"Applications", "xl-deploy/v1alpha1", nil, nil}, ""}
		xlDeployServer := XLDeployServer{&DummyHTTPServer{}, "", "", "", ""}

		assert.True(t, xlDeployServer.AcceptsDoc(&doc))
	})

	t.Run("XL Deploy should not accept xl-release/v1 documents", func(t *testing.T) {
		doc := Document{unmarshalleddocument{"Applications", "xl-release/v1", nil, nil}, ""}
		xlDeployServer := XLDeployServer{&DummyHTTPServer{}, "", "", "", ""}

		assert.False(t, xlDeployServer.AcceptsDoc(&doc))
	})

	t.Run("XL Release should accept xl-release/v1 documents", func(t *testing.T) {
		doc := Document{unmarshalleddocument{"Applications", "xl-release/v1", nil, nil}, ""}
		xlReleaseServer := XLReleaseServer{&DummyHTTPServer{}, ""}

		assert.True(t, xlReleaseServer.AcceptsDoc(&doc))
	})

	t.Run("XL Release should not accept xl-deploy/v1alpha1 documents", func(t *testing.T) {
		doc := Document{unmarshalleddocument{"Applications", "xl-deploy/v1alpha1", nil, nil}, ""}
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
		ioutil.WriteFile(filepath.Join(artifactsDir, "petclinic.properties"), []byte(artifactContents), 0644)

		yamlDoc := `apiVersion: xl-deploy/v1
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
      file: !file petclinic.properties`

		doc, err := ParseYamlDocument(yamlDoc)

		assert.Nil(t, err)
		assert.NotNil(t, doc)

		dummyServer := DummyHTTPServer{}
		xlDeployServer := XLDeployServer{Server: &dummyServer}
		context := &Context{&xlDeployServer, &XLReleaseServer{Server: &DummyHTTPServer{}}}

		doc.Preprocess(context, artifactsDir)
		defer doc.Cleanup()

		err = xlDeployServer.SendDoc(doc)

		assert.Nil(t, err)
		assert.Equal(t, "deployit/ascode", dummyServer.capturedPath)
		assert.Nil(t, dummyServer.capturedBytes)
		assert.Equal(t, doc.ApplyZip, dummyServer.capturedFilename)
	})
}


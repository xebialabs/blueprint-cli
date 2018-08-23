package lib

import (
	"archive/zip"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"github.com/stretchr/testify/assert"
	"github.com/xebialabs/yaml"
)

func TestDocument(t *testing.T) {
	t.Run("should parse YAML document", func(t *testing.T) {
		yamlDoc := `apiVersion: xl-deploy/v1
kind: Applications
metadata:
  Applications-home: Applications/Cloud
spec:
- name: Applications/AWS
  type: core.Directory
  children:
    - name: rest-o-rant-ecs-service
      type: udm.Application
      children:
      - name: 1.0
        type: udm.DeploymentPackage
`

		doc, err := ParseYamlDocument(yamlDoc)

		assert.Nil(t, err)
		assert.NotNil(t, doc)
		assert.Equal(t, "xl-deploy/v1", doc.ApiVersion)
		assert.Equal(t, "Applications", doc.Kind)
		assert.NotNil(t, doc.Metadata)
		assert.Equal(t, "Applications/Cloud", doc.Metadata["Applications-home"])
		assert.NotNil(t, doc.Spec)
		assert.Equal(t, "Applications/AWS", doc.Spec[0]["name"])
		children := doc.Spec[0]["children"].([]interface{})
		firstChild := children[0].(map[interface{}]interface{})
		assert.Equal(t, "rest-o-rant-ecs-service", firstChild["name"])
		children2 := firstChild["children"].([]interface{})
		firstChild2 := children2[0].(map[interface{}]interface{})
		assert.Equal(t, "udm.DeploymentPackage", firstChild2["type"])
	})

	t.Run("should parse YAML file with multiple documents", func(t *testing.T) {
		yamlDocs := `apiVersion: xl-deploy/v1
kind: Applications
spec:
- name: Applications/AWS1
---
apiVersion: xl-release/v1
kind: Template
spec:
- name: Template1
---
apiVersion: xl-release/v1
kind: Template
spec:
- name: Template2
`

		docreader := NewDocumentReader(strings.NewReader(yamlDocs))
		doc, err := docreader.ReadNextYamlDocument()

		assert.Nil(t, err)
		assert.NotNil(t, doc)
		assert.Equal(t, "xl-deploy/v1", doc.ApiVersion)
		assert.Equal(t, "Applications", doc.Kind)
		assert.NotNil(t, doc.Spec)
		assert.Equal(t, "Applications/AWS1", doc.Spec[0]["name"])

		doc, err = docreader.ReadNextYamlDocument()

		assert.Nil(t, err)
		assert.NotNil(t, doc)
		assert.Equal(t, "xl-release/v1", doc.ApiVersion)
		assert.Equal(t, "Template", doc.Kind)
		assert.NotNil(t, doc.Spec)
		assert.Equal(t, "Template1", doc.Spec[0]["name"])

		doc, err = docreader.ReadNextYamlDocument()

		assert.Nil(t, err)
		assert.NotNil(t, doc)
		assert.NotNil(t, doc.Spec)
		assert.Equal(t, "Template2", doc.Spec[0]["name"])
	})


	t.Run("should parse YAML documents with custom tags", func(t *testing.T) {
		yamlDoc := `apiVersion: xl-deploy/v1
kind: Applications
spec:
- name: PetClinic-ear
  type: udm.Application
  children:
  - name: 1.0
    type: udm.DeploymentPackage
    children:
    - name: ear
      type: jee.Ear
      file: !file PetClinic-1.0.ear
---
apiVersion: xl-deploy/v1
kind: Intrastructure
spec:
- name: server
  type: overthere.SshHost
  address: server.example.com
  username: root
  password: !secret server.root.password
`

		docreader := NewDocumentReader(strings.NewReader(yamlDoc))
		doc1, err1 := docreader.ReadNextYamlDocument()

		assert.Nil(t, err1)
		assert.NotNil(t, doc1)
		assert.Equal(t, "PetClinic-ear", doc1.Spec[0]["name"])
		v1 := doc1.Spec[0]["children"]                     // spec[0].children
		v2 := v1.([]interface{})[0]                        // spec[0].children[0]
		v3 := v2.(map[interface{}]interface{})["children"] // spec[0].children[0].children
		v4 := v3.([]interface{})[0]                        // spec[0].children[0].children[0]
		v5 := v4.(map[interface{}]interface{})["file"]     // spec[0].children[0].children[0].file
		assert.Equal(t, yaml.CustomTag{"!file", "PetClinic-1.0.ear"}, v5)

		doc2, err2 := docreader.ReadNextYamlDocument()

		assert.Nil(t, err2)
		assert.NotNil(t, doc2)
		assert.Equal(t, "server", doc2.Spec[0]["name"])
		assert.Equal(t, yaml.CustomTag{"!secret", "server.root.password"}, doc2.Spec[0]["password"])
	})

	t.Run("should add xl-deploy homes if missing in yaml document", func(t *testing.T) {
		yamlDoc := `apiVersion: xl-deploy/v1alpha1
kind: Applications`

		doc, err := ParseYamlDocument(yamlDoc)

		assert.Nil(t, err)
		assert.NotNil(t, doc)

		context := &Context{&XLDeployServer{Server: &DummyHTTPServer{},
			ApplicationsHome:   "Applications/MyHome",
			EnvironmentsHome:   "Environments/MyHome",
			ConfigurationHome:  "Configuration/MyHome",
			InfrastructureHome: "Infrastructure/MyHome"}, &XLReleaseServer{Server: &DummyHTTPServer{}}}
		err = doc.Preprocess(context, "")
		defer doc.Cleanup()

		assert.Nil(t, err)
		assert.NotNil(t, doc)
		assert.Equal(t, "xl-deploy/v1alpha1", doc.ApiVersion)
		assert.Equal(t, "Applications", doc.Kind)
		assert.Equal(t, "Applications/MyHome", doc.Metadata["Applications-home"])
		assert.Equal(t, "Environments/MyHome", doc.Metadata["Environments-home"])
		assert.Equal(t, "Configuration/MyHome", doc.Metadata["Configuration-home"])
		assert.Equal(t, "Infrastructure/MyHome", doc.Metadata["Infrastructure-home"])
	})

	t.Run("should add xl-release home if missing in yaml document", func(t *testing.T) {
		yaml := `apiVersion: xl-release/v1
kind: Templates`

		doc, err := ParseYamlDocument(yaml)

		assert.Nil(t, err)
		assert.NotNil(t, doc)

		context := &Context{&XLDeployServer{Server: &DummyHTTPServer{}}, &XLReleaseServer{Server: &DummyHTTPServer{}, Home: "MyHome"}}
		err = doc.Preprocess(context, "")
		defer doc.Cleanup()

		assert.Nil(t, err)
		assert.NotNil(t, doc)
		assert.Equal(t, "xl-release/v1", doc.ApiVersion)
		assert.Equal(t, "MyHome", doc.Metadata["home"])
	})

	t.Run("should not replace homes if included in yaml document", func(t *testing.T) {
		yaml := `apiVersion: xl-deploy/v1
kind: Applications
metadata:
  Applications-home: Applications/DoNotTouch
  Environments-home: Environments/DoNotTouch
  Configuration-home: Configuration/DoNotTouch
  Infrastructure-home: Infrastructure/DoNotTouch`

		doc, err := ParseYamlDocument(yaml)

		assert.Nil(t, err)
		assert.NotNil(t, doc)

		context := &Context{&XLDeployServer{Server: &DummyHTTPServer{},
			ApplicationsHome:   "Applications/MyHome",
			EnvironmentsHome:   "Environments/MyHome",
			ConfigurationHome:  "Configuration/MyHome",
			InfrastructureHome: "Infrastructure/MyHome"}, &XLReleaseServer{Server: &DummyHTTPServer{}}}
		err = doc.Preprocess(context, "")
		defer doc.Cleanup()

		assert.Nil(t, err)
		assert.NotNil(t, doc)
		assert.Equal(t, "Applications/DoNotTouch", doc.Metadata["Applications-home"])
		assert.Equal(t, "Environments/DoNotTouch", doc.Metadata["Environments-home"])
		assert.Equal(t, "Configuration/DoNotTouch", doc.Metadata["Configuration-home"])
		assert.Equal(t, "Infrastructure/DoNotTouch", doc.Metadata["Infrastructure-home"])
	})

	t.Run("should not add homes if empty", func(t *testing.T) {
		yaml := `apiVersion: xl-deploy/v1
kind: Applications`

		doc, err := ParseYamlDocument(yaml)

		assert.Nil(t, err)
		assert.NotNil(t, doc)

		context := &Context{&XLDeployServer{Server: &DummyHTTPServer{},
			ApplicationsHome:   "",
			EnvironmentsHome:   "",
			ConfigurationHome:  "",
			InfrastructureHome: ""}, &XLReleaseServer{Server: &DummyHTTPServer{}}}
		err = doc.Preprocess(context, "")
		defer doc.Cleanup()

		assert.Nil(t, err)
		assert.NotNil(t, doc)
		assert.Nil(t, doc.Metadata["Applications-home"])
		assert.Nil(t, doc.Metadata["Environments-home"])
		assert.Nil(t, doc.Metadata["Configuration-home"])
		assert.Nil(t, doc.Metadata["Infrastructure-home"])
	})

	t.Run("should process !file tags", func(t *testing.T) {
		artifactsDir, err := ioutil.TempDir("", "should_process_file_tags")
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

		doc, err := NewDocumentReader(strings.NewReader(yamlDoc)).ReadNextYamlDocument()

		assert.Nil(t, err)
		assert.NotNil(t, doc)

		errp := doc.Preprocess(nil, artifactsDir)
		defer doc.Cleanup()

		assert.Nil(t, errp)
		assert.NotNil(t, doc.ApplyZip)
		zipR, err := zip.OpenReader(doc.ApplyZip)
		if err != nil {
			assert.FailNow(t, "cannot open generated ZIP file [%s]: %s", doc.ApplyZip, err)
		}

		fileContents := make(map[string]string)
		for _, f := range zipR.File {
			fr, err := f.Open()
			if err != nil {
				assert.FailNow(t, "cannot open entry [%s] in generated ZIP file [%s]: %s", f.Name, doc.ApplyZip, err)
			}
			contents, err := ioutil.ReadAll(fr)
			if err != nil {
				assert.FailNow(t, "cannot read entry [%s] in generated ZIP file [%s]: %s", f.Name, doc.ApplyZip, err)
			}
			fileContents[f.Name] = string(contents)
		}

		assert.Contains(t, fileContents, "index.yaml")
		indexDocument, err := ParseYamlDocument(fileContents["index.yaml"])
		Applications_PetClinic_1_0_conf_file := indexDocument.Spec[0]["children"].([]interface{})[0].(map[interface{}]interface{})["children"].([]interface{})[0].(map[interface{}]interface{})["file"].(yaml.CustomTag)
		assert.Contains(t, fileContents, Applications_PetClinic_1_0_conf_file.Value)
		assert.Equal(t, artifactContents, fileContents[Applications_PetClinic_1_0_conf_file.Value])
	})

	t.Run("should report error when !file tag contains absolute path", func(t *testing.T) {
		artifactsDir, err := ioutil.TempDir("", "should_process_file_tags")
		if err != nil {
			assert.FailNow(t, "cannot open temporary directory", err)
		}
		defer os.RemoveAll(artifactsDir)

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
      file: !file /etc/passwd`

		doc, err := ParseYamlDocument(yamlDoc)

		assert.Nil(t, err)
		assert.NotNil(t, doc)

		err = doc.Preprocess(nil, artifactsDir)
		assert.NotNil(t, err)
		assert.Contains(t,  err.Error(), "absolute path")
	})

	t.Run("should report error when !file tag contains relative path that starts with ..", func(t *testing.T) {
		artifactsDir, err := ioutil.TempDir("", "should_process_file_tags")
		if err != nil {
			assert.FailNow(t, "cannot open temporary directory", err)
		}
		defer os.RemoveAll(artifactsDir)

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
      file: !file ../../../../../../../../../../etc/passwd`

		doc, err := ParseYamlDocument(yamlDoc)

		assert.Nil(t, err)
		assert.NotNil(t, doc)

		err = doc.Preprocess(nil, artifactsDir)
		assert.NotNil(t, err)
		assert.Contains(t,  err.Error(), "relative path")
	})

	t.Run("should render YAML document", func(t *testing.T) {
		yamlDoc := `apiVersion: xl-deploy/v1
kind: Applications
metadata:
  Applications-home: Applications/Cloud
spec:
- name: Applications/AWS
  type: core.Directory
  children:
    - name: rest-o-rant-ecs-service
      type: udm.Application
      children:
      - name: '1.0'
        type: udm.DeploymentPackage
`

		doc, err := ParseYamlDocument(yamlDoc)

		assert.Nil(t, err)
		assert.NotNil(t, doc)

		renderedYamlDocBytes, err := doc.RenderYamlDocument()
		assert.Nil(t, err)
		assert.NotNil(t, renderedYamlDocBytes)
		renderedDoc, err := ParseYamlDocument(string(renderedYamlDocBytes))
		assert.Equal(t, doc.ApiVersion, renderedDoc.ApiVersion)
		assert.Equal(t, doc.Kind, renderedDoc.Kind)
		assert.Equal(t, doc.Metadata, renderedDoc.Metadata)
		assert.Equal(t, doc.Spec, renderedDoc.Spec)
	})

	t.Run("should not render metadata if empty", func(t *testing.T) {
		yamlDoc := `apiVersion: xl-deploy/v1
kind: Applications`

		doc, err := ParseYamlDocument(yamlDoc)

		assert.Nil(t, err)
		assert.NotNil(t, doc)

		renderedYaml, err := doc.RenderYamlDocument()
		assert.Nil(t, err)
		assert.NotContains(t, string(renderedYaml), "metadata")
	})

	t.Run("should not render spec if empty", func(t *testing.T) {
		yamlDoc := `apiVersion: xl-deploy/v1
kind: Applications`

		doc, err := ParseYamlDocument(yamlDoc)

		assert.Nil(t, err)
		assert.NotNil(t, doc)

		renderedYaml, err := doc.RenderYamlDocument()
		assert.Nil(t, err)
		assert.NotContains(t, string(renderedYaml), "spec")
	})

	t.Run("should render YAML document with empty metadata", func(t *testing.T) {
		yamlDoc := `apiVersion: xl-deploy/v1
kind: Infrastructure
spec:
- name: Localhost
  type: overthere.LocalHost`

		doc, err := ParseYamlDocument(yamlDoc)

		assert.Nil(t, err)
		assert.NotNil(t, doc)

		renderedYamlDocBytes, err := doc.RenderYamlDocument()
		assert.Nil(t, err)
		assert.NotNil(t, renderedYamlDocBytes)
		renderedDoc, err := ParseYamlDocument(string(renderedYamlDocBytes))
		assert.Equal(t, doc.ApiVersion, renderedDoc.ApiVersion)
		assert.Equal(t, doc.Kind, renderedDoc.Kind)
		assert.Equal(t, doc.Metadata, renderedDoc.Metadata)
		assert.Equal(t, doc.Spec, renderedDoc.Spec)
	})

	t.Run("should render YAML document with custom file tags", func(t *testing.T) {
		yamlDoc :=
			`apiVersion: xl-deploy/v1alpha1
kind: Applications
spec:
- name: PetClinic-ear
  type: jee.Ear
  file: !file PetClinic-1.0.ear
`

		doc, err := ParseYamlDocument(yamlDoc)

		assert.Nil(t, err)
		assert.NotNil(t, doc)

		renderedYamlDocBytes, err := doc.RenderYamlDocument()
		assert.Nil(t, err)
		assert.NotNil(t, renderedYamlDocBytes)
		renderedDoc, err := ParseYamlDocument(string(renderedYamlDocBytes))
		assert.Equal(t, doc.ApiVersion, renderedDoc.ApiVersion)
		assert.Equal(t, doc.Kind, renderedDoc.Kind)
		assert.Equal(t, doc.Metadata, renderedDoc.Metadata)
		assert.Equal(t, doc.Spec, renderedDoc.Spec)
	})

}
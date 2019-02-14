package xl

import (
	"archive/zip"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/xebialabs/xl-cli/pkg/util"
	"github.com/xebialabs/yaml"
)

func writeFile(path string, content []byte) {
	ioutil.WriteFile(path, content, 0644)
}

func writeTempFile(content []byte) string {
	location, _ := ioutil.TempFile("", "tmpFile")
	writeFile(location.Name(), content)
	return location.Name()
}

func prepareArtifactsDir(t *testing.T, dir string, folderName string, fileContents map[string]string) string {
	artifactsDir, err := ioutil.TempDir(dir, folderName)
	if err != nil {
		assert.FailNow(t, "cannot open temporary directory", err)
	}

	for fileName, fileContent := range fileContents {
		writeFile(filepath.Join(artifactsDir, fileName), []byte(fileContent))
	}
	return artifactsDir
}

func readZipContent(t *testing.T, doc *Document, path string) map[string][]byte {
	zipR, err := zip.OpenReader(path)
	if err != nil {
		assert.FailNow(t, "cannot open generated ZIP file [%s]: %s", doc.ApplyZip, err)
	}

	fileContents := make(map[string][]byte)
	for _, f := range zipR.File {
		fr, err := f.Open()
		if err != nil {
			assert.FailNow(t, "cannot open entry [%s] in generated ZIP file [%s]: %s", f.Name, doc.ApplyZip, err)
		}
		contents, err := ioutil.ReadAll(fr)
		if err != nil {
			assert.FailNow(t, "cannot read entry [%s] in generated ZIP file [%s]: %s", f.Name, doc.ApplyZip, err)
		}
		fileContents[f.Name] = contents
	}
	return fileContents
}

func TestDocument(t *testing.T) {
	t.Run("should parse YAML document", func(t *testing.T) {
		yamlDoc := fmt.Sprintf(`apiVersion: %s
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
`, XldApiVersion)

		doc, err := ParseYamlDocument(yamlDoc)

		assert.Nil(t, err)
		assert.NotNil(t, doc)
		assert.Equal(t, XldApiVersion, doc.ApiVersion)
		assert.Equal(t, "Applications", doc.Kind)
		assert.NotNil(t, doc.Metadata)
		assert.Equal(t, "Applications/Cloud", doc.Metadata["Applications-home"])
		assert.NotNil(t, doc.Spec)
		spec := util.TransformToMap(doc.Spec)
		assert.Equal(t, "Applications/AWS", spec[0]["name"])
		children := spec[0]["children"].([]interface{})
		firstChild := children[0].(map[interface{}]interface{})
		assert.Equal(t, "rest-o-rant-ecs-service", firstChild["name"])
		children2 := firstChild["children"].([]interface{})
		firstChild2 := children2[0].(map[interface{}]interface{})
		assert.Equal(t, "udm.DeploymentPackage", firstChild2["type"])
	})

	t.Run("should parse YAML file with multiple documents", func(t *testing.T) {
		yamlDocs := fmt.Sprintf(`apiVersion: %s
kind: Applications
spec:
- name: Applications/AWS1
---
apiVersion: %s
kind: Template
spec:
- name: Template1
---
apiVersion: %s
kind: Template
spec:
- name: Template2
`, XldApiVersion, XlrApiVersion, XlrApiVersion)
		docreader := NewDocumentReader(strings.NewReader(yamlDocs))
		doc, err := docreader.ReadNextYamlDocument()

		assert.Nil(t, err)
		assert.NotNil(t, doc)
		assert.Equal(t, XldApiVersion, doc.ApiVersion)
		assert.Equal(t, "Applications", doc.Kind)
		assert.NotNil(t, doc.Spec)
		spec := util.TransformToMap(doc.Spec)
		assert.Equal(t, "Applications/AWS1", spec[0]["name"])

		doc, err = docreader.ReadNextYamlDocument()

		assert.Nil(t, err)
		assert.NotNil(t, doc)
		assert.Equal(t, XlrApiVersion, doc.ApiVersion)
		assert.Equal(t, "Template", doc.Kind)
		assert.NotNil(t, doc.Spec)
		spec = util.TransformToMap(doc.Spec)
		assert.Equal(t, "Template1", spec[0]["name"])

		doc, err = docreader.ReadNextYamlDocument()

		assert.Nil(t, err)
		assert.NotNil(t, doc)
		assert.NotNil(t, doc.Spec)
		spec = util.TransformToMap(doc.Spec)
		assert.Equal(t, "Template2", spec[0]["name"])
	})

	t.Run("should parse YAML documents with custom tags", func(t *testing.T) {
		yamlDoc := fmt.Sprintf(`apiVersion: %s
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
apiVersion: %s
kind: Intrastructure
spec:
- name: server
  type: overthere.SshHost
  address: server.example.com
  username: root
  password: !value server.root.password
---
apiVersion: %s
kind: Intrastructure
spec:
- name: server
  type: overthere.SshHost
  address: !format '%%server_url%%%%%%35%%%%'
  username: root
  password: pass
`, XldApiVersion, XldApiVersion, XldApiVersion)
		docreader := NewDocumentReader(strings.NewReader(yamlDoc))
		doc1, err1 := docreader.ReadNextYamlDocument()
		var v1 interface{}

		assert.Nil(t, err1)
		assert.NotNil(t, doc1)
		spec1 := util.TransformToMap(doc1.Spec)
		assert.Equal(t, "PetClinic-ear", spec1[0]["name"])
		v1 = spec1[0]["children"]                          // spec[0].children
		v2 := v1.([]interface{})[0]                        // spec[0].children[0]
		v3 := v2.(map[interface{}]interface{})["children"] // spec[0].children[0].children
		v4 := v3.([]interface{})[0]                        // spec[0].children[0].children[0]
		v5 := v4.(map[interface{}]interface{})["file"]     // spec[0].children[0].children[0].file
		assert.Equal(t, yaml.CustomTag{"!file", "PetClinic-1.0.ear"}, v5)

		doc2, err2 := docreader.ReadNextYamlDocument()

		assert.Nil(t, err2)
		assert.NotNil(t, doc2)
		spec2 := util.TransformToMap(doc2.Spec)
		assert.Equal(t, "server", spec2[0]["name"])
		assert.Equal(t, yaml.CustomTag{"!value", "server.root.password"}, spec2[0]["password"])

		doc3, err3 := docreader.ReadNextYamlDocument()

		assert.Nil(t, err3)
		assert.NotNil(t, doc3)
		spec3 := util.TransformToMap(doc3.Spec)
		fmt.Println(fmt.Sprintf("%s", doc3.Spec))
		assert.Equal(t, yaml.CustomTag{"!format", "%server_url%%%35%%"}, spec3[0]["address"])
	})

	t.Run("should add xl-deploy homes if missing in yaml document", func(t *testing.T) {
		yamlDoc := fmt.Sprintf(`apiVersion: %s
kind: Applications`, XldApiVersion)
		doc, err := ParseYamlDocument(yamlDoc)

		assert.Nil(t, err)
		assert.NotNil(t, doc)

		context := &Context{XLDeploy: &XLDeployServer{Server: &DummyHTTPServer{},
			ApplicationsHome:   "Applications/MyHome",
			EnvironmentsHome:   "Environments/MyHome",
			ConfigurationHome:  "Configuration/MyHome",
			InfrastructureHome: "Infrastructure/MyHome"},
			XLRelease: &XLReleaseServer{Server: &DummyHTTPServer{}}}
		err = doc.Preprocess(context, "")
		defer doc.Cleanup()

		assert.Nil(t, err)
		assert.NotNil(t, doc)
		assert.Equal(t, XldApiVersion, doc.ApiVersion)
		assert.Equal(t, "Applications", doc.Kind)
		assert.Equal(t, "Applications/MyHome", doc.Metadata["Applications-home"])
		assert.Equal(t, "Environments/MyHome", doc.Metadata["Environments-home"])
		assert.Equal(t, "Configuration/MyHome", doc.Metadata["Configuration-home"])
		assert.Equal(t, "Infrastructure/MyHome", doc.Metadata["Infrastructure-home"])
	})

	t.Run("should add xl-release home if missing in yaml document", func(t *testing.T) {
		yaml := fmt.Sprintf(`apiVersion: %s
kind: Templates`, XlrApiVersion)
		doc, err := ParseYamlDocument(yaml)

		assert.Nil(t, err)
		assert.NotNil(t, doc)

		context := &Context{XLDeploy: &XLDeployServer{Server: &DummyHTTPServer{}}, XLRelease: &XLReleaseServer{Server: &DummyHTTPServer{}, Home: "MyHome"}}
		err = doc.Preprocess(context, "")
		defer doc.Cleanup()

		assert.Nil(t, err)
		assert.NotNil(t, doc)
		assert.Equal(t, XlrApiVersion, doc.ApiVersion)
		assert.Equal(t, "MyHome", doc.Metadata["home"])
	})

	t.Run("should not replace homes if included in yaml document", func(t *testing.T) {
		yaml := fmt.Sprintf(`apiVersion: %s
kind: Applications
metadata:
  Applications-home: Applications/DoNotTouch
  Environments-home: Environments/DoNotTouch
  Configuration-home: Configuration/DoNotTouch
  Infrastructure-home: Infrastructure/DoNotTouch`, XldApiVersion)
		doc, err := ParseYamlDocument(yaml)

		assert.Nil(t, err)
		assert.NotNil(t, doc)

		context := &Context{XLDeploy: &XLDeployServer{Server: &DummyHTTPServer{},
			ApplicationsHome:   "Applications/MyHome",
			EnvironmentsHome:   "Environments/MyHome",
			ConfigurationHome:  "Configuration/MyHome",
			InfrastructureHome: "Infrastructure/MyHome"},
			XLRelease: &XLReleaseServer{Server: &DummyHTTPServer{}}}
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
		yaml := fmt.Sprintf(`apiVersion: %s
kind: Applications`, XldApiVersion)

		doc, err := ParseYamlDocument(yaml)

		assert.Nil(t, err)
		assert.NotNil(t, doc)

		context := &Context{XLDeploy: &XLDeployServer{Server: &DummyHTTPServer{},
			ApplicationsHome:   "",
			EnvironmentsHome:   "",
			ConfigurationHome:  "",
			InfrastructureHome: ""},
			XLRelease: &XLReleaseServer{Server: &DummyHTTPServer{}}}
		err = doc.Preprocess(context, "")
		defer doc.Cleanup()

		assert.Nil(t, err)
		assert.NotNil(t, doc)
		assert.Nil(t, doc.Metadata["Applications-home"])
		assert.Nil(t, doc.Metadata["Environments-home"])
		assert.Nil(t, doc.Metadata["Configuration-home"])
		assert.Nil(t, doc.Metadata["Infrastructure-home"])
	})

	t.Run("should process !value tag", func(t *testing.T) {
		yamlDoc := fmt.Sprintf(`apiVersion: %s
kind: Infrastructure
spec:
- name: serverHost
  type: overthere.SshHost
  address: !value server.host
  username: root
  password: r00t`, XldApiVersion)

		doc, err := ParseYamlDocument(yamlDoc)

		require.Nil(t, err)
		require.NotNil(t, doc)

		values := map[string]string{
			"server.host": "server.example.com",
		}
		context := &Context{values: values}

		err = doc.Preprocess(context, "")

		assert.Nil(t, err)
		assert.Equal(t, "server.example.com", util.TransformToMap(doc.Spec)[0]["address"])
	})

	t.Run("should report error when !value tag refers to unknown value", func(t *testing.T) {
		yamlDoc := fmt.Sprintf(`apiVersion: %s
kind: Infrastructure
spec:
- name: serverHost
  type: overthere.SshHost
  address: server.example.com
  username: !value server.username
  password: r00t`, XldApiVersion)

		doc, err := ParseYamlDocument(yamlDoc)

		require.Nil(t, err)
		require.NotNil(t, doc)

		values := map[string]string{
			"server.host": "server.example.com",
		}
		context := &Context{values: values}

		err = doc.Preprocess(context, "")

		assert.NotNil(t, err)
		assert.Contains(t, err.Error(), "No value")
	})

	t.Run("should process !format tag", func(t *testing.T) {
		yamlDoc := fmt.Sprintf(`apiVersion: %s
kind: Infrastructure
spec:`, XldApiVersion) + `
- name: serverHost
  type: overthere.SshHost
  address: server.example.com
  username: !format '%server_url%%%35%%'
  password: r00t`

		doc, err := ParseYamlDocument(yamlDoc)

		require.Nil(t, err)
		require.NotNil(t, doc)

		values := map[string]string{
			"server_url": "theuselessweb.com/",
		}
		context := &Context{values: values}

		err = doc.Preprocess(context, "")

		assert.Nil(t, err)
		assert.Equal(t, "theuselessweb.com/%35%", util.TransformToMap(doc.Spec)[0]["username"])
	})

	t.Run("should process !format tag", func(t *testing.T) {
		yamlDoc := fmt.Sprintf(`apiVersion: %s
kind: Infrastructure
spec:`, XldApiVersion) + `
- name: serverHost
  url: !format '%server_url%%%35%%'
`
		doc, err := ParseYamlDocument(yamlDoc)

		require.Nil(t, err)
		require.NotNil(t, doc)

		values := map[string]string{"server_url": "testhost"}
		context := &Context{values: values}

		err = doc.Preprocess(context, "")

		assert.Nil(t, err)
		assert.Equal(t, "testhost%35%", util.TransformToMap(doc.Spec)[0]["url"])
	})

	t.Run("should report error when !format tag have a reference to unknown value", func(t *testing.T) {
		yamlDoc := fmt.Sprintf(`apiVersion: %s
kind: Infrastructure
spec:`, XldApiVersion) + `
- name: serverHost
  url: !format '%server_url%%%35%%'
`
		doc, err := ParseYamlDocument(yamlDoc)

		require.Nil(t, err)
		require.NotNil(t, doc)

		values := map[string]string{}
		context := &Context{values: values}

		err = doc.Preprocess(context, "")

		assert.NotNil(t, err)
		assert.Contains(t, err.Error(), "unknown value: `server_url`")
	})

	t.Run("should process !file tag", func(t *testing.T) {
		artifactContents := "cats=5\ndogs=8\n"
		artifactsDir := prepareArtifactsDir(t, "", "should_process_file_tags", map[string]string{
			"petclinic.properties": artifactContents,
		})
		defer os.RemoveAll(artifactsDir)
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
		doc, err := NewDocumentReader(strings.NewReader(yamlDoc)).ReadNextYamlDocument()

		assert.Nil(t, err)
		assert.NotNil(t, doc)

		errp := doc.Preprocess(nil, artifactsDir)
		defer doc.Cleanup()

		assert.Nil(t, errp)
		assert.NotNil(t, doc.ApplyZip)
		fileContents := readZipContent(t, doc, doc.ApplyZip)
		assert.Contains(t, fileContents, "index.yaml")
		indexDocument, err := ParseYamlDocument(string(fileContents["index.yaml"]))
		indexDocumentSpec := util.TransformToMap(indexDocument.Spec)
		Applications_PetClinic_1_0_conf_file := indexDocumentSpec[0]["children"].([]interface{})[0].(map[interface{}]interface{})["children"].([]interface{})[0].(map[interface{}]interface{})["file"].(yaml.CustomTag)
		assert.Contains(t, fileContents, Applications_PetClinic_1_0_conf_file.Value)
		assert.Equal(t, artifactContents, string(fileContents[Applications_PetClinic_1_0_conf_file.Value]))
	})

	t.Run("should report error when !file tag contains absolute path", func(t *testing.T) {
		artifactsDir, err := ioutil.TempDir("", "should_process_file_tags")
		if err != nil {
			assert.FailNow(t, "cannot open temporary directory", err)
		}
		defer os.RemoveAll(artifactsDir)

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
      file: !file /etc/passwd`, XldApiVersion)

		doc, err := ParseYamlDocument(yamlDoc)

		assert.Nil(t, err)
		assert.NotNil(t, doc)

		err = doc.Preprocess(nil, artifactsDir)
		assert.NotNil(t, err)
		assert.Contains(t, err.Error(), "absolute path")
	})

	t.Run("should process !file tag with relative path", func(t *testing.T) {
		artifactContents := "cats=5\ndogs=8\n"
		artifactsDir, err := ioutil.TempDir("", "should_process_file_tags")
		if err != nil {
			assert.FailNow(t, "cannot open temporary directory", err)
		}
		defer os.RemoveAll(artifactsDir)

		artifactsSubDir, err := ioutil.TempDir(artifactsDir, "subDir")
		if err != nil {
			assert.FailNow(t, "cannot open temporary directory", err)
		}

		writeFile(filepath.Join(artifactsDir, "test.txt"), []byte(artifactContents))

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
      file: !file ../test.txt`, XldApiVersion)
		doc, err := NewDocumentReader(strings.NewReader(yamlDoc)).ReadNextYamlDocument()

		assert.Nil(t, err)
		assert.NotNil(t, doc)

		errp := doc.Preprocess(nil, artifactsSubDir)
		defer doc.Cleanup()

		assert.Nil(t, errp)
		assert.NotNil(t, doc.ApplyZip)
		fileContents := readZipContent(t, doc, doc.ApplyZip)
		assert.Contains(t, fileContents, "index.yaml")
		indexDocument, err := ParseYamlDocument(string(fileContents["index.yaml"]))
		indexDocumentSpec := util.TransformToMap(indexDocument.Spec)
		Applications_PetClinic_1_0_conf_file := indexDocumentSpec[0]["children"].([]interface{})[0].(map[interface{}]interface{})["children"].([]interface{})[0].(map[interface{}]interface{})["file"].(yaml.CustomTag)
		assert.Contains(t, fileContents, Applications_PetClinic_1_0_conf_file.Value)
		assert.Equal(t, artifactContents, string(fileContents[Applications_PetClinic_1_0_conf_file.Value]))
	})

	t.Run("should render YAML document", func(t *testing.T) {
		yamlDoc := fmt.Sprintf(`apiVersion: %s
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
`, XldApiVersion)
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
		yamlDoc := fmt.Sprintf(`apiVersion: %s
kind: Applications`, XldApiVersion)

		doc, err := ParseYamlDocument(yamlDoc)

		assert.Nil(t, err)
		assert.NotNil(t, doc)

		renderedYaml, err := doc.RenderYamlDocument()
		assert.Nil(t, err)
		assert.NotContains(t, string(renderedYaml), "metadata")
	})

	t.Run("should not render spec if empty", func(t *testing.T) {
		yamlDoc := fmt.Sprintf(`apiVersion: %s
kind: Applications`, XldApiVersion)
		doc, err := ParseYamlDocument(yamlDoc)

		assert.Nil(t, err)
		assert.NotNil(t, doc)

		renderedYaml, err := doc.RenderYamlDocument()
		assert.Nil(t, err)
		assert.NotContains(t, string(renderedYaml), "spec")
	})

	t.Run("should render YAML document with empty metadata", func(t *testing.T) {
		yamlDoc := fmt.Sprintf(`apiVersion: %s
kind: Infrastructure
spec:
- name: Localhost
  type: overthere.LocalHost`, XldApiVersion)
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
			fmt.Sprintf(`apiVersion: %s
kind: Applications
spec:
- name: PetClinic-ear
  type: jee.Ear
  file: !file PetClinic-1.0.ear
`, XldApiVersion)
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

	t.Run("should support folders in !file tags", func(t *testing.T) {
		baseDir := prepareArtifactsDir(t, "", "should_process_folders_in_file_tags", map[string]string{})
		artifactsDir := prepareArtifactsDir(t, baseDir, "should_process_folders_in_file_tags", map[string]string{
			"users.conf":  "admin\njohn\n",
			"system.conf": "autoShutdown: false",
		})
		defer os.RemoveAll(baseDir)
		folderDirZip := filepath.Base(artifactsDir)

		yamlDoc := fmt.Sprintf(`apiVersion: %s
kind: Applications
spec:
- name: 
  type: file.Folder
  file: !file %s`, XldApiVersion, folderDirZip)
		doc, err := NewDocumentReader(strings.NewReader(yamlDoc)).ReadNextYamlDocument()

		assert.Nil(t, err)
		assert.NotNil(t, doc)

		errp := doc.Preprocess(nil, baseDir)
		defer doc.Cleanup()

		assert.Nil(t, errp)
		assert.NotNil(t, doc.ApplyZip)

		fileContents := readZipContent(t, doc, doc.ApplyZip)
		assert.Contains(t, fileContents, "index.yaml")
		fileName := "1/" + folderDirZip
		assert.Contains(t, fileContents, fileName)

		internalFolderZip := writeTempFile(fileContents[fileName])
		defer os.Remove(internalFolderZip)

		internalFolderZipContents := readZipContent(t, doc, internalFolderZip)
		assert.Contains(t, internalFolderZipContents, "users.conf")
		assert.Contains(t, internalFolderZipContents, "system.conf")
		assert.Equal(t, string(internalFolderZipContents["users.conf"]), "admin\njohn\n")
		assert.Equal(t, string(internalFolderZipContents["system.conf"]), "autoShutdown: false")
	})

}

package cmd

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
	"github.com/xebialabs/xl-cli/pkg/models"
	"github.com/xebialabs/xl-cli/pkg/util"
	"github.com/xebialabs/xl-cli/pkg/xl"
)

var TestCmd = &cobra.Command{
	Use:   "xl",
	Short: "Test command",
}

type TestInfra struct {
	documents []xl.Document
	xldServer *httptest.Server
	xlrServer *httptest.Server
}

func (infra *TestInfra) appendDoc(document xl.Document) {
	infra.documents = append(infra.documents, document)
}

func (infra *TestInfra) shutdown() {
	infra.xldServer.Close()
	infra.xlrServer.Close()
}

func (infra *TestInfra) doc(index int) xl.Document {
	return infra.documents[index]
}

func (infra *TestInfra) spec(index int) []map[interface{}]interface{} {
	return xl.TransformToMap(infra.doc(index).Spec)
}

func (infra *TestInfra) metadata(index int) map[interface{}]interface{} {
	return infra.doc(index).Metadata
}

func CreateTestInfra(viper *viper.Viper) *TestInfra {
	infra := &TestInfra{documents: make([]xl.Document, 0)}

	xldHandler := func(responseWriter http.ResponseWriter, request *http.Request) {
		body, err := ioutil.ReadAll(request.Body)
		check(err)
		doc, err := xl.ParseYamlDocument(string(body))
		check(err)
		infra.appendDoc(*doc)
		_, _ = responseWriter.Write([]byte("{}"))
	}

	xlrHandler := func(responseWriter http.ResponseWriter, request *http.Request) {
		body, err := ioutil.ReadAll(request.Body)
		check(err)
		doc, err := xl.ParseYamlDocument(string(body))
		check(err)
		infra.appendDoc(*doc)
		_, _ = responseWriter.Write([]byte("{}"))
	}

	infra.xldServer = httptest.NewServer(http.HandlerFunc(xldHandler))
	infra.xlrServer = httptest.NewServer(http.HandlerFunc(xlrHandler))

	viper.Set("xl-deploy.url", infra.xldServer.URL)
	viper.Set("xl-release.url", infra.xlrServer.URL)
	return infra
}

func TestApply(t *testing.T) {

	util.IsVerbose = true

	t.Run("should apply multiple yaml files in right order with value replacement to both xlr and xld", func(t *testing.T) {
		tempDir1 := createTempDir("firstDir")
		writeToFile(filepath.Join(tempDir1, "prop1.xlvals"), "replaceme=success1")
		yaml := writeToTempFile(tempDir1, "yaml1", fmt.Sprintf(`
apiVersion: %s
kind: Template
spec:
- name: Template1
- replaceTest: !value replaceme

---

apiVersion: %s
kind: Applications
spec:
- name: App1
`, xl.XlrApiVersion, xl.XldApiVersion))
		defer os.RemoveAll(tempDir1)

		tempDir2 := createTempDir("secondDir")
		writeToFile(filepath.Join(tempDir2, "prop2.xlvals"), "replaceme=success2\noverrideme=notoverriden")
		writeToFile(filepath.Join(tempDir2, "prop3.xlvals"), "overrideme=OVERRIDDEN")
		yaml2 := writeToTempFile(tempDir2, "yaml2", fmt.Sprintf(`
apiVersion: %s
kind: Template
spec:
- name: Template2
- replaceTest: !value replaceme
- overrideTest: !value overrideme
---

apiVersion: %s
kind: Applications
spec:
- name: App2
`, xl.XlrApiVersion, xl.XldApiVersion))
		defer os.RemoveAll(tempDir2)

		v := viper.GetViper()
		v.Set("xl-deploy.applications-home", "Applications/XL")
		v.Set("xl-release.home", "XL")

		infra := CreateTestInfra(v)
		defer infra.shutdown()

		DoApply(TestCmd, []string{yaml.Name(), yaml2.Name()})

		assert.Equal(t, infra.spec(0)[0]["name"], "Template1")
		assert.Equal(t, infra.spec(0)[1]["replaceTest"], "success1")
		assert.Equal(t, infra.metadata(0)["home"], "XL")

		assert.Equal(t, infra.spec(1)[0]["name"], "App1")
		assert.Equal(t, infra.metadata(1)["Applications-home"], "Applications/XL")

		assert.Equal(t, infra.spec(2)[0]["name"], "Template2")
		assert.Equal(t, infra.spec(2)[1]["replaceTest"], "success2")
		assert.Equal(t, infra.spec(2)[2]["overrideTest"], "OVERRIDDEN")
		assert.Equal(t, infra.metadata(2)["home"], "XL")

		assert.Equal(t, infra.spec(3)[0]["name"], "App2")
		assert.Equal(t, infra.metadata(3)["Applications-home"], "Applications/XL")
	})

	t.Run("should take imports into account", func(t *testing.T) {
		tempDir := createTempDir("imports")
		provisionFile := writeToTempFile(tempDir, "provision.yaml", fmt.Sprintf(`
apiVersion: %s
kind: Applications
spec:
- name: App1
`, xl.XldApiVersion))

		deployFile := writeToTempFile(tempDir, "deploy.yaml", fmt.Sprintf(`
apiVersion: %s
kind: Deployment
metadata:
  imports:
    - %s
spec:
  package: Applications/PetPortal/1.0
  environment: Environments/AWS Environment
  undeployDependencies: true
  orchestrators:
  - parallel-by-deployment-group
  - sequential-by-container
`, xl.XldApiVersion, filepath.Base(provisionFile.Name())))
		defer os.RemoveAll(tempDir)

		v := viper.GetViper()
		infra := CreateTestInfra(v)
		defer infra.shutdown()

		DoApply(TestCmd, []string{deployFile.Name()})

		assert.Equal(t, len(infra.documents), 2)
		assert.Equal(t, infra.doc(0).Kind, "Applications")
		assert.Equal(t, infra.doc(1).Kind, "Deployment")
		assert.Nil(t, infra.metadata(1)["imports"])
	})

	t.Run("should support 'imports' file together with imports inside of imported files", func(t *testing.T) {
		tempDir := createTempDir("imports2")
		provisionFile := writeToTempFile(tempDir, "provision.yaml", fmt.Sprintf(`
apiVersion: %s
kind: Applications
spec:
- name: App1
`, xl.XldApiVersion))

		deployFile := writeToTempFile(tempDir, "deploy.yaml", fmt.Sprintf(`
apiVersion: %s
kind: Deployment
metadata:
  imports:
    - %s
spec:
  package: Applications/PetPortal/1.0
  environment: Environments/AWS Environment
  undeployDependencies: true
  orchestrators:
  - parallel-by-deployment-group
  - sequential-by-container
`, xl.XldApiVersion, filepath.Base(provisionFile.Name())))

		importsFile := writeToTempFile(tempDir, "imports.yaml", fmt.Sprintf(`
apiVersion: %s
kind: Import
metadata:
  imports:
    - %s
`, models.YamlFormatVersion, filepath.Base(deployFile.Name())))
		defer os.RemoveAll(tempDir)

		v := viper.GetViper()
		infra := CreateTestInfra(v)
		defer infra.shutdown()

		DoApply(TestCmd, []string{importsFile.Name()})

		assert.Equal(t, len(infra.documents), 2)
		assert.Equal(t, infra.doc(0).Kind, "Applications")
		assert.Equal(t, infra.doc(1).Kind, "Deployment")
		assert.Nil(t, infra.metadata(1)["imports"])
	})

	t.Run("should list xlvals files from relative directory in alphabetical order", func(t *testing.T) {
		dir, err := ioutil.TempDir("", "")
		defer os.Remove(dir)
		check(err)

		_, err1 := os.Create(filepath.Join(dir, "b.xlvals"))
		check(err1)

		_, err2 := os.Create(filepath.Join(dir, "c.xlvals"))
		check(err2)

		_, err3 := os.Create(filepath.Join(dir, "a.xlvals"))
		check(err3)

		subdir, err := ioutil.TempDir(dir, "")
		check(err)

		_, err4 := os.Create(filepath.Join(subdir, "d.xlvals"))
		check(err4)

		files, err := listRelativeXlValsFiles(dir)
		check(err)

		assert.Len(t, files, 3)
		assert.Equal(t, filepath.Base(files[0]), "a.xlvals")
		assert.Equal(t, filepath.Base(files[1]), "b.xlvals")
		assert.Equal(t, filepath.Base(files[2]), "c.xlvals")
		assert.NotContains(t, files, "d.xlvals")
	})
}

func createTempDir(name string) string {
	dir, err := ioutil.TempDir("", name)
	check(err)
	return dir
}

func writeToTempFile(tempDir string, fileName string, value string) *os.File {
	file, err := ioutil.TempFile(tempDir, fileName)
	check(err)
	return writeTo(file, value)
}

func writeToFile(filePath string, value string) *os.File {
	file, err := os.Create(filePath)
	check(err)
	return writeTo(file, value)
}

func writeTo(file *os.File, value string) *os.File {
	defer file.Close()
	_, err2 := file.WriteString(value)
	file.Sync()
	check(err2)
	return file
}

func check(e error) {
	if e != nil {
		panic(e)
	}
}

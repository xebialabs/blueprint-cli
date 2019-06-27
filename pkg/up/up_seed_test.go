package up

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/xebialabs/xl-cli/pkg/xl"

	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
	"github.com/xebialabs/xl-cli/pkg/blueprint"
	"github.com/xebialabs/xl-cli/pkg/models"
	"github.com/xebialabs/xl-cli/pkg/util"
	"github.com/xebialabs/yaml"
)

const blueprintDir = Xebialabs

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
	return util.TransformToMap(infra.doc(index).Spec)
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
	viper.Set("xl-deploy.applications-home", "Applications/XL")
	viper.Set("xl-release.home", "XL")
	viper.Set(models.ViperKeyXLDUsername, "deployer")
	viper.Set(models.ViperKeyXLDPassword, "d3ploy1t")
	viper.Set(models.ViperKeyXLRUsername, "releaser")
	viper.Set(models.ViperKeyXLRPassword, "r3l34s3")

	return infra
}

func TestFakeApplyFiles(t *testing.T) {
	t.Run("should not change the file tag", func(t *testing.T) {

		err := os.Mkdir(blueprintDir, os.ModePerm)
		defer os.RemoveAll(blueprintDir)
		check(err)

		xlvFile, err := os.Create(filepath.Join(blueprintDir, "prop1.xlvals"))
		check(err)
		xlvFile.WriteString("replaceme=success1")

		yFile, err := os.Create(filepath.Join(blueprintDir, "yaml1.yaml"))
		check(err)
		yFile.WriteString(fmt.Sprintf(`
apiVersion: %s
kind: Template
spec:
- name: Template1
- replaceTest: !value replaceme
- file: !file "../path/to/the/file/"

---

apiVersion: %s
kind: Applications
spec:
- name: App1
`, xl.XlrApiVersion, xl.XldApiVersion))
		blueprint.WriteConfigFile = false
		v, _, _, _ := blueprint.GetDefaultBlueprintViperConfig(viper.GetViper(), "")
		infra := CreateTestInfra(v)
		defer infra.shutdown()

		applyFilesAndSave()

		fileContents, err := ioutil.ReadFile(filepath.Join(blueprintDir, "yaml1.yaml"))
		check(err)

		doc, _ := xl.ParseYamlDocument(string(fileContents))
		infra.appendDoc(*doc)

		assert.Equal(t, infra.spec(0)[1]["replaceTest"], "success1")
		assert.Equal(t, infra.spec(0)[2]["file"], yaml.CustomTag(yaml.CustomTag{Tag: "!file", Value: "../path/to/the/file/"}))
	})
}

func check(e error) {
	if e != nil {
		panic(e)
	}
}

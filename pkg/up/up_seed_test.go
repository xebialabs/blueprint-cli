package up

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"os"
	"path"
	"path/filepath"
	"strings"
	"testing"

	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/xebialabs/xl-cli/pkg/blueprint"
	"github.com/xebialabs/xl-cli/pkg/models"
	"github.com/xebialabs/xl-cli/pkg/util"
	"github.com/xebialabs/xl-cli/pkg/xl"
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

		if !util.PathExists(blueprintDir, true) {
			defer os.RemoveAll(blueprintDir)
			err := os.Mkdir(blueprintDir, os.ModePerm)
			check(err)
		}

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

func GetViperConf(t *testing.T, yaml string) *viper.Viper {
	configdir, err := ioutil.TempDir("", "xebialabsconfig")
	if err != nil {
		t.Error(err)
	}
	defer os.RemoveAll(configdir)
	configfile := filepath.Join(configdir, "config.yaml")
	originalConfigBytes := []byte(yaml)
	ioutil.WriteFile(configfile, originalConfigBytes, 0755)

	v := viper.New()
	v.SetConfigFile(configfile)
	v.ReadInConfig()
	return v
}

var BlueprintTestPath = ""

func getLocalTestBlueprintContext(t *testing.T) *blueprint.BlueprintContext {
	configdir, _ := ioutil.TempDir("", "xebialabsconfig")
	configfile := filepath.Join(configdir, "config.yaml")
	v := GetViperConf(t, "")
	c, err := blueprint.ConstructBlueprintContext(v, configfile, "9.0.0")
	if err != nil {
		t.Error(err)
	}
	return c
}

func GetTestTemplateDir(blueprint string) string {
	pwd, _ := os.Getwd()
	return strings.Replace(pwd, path.Join("pkg", "up"), path.Join("templates", "test", blueprint), -1)
}

func TestInvokeBlueprintAndSeed(t *testing.T) {
	SkipSeed = true
	SkipKube = true
	SkipPrompts = true
	blueprint.SkipFinalPrompt = true

	t.Run("should create output files for valid xl-up template with answers file", func(t *testing.T) {
		// This can be used to debug a local blueprint if you have the repo in ../xl-up-blueprint relative to xl-cli
		// pwd, _ := os.Getwd()
		// BlueprintTestPath = strings.Replace(pwd, path.Join("xl-cli", "pkg", "up"), path.Join("xl-up-blueprint"), -1)
		gb, err := InvokeBlueprintAndSeed(
			getLocalTestBlueprintContext(t),
			UpParams{
				LocalPath:         "../../../xl-up-blueprint",
				QuickSetup:        true,
				AdvancedSetup:     false,
				BlueprintTemplate: "xl-infra",
				CfgOverridden:     false,
				AnswerFile:        GetTestTemplateDir("answer-xl-up.yaml"),
				NoCleanup:         false,
				Destroy:           false,
			},
			"",
		)
		defer gb.Cleanup()

		require.Nil(t, err)

		// assertions
		// assert.FileExists(t, "xld-environment.yml")
		// assert.FileExists(t, "xld-infrastructure.yml")
		// assert.FileExists(t, "xlr-pipeline.yml")
		// assert.FileExists(t, path.Join(gb.OutputDir, valuesFile))
		// assert.FileExists(t, path.Join(gb.OutputDir, secretsFile))
		// assert.FileExists(t, path.Join(gb.OutputDir, gitignoreFile))

		// // check __test__ directory is not there
		// _, err = os.Stat("__test__")
		// assert.True(t, os.IsNotExist(err))

		// // check encoded string value in env template
		// envTemplateFile := GetFileContent("xld-environment.yml")
		// assert.Contains(t, envTemplateFile, fmt.Sprintf("accessSecret: %s", b64.StdEncoding.EncodeToString([]byte("accesssecret"))))

		// // check values file
		// valsFile := GetFileContent(path.Join(gb.OutputDir, valuesFile))
		// valueMap := map[string]string{
		// 	"Test":               "testing",
		// 	"ClientCert":         "FshYmQzRUNbYTA4Icc3V7JEgLXMNjcSLY9L1H4XQD79coMBRbbJFtOsp0Yk2btCKCAYLio0S8Jw85W5mgpLkasvCrXO5\\nQJGxFvtQc2tHGLj0kNzM9KyAqbUJRe1l40TqfMdscEaWJimtd4oygqVc6y7zW1Wuj1EcDUvMD8qK8FEWfQgm5ilBIldQ\\n",
		// 	"AppName":            "TestApp",
		// 	"SuperSecret":        "invisible",
		// 	"AWSRegion":          "eu-central-1",
		// 	"DiskSize":           "100",
		// 	"DiskSizeWithBuffer": "125.1",
		// 	"ShouldNotBeThere":   "",
		// }
		// for k, v := range valueMap {
		// 	assert.Contains(t, valsFile, fmt.Sprintf("%s = %s", k, v))
		// }

		// // check secrets file
		// secretsFile := GetFileContent(path.Join(gb.OutputDir, secretsFile))
		// secretsMap := map[string]string{
		// 	"AWSAccessKey":    "accesskey",
		// 	"AWSAccessSecret": "accesssecret",
		// }
		// for k, v := range secretsMap {
		// 	assert.Contains(t, secretsFile, fmt.Sprintf("%s = %s", k, v))
		// }
	})
}

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

		defer os.RemoveAll(blueprintDir)
		if !util.PathExists(blueprintDir, true) {
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

func getLocalTestBlueprintContext(t *testing.T) *blueprint.BlueprintContext {
	configdir, _ := ioutil.TempDir("", "xebialabsconfig")
	configfile := filepath.Join(configdir, "config.yaml")
	v := viper.New()
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

func GetFileContent(filePath string) string {
	f, err := ioutil.ReadFile(filePath)
	if err != nil {
		panic(err)
	}
	return string(f)
}

func TestInvokeBlueprintAndSeed(t *testing.T) {
	SkipSeed = true
	SkipKube = true
	SkipPrompts = true
	blueprint.SkipFinalPrompt = true
	blueprint.SkipUpFinalPrompt = true

	t.Run("should create output files for valid xl-up template with answers file", func(t *testing.T) {
		gb := &blueprint.GeneratedBlueprint{OutputDir: models.BlueprintOutputDir}
		defer gb.Cleanup()
		err := InvokeBlueprintAndSeed(
			getLocalTestBlueprintContext(t),
			UpParams{
				// enable for local testing
				// LocalPath:         "../../../xl-up-blueprint",
				BlueprintTemplate: "xl-infra",
				AnswerFile:        GetTestTemplateDir(path.Join("xl-up", "answer-xl-up.yaml")),
				QuickSetup:        true,
				AdvancedSetup:     false,
				CfgOverridden:     false,
				NoCleanup:         false,
				Undeploy:          false,
			},
			"beta",
			gb,
		)

		require.Nil(t, err)

		// assertions

		// certs
		assert.FileExists(t, "cert.crt")
		assert.FileExists(t, "cert.key")

		//answer files
		assert.FileExists(t, GeneratedAnswerFile)
		assert.FileExists(t, GeneratedFinalAnswerFile)
		assert.FileExists(t, MergedAnswerFile)
		assert.FileExists(t, TempAnswerFile)

		//xl files
		assert.FileExists(t, "xebialabs.yaml")
		assert.FileExists(t, path.Join(gb.OutputDir, "values.xlvals"))
		assert.FileExists(t, path.Join(gb.OutputDir, "secrets.xlvals"))
		assert.FileExists(t, path.Join(gb.OutputDir, ".gitignore"))

		assert.FileExists(t, path.Join(gb.OutputDir, "answers.yaml"))
		assert.FileExists(t, path.Join(gb.OutputDir, "common.yaml"))
		assert.FileExists(t, path.Join(gb.OutputDir, "deploy-it.lic"))
		assert.FileExists(t, path.Join(gb.OutputDir, "xl-release.lic"))
		assert.FileExists(t, path.Join(gb.OutputDir, "deployments.yaml"))
		assert.FileExists(t, path.Join(gb.OutputDir, "keystore.jceks"))
		assert.FileExists(t, path.Join(gb.OutputDir, "xl-deploy.yaml"))
		assert.FileExists(t, path.Join(gb.OutputDir, "xl-release.yaml"))

		// check __test__ directory is not there
		_, err = os.Stat("__test__")
		assert.True(t, os.IsNotExist(err))

		// check encoded string value in commom.yaml
		commonFile := GetFileContent(path.Join(gb.OutputDir, "common.yaml"))
		assert.Contains(t, commonFile, fmt.Sprintf("tlsCert: %s", `|
      -----BEGIN CERTIFICATE-----
      MIIDDDCCAfSgAwIBAgIRAJpYCmNgnRC42l6lqK7rxOowDQYJKoZIhvcNAQELBQAw
      LzEtMCsGA1UEAxMkMzMzOTBhMDEtMTJiNi00NzViLWFiZjYtNmY4OGRhZTEyYmMz
      MB4XDTE5MDgxNjEzNTkxMVoXDTI0MDgxNDE0NTkxMVowLzEtMCsGA1UEAxMkMzMz
      OTBhMDEtMTJiNi00NzViLWFiZjYtNmY4OGRhZTEyYmMzMIIBIjANBgkqhkiG9w0B
      AQEFAAOCAQ8AMIIBCgKCAQEAxkkd68aG1Sy+S1P83iwMc5pFnehmVWsI7/fm6VK8
      igrzO1MAAUve4WxGR9kDQgOFO9xia2uSUAm7tJ+Hr8oE0ka8c0aLzZizfonsmlRH
      +5QidjwOEtztgEfenuUmlnN2yj1X0Fqd//XB9pyMAlRBVMiXjiJNwWEXWKvGrdna
      8dXEoKIGizhvroGFYThjhgjhdtLnLWz1RKQtcjcnmOX4V/SangsIgkEzSvdj2TfD
      wZon5q4zBasaGmhXr8xA2kRPXKyALaiThoJsRoW0haxNOXJvLNbRDheuNWe7ZGkV
      E/XLqrQguamIvjyFET+2bHZZWlLInJRpSFAvZ3RCtMdknQIDAQABoyMwITAOBgNV
      HQ8BAf8EBAMCAgQwDwYDVR0TAQH/BAUwAwEB/zANBgkqhkiG9w0BAQsFAAOCAQEA
      hdUZZKy41R4YgoAPdIq5ftm3wX4yvB01WzB795U5SJ9ME25QaUw2JNdD/yBwC6wH
      72RcA4L9SlJW0I2mUUtT9uYzF0r+NJJO1QJi6ek8Gu57WzQahs/JtN3giJNLH3eN
      EYwMldMe9Z/6aa4PSKVaq130lLrAty7R/YFA0EDzSjZhea+mpTrpLL+4Ma+PbPCw
      OP7FPOeFAnLXUajrwly1CIL7F/q9HNOlpGcebaS9Ea5a8xkGxPqEqf7M1PK2pn7l
      hHxzUjQUG57tb4tKtUmS8/DchrT1crM4i3AMKzvLLOCX4PnDbhmHJlhcNTKJL6y9
      LxjYOSJ5loUikwq6lQBA5Q==
      -----END CERTIFICATE-----`))

		// check values file
		valsFile := GetFileContent(path.Join(gb.OutputDir, "values.xlvals"))
		valueMap := map[string]string{
			"XlrLic":             "-----BEGIN CERTIFICATE-----\\nMIIDDDCCAfSgAwIBAgIRAJpYCmNgnRC42l6lqK7rxOowDQYJKoZIhvcNAQELBQAw\\nLzEtMCsGA1UEAxMkMzMzOTBhMDEtMTJiNi00NzViLWFiZjYtNmY4OGRhZTEyYmMz\\nMB4XDTE5MDgxNjEzNTkxMVoXDTI0MDgxNDE0NTkxMVowLzEtMCsGA1UEAxMkMzMz\\nOTBhMDEtMTJiNi00NzViLWFiZjYtNmY4OGRhZTEyYmMzMIIBIjANBgkqhkiG9w0B\\nAQEFAAOCAQ8AMIIBCgKCAQEAxkkd68aG1Sy+S1P83iwMc5pFnehmVWsI7/fm6VK8\\nigrzO1MAAUve4WxGR9kDQgOFO9xia2uSUAm7tJ+Hr8oE0ka8c0aLzZizfonsmlRH\\n+5QidjwOEtztgEfenuUmlnN2yj1X0Fqd//XB9pyMAlRBVMiXjiJNwWEXWKvGrdna\\n8dXEoKIGizhvroGFYThjhgjhdtLnLWz1RKQtcjcnmOX4V/SangsIgkEzSvdj2TfD\\nwZon5q4zBasaGmhXr8xA2kRPXKyALaiThoJsRoW0haxNOXJvLNbRDheuNWe7ZGkV\\nE/XLqrQguamIvjyFET+2bHZZWlLInJRpSFAvZ3RCtMdknQIDAQABoyMwITAOBgNV\\nHQ8BAf8EBAMCAgQwDwYDVR0TAQH/BAUwAwEB/zANBgkqhkiG9w0BAQsFAAOCAQEA\\nhdUZZKy41R4YgoAPdIq5ftm3wX4yvB01WzB795U5SJ9ME25QaUw2JNdD/yBwC6wH\\n72RcA4L9SlJW0I2mUUtT9uYzF0r+NJJO1QJi6ek8Gu57WzQahs/JtN3giJNLH3eN\\nEYwMldMe9Z/6aa4PSKVaq130lLrAty7R/YFA0EDzSjZhea+mpTrpLL+4Ma+PbPCw\\nOP7FPOeFAnLXUajrwly1CIL7F/q9HNOlpGcebaS9Ea5a8xkGxPqEqf7M1PK2pn7l\\nhHxzUjQUG57tb4tKtUmS8/DchrT1crM4i3AMKzvLLOCX4PnDbhmHJlhcNTKJL6y9\\nLxjYOSJ5loUikwq6lQBA5Q==\\n-----END CERTIFICATE-----\\n",
			"InstallMonitoring":  "true",
			"K8sAuthentication":  "FilePath",
			"PostgresMaxConn":    "400",
			"XlrOfficialVersion": "8.6.1",
			"XldOfficialVersion": "8.6.1",
		}
		for k, v := range valueMap {
			assert.Contains(t, valsFile, fmt.Sprintf("%s = %s", k, v))
		}

		// check secrets file
		secretsFile := GetFileContent(path.Join(gb.OutputDir, "secrets.xlvals"))
		secretsMap := map[string]string{
			"K8sClientCert":      "Li4vLi4vdGVtcGxhdGVzL3Rlc3QveGwtdXAvY2VydA==",
			"MonitoringUserPass": "mon-pass",
		}
		for k, v := range secretsMap {
			assert.Contains(t, secretsFile, fmt.Sprintf("%s = %s", k, v))
		}
	})

	t.Run("should error when passing --undeploy flag for non existing config", func(t *testing.T) {
		gb := &blueprint.GeneratedBlueprint{OutputDir: models.BlueprintOutputDir}
		defer gb.Cleanup()
		err := InvokeBlueprintAndSeed(
			getLocalTestBlueprintContext(t),
			UpParams{
				// enable for local testing
				// LocalPath:         "../../../xl-up-blueprint",
				BlueprintTemplate: "xl-infra",
				AnswerFile:        GetTestTemplateDir(path.Join("xl-up", "answer-xl-up.yaml")),
				QuickSetup:        true,
				AdvancedSetup:     false,
				CfgOverridden:     false,
				NoCleanup:         false,
				Undeploy:          true,
			},
			"beta",
			gb,
		)

		require.NotNil(t, err)
	})
}

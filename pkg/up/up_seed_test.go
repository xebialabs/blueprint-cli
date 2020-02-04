package up

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"os"
	"path"
	"path/filepath"
	"reflect"
	"runtime"
	"strings"
	"testing"

	"gopkg.in/AlecAivazis/survey.v1"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"

	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/xebialabs/xl-cli/pkg/blueprint"
	"github.com/xebialabs/xl-cli/pkg/models"
	"github.com/xebialabs/xl-cli/pkg/util"
	versionHelper "github.com/xebialabs/xl-cli/pkg/version"
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

func getK8sLocalApiServerURLVal() string {
	if runtime.GOOS == "windows" || runtime.GOOS == "darwin" {
		return "https://host.docker.internal:6443"
	}
	return "https://172.16.16.21:6443"
}

var simpleSampleKubeConfig = `apiVersion: v1
clusters:
- cluster:
    certificate-authority-data: 123==123
    insecure-skip-tls-verify: true
    server: https://test.io:443
  name: testCluster
contexts:
- context:
    cluster: testCluster
    namespace: test
    user: testCluster_user
  name: testCluster
current-context: testCluster
kind: Config
preferences: {}
users:
- name: testCluster_user
  user:
    client-certificate-data: 123==123
    client-key-data: 123==123
    token: 6555565666666666666`

var TestLocalPath = ""
var CLIVersion = "9.1.0"
var GITBranch = "master"

func TestInvokeBlueprintAndSeed(t *testing.T) {

	// enable for local testing
	// TestLocalPath = "../../../xl-up-blueprint"
	// ForceConfigMap = true
	// MockConfigMap = ``

	// enable for local testing with HTTP repo or a different git branch
	// GITBranch = ""
	// CLIVersion = "9.1.0"

	SkipPrompts = true
	blueprint.SkipFinalPrompt = true
	blueprint.SkipUpFinalPrompt = true
	versionHelper.AvailableXldVersions = []string{"9.0.2"}
	versionHelper.AvailableXlrVersions = []string{"9.0.2"}

	// initialize temp dir for tests
	tmpDir, err := ioutil.TempDir("", "xltest")
	if err != nil {
		t.Error(err)
	}
	defer os.RemoveAll(tmpDir)

	// create test k8s config file
	d1 := []byte(simpleSampleKubeConfig)
	ioutil.WriteFile(filepath.Join(tmpDir, "config"), d1, os.ModePerm)
	os.Setenv("KUBECONFIG", filepath.Join(tmpDir, "config"))
	os.Setenv("AWS_ACCESS_KEY_ID", "dummy_val")
	os.Setenv("AWS_SECRET_ACCESS_KEY", "dummy_val")

	t.Run("should create output files for valid xl-up template with answers file", func(t *testing.T) {
		gb := &blueprint.GeneratedBlueprint{OutputDir: models.BlueprintOutputDir}
		defer gb.Cleanup()
		err := InvokeBlueprintAndSeed(
			getLocalTestBlueprintContext(t),
			UpParams{
				LocalPath:         TestLocalPath,
				BlueprintTemplate: "xl-infra",
				AnswerFile:        GetTestTemplateDir(path.Join("xl-up", "answer-xl-up.yaml")),
				QuickSetup:        true,
				AdvancedSetup:     false,
				CfgOverridden:     false,
				NoCleanup:         false,
				Undeploy:          false,
				DryRun:            true,
				SkipK8sConnection: true,
				GITBranch:         GITBranch,
				XLDVersions:       "9.0.2, 9.0.5",
				XLRVersions:       "9.0.2, 9.0.6",
			},
			CLIVersion,
			gb,
		)

		require.Nil(t, err)

		// assertions

		// certs
		assert.FileExists(t, "cert.crt")
		assert.FileExists(t, "cert.key")

		//answer files
		assert.FileExists(t, GeneratedAnswerFile)

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
			"MonitoringInstall":  "true",
			"K8sSetup":           "PlainK8SCluster",
			"K8sAuthentication":  "FilePath",
			"PostgresMaxConn":    "400",
			"XlrOfficialVersion": "9.0.2",
			"XldOfficialVersion": "9.0.2",
			"UseKubeconfig":      "false",
			"K8sApiServerURL":    "https://k8s.com:6443",
		}
		for k, v := range valueMap {
			assert.Contains(t, valsFile, fmt.Sprintf("%s = %s", k, v))
		}

		// check secrets file
		secretsFile := GetFileContent(path.Join(gb.OutputDir, "secrets.xlvals"))
		secretsMap := map[string]string{
			"XlrLic":             "-----BEGIN CERTIFICATE-----\\nMIIDDDCCAfSgAwIBAgIRAJpYCmNgnRC42l6lqK7rxOowDQYJKoZIhvcNAQELBQAw\\nLzEtMCsGA1UEAxMkMzMzOTBhMDEtMTJiNi00NzViLWFiZjYtNmY4OGRhZTEyYmMz\\nMB4XDTE5MDgxNjEzNTkxMVoXDTI0MDgxNDE0NTkxMVowLzEtMCsGA1UEAxMkMzMz\\nOTBhMDEtMTJiNi00NzViLWFiZjYtNmY4OGRhZTEyYmMzMIIBIjANBgkqhkiG9w0B\\nAQEFAAOCAQ8AMIIBCgKCAQEAxkkd68aG1Sy+S1P83iwMc5pFnehmVWsI7/fm6VK8\\nigrzO1MAAUve4WxGR9kDQgOFO9xia2uSUAm7tJ+Hr8oE0ka8c0aLzZizfonsmlRH\\n+5QidjwOEtztgEfenuUmlnN2yj1X0Fqd//XB9pyMAlRBVMiXjiJNwWEXWKvGrdna\\n8dXEoKIGizhvroGFYThjhgjhdtLnLWz1RKQtcjcnmOX4V/SangsIgkEzSvdj2TfD\\nwZon5q4zBasaGmhXr8xA2kRPXKyALaiThoJsRoW0haxNOXJvLNbRDheuNWe7ZGkV\\nE/XLqrQguamIvjyFET+2bHZZWlLInJRpSFAvZ3RCtMdknQIDAQABoyMwITAOBgNV\\nHQ8BAf8EBAMCAgQwDwYDVR0TAQH/BAUwAwEB/zANBgkqhkiG9w0BAQsFAAOCAQEA\\nhdUZZKy41R4YgoAPdIq5ftm3wX4yvB01WzB795U5SJ9ME25QaUw2JNdD/yBwC6wH\\n72RcA4L9SlJW0I2mUUtT9uYzF0r+NJJO1QJi6ek8Gu57WzQahs/JtN3giJNLH3eN\\nEYwMldMe9Z/6aa4PSKVaq130lLrAty7R/YFA0EDzSjZhea+mpTrpLL+4Ma+PbPCw\\nOP7FPOeFAnLXUajrwly1CIL7F/q9HNOlpGcebaS9Ea5a8xkGxPqEqf7M1PK2pn7l\\nhHxzUjQUG57tb4tKtUmS8/DchrT1crM4i3AMKzvLLOCX4PnDbhmHJlhcNTKJL6y9\\nLxjYOSJ5loUikwq6lQBA5Q==\\n-----END CERTIFICATE-----",
			"K8sClientCertFile":  "-----BEGIN CERTIFICATE-----\\nMIIDDDCCAfSgAwIBAgIRAJpYCmNgnRC42l6lqK7rxOowDQYJKoZIhvcNAQELBQAw\\nLzEtMCsGA1UEAxMkMzMzOTBhMDEtMTJiNi00NzViLWFiZjYtNmY4OGRhZTEyYmMz\\nMB4XDTE5MDgxNjEzNTkxMVoXDTI0MDgxNDE0NTkxMVowLzEtMCsGA1UEAxMkMzMz\\nOTBhMDEtMTJiNi00NzViLWFiZjYtNmY4OGRhZTEyYmMzMIIBIjANBgkqhkiG9w0B\\nAQEFAAOCAQ8AMIIBCgKCAQEAxkkd68aG1Sy+S1P83iwMc5pFnehmVWsI7/fm6VK8\\nigrzO1MAAUve4WxGR9kDQgOFO9xia2uSUAm7tJ+Hr8oE0ka8c0aLzZizfonsmlRH\\n+5QidjwOEtztgEfenuUmlnN2yj1X0Fqd//XB9pyMAlRBVMiXjiJNwWEXWKvGrdna\\n8dXEoKIGizhvroGFYThjhgjhdtLnLWz1RKQtcjcnmOX4V/SangsIgkEzSvdj2TfD\\nwZon5q4zBasaGmhXr8xA2kRPXKyALaiThoJsRoW0haxNOXJvLNbRDheuNWe7ZGkV\\nE/XLqrQguamIvjyFET+2bHZZWlLInJRpSFAvZ3RCtMdknQIDAQABoyMwITAOBgNV\\nHQ8BAf8EBAMCAgQwDwYDVR0TAQH/BAUwAwEB/zANBgkqhkiG9w0BAQsFAAOCAQEA\\nhdUZZKy41R4YgoAPdIq5ftm3wX4yvB01WzB795U5SJ9ME25QaUw2JNdD/yBwC6wH\\n72RcA4L9SlJW0I2mUUtT9uYzF0r+NJJO1QJi6ek8Gu57WzQahs/JtN3giJNLH3eN\\nEYwMldMe9Z/6aa4PSKVaq130lLrAty7R/YFA0EDzSjZhea+mpTrpLL+4Ma+PbPCw\\nOP7FPOeFAnLXUajrwly1CIL7F/q9HNOlpGcebaS9Ea5a8xkGxPqEqf7M1PK2pn7l\\nhHxzUjQUG57tb4tKtUmS8/DchrT1crM4i3AMKzvLLOCX4PnDbhmHJlhcNTKJL6y9\\nLxjYOSJ5loUikwq6lQBA5Q==\\n-----END CERTIFICATE-----",
			"MonitoringUserPass": "mon-pass",
		}
		for k, v := range secretsMap {
			assert.Contains(t, secretsFile, fmt.Sprintf("%s = %s", k, v))
		}
	})

	t.Run("should create output files for valid xl-up template with answers from config map for update scenario", func(t *testing.T) {
		MockConfigMap = `
        K8sSetup: PlainK8SCluster
        UseKubeconfig: false
        K8sApiServerURL: https://k8s.com:6443
        K8sAuthentication: FilePath
        K8sClientCertFile: ../../templates/test/xl-up/cert
        K8sClientKeyFile: ../../templates/test/xl-up/cert
        InstallXLD: true
        InstallXLR: true
        UseCustomRegistry: true
        XlrVersion: xl-release:9.0.2
        XldVersion: xl-deploy:9.0.2
        DockerUser: yo
        DockerPass: yo
        RegistryURL: docker.io/xebialabs
        XldAdminPass: password
        XldLic: ../../templates/test/xl-up/cert
        XldDbName: xl-deploy
        XldDbUser: xl-deploy
        XldDbPass: xl-deploy
        NfsServerHost: 12.2.2.2
        NfsSharePath: /xebialabs
        XlrAdminPass: password
        XlrLic: ../../templates/test/xl-up/cert
        XlrDbName: xl-release
        XlrDbUser: xl-release
        XlrDbPass: xl-release
        XlrReportDbName: xl-release-report
        XlrReportDbUser: xl-release-report
        XlrReportDbPass: xl-release-report
        XlKeyStore: ../../templates/test/xl-up/cert
        XlKeyStorePass: test123
        MonitoringInstall: true
        MonitoringUser: mon-user
        MonitoringUserPass: mon-pass
        PostgresMaxConn: 400
        PostgresSharedBuff: 612MB
        PostgresEffectCacheSize: 2GB
        PostgresSyncCommit: "off"
        PostgresMaxWallSize: 512MB
        PostgresqlWorkHostpath: some/path
        OsType: linux
        `
		ForceConfigMap = true
		EmptyVersion = "9.0.2"
		defer func() {
			MockConfigMap = ""
			ForceConfigMap = false
			EmptyVersion = ""
		}()
		gb := &blueprint.GeneratedBlueprint{OutputDir: models.BlueprintOutputDir}
		defer gb.Cleanup()
		err := InvokeBlueprintAndSeed(
			getLocalTestBlueprintContext(t),
			UpParams{
				// enable for local testing
				LocalPath:         TestLocalPath,
				BlueprintTemplate: "xl-infra",
				AnswerFile:        GetTestTemplateDir(path.Join("xl-up", "answer-xl-up-update.yaml")),
				QuickSetup:        true,
				AdvancedSetup:     false,
				CfgOverridden:     false,
				NoCleanup:         false,
				Undeploy:          false,
				DryRun:            true,
				SkipK8sConnection: true,
				GITBranch:         GITBranch,
				XLDVersions:       "9.0.2, 9.0.5",
				XLRVersions:       "9.0.2, 9.0.6",
			},
			CLIVersion,
			gb,
		)

		require.Nil(t, err)

		// assertions

		// certs
		assert.FileExists(t, "cert.crt")
		assert.FileExists(t, "cert.key")

		//answer files
		assert.FileExists(t, GeneratedAnswerFile)

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
			"MonitoringInstall": "true",
			"K8sSetup":          "PlainK8SCluster",
			"K8sAuthentication": "FilePath",
			"PostgresMaxConn":   "400",
			"UseCustomRegistry": "true",
			"XlrVersion":        "xl-release:9.0.2",
			"XldVersion":        "xl-deploy:9.0.2",
			"DockerUser":        "yo",
			"RegistryURL":       "docker.io/xebialabs",
			"UseKubeconfig":     "false",
			"K8sApiServerURL":   "https://k8s.com:6443",
		}
		for k, v := range valueMap {
			assert.Contains(t, valsFile, fmt.Sprintf("%s = %s", k, v))
		}

		// check secrets file
		secretsFile := GetFileContent(path.Join(gb.OutputDir, "secrets.xlvals"))
		secretsMap := map[string]string{
			"XlrLic":             "-----BEGIN CERTIFICATE-----\\nMIIDDDCCAfSgAwIBAgIRAJpYCmNgnRC42l6lqK7rxOowDQYJKoZIhvcNAQELBQAw\\nLzEtMCsGA1UEAxMkMzMzOTBhMDEtMTJiNi00NzViLWFiZjYtNmY4OGRhZTEyYmMz\\nMB4XDTE5MDgxNjEzNTkxMVoXDTI0MDgxNDE0NTkxMVowLzEtMCsGA1UEAxMkMzMz\\nOTBhMDEtMTJiNi00NzViLWFiZjYtNmY4OGRhZTEyYmMzMIIBIjANBgkqhkiG9w0B\\nAQEFAAOCAQ8AMIIBCgKCAQEAxkkd68aG1Sy+S1P83iwMc5pFnehmVWsI7/fm6VK8\\nigrzO1MAAUve4WxGR9kDQgOFO9xia2uSUAm7tJ+Hr8oE0ka8c0aLzZizfonsmlRH\\n+5QidjwOEtztgEfenuUmlnN2yj1X0Fqd//XB9pyMAlRBVMiXjiJNwWEXWKvGrdna\\n8dXEoKIGizhvroGFYThjhgjhdtLnLWz1RKQtcjcnmOX4V/SangsIgkEzSvdj2TfD\\nwZon5q4zBasaGmhXr8xA2kRPXKyALaiThoJsRoW0haxNOXJvLNbRDheuNWe7ZGkV\\nE/XLqrQguamIvjyFET+2bHZZWlLInJRpSFAvZ3RCtMdknQIDAQABoyMwITAOBgNV\\nHQ8BAf8EBAMCAgQwDwYDVR0TAQH/BAUwAwEB/zANBgkqhkiG9w0BAQsFAAOCAQEA\\nhdUZZKy41R4YgoAPdIq5ftm3wX4yvB01WzB795U5SJ9ME25QaUw2JNdD/yBwC6wH\\n72RcA4L9SlJW0I2mUUtT9uYzF0r+NJJO1QJi6ek8Gu57WzQahs/JtN3giJNLH3eN\\nEYwMldMe9Z/6aa4PSKVaq130lLrAty7R/YFA0EDzSjZhea+mpTrpLL+4Ma+PbPCw\\nOP7FPOeFAnLXUajrwly1CIL7F/q9HNOlpGcebaS9Ea5a8xkGxPqEqf7M1PK2pn7l\\nhHxzUjQUG57tb4tKtUmS8/DchrT1crM4i3AMKzvLLOCX4PnDbhmHJlhcNTKJL6y9\\nLxjYOSJ5loUikwq6lQBA5Q==\\n-----END CERTIFICATE-----",
			"K8sClientCertFile":  "-----BEGIN CERTIFICATE-----\\nMIIDDDCCAfSgAwIBAgIRAJpYCmNgnRC42l6lqK7rxOowDQYJKoZIhvcNAQELBQAw\\nLzEtMCsGA1UEAxMkMzMzOTBhMDEtMTJiNi00NzViLWFiZjYtNmY4OGRhZTEyYmMz\\nMB4XDTE5MDgxNjEzNTkxMVoXDTI0MDgxNDE0NTkxMVowLzEtMCsGA1UEAxMkMzMz\\nOTBhMDEtMTJiNi00NzViLWFiZjYtNmY4OGRhZTEyYmMzMIIBIjANBgkqhkiG9w0B\\nAQEFAAOCAQ8AMIIBCgKCAQEAxkkd68aG1Sy+S1P83iwMc5pFnehmVWsI7/fm6VK8\\nigrzO1MAAUve4WxGR9kDQgOFO9xia2uSUAm7tJ+Hr8oE0ka8c0aLzZizfonsmlRH\\n+5QidjwOEtztgEfenuUmlnN2yj1X0Fqd//XB9pyMAlRBVMiXjiJNwWEXWKvGrdna\\n8dXEoKIGizhvroGFYThjhgjhdtLnLWz1RKQtcjcnmOX4V/SangsIgkEzSvdj2TfD\\nwZon5q4zBasaGmhXr8xA2kRPXKyALaiThoJsRoW0haxNOXJvLNbRDheuNWe7ZGkV\\nE/XLqrQguamIvjyFET+2bHZZWlLInJRpSFAvZ3RCtMdknQIDAQABoyMwITAOBgNV\\nHQ8BAf8EBAMCAgQwDwYDVR0TAQH/BAUwAwEB/zANBgkqhkiG9w0BAQsFAAOCAQEA\\nhdUZZKy41R4YgoAPdIq5ftm3wX4yvB01WzB795U5SJ9ME25QaUw2JNdD/yBwC6wH\\n72RcA4L9SlJW0I2mUUtT9uYzF0r+NJJO1QJi6ek8Gu57WzQahs/JtN3giJNLH3eN\\nEYwMldMe9Z/6aa4PSKVaq130lLrAty7R/YFA0EDzSjZhea+mpTrpLL+4Ma+PbPCw\\nOP7FPOeFAnLXUajrwly1CIL7F/q9HNOlpGcebaS9Ea5a8xkGxPqEqf7M1PK2pn7l\\nhHxzUjQUG57tb4tKtUmS8/DchrT1crM4i3AMKzvLLOCX4PnDbhmHJlhcNTKJL6y9\\nLxjYOSJ5loUikwq6lQBA5Q==\\n-----END CERTIFICATE-----",
			"MonitoringUserPass": "mon-pass",
			"DockerPass":         "yo",
		}
		for k, v := range secretsMap {
			assert.Contains(t, secretsFile, fmt.Sprintf("%s = %s", k, v))
		}
	})

	t.Run("should create output files for valid xl-up template with answers file for local setup", func(t *testing.T) {
		gb := &blueprint.GeneratedBlueprint{OutputDir: models.BlueprintOutputDir}
		defer gb.Cleanup()
		err := InvokeBlueprintAndSeed(
			getLocalTestBlueprintContext(t),
			UpParams{
				// enable for local testing
				LocalPath:         TestLocalPath,
				BlueprintTemplate: "xl-infra",
				AnswerFile:        GetTestTemplateDir(path.Join("xl-up", "answer-xl-up-local.yaml")),
				QuickSetup:        true,
				AdvancedSetup:     false,
				CfgOverridden:     false,
				NoCleanup:         false,
				Undeploy:          false,
				DryRun:            true,
				SkipK8sConnection: true,
				GITBranch:         GITBranch,
				XLDVersions:       "9.0.2, 9.0.5",
				XLRVersions:       "9.0.2, 9.0.6",
			},
			CLIVersion,
			gb,
		)

		require.Nil(t, err)

		// assertions

		// certs
		assert.False(t, util.PathExists("cert.crt", false))
		assert.False(t, util.PathExists("cert.key", false))

		//answer files
		assert.FileExists(t, GeneratedAnswerFile)

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
		assert.Contains(t, commonFile, fmt.Sprintf("tlsCert: %s", `123==123`))

		// check values file
		valsFile := GetFileContent(path.Join(gb.OutputDir, "values.xlvals"))
		valueMap := map[string]string{
			"K8sSetup":             "DockerDesktopK8s",
			"PostgresMaxConn":      "512",
			"XlrOfficialVersion":   "9.0.2",
			"XldOfficialVersion":   "9.0.2",
			"UseKubeconfig":        "true",
			"K8sLocalApiServerURL": getK8sLocalApiServerURLVal(),
		}
		for k, v := range valueMap {
			assert.Contains(t, valsFile, fmt.Sprintf("%s = %s", k, v))
		}

		// check secrets file
		secretsFile := GetFileContent(path.Join(gb.OutputDir, "secrets.xlvals"))
		secretsMap := map[string]string{
			"XlrLic":        "-----BEGIN CERTIFICATE-----\\nMIIDDDCCAfSgAwIBAgIRAJpYCmNgnRC42l6lqK7rxOowDQYJKoZIhvcNAQELBQAw\\nLzEtMCsGA1UEAxMkMzMzOTBhMDEtMTJiNi00NzViLWFiZjYtNmY4OGRhZTEyYmMz\\nMB4XDTE5MDgxNjEzNTkxMVoXDTI0MDgxNDE0NTkxMVowLzEtMCsGA1UEAxMkMzMz\\nOTBhMDEtMTJiNi00NzViLWFiZjYtNmY4OGRhZTEyYmMzMIIBIjANBgkqhkiG9w0B\\nAQEFAAOCAQ8AMIIBCgKCAQEAxkkd68aG1Sy+S1P83iwMc5pFnehmVWsI7/fm6VK8\\nigrzO1MAAUve4WxGR9kDQgOFO9xia2uSUAm7tJ+Hr8oE0ka8c0aLzZizfonsmlRH\\n+5QidjwOEtztgEfenuUmlnN2yj1X0Fqd//XB9pyMAlRBVMiXjiJNwWEXWKvGrdna\\n8dXEoKIGizhvroGFYThjhgjhdtLnLWz1RKQtcjcnmOX4V/SangsIgkEzSvdj2TfD\\nwZon5q4zBasaGmhXr8xA2kRPXKyALaiThoJsRoW0haxNOXJvLNbRDheuNWe7ZGkV\\nE/XLqrQguamIvjyFET+2bHZZWlLInJRpSFAvZ3RCtMdknQIDAQABoyMwITAOBgNV\\nHQ8BAf8EBAMCAgQwDwYDVR0TAQH/BAUwAwEB/zANBgkqhkiG9w0BAQsFAAOCAQEA\\nhdUZZKy41R4YgoAPdIq5ftm3wX4yvB01WzB795U5SJ9ME25QaUw2JNdD/yBwC6wH\\n72RcA4L9SlJW0I2mUUtT9uYzF0r+NJJO1QJi6ek8Gu57WzQahs/JtN3giJNLH3eN\\nEYwMldMe9Z/6aa4PSKVaq130lLrAty7R/YFA0EDzSjZhea+mpTrpLL+4Ma+PbPCw\\nOP7FPOeFAnLXUajrwly1CIL7F/q9HNOlpGcebaS9Ea5a8xkGxPqEqf7M1PK2pn7l\\nhHxzUjQUG57tb4tKtUmS8/DchrT1crM4i3AMKzvLLOCX4PnDbhmHJlhcNTKJL6y9\\nLxjYOSJ5loUikwq6lQBA5Q==\\n-----END CERTIFICATE-----\\n",
			"K8sClientCert": "123==123",
		}
		for k, v := range secretsMap {
			assert.Contains(t, secretsFile, fmt.Sprintf("%s = %s", k, v))
		}
	})

	undeployCalled := false
	undeployAll = func(client *kubernetes.Clientset) error {
		undeployCalled = true
		return nil
	}

	getKubeClient = func(answerMap map[string]string) (clientset *kubernetes.Clientset, err error) {
		client := kubernetes.Clientset{}
		return &client, nil
	}

	getK8sConfigMaps = func(client *kubernetes.Clientset, opts metav1.ListOptions) (*v1.ConfigMapList, error) {
		return &v1.ConfigMapList{
			Items: []v1.ConfigMap{
				v1.ConfigMap{
					ObjectMeta: metav1.ObjectMeta{Name: ConfigMapName},
					Data: map[string]string{
						DataFile: `
                        InstallXLD: true
                        InstallXLR: true
                        `,
					},
				},
			},
		}, nil
	}
	getK8sNamespaces = func(client *kubernetes.Clientset, opts metav1.ListOptions) (*v1.NamespaceList, error) {
		return &v1.NamespaceList{
			Items: []v1.Namespace{
				v1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: NAMESPACE}},
			},
		}, nil
	}

	t.Run("should undeploy when passing --undeploy flag for existing config", func(t *testing.T) {
		gb := &blueprint.GeneratedBlueprint{OutputDir: models.BlueprintOutputDir}
		defer gb.Cleanup()
		err := InvokeBlueprintAndSeed(
			getLocalTestBlueprintContext(t),
			UpParams{
				// enable for local testing
				LocalPath:         TestLocalPath,
				AnswerFile:        GetTestTemplateDir(path.Join("xl-up", "answer-xl-up.yaml")),
				QuickSetup:        true,
				AdvancedSetup:     false,
				CfgOverridden:     false,
				NoCleanup:         false,
				Undeploy:          true,
				DryRun:            true,
				SkipK8sConnection: false,
				GITBranch:         GITBranch,
				XLDVersions:       "9.0.2, 9.0.5",
				XLRVersions:       "9.0.2, 9.0.6",
			},
			CLIVersion,
			gb,
		)

		require.Nil(t, err)
		assert.True(t, undeployCalled)
	})

	askToSaveToConfig = func(surveyOpts ...survey.AskOpt) (b bool, err error) {
		return true, nil
	}

	updateCalled := false
	tempUpdateXebialabsConfig := updateXebialabsConfig
	updateXebialabsConfig = func(client *kubernetes.Clientset, answers map[string]string, v *viper.Viper) error {
		updateCalled = true
		return nil
	}

	getKubeClient = func(answerMap map[string]string) (clientset *kubernetes.Clientset, err error) {
		client := kubernetes.Clientset{}
		return &client, nil
	}

	GetIp = func(client *kubernetes.Clientset) (string, error) {
		return "http://testhost", nil
	}

	t.Run("should save config when save config is answered yes", func(t *testing.T) {
		gb := &blueprint.GeneratedBlueprint{OutputDir: models.BlueprintOutputDir}
		defer gb.Cleanup()
		err := InvokeBlueprintAndSeed(
			getLocalTestBlueprintContext(t),
			UpParams{
				// enable for local testing
				LocalPath:         TestLocalPath,
				AnswerFile:        GetTestTemplateDir(path.Join("xl-up", "answer-xl-up.yaml")),
				QuickSetup:        true,
				AdvancedSetup:     false,
				CfgOverridden:     false,
				NoCleanup:         false,
				Undeploy:          false,
				DryRun:            true,
				SkipK8sConnection: false,
				GITBranch:         GITBranch,
				XLDVersions:       "9.0.2, 9.0.5",
				XLRVersions:       "9.0.2, 9.0.6",
			},
			CLIVersion,
			gb,
		)
		require.Nil(t, err)
		assert.True(t, updateCalled)
	})

	askToSaveToConfig = func(surveyOpts ...survey.AskOpt) (b bool, err error) {
		return false, nil
	}

	updateCalled = false
	t.Run("should not save config when save config is answered no", func(t *testing.T) {
		gb := &blueprint.GeneratedBlueprint{OutputDir: models.BlueprintOutputDir}
		defer gb.Cleanup()
		err := InvokeBlueprintAndSeed(
			getLocalTestBlueprintContext(t),
			UpParams{
				// enable for local testing
				LocalPath:         TestLocalPath,
				AnswerFile:        GetTestTemplateDir(path.Join("xl-up", "answer-xl-up.yaml")),
				QuickSetup:        true,
				AdvancedSetup:     false,
				CfgOverridden:     false,
				NoCleanup:         false,
				Undeploy:          false,
				DryRun:            true,
				SkipK8sConnection: true,
				GITBranch:         GITBranch,
				XLDVersions:       "9.0.2, 9.0.5",
				XLRVersions:       "9.0.2, 9.0.6",
			},
			CLIVersion,
			gb,
		)
		assert.Nil(t, err)
		assert.False(t, updateCalled)
	})
	updateXebialabsConfig = tempUpdateXebialabsConfig
}

func Test_getVersion(t *testing.T) {
	type args struct {
		answerMapFromConfigMap map[string]string
		key                    string
		prevKey                string
	}
	tests := []struct {
		name string
		args args
		want string
	}{
		{
			"get current key",
			args{
				map[string]string{
					"current": "current",
					"prev":    "prev",
				},
				"current",
				"prev",
			},
			"current",
		},
		{
			"get prev key when current not found",
			args{
				map[string]string{
					"prev": "prev",
				},
				"current",
				"prev",
			},
			"prev",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := getVersion(tt.args.answerMapFromConfigMap, tt.args.key, tt.args.prevKey); got != tt.want {
				t.Errorf("getVersion() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_getAvailableVersions(t *testing.T) {
	tests := []struct {
		name            string
		versions        string
		defaultVersions []string
		want            []string
	}{
		{
			"return default when no version given",
			"",
			[]string{"9.0.0"},
			[]string{"9.0.0"},
		},
		{
			"return version array when given",
			"9.0.5, 9.6.5 ,9.5",
			[]string{"9.0.0"},
			[]string{"9.0.5", "9.6.5", "9.5"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := getAvailableVersions(tt.versions, tt.defaultVersions); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("getAvailableVersions() = %v, want %v", got, tt.want)
			}
		})
	}
}

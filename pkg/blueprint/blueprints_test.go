package blueprint

import (
	b64 "encoding/base64"
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"path/filepath"
	"strings"
	"testing"

	"github.com/jarcoal/httpmock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/xebialabs/xl-cli/pkg/models"
	"github.com/xebialabs/xl-cli/pkg/util"
	survey "gopkg.in/AlecAivazis/survey.v1"
)

// auxiliary functions
func GetFileContent(filePath string) string {
	f, err := ioutil.ReadFile(filePath)
	if err != nil {
		panic(err)
	}
	return string(f)
}

func RemoveFiles(glob string) {
	files, err := filepath.Glob(glob)
	if err != nil {
		panic(err)
	}
	for _, f := range files {
		if err := os.Remove(f); err != nil {
			panic(err)
		}
	}
}

func GetTestTemplateDir(blueprint string) string {
	pwd, _ := os.Getwd()
	return strings.Replace(pwd, path.Join("pkg", "blueprint"), path.Join("templates", "test", blueprint), -1)
}

func TestWriteDataToFile(t *testing.T) {
	t.Run("should write template data to output file", func(t *testing.T) {
		gb := new(GeneratedBlueprint)
		defer gb.Cleanup()
		data := "test\ndata\n"
		filePath := "test.yml"
		err := writeDataToFile(gb, filePath, &data)
		require.Nil(t, err)
		assert.FileExists(t, filePath)
		assert.Equal(t, GetFileContent(filePath), data)
	})
	t.Run("should write template data to output file in a folder", func(t *testing.T) {
		gb := new(GeneratedBlueprint)
		defer gb.Cleanup()
		data := "test\ndata\n"
		filePath := path.Join("test", "test.yml")
		err := writeDataToFile(gb, filePath, &data)
		require.Nil(t, err)
		assert.FileExists(t, filePath)
		assert.Equal(t, GetFileContent(filePath), data)
	})
}

func TestWriteConfigToFile(t *testing.T) {
	t.Run("should write config data to output file sorted", func(t *testing.T) {
		config := make(map[string]interface{}, 3)
		config["d"] = 1
		config["a"] = true
		config["z"] = "test"
		filePath := "test.xlvals"
		gb := new(GeneratedBlueprint)
		err := writeConfigToFile("#comment", config, gb, filePath)
		defer gb.Cleanup()
		require.Nil(t, err)
		assert.FileExists(t, filePath)
		assert.Equal(t, "#comment\na = true\nd = 1\nz = test", strings.TrimSpace(GetFileContent(filePath)))
	})
	t.Run("should write config data to output file in folder", func(t *testing.T) {
		gb := new(GeneratedBlueprint)
		defer gb.Cleanup()
		config := make(map[string]interface{}, 3)
		config["d"] = 1
		config["a"] = true
		config["z"] = "test"
		filePath := path.Join("test", "test.xlvals")
		err := writeConfigToFile("#comment", config, gb, filePath)
		require.Nil(t, err)
		assert.FileExists(t, filePath)
		assert.Equal(t, "#comment\na = true\nd = 1\nz = test", strings.TrimSpace(GetFileContent(filePath)))
	})
}

func TestInstantiateBlueprint(t *testing.T) {
	SkipFinalPrompt = true
	// V2 schema test
	t.Run("should error on unknown template", func(t *testing.T) {
		gb := &GeneratedBlueprint{OutputDir: "xebialabs"}
		defer gb.Cleanup()
		_, _, err := InstantiateBlueprint(
			BlueprintParams{
				TemplatePath:       "abc",
				AnswersFile:        "",
				StrictAnswers:      false,
				UseDefaultsAsValue: false,
				FromUpCommand:      false,
				PrintSummaryTable:  true,
			},
			getLocalTestBlueprintContext(t),
			gb,
		)

		require.NotNil(t, err)
		assert.Equal(t, "blueprint [abc] not found in repository Test", err.Error())
	})

	t.Run("should error on invalid test template", func(t *testing.T) {
		gb := &GeneratedBlueprint{OutputDir: "xebialabs"}
		defer gb.Cleanup()
		_, _, err := InstantiateBlueprint(
			BlueprintParams{
				TemplatePath:       "invalid",
				AnswersFile:        "",
				StrictAnswers:      false,
				UseDefaultsAsValue: false,
				FromUpCommand:      false,
				PrintSummaryTable:  true,
			},
			getLocalTestBlueprintContext(t),
			gb,
		)
		require.NotNil(t, err)
		assert.Equal(t, "parameter AppName must have a 'prompt' field", err.Error())
	})

	t.Run("should create output files for valid test template with answers map", func(t *testing.T) {
		gb := &GeneratedBlueprint{OutputDir: "xebialabs"}
		defer gb.Cleanup()
		data, doc, err := InstantiateBlueprint(
			BlueprintParams{
				TemplatePath: "answer-input",
				AnswersMap: map[string]string{
					"Test":               "testing",
					"ClientCert":         "FshYmQzRUNbYTA4Icc3V7JEgLXMNjcSLY9L1H4XQD79coMBRbbJFtOsp0Yk2btCKCAYLio0S8Jw85W5mgpLkasvCrXO5\\nQJGxFvtQc2tHGLj0kNzM9KyAqbUJRe1l40TqfMdscEaWJimtd4oygqVc6y7zW1Wuj1EcDUvMD8qK8FEWfQgm5ilBIldQ\\n",
					"TestDepends":        "true",
					"TestDepends2":       "false",
					"TestDepends3":       "false",
					"AppName":            "TestApp",
					"AWSAccessKey":       "accesskey",
					"AWSAccessSecret":    "accesssecret",
					"ShouldNotBeThere":   "nope",
					"SuperSecret":        "invisible",
					"AWSRegion":          "eu-central-1",
					"DiskSize":           "100.0",
					"DiskSizeWithBuffer": "125.1",
				},
				StrictAnswers:      true,
				UseDefaultsAsValue: false,
				FromUpCommand:      false,
				PrintSummaryTable:  true,
			},
			getLocalTestBlueprintContext(t),
			gb,
		)
		require.Nil(t, err)
		require.NotNil(t, data)
		require.NotNil(t, doc)

		// assertions
		assert.FileExists(t, "xld-environment.yml")
		assert.FileExists(t, "xld-infrastructure.yml")
		assert.FileExists(t, "xlr-pipeline.yml")
		assert.FileExists(t, path.Join(gb.OutputDir, valuesFile))
		assert.FileExists(t, path.Join(gb.OutputDir, secretsFile))
		assert.FileExists(t, path.Join(gb.OutputDir, gitignoreFile))

		// check __test__ directory is not there
		_, err = os.Stat("__test__")
		assert.True(t, os.IsNotExist(err))

		// check encoded string value in env template
		envTemplateFile := GetFileContent("xld-environment.yml")
		assert.Contains(t, envTemplateFile, fmt.Sprintf("accessSecret: %s", b64.StdEncoding.EncodeToString([]byte("accesssecret"))))

		// check values file
		valsFile := GetFileContent(path.Join(gb.OutputDir, valuesFile))
		valueMap := map[string]string{
			"Test":               "testing",
			"ClientCert":         "FshYmQzRUNbYTA4Icc3V7JEgLXMNjcSLY9L1H4XQD79coMBRbbJFtOsp0Yk2btCKCAYLio0S8Jw85W5mgpLkasvCrXO5\\nQJGxFvtQc2tHGLj0kNzM9KyAqbUJRe1l40TqfMdscEaWJimtd4oygqVc6y7zW1Wuj1EcDUvMD8qK8FEWfQgm5ilBIldQ\\n",
			"AppName":            "TestApp",
			"SuperSecret":        "invisible",
			"AWSRegion":          "eu-central-1",
			"DiskSize":           "100",
			"DiskSizeWithBuffer": "125.1",
			"ShouldNotBeThere":   "",
		}
		for k, v := range valueMap {
			assert.Contains(t, valsFile, fmt.Sprintf("%s = %s", k, v))
		}

		// check secrets file
		secretsFile := GetFileContent(path.Join(gb.OutputDir, secretsFile))
		secretsMap := map[string]string{
			"AWSAccessKey":    "accesskey",
			"AWSAccessSecret": "accesssecret",
		}
		for k, v := range secretsMap {
			assert.Contains(t, secretsFile, fmt.Sprintf("%s = %s", k, v))
		}
	})

	t.Run("should create output files for valid test template with answers file", func(t *testing.T) {
		gb := &GeneratedBlueprint{OutputDir: "xebialabs"}
		defer gb.Cleanup()
		data, doc, err := InstantiateBlueprint(
			BlueprintParams{
				TemplatePath:       "answer-input",
				AnswersFile:        GetTestTemplateDir("answer-input.yaml"),
				StrictAnswers:      true,
				UseDefaultsAsValue: false,
				FromUpCommand:      false,
				PrintSummaryTable:  true,
			},
			getLocalTestBlueprintContext(t),
			gb,
		)
		require.Nil(t, err)
		require.NotNil(t, data)
		require.NotNil(t, doc)

		// assertions
		assert.FileExists(t, "xld-environment.yml")
		assert.FileExists(t, "xld-infrastructure.yml")
		assert.FileExists(t, "xlr-pipeline.yml")
		assert.FileExists(t, path.Join(gb.OutputDir, valuesFile))
		assert.FileExists(t, path.Join(gb.OutputDir, secretsFile))
		assert.FileExists(t, path.Join(gb.OutputDir, gitignoreFile))

		// check __test__ directory is not there
		_, err = os.Stat("__test__")
		assert.True(t, os.IsNotExist(err))

		// check encoded string value in env template
		envTemplateFile := GetFileContent("xld-environment.yml")
		assert.Contains(t, envTemplateFile, fmt.Sprintf("accessSecret: %s", b64.StdEncoding.EncodeToString([]byte("accesssecret"))))

		// check values file
		valsFile := GetFileContent(path.Join(gb.OutputDir, valuesFile))
		valueMap := map[string]string{
			"Test":               "testing",
			"ClientCert":         "FshYmQzRUNbYTA4Icc3V7JEgLXMNjcSLY9L1H4XQD79coMBRbbJFtOsp0Yk2btCKCAYLio0S8Jw85W5mgpLkasvCrXO5\\nQJGxFvtQc2tHGLj0kNzM9KyAqbUJRe1l40TqfMdscEaWJimtd4oygqVc6y7zW1Wuj1EcDUvMD8qK8FEWfQgm5ilBIldQ\\n",
			"AppName":            "TestApp",
			"SuperSecret":        "invisible",
			"AWSRegion":          "eu-central-1",
			"DiskSize":           "100",
			"DiskSizeWithBuffer": "125.1",
			"ShouldNotBeThere":   "",
		}
		for k, v := range valueMap {
			assert.Contains(t, valsFile, fmt.Sprintf("%s = %s", k, v))
		}

		// check secrets file
		secretsFile := GetFileContent(path.Join(gb.OutputDir, secretsFile))
		secretsMap := map[string]string{
			"AWSAccessKey":    "accesskey",
			"AWSAccessSecret": "accesssecret",
		}
		for k, v := range secretsMap {
			assert.Contains(t, secretsFile, fmt.Sprintf("%s = %s", k, v))
		}
	})

	t.Run("should create output files for valid test template with promptIf on parameters", func(t *testing.T) {
		gb := &GeneratedBlueprint{OutputDir: "xebialabs"}
		defer gb.Cleanup()
		data, doc, err := InstantiateBlueprint(
			BlueprintParams{
				TemplatePath:       "dependson-test",
				AnswersFile:        GetTestTemplateDir("answer-input-dependson.yaml"),
				StrictAnswers:      false,
				UseDefaultsAsValue: false,
				FromUpCommand:      false,
				PrintSummaryTable:  true,
			},
			getLocalTestBlueprintContext(t),
			gb,
		)
		require.Nil(t, err)
		require.NotNil(t, data)
		require.NotNil(t, doc)

		// assertions
		assert.FileExists(t, path.Join(gb.OutputDir, valuesFile))
		assert.FileExists(t, path.Join(gb.OutputDir, secretsFile))
		assert.FileExists(t, path.Join(gb.OutputDir, gitignoreFile))

		// check __test__ directory is not there
		_, err = os.Stat("__test__")
		assert.True(t, os.IsNotExist(err))

		// check values file
		valsFile := GetFileContent(path.Join(gb.OutputDir, valuesFile))
		valueMap := map[string]string{
			"UseAWSCredentialsFromSystem": "true",
		}
		for k, v := range valueMap {
			assert.Contains(t, valsFile, fmt.Sprintf("%s = %s", k, v))
		}

		// check secrets file
		secretsFile := GetFileContent(path.Join(gb.OutputDir, secretsFile))
		secretsMap := map[string]string{
			"AWSAccessSecret":      "testSecret",
			"AWSAccessSuperSecret": "superSecret",
		}
		for k, v := range secretsMap {
			assert.Contains(t, secretsFile, fmt.Sprintf("%s = %s", k, v))
		}
		assert.NotContains(t, secretsFile, "AWSAccessKey")
	})

	t.Run("should create output files for valid test template in use defaults as values mode", func(t *testing.T) {
		gb := &GeneratedBlueprint{OutputDir: "xebialabs"}
		defer gb.Cleanup()
		data, doc, err := InstantiateBlueprint(
			BlueprintParams{
				TemplatePath:       "defaults-as-values",
				AnswersFile:        "",
				StrictAnswers:      false,
				UseDefaultsAsValue: true,
				FromUpCommand:      false,
				PrintSummaryTable:  true,
			},
			getLocalTestBlueprintContext(t),
			gb,
		)
		require.Nil(t, err)
		require.NotNil(t, data)
		require.NotNil(t, doc)

		// assertions
		assert.FileExists(t, "xld-environment.yml")
		assert.FileExists(t, "xld-infrastructure.yml")
		assert.FileExists(t, "xlr-pipeline.yml")
		assert.FileExists(t, path.Join(gb.OutputDir, valuesFile))
		assert.FileExists(t, path.Join(gb.OutputDir, secretsFile))
		assert.FileExists(t, path.Join(gb.OutputDir, gitignoreFile))

		// check __test__ directory is not there
		_, err = os.Stat("__test__")
		assert.True(t, os.IsNotExist(err))

		// check values file
		valsFile := GetFileContent(path.Join(gb.OutputDir, valuesFile))
		valueMap := map[string]string{
			"Test":               "testing",
			"ClientCert":         "this is a multiline\\ntext\\n\\nwith escape chars\\n",
			"AppName":            "TestApp",
			"SuperSecret":        "supersecret",
			"AWSRegion":          "eu-central-1",
			"DiskSize":           "10",
			"DiskSizeWithBuffer": "125.6",
			"ShouldNotBeThere":   "shouldnotbehere",
			"File":               "-----BEGIN CERTIFICATE-----\\nMIIDDDCCAfSgAwIBAgIRAJpYCmNgnRC42l6lqK7rxOowDQYJKoZIhvcNAQELBQAw\\nLzEtMCsGA1UEAxMkMzMzOTBhMDEtMTJiNi00NzViLWFiZjYtNmY4OGRhZTEyYmMz\\n-----END CERTIFICATE-----\\n",
		}
		for k, v := range valueMap {
			assert.Contains(t, valsFile, fmt.Sprintf("%s = %s", k, v))
		}

		// check secrets file
		secretsFile := GetFileContent(path.Join(gb.OutputDir, secretsFile))
		secretsMap := map[string]string{
			"AWSAccessKey":    "accesskey",
			"AWSAccessSecret": "accesssecret",
			"SecretFile":      "-----BEGIN CERTIFICATE-----\\nMIIDDDCCAfSgAwIBAgIRAJpYCmNgnRC42l6lqK7rxOowDQYJKoZIhvcNAQELBQAw\\nLzEtMCsGA1UEAxMkMzMzOTBhMDEtMTJiNi00NzViLWFiZjYtNmY4OGRhZTEyYmMz\\n-----END CERTIFICATE-----\\n",
		}
		for k, v := range secretsMap {
			assert.Contains(t, secretsFile, fmt.Sprintf("%s = %s", k, v))
		}
	})

	t.Run("should create output files for valid test template in use defaults as values mode with existing data passed on", func(t *testing.T) {
		gb := &GeneratedBlueprint{OutputDir: "xebialabs"}
		defer gb.Cleanup()
		data, doc, err := InstantiateBlueprint(
			BlueprintParams{
				TemplatePath:       "defaults-as-values-existing-data",
				AnswersFile:        "",
				StrictAnswers:      false,
				UseDefaultsAsValue: true,
				FromUpCommand:      false,
				PrintSummaryTable:  true,
				ExistingPreparedData: &PreparedData{
					TemplateData: map[string]interface{}{"Test": "testing", "TestDepends2": false, "AppName": "TestApp", "AWSAccessSecret": "accesssecret", "DiskSize": "10.0", "SecretFile": "../../templates/test/defaults-as-values/cert"},
					SummaryData:  map[string]interface{}{"Test Label": "testing", "TestDepends2": false, "AppName": "TestApp", "AWSAccessSecret": "*****", "DiskSize": "10.0", "SecretFile": "*****"},
					Values:       map[string]interface{}{"Test": "testing", "AppName": "TestApp", "DiskSize": "10.0"},
					Secrets:      map[string]interface{}{"AWSAccessSecret": "accesssecret", "SecretFile": "-----BEGIN CERTIFICATE-----\\nMIIDDDCCAfSgAwIBAgIRAJpYCmNgnRC42l6lqK7rxOowDQYJKoZIhvcNAQELBQAw\\n-----END CERTIFICATE-----\\n"},
				},
			},
			getLocalTestBlueprintContext(t),
			gb,
		)
		require.Nil(t, err)
		require.NotNil(t, data)
		require.NotNil(t, doc)

		// assertions
		assert.FileExists(t, "xld-environment.yml")
		assert.FileExists(t, "xld-infrastructure.yml")
		assert.FileExists(t, "xlr-pipeline.yml")
		assert.FileExists(t, path.Join(gb.OutputDir, valuesFile))
		assert.FileExists(t, path.Join(gb.OutputDir, secretsFile))
		assert.FileExists(t, path.Join(gb.OutputDir, gitignoreFile))

		// check __test__ directory is not there
		_, err = os.Stat("__test__")
		assert.True(t, os.IsNotExist(err))

		// check values file
		valsFile := GetFileContent(path.Join(gb.OutputDir, valuesFile))
		valueMap := map[string]string{
			"Test":               "testing",
			"ClientCert":         "this is a multiline\\ntext\\n\\nwith escape chars\\n",
			"AppName":            "TestApp",
			"SuperSecret":        "supersecret",
			"AWSRegion":          "eu-central-1",
			"DiskSize":           "10",
			"DiskSizeWithBuffer": "125.6",
			"ShouldNotBeThere":   "shouldnotbehere",
			"File":               "-----BEGIN CERTIFICATE-----\\nMIIDDDCCAfSgAwIBAgIRAJpYCmNgnRC42l6lqK7rxOowDQYJKoZIhvcNAQELBQAw\\nLzEtMCsGA1UEAxMkMzMzOTBhMDEtMTJiNi00NzViLWFiZjYtNmY4OGRhZTEyYmMz\\n-----END CERTIFICATE-----\\n",
		}
		for k, v := range valueMap {
			assert.Contains(t, valsFile, fmt.Sprintf("%s = %s", k, v))
		}

		// check secrets file
		secretsFile := GetFileContent(path.Join(gb.OutputDir, secretsFile))
		secretsMap := map[string]string{
			"AWSAccessKey":    "accesskey",
			"AWSAccessSecret": "accesssecret",
			"SecretFile":      "-----BEGIN CERTIFICATE-----\\nMIIDDDCCAfSgAwIBAgIRAJpYCmNgnRC42l6lqK7rxOowDQYJKoZIhvcNAQELBQAw\\n-----END CERTIFICATE-----\\n",
		}
		for k, v := range secretsMap {
			assert.Contains(t, secretsFile, fmt.Sprintf("%s = %s", k, v))
		}
	})

	t.Run("should create output files for valid test template in use defaults as values mode using expressions with valid k8s & aws config", func(t *testing.T) {

		// initialize temp dir for tests
		tmpDir, err := ioutil.TempDir("", "xltest")
		if err != nil {
			t.Error(err)
		}
		defer os.RemoveAll(tmpDir)

		// create test k8s config file
		// create test k8s config file
		d1 := []byte(SampleKubeConfig)
		ioutil.WriteFile(filepath.Join(tmpDir, "config"), d1, os.ModePerm)
		os.Setenv("KUBECONFIG", filepath.Join(tmpDir, "config"))
		os.Setenv("AWS_ACCESS_KEY_ID", "dummy_val")
		os.Setenv("AWS_SECRET_ACCESS_KEY", "dummy_val")

		gb := &GeneratedBlueprint{OutputDir: "xebialabs"}
		defer gb.Cleanup()

		data, doc, err := InstantiateBlueprint(
			BlueprintParams{
				TemplatePath:       "input-expression-tests",
				AnswersFile:        "",
				StrictAnswers:      false,
				UseDefaultsAsValue: true,
				FromUpCommand:      false,
				PrintSummaryTable:  true,
			},
			getLocalTestBlueprintContext(t),
			gb,
		)
		require.Nil(t, err)
		require.NotNil(t, data)
		require.NotNil(t, doc)

		// assertions
		assert.FileExists(t, path.Join(gb.OutputDir, valuesFile))
		assert.FileExists(t, path.Join(gb.OutputDir, secretsFile))
		assert.FileExists(t, path.Join(gb.OutputDir, gitignoreFile))

		// check __test__ directory is not there
		_, err = os.Stat("__test__")
		assert.True(t, os.IsNotExist(err))

		// check values file
		valsFile := GetFileContent(path.Join(gb.OutputDir, valuesFile))
		valueMap := map[string]string{
			"Provider":                    "AWS",
			"Service":                     "EKS",
			"K8sConfig":                   "true",
			"K8sClusterName":              "https://test.hcp.eastus.azmk8s.io:443",
			"UseAWSCredentialsFromSystem": "true",
			"AWSRegion":                   "ap-northeast-1",
		}
		for k, v := range valueMap {
			assert.Contains(t, valsFile, fmt.Sprintf("%s = %s", k, v))
		}

		// check secrets file
		secretsFile := GetFileContent(path.Join(gb.OutputDir, secretsFile))
		secretsMap := map[string]string{
			"AWSAccessKey":    "dummy_val",
			"AWSAccessSecret": "dummy_val",
		}
		for k, v := range secretsMap {
			assert.Contains(t, secretsFile, fmt.Sprintf("%s = %s", k, v))
		}
	})

	t.Run("should create output files for valid test template in use defaults as values mode using expressions without k8s & aws config", func(t *testing.T) {

		// initialize temp dir for tests
		tmpDir, err := ioutil.TempDir("", "xltest")
		if err != nil {
			t.Error(err)
		}
		defer os.RemoveAll(tmpDir)

		// create test k8s config file
		os.Setenv("KUBECONFIG", filepath.Join(tmpDir, "config"))
		os.Setenv("AWS_ACCESS_KEY_ID", "")
		os.Setenv("AWS_SECRET_ACCESS_KEY", "")
		os.Setenv("AWS_CONFIG_FILE", filepath.Join(tmpDir, "aws", "config"))
		os.Setenv("AWS_SHARED_CREDENTIALS_FILE", filepath.Join(tmpDir, "aws", "credentials"))

		gb := &GeneratedBlueprint{OutputDir: "xebialabs"}
		defer gb.Cleanup()

		data, doc, err := InstantiateBlueprint(
			BlueprintParams{
				TemplatePath:       "input-expression-tests",
				AnswersFile:        "",
				StrictAnswers:      false,
				UseDefaultsAsValue: true,
				FromUpCommand:      false,
				PrintSummaryTable:  true,
			},
			getLocalTestBlueprintContext(t),
			gb,
		)
		require.Nil(t, err)
		require.NotNil(t, data)
		require.NotNil(t, doc)

		// assertions
		assert.FileExists(t, path.Join(gb.OutputDir, valuesFile))
		assert.FileExists(t, path.Join(gb.OutputDir, secretsFile))
		assert.FileExists(t, path.Join(gb.OutputDir, gitignoreFile))

		// check __test__ directory is not there
		_, err = os.Stat("__test__")
		assert.True(t, os.IsNotExist(err))

		// check values file
		valsFile := GetFileContent(path.Join(gb.OutputDir, valuesFile))
		valueMap := map[string]string{
			"Provider":                    "AWS",
			"Service":                     "EKS",
			"K8sConfig":                   "false",
			"K8sClusterName":              "defaultVal",
			"UseAWSCredentialsFromSystem": "false",
			"AWSRegion":                   "ap-northeast-1",
		}
		for k, v := range valueMap {
			assert.Contains(t, valsFile, fmt.Sprintf("%s = %s", k, v))
		}

		// check secrets file
		secretsFile := GetFileContent(path.Join(gb.OutputDir, secretsFile))
		secretsMap := map[string]string{
			"AWSAccessKey":    "defaultVal",
			"AWSAccessSecret": "defaultVal",
		}
		for k, v := range secretsMap {
			assert.Contains(t, secretsFile, fmt.Sprintf("%s = %s", k, v))
		}
	})

	t.Run("should create output files for valid test template from local path", func(t *testing.T) {
		gb := &GeneratedBlueprint{OutputDir: "xebialabs"}
		defer gb.Cleanup()
		data, doc, err := InstantiateBlueprint(
			BlueprintParams{
				TemplatePath:       "valid-no-prompt",
				AnswersFile:        "",
				StrictAnswers:      false,
				UseDefaultsAsValue: false,
				FromUpCommand:      false,
				PrintSummaryTable:  true,
			},
			getLocalTestBlueprintContext(t),
			gb,
		)
		require.Nil(t, err)
		require.NotNil(t, data)
		require.NotNil(t, doc)

		// assertions
		assert.FileExists(t, "xld-environment.yml")
		assert.FileExists(t, "xld-infrastructure.yml")
		assert.FileExists(t, "xlr-pipeline-2.yml")
		assert.False(t, util.PathExists("xlr-pipeline.yml", false))
		assert.FileExists(t, path.Join(gb.OutputDir, valuesFile))
		assert.FileExists(t, path.Join(gb.OutputDir, secretsFile))
		assert.FileExists(t, path.Join(gb.OutputDir, gitignoreFile))
		envFile := GetFileContent("xld-environment.yml")
		assert.Contains(t, envFile, fmt.Sprintf("region: %s", "us-west"))
		infraFile := GetFileContent("xld-infrastructure.yml")
		infraChecks := []string{
			fmt.Sprintf("- name: %s-ecs-fargate-cluster", "testApp"),
			fmt.Sprintf("- name: %s-ecs-vpc", "testApp"),
			fmt.Sprintf("- name: %s-ecs-subnet-ipv4-az-1a", "testApp"),
			fmt.Sprintf("- name: %s-ecs-route-table", "testApp"),
			fmt.Sprintf("- name: %s-ecs-security-group", "testApp"),
			fmt.Sprintf("- name: %s-targetgroup", "testApp"),
			fmt.Sprintf("- name: %s-ecs-alb", "testApp"),
			fmt.Sprintf("- name: %s-ecs-db-subnet-group", "testApp"),
			fmt.Sprintf("- name: %s-ecs-dictionary", "testApp"),
			"MYSQL_DB_ADDRESS: '{{%address%}}'",
		}
		for _, infraCheck := range infraChecks {
			assert.Contains(t, infraFile, infraCheck)
		}

		// Check if only saveInXlvals marked fields are in values.xlvals
		valuesFileContent := GetFileContent(models.BlueprintOutputDir + string(os.PathSeparator) + valuesFile)
		assert.Contains(t, valuesFileContent, "TestFoo = testing")
		assert.Contains(t, GetFileContent(path.Join(gb.OutputDir, valuesFile)), `FshYmQzRUNbYTA4Icc3V7JEgLXMNjcSLY9L1H4XQD79coMBRbbJFtOsp0Yk2btCKCAYLio0S8Jw85W5mgpLkasvCrXO5\nQJGxFvtQc2tHGLj0kNzM9KyAqbUJRe1l40TqfMdscEaWJimtd4oygqVc6y7zW1Wuj1EcDUvMD8qK8FEWfQgm5ilBIldQ\nomhDPbq8F84KRsRwCgT05mTrxhBtgqGuCHXcr115iUuUNW7dzzP5iXAgEp4Apa30NHzNsy5TUoIZGDJceO2BmAAmG4HS0cZ\notIXJ2BJEx95SGnqO4kZFoRJzghlZMWs50PkskI5JTM6tHFmtZIdYbo7ZvbA0LP71QtTbSDziqDzXoi5uwxpwaDO95fPYv0\nN1ajgotzn4czgX4hA8gFIipmUUA2AYfgQ5jZQ4I9zO5rxxj80lPWFNOnrHzD1jWZAhLgdpyWldWLt9NbcWegrgLpI\nhRA08PILJnV2z79aTfylL7Y3zJ2urSjr0XIbTWQlWwZ1VXBm13IbRffbku0qjFmSuxDrKFCwGEBtRZ4RnseholT8DA0yDIjPCsfY2jo\nCjljgZHYRoIe4E8WsMt0zzp9G0UP7It6jzJok3yk9Ril48yLthkPvyJ4qoH2PTLx8xBeGBJLKmHT9ojDbWQxOXpml72ati\n4jcxmZfSgDUqMPmTRHPqZ47k6f3XTrPxqIDJ8SzOj09OaKzjSYyZnxIEokm1JotTaqhZa64zptKlbuY0kblSbFAGFFQZnn7RjkU3ZKq872gTDh\nAdteR98sbMdmMGipaxgYbCfuomBEdxldjlApbwDiswJkOQIY0Vypwt95M3LAWha4zACRwrYz7rVqDBJqpo6hFh3V6zBRQR2C6GINUJZq3KWWz\nXAI0ncPo95GDraIFnaStGFHu6R1WC7oopSFS6kgbhJL6noGgMjxbmnPzDA8sXVo1GEtyq79oG2CTHBbrODI9KhsKYy3B0\n8Prpu561H6kDtwIyZqZQXHppVaeFbrGlWAsQpp5su5iHhfFllVaCsDI8kYmmy4JdtOEmPYNL3pF7Uf35X0LIdJKb54czjwBuc2rbbifX9mIn30I8tTgq\n9ldZFjj0SwtTxN1hjYh5pRRTdKZkuwNv6v9L0iPitR6YwuCQaIx1LlymGwfR1Zo6u4gLDCqBYjLz2s1jc7o5dhdmVXmMHKFjWrTaVbanLiwJuNWDQb1e14UikLg\nP4l6RiCx5nNF2wbSQ7uYrvDpYa6ToKysXVUTAPLxG3C4BirrQDaSnTThTzMC7GUAmxKAK3tnBHXEqOIsnYZ3rD92iUr2XI65oFIbIT\nXUrYNapiDWYsPEGTaQTX8L1ZkrFaQTL8wC1Zko8aZFfzqmYbNi5OvJydnWWoaRc0eyvnFmtNh0utLQZEME4DXCU3RxET3q6pwsid8DolT1FZtWBE0V3F0XM\nffWx27IYj63dyTtT4UoJwtTgdtXeHAG4a0AGvbfM9p462qEbV3rMNynLWyzQDc3sN6nI-`)
		assert.Contains(t, valuesFileContent, "DiskSizeWithBuffer = 125.1")

		// Check if only secret marked fields are in values.xlvals
		secretsFileContent := GetFileContent(models.BlueprintOutputDir + string(os.PathSeparator) + secretsFile)

		assert.Contains(t, secretsFileContent, "AWSAccessKey = accesskey")
		assert.Contains(t, secretsFileContent, "AWSAccessSecret = accesssecret")
		assert.NotContains(t, secretsFileContent, "SuperSecret = invisible")

	})

	t.Run("should create output files for valid test template with SuppressXebiaLabsFolder enabled", func(t *testing.T) {
		gb := &GeneratedBlueprint{OutputDir: "xebialabs"}
		defer gb.Cleanup()
		data, doc, err := InstantiateBlueprint(
			BlueprintParams{
				TemplatePath:       "valid-no-prompt-suppress-xebia-labs-folder",
				AnswersFile:        "",
				StrictAnswers:      false,
				UseDefaultsAsValue: false,
				FromUpCommand:      false,
				PrintSummaryTable:  true,
			},
			getLocalTestBlueprintContext(t),
			gb,
		)
		require.Nil(t, err)
		require.NotNil(t, data)
		require.NotNil(t, doc)

		// assertions
		assert.FileExists(t, "xld-environment.yml")
		assert.FileExists(t, "xld-infrastructure.yml")
		assert.FileExists(t, "xlr-pipeline-2.yml")
		assert.False(t, util.PathExists("xlr-pipeline.yml", false))

		assert.False(t, util.PathExists(path.Join(gb.OutputDir, valuesFile), false))
		assert.False(t, util.PathExists(path.Join(gb.OutputDir, secretsFile), false))
		assert.False(t, util.PathExists(path.Join(gb.OutputDir, gitignoreFile), false))

		envFile := GetFileContent("xld-environment.yml")
		assert.Contains(t, envFile, fmt.Sprintf("region: %s", "us-west"))
		infraFile := GetFileContent("xld-infrastructure.yml")
		infraChecks := []string{
			fmt.Sprintf("- name: %s-ecs-fargate-cluster", "testApp"),
			fmt.Sprintf("- name: %s-ecs-vpc", "testApp"),
			fmt.Sprintf("- name: %s-ecs-subnet-ipv4-az-1a", "testApp"),
			fmt.Sprintf("- name: %s-ecs-route-table", "testApp"),
			fmt.Sprintf("- name: %s-ecs-security-group", "testApp"),
			fmt.Sprintf("- name: %s-targetgroup", "testApp"),
			fmt.Sprintf("- name: %s-ecs-alb", "testApp"),
			fmt.Sprintf("- name: %s-ecs-db-subnet-group", "testApp"),
			fmt.Sprintf("- name: %s-ecs-dictionary", "testApp"),
			"MYSQL_DB_ADDRESS: '{{%address%}}'",
		}
		for _, infraCheck := range infraChecks {
			assert.Contains(t, infraFile, infraCheck)
		}
	})

	t.Run("should create output files for valid test template composed from local path", func(t *testing.T) {
		gb := &GeneratedBlueprint{OutputDir: "xebialabs"}
		defer gb.Cleanup()
		data, doc, err := InstantiateBlueprint(
			BlueprintParams{
				TemplatePath:       "composed",
				AnswersFile:        "",
				StrictAnswers:      false,
				UseDefaultsAsValue: true,
				FromUpCommand:      false,
				PrintSummaryTable:  true,
			},
			getLocalTestBlueprintContext(t),
			gb,
		)
		require.Nil(t, err)
		require.NotNil(t, data)
		require.NotNil(t, doc)

		// assertions
		assert.False(t, util.PathExists("xld-environment.yml", false))   // this file is skipped when composing
		assert.True(t, util.PathExists("xld-infrastructure.yml", false)) // this comes from composed blueprint 'defaults-as-values'
		assert.False(t, util.PathExists("xlr-pipeline.yml", false))      // this file is renamed when composing
		assert.True(t, util.PathExists("xlr-pipeline-new2.yml", false))  // this comes from composed blueprint 'defaults-as-values'
		assert.True(t, util.PathExists("xlr-pipeline-new.yml", false))   // this comes from composed blueprint 'valid-no-prompt'
		assert.True(t, util.PathExists("xlr-pipeline-4.yml", false))     // this comes from blueprint 'composed'

		// these files are from the main blueprint 'composed'
		assert.FileExists(t, path.Join(gb.OutputDir, valuesFile))
		assert.FileExists(t, path.Join(gb.OutputDir, secretsFile))
		assert.FileExists(t, path.Join(gb.OutputDir, gitignoreFile))

		infraFile := GetFileContent("xld-infrastructure.yml")
		// the values are overridden by the last blueprint composed
		infraChecks := []string{
			fmt.Sprintf("- name: %s-ecs-fargate-cluster", "TestApp"),
			fmt.Sprintf("- name: %s-ecs-vpc", "TestApp"),
			fmt.Sprintf("- name: %s-ecs-subnet-ipv4-az-1a", "TestApp"),
			fmt.Sprintf("- name: %s-ecs-route-table", "TestApp"),
			fmt.Sprintf("- name: %s-ecs-security-group", "TestApp"),
			fmt.Sprintf("- name: %s-targetgroup", "TestApp"),
			fmt.Sprintf("- name: %s-ecs-alb", "TestApp"),
			fmt.Sprintf("- name: %s-ecs-db-subnet-group", "TestApp"),
			fmt.Sprintf("- name: %s-ecs-dictionary", "TestApp"),
			"MYSQL_DB_ADDRESS: '{{%address%}}'",
		}
		for _, infraCheck := range infraChecks {
			assert.Contains(t, infraFile, infraCheck)
		}

		// the values are overridden by the last blueprint composed
		// Check if only secret marked fields are in values.xlvals
		secretsFileContent := GetFileContent(models.BlueprintOutputDir + string(os.PathSeparator) + secretsFile)

		assert.Contains(t, secretsFileContent, "AWSAccessKey = accesskey")
		assert.Contains(t, secretsFileContent, "AWSAccessSecret = accesssecret")
		assert.NotContains(t, secretsFileContent, "SuperSecret = invisible")

		// check __test__ directory is not there
		_, err = os.Stat("__test__")
		assert.True(t, os.IsNotExist(err))

		// check values file
		valsFile := GetFileContent(path.Join(gb.OutputDir, valuesFile))
		valueMap := map[string]string{
			"Test":               "hello", // value from parameterOverride using !expr "TestCompose"
			"TestFoo":            "hello", // value from parameterOverride
			"TestCompose":        "hello", // value from parameterOverride using !expr "TestFoo"
			"ClientCert":         "this is a multiline\\ntext\\n\\nwith escape chars\\n",
			"AppName":            "TestApp",
			"SuperSecret":        "supersecret",
			"AWSRegion":          "eu-central-1",
			"DiskSize":           "10",
			"DiskSizeWithBuffer": "125.6",
			"ShouldNotBeThere":   "shouldnotbehere",
		}
		for k, v := range valueMap {
			assert.Contains(t, valsFile, fmt.Sprintf("%s = %s", k, v))
		}

		// check secrets file
		secretsFile := GetFileContent(path.Join(gb.OutputDir, secretsFile))
		secretsMap := map[string]string{
			"AWSAccessKey":    "accesskey",
			"AWSAccessSecret": "accesssecret",
		}
		for k, v := range secretsMap {
			assert.Contains(t, secretsFile, fmt.Sprintf("%s = %s", k, v))
		}

	})

	t.Run("should create output files for valid test nested templates composed from local path", func(t *testing.T) {
		gb := &GeneratedBlueprint{OutputDir: "xebialabs"}
		defer gb.Cleanup()
		// This can be used to debug a local blueprint if you have the repo in ../blueprints relative to xl-cli
		/* 		pwd, _ := os.Getwd()
		   		BlueprintTestPath = strings.Replace(pwd, path.Join("xl-cli", "pkg", "blueprint"), path.Join("blueprints"), -1)
		   		data, doc, err := InstantiateBlueprint(
		   			BlueprintParams{
		   				TemplatePath:       "gcp/microservice-ecommerce",
		   				AnswersFile:        BlueprintTestPath + "/gcp/microservice-ecommerce/__test__/answers-with-cluster-with-cicd.yaml",
		   				StrictAnswers:      false,
		   				UseDefaultsAsValue: true,
		   				FromUpCommand:      false,
		   				PrintSummaryTable:  true,
		   			},
		   			getLocalTestBlueprintContext(t),
		   			gb,
		   		) */
		data, doc, err := InstantiateBlueprint(
			BlueprintParams{
				TemplatePath:       "compose-nested",
				AnswersFile:        "",
				StrictAnswers:      false,
				UseDefaultsAsValue: true,
				FromUpCommand:      false,
				PrintSummaryTable:  true,
			},
			getLocalTestBlueprintContext(t),
			gb,
		)
		require.Nil(t, err)
		require.NotNil(t, data)
		require.NotNil(t, doc)

		// assertions
		assert.False(t, util.PathExists("xld-environment.yml", false))   // this file is skipped when composing
		assert.True(t, util.PathExists("xld-infrastructure.yml", false)) // this comes from composed blueprint 'defaults-as-values'
		assert.False(t, util.PathExists("xlr-pipeline.yml", false))      // this file is renamed when composing
		assert.True(t, util.PathExists("xlr-pipeline-new2.yml", false))  // this comes from composed blueprint 'defaults-as-values'
		assert.True(t, util.PathExists("xlr-pipeline-new.yml", false))   // this comes from composed blueprint 'valid-no-prompt'
		assert.True(t, util.PathExists("xlr-pipeline-4.yml", false))     // this comes from blueprint 'composed'

		// these files are from the main blueprint 'composed'
		assert.FileExists(t, path.Join(gb.OutputDir, valuesFile))
		assert.FileExists(t, path.Join(gb.OutputDir, secretsFile))
		assert.FileExists(t, path.Join(gb.OutputDir, gitignoreFile))

		infraFile := GetFileContent("xld-infrastructure.yml")
		// the values are overridden by the last blueprint composed
		infraChecks := []string{
			fmt.Sprintf("- name: %s-ecs-fargate-cluster", "TestApp"),
			fmt.Sprintf("- name: %s-ecs-vpc", "TestApp"),
			fmt.Sprintf("- name: %s-ecs-subnet-ipv4-az-1a", "TestApp"),
			fmt.Sprintf("- name: %s-ecs-route-table", "TestApp"),
			fmt.Sprintf("- name: %s-ecs-security-group", "TestApp"),
			fmt.Sprintf("- name: %s-targetgroup", "TestApp"),
			fmt.Sprintf("- name: %s-ecs-alb", "TestApp"),
			fmt.Sprintf("- name: %s-ecs-db-subnet-group", "TestApp"),
			fmt.Sprintf("- name: %s-ecs-dictionary", "TestApp"),
			"MYSQL_DB_ADDRESS: '{{%address%}}'",
		}
		for _, infraCheck := range infraChecks {
			assert.Contains(t, infraFile, infraCheck)
		}

		// the values are overridden by the last blueprint composed
		// Check if only secret marked fields are in values.xlvals
		secretsFileContent := GetFileContent(models.BlueprintOutputDir + string(os.PathSeparator) + secretsFile)

		assert.Contains(t, secretsFileContent, "AWSAccessKey = accesskey")
		assert.Contains(t, secretsFileContent, "AWSAccessSecret = accesssecret")
		assert.NotContains(t, secretsFileContent, "SuperSecret = invisible")

		// check __test__ directory is not there
		_, err = os.Stat("__test__")
		assert.True(t, os.IsNotExist(err))

		// check values file
		valsFile := GetFileContent(path.Join(gb.OutputDir, valuesFile))
		valueMap := map[string]string{
			"Test":               "TestComposeTrue", // value from parameterOverride using !expr "TestCompose"
			"TestFoo":            "hello",           // value from parameterOverride
			"TestCompose":        "TestComposeTrue", // value from parameterOverride using !expr "TestComposeTrue"
			"ClientCert":         "this is a multiline\\ntext\\n\\nwith escape chars\\n",
			"AppName":            "TestApp",
			"SuperSecret":        "supersecret",
			"AWSRegion":          "eu-central-1",
			"DiskSize":           "10",
			"DiskSizeWithBuffer": "125.6",
			"ShouldNotBeThere":   "shouldnotbehere",
		}
		for k, v := range valueMap {
			assert.Contains(t, valsFile, fmt.Sprintf("%s = %s", k, v))
		}

		// check secrets file
		secretsFile := GetFileContent(path.Join(gb.OutputDir, secretsFile))
		secretsMap := map[string]string{
			"AWSAccessKey":    "accesskey",
			"AWSAccessSecret": "accesssecret",
		}
		for k, v := range secretsMap {
			assert.Contains(t, secretsFile, fmt.Sprintf("%s = %s", k, v))
		}

	})

	// V1 schema test
	t.Run("should create output files for valid test template from local path for schema V1", func(t *testing.T) {
		gb := &GeneratedBlueprint{OutputDir: "xebialabs"}
		defer gb.Cleanup()
		data, doc, err := InstantiateBlueprint(
			BlueprintParams{
				TemplatePath:       "valid-no-prompt-v1",
				AnswersFile:        "",
				StrictAnswers:      false,
				UseDefaultsAsValue: false,
				FromUpCommand:      false,
				PrintSummaryTable:  true,
			},
			getLocalTestBlueprintContext(t),
			gb,
		)
		require.Nil(t, err)
		require.NotNil(t, data)
		require.NotNil(t, doc)

		// assertions
		assert.FileExists(t, "xld-environment.yml")
		assert.FileExists(t, "xld-infrastructure.yml")
		assert.FileExists(t, "xlr-pipeline-2.yml")
		assert.False(t, util.PathExists("xlr-pipeline.yml", false))
		assert.FileExists(t, path.Join(gb.OutputDir, valuesFile))
		assert.FileExists(t, path.Join(gb.OutputDir, secretsFile))
		assert.FileExists(t, path.Join(gb.OutputDir, gitignoreFile))
		envFile := GetFileContent("xld-environment.yml")
		assert.Contains(t, envFile, fmt.Sprintf("region: %s", "us-west"))
		infraFile := GetFileContent("xld-infrastructure.yml")
		infraChecks := []string{
			fmt.Sprintf("- name: %s-ecs-fargate-cluster", "testApp"),
			fmt.Sprintf("- name: %s-ecs-vpc", "testApp"),
			fmt.Sprintf("- name: %s-ecs-subnet-ipv4-az-1a", "testApp"),
			fmt.Sprintf("- name: %s-ecs-route-table", "testApp"),
			fmt.Sprintf("- name: %s-ecs-security-group", "testApp"),
			fmt.Sprintf("- name: %s-targetgroup", "testApp"),
			fmt.Sprintf("- name: %s-ecs-alb", "testApp"),
			fmt.Sprintf("- name: %s-ecs-db-subnet-group", "testApp"),
			fmt.Sprintf("- name: %s-ecs-dictionary", "testApp"),
			"MYSQL_DB_ADDRESS: '{{%address%}}'",
		}
		for _, infraCheck := range infraChecks {
			assert.Contains(t, infraFile, infraCheck)
		}

		// Check if only saveInXlvals marked fields are in values.xlvals
		valuesFileContent := GetFileContent(models.BlueprintOutputDir + string(os.PathSeparator) + valuesFile)
		assert.Contains(t, valuesFileContent, "TestFoo = testing")
		assert.Contains(t, GetFileContent(path.Join(gb.OutputDir, valuesFile)), `FshYmQzRUNbYTA4Icc3V7JEgLXMNjcSLY9L1H4XQD79coMBRbbJFtOsp0Yk2btCKCAYLio0S8Jw85W5mgpLkasvCrXO5\nQJGxFvtQc2tHGLj0kNzM9KyAqbUJRe1l40TqfMdscEaWJimtd4oygqVc6y7zW1Wuj1EcDUvMD8qK8FEWfQgm5ilBIldQ\nomhDPbq8F84KRsRwCgT05mTrxhBtgqGuCHXcr115iUuUNW7dzzP5iXAgEp4Apa30NHzNsy5TUoIZGDJceO2BmAAmG4HS0cZ\notIXJ2BJEx95SGnqO4kZFoRJzghlZMWs50PkskI5JTM6tHFmtZIdYbo7ZvbA0LP71QtTbSDziqDzXoi5uwxpwaDO95fPYv0\nN1ajgotzn4czgX4hA8gFIipmUUA2AYfgQ5jZQ4I9zO5rxxj80lPWFNOnrHzD1jWZAhLgdpyWldWLt9NbcWegrgLpI\nhRA08PILJnV2z79aTfylL7Y3zJ2urSjr0XIbTWQlWwZ1VXBm13IbRffbku0qjFmSuxDrKFCwGEBtRZ4RnseholT8DA0yDIjPCsfY2jo\nCjljgZHYRoIe4E8WsMt0zzp9G0UP7It6jzJok3yk9Ril48yLthkPvyJ4qoH2PTLx8xBeGBJLKmHT9ojDbWQxOXpml72ati\n4jcxmZfSgDUqMPmTRHPqZ47k6f3XTrPxqIDJ8SzOj09OaKzjSYyZnxIEokm1JotTaqhZa64zptKlbuY0kblSbFAGFFQZnn7RjkU3ZKq872gTDh\nAdteR98sbMdmMGipaxgYbCfuomBEdxldjlApbwDiswJkOQIY0Vypwt95M3LAWha4zACRwrYz7rVqDBJqpo6hFh3V6zBRQR2C6GINUJZq3KWWz\nXAI0ncPo95GDraIFnaStGFHu6R1WC7oopSFS6kgbhJL6noGgMjxbmnPzDA8sXVo1GEtyq79oG2CTHBbrODI9KhsKYy3B0\n8Prpu561H6kDtwIyZqZQXHppVaeFbrGlWAsQpp5su5iHhfFllVaCsDI8kYmmy4JdtOEmPYNL3pF7Uf35X0LIdJKb54czjwBuc2rbbifX9mIn30I8tTgq\n9ldZFjj0SwtTxN1hjYh5pRRTdKZkuwNv6v9L0iPitR6YwuCQaIx1LlymGwfR1Zo6u4gLDCqBYjLz2s1jc7o5dhdmVXmMHKFjWrTaVbanLiwJuNWDQb1e14UikLg\nP4l6RiCx5nNF2wbSQ7uYrvDpYa6ToKysXVUTAPLxG3C4BirrQDaSnTThTzMC7GUAmxKAK3tnBHXEqOIsnYZ3rD92iUr2XI65oFIbIT\nXUrYNapiDWYsPEGTaQTX8L1ZkrFaQTL8wC1Zko8aZFfzqmYbNi5OvJydnWWoaRc0eyvnFmtNh0utLQZEME4DXCU3RxET3q6pwsid8DolT1FZtWBE0V3F0XM\nffWx27IYj63dyTtT4UoJwtTgdtXeHAG4a0AGvbfM9p462qEbV3rMNynLWyzQDc3sN6nI-`)
		assert.Contains(t, valuesFileContent, "DiskSizeWithBuffer = 125.1")

		// Check if only secret marked fields are in values.xlvals
		secretsFileContent := GetFileContent(models.BlueprintOutputDir + string(os.PathSeparator) + secretsFile)

		assert.Contains(t, secretsFileContent, "AWSAccessKey = accesskey")
		assert.Contains(t, secretsFileContent, "AWSAccessSecret = accesssecret")
		assert.NotContains(t, secretsFileContent, "SuperSecret = invisible")

	})
}

func TestShouldSkipFile(t *testing.T) {
	type args struct {
		templateConfig TemplateConfig
		params         map[string]interface{}
	}
	tests := []struct {
		name    string
		args    args
		want    bool
		wantErr bool
	}{
		{
			"should return false if dependsOn not defined",
			args{
				TemplateConfig{Path: "foo.yaml"},
				nil,
			},
			false,
			false,
		},
		{
			"should return true if dependsOn is defined and its value is false",
			args{
				TemplateConfig{Path: "foo.yaml", DependsOn: VarField{Value: "foo"}},
				map[string]interface{}{
					"foo": false,
				},
			},
			true,
			false,
		},
		{
			"should return true if dependsOnFalse is defined and its value is true",
			args{
				TemplateConfig{Path: "foo.yaml", DependsOn: VarField{Value: "foo", InvertBool: true}},
				map[string]interface{}{
					"foo": true,
				},
			},
			true,
			false,
		},
		{
			"should return error if dependsOn value cannot be processed",
			args{
				TemplateConfig{Path: "foo.yaml", DependsOn: VarField{Value: "foo", InvertBool: true}},
				map[string]interface{}{},
			},
			false,
			true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := shouldSkipFile(tt.args.templateConfig, tt.args.params)
			if (err != nil) != tt.wantErr {
				t.Errorf("shouldSkipFile() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("shouldSkipFile() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_getBlueprintConfig(t *testing.T) {
	defer httpmock.DeactivateAndReset()
	repo := getMockHttpBlueprintContext(t)
	blueprints, err := repo.initCurrentRepoClient()
	require.Nil(t, err)
	require.NotNil(t, blueprints)

	type args struct {
		blueprintContext *BlueprintContext
		blueprints       map[string]*models.BlueprintRemote
		templatePath     string
		dependsOn        VarField
		parentName       string
	}
	type ComposedBlueprintPtr = []*ComposedBlueprint
	tests := []struct {
		name            string
		args            args
		wantProjectName string
		wantArray       ComposedBlueprintPtr
		wantErr         bool
	}{
		{
			"should error when invalid path is passed",
			args{
				repo,
				blueprints,
				"test",
				VarField{},
				"",
			},
			"",
			nil,
			true,
		},
		{
			"should get blueprint config for a simple definition without composed includes",
			args{
				repo,
				blueprints,
				"aws/monolith",
				VarField{},
				"",
			},
			"Test Project",
			ComposedBlueprintPtr{
				{
					Name: "aws/monolith",
					BlueprintConfig: &BlueprintConfig{
						ApiVersion: "xl/v2",
						Kind:       "Blueprint",
						Metadata:   Metadata{Name: "Test Project"},
						TemplateConfigs: []TemplateConfig{
							{Path: "xld-environment.yml.tmpl", FullPath: "aws/monolith/xld-environment.yml.tmpl"},
							{Path: "xld-infrastructure.yml.tmpl", FullPath: "aws/monolith/xld-infrastructure.yml.tmpl"},
							{Path: "xlr-pipeline.yml", FullPath: "aws/monolith/xlr-pipeline.yml"},
						},
						Include: []IncludedBlueprintProcessed{},
						Variables: []Variable{
							{Name: VarField{Value: "Test"}, Label: VarField{Value: "Test"}, Value: VarField{Value: "testing"}, SaveInXlvals: VarField{Value: "true", Bool: true}},
						},
					},
					DependsOn: []VarField{{}},
				},
			},
			false,
		},
		{
			"should get blueprint config for a simple definition with composed includes",
			args{
				repo,
				blueprints,
				"aws/compose",
				VarField{},
				"",
			},
			"Test Project",
			ComposedBlueprintPtr{
				{
					Name:   "aws/monolith",
					Parent: "aws/compose",
					BlueprintConfig: &BlueprintConfig{
						ApiVersion: "xl/v2",
						Kind:       "Blueprint",
						Metadata:   Metadata{Name: "Test Project"},
						Include:    []IncludedBlueprintProcessed{},
						Variables: []Variable{
							{
								Name:         VarField{Value: "Test"},
								Label:        VarField{Value: "Test"},
								Value:        VarField{Value: "hello"},
								SaveInXlvals: VarField{Value: "true", Bool: true},
								DependsOn:    VarField{Tag: tagExpressionV2, Value: "2 > 1"},
							},
						},
						TemplateConfigs: []TemplateConfig{
							{Path: "xld-environment.yml.tmpl", FullPath: "aws/monolith/xld-environment.yml.tmpl"},
							{Path: "xld-infrastructure.yml.tmpl", FullPath: "aws/monolith/xld-infrastructure.yml.tmpl", DependsOn: VarField{Tag: tagExpressionV2, Value: "false"}},
							{Path: "xlr-pipeline.yml", FullPath: "aws/monolith/xlr-pipeline.yml"},
						},
					},
					DependsOn: []VarField{{}},
				},
				{
					Name:   "aws/compose",
					Parent: "",
					BlueprintConfig: &BlueprintConfig{
						ApiVersion: "xl/v2",
						Kind:       "Blueprint",
						Metadata:   Metadata{Name: "Test Project"},
						Include: []IncludedBlueprintProcessed{
							IncludedBlueprintProcessed{
								Blueprint: "aws/monolith",
								Stage:     "before",
								ParameterOverrides: []Variable{
									{
										Name:      VarField{Value: "Test"},
										Value:     VarField{Value: "hello"},
										DependsOn: VarField{Tag: tagExpressionV2, Value: "2 > 1"},
									},
								},
								FileOverrides: []TemplateConfig{
									{
										Path:      "xld-infrastructure.yml.tmpl",
										DependsOn: VarField{Tag: tagExpressionV2, Value: "false"},
									},
								},
							},
							IncludedBlueprintProcessed{
								Blueprint: "aws/datalake",
								Stage:     "after",
								ParameterOverrides: []Variable{
									{
										Name:  VarField{Value: "Foo"},
										Value: VarField{Value: "hello"},
									},
								},
								DependsOn: VarField{Tag: tagExpressionV2, Value: "Bar == 'testing'"},
								FileOverrides: []TemplateConfig{
									{
										Path:      "xlr-pipeline.yml",
										RenameTo:  VarField{Value: "xlr-pipeline2-new.yml"},
										DependsOn: VarField{Value: "TestDepends"},
									},
								},
							},
						},
						Variables: []Variable{
							{Name: VarField{Value: "Bar"}, Label: VarField{Value: "Bar"}, Value: VarField{Value: "testing"}},
						},
						TemplateConfigs: []TemplateConfig{
							{Path: "xld-environment.yml.tmpl", FullPath: "aws/compose/xld-environment.yml.tmpl"},
							{Path: "xld-infrastructure.yml.tmpl", FullPath: "aws/compose/xld-infrastructure.yml.tmpl"},
							{Path: "xlr-pipeline.yml", FullPath: "aws/compose/xlr-pipeline.yml"},
						},
					},
					DependsOn: []VarField{{}},
				},
				{
					Name:   "aws/datalake",
					Parent: "aws/compose",
					BlueprintConfig: &BlueprintConfig{
						ApiVersion: "xl/v2",
						Kind:       "Blueprint",
						Metadata:   Metadata{Name: "Test Project 2"},
						Include:    []IncludedBlueprintProcessed{},
						Variables: []Variable{
							{Name: VarField{Value: "Foo"}, Label: VarField{Value: "Foo"}, Value: VarField{Value: "hello"}},
						},
						TemplateConfigs: []TemplateConfig{
							{Path: "xld-app.yml.tmpl", FullPath: "aws/datalake/xld-app.yml.tmpl"},
							{
								Path: "xlr-pipeline.yml", FullPath: "aws/datalake/xlr-pipeline.yml",
								RenameTo:  VarField{Value: "xlr-pipeline2-new.yml"},
								DependsOn: VarField{Value: "TestDepends"},
							},
						},
					},
					DependsOn: []VarField{{}, {Tag: tagExpressionV2, Value: "Bar == 'testing'"}},
				},
			},
			false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotArray, got, err := getBlueprintConfig(tt.args.blueprintContext, tt.args.blueprints, tt.args.templatePath, []VarField{tt.args.dependsOn}, tt.args.parentName)
			if (err != nil) != tt.wantErr {
				t.Errorf("getBlueprintConfig() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			assert.Equal(t, tt.wantArray, gotArray)
			if !tt.wantErr {
				assert.Equal(t, tt.wantProjectName, got.Metadata.Name)
			}
		})
	}
}

func Test_composeBlueprints(t *testing.T) {
	defer httpmock.DeactivateAndReset()
	repo := getMockHttpBlueprintContext(t)
	blueprints, err := repo.initCurrentRepoClient()
	require.Nil(t, err)
	require.NotNil(t, blueprints)

	type args struct {
		blueprintName    string
		blueprintDoc     *BlueprintConfig
		blueprintContext *BlueprintContext
		blueprints       map[string]*models.BlueprintRemote
		dependsOn        VarField
		parentName       string
	}

	type ComposedBlueprintPtr = []*ComposedBlueprint
	tests := []struct {
		name    string
		args    args
		want    ComposedBlueprintPtr
		wantErr bool
	}{
		{
			"should error when invalid config is passed",
			args{
				"aws/test",
				&BlueprintConfig{
					Include: []IncludedBlueprintProcessed{
						{
							Blueprint: "aws/nonexisting",
							Stage:     "after",
						},
					},
				},
				repo,
				blueprints,
				VarField{},
				"bpParent",
			},
			nil,
			true,
		},
		{
			"should not fail when blueprints with empty param or files are passed",
			args{
				"aws/test",
				&BlueprintConfig{
					ApiVersion: "xl/v2",
					Kind:       "Blueprint",
					Metadata:   Metadata{Name: "Test Project"},
					Include: []IncludedBlueprintProcessed{
						{
							Blueprint: "aws/emptyfiles",
						},
						{
							Blueprint: "aws/emptyparams",
						},
					},
					Variables: []Variable{
						{Name: VarField{Value: "Bar"}, Label: VarField{Value: "Bar"}, Value: VarField{Value: "testing"}},
					},
					TemplateConfigs: []TemplateConfig{
						{Path: "xld-environment.yml.tmpl", FullPath: "aws/compose/xld-environment.yml.tmpl"},
						{Path: "xld-infrastructure.yml.tmpl", FullPath: "aws/compose/xld-infrastructure.yml.tmpl"},
						{Path: "xlr-pipeline.yml", FullPath: "aws/compose/xlr-pipeline.yml"},
					},
				},
				repo,
				blueprints,
				VarField{},
				"bpParent",
			},
			ComposedBlueprintPtr{
				{
					Name:   "aws/test",
					Parent: "bpParent",
					BlueprintConfig: &BlueprintConfig{
						ApiVersion: "xl/v2",
						Kind:       "Blueprint",
						Metadata:   Metadata{Name: "Test Project"},
						Include: []IncludedBlueprintProcessed{
							{
								Blueprint: "aws/emptyfiles",
							},
							{
								Blueprint: "aws/emptyparams",
							},
						},
						Variables: []Variable{
							{Name: VarField{Value: "Bar"}, Label: VarField{Value: "Bar"}, Value: VarField{Value: "testing"}},
						},
						TemplateConfigs: []TemplateConfig{
							{Path: "xld-environment.yml.tmpl", FullPath: "aws/compose/xld-environment.yml.tmpl"},
							{Path: "xld-infrastructure.yml.tmpl", FullPath: "aws/compose/xld-infrastructure.yml.tmpl"},
							{Path: "xlr-pipeline.yml", FullPath: "aws/compose/xlr-pipeline.yml"},
						},
					},
					DependsOn: []VarField{{}},
				},
				{
					Name:   "aws/emptyfiles",
					Parent: "aws/test",
					BlueprintConfig: &BlueprintConfig{
						ApiVersion: "xl/v2",
						Kind:       "Blueprint",
						Metadata:   Metadata{Name: "Test Project 3"},
						Include:    []IncludedBlueprintProcessed{},
						Variables: []Variable{
							{Name: VarField{Value: "Foo"}, Label: VarField{Value: "Foo"}, Value: VarField{Value: "testing"}},
						},
						TemplateConfigs: []TemplateConfig{},
					},
					DependsOn: []VarField{{}},
				},
				{
					Name:   "aws/emptyparams",
					Parent: "aws/test",
					BlueprintConfig: &BlueprintConfig{
						ApiVersion: "xl/v2",
						Kind:       "Blueprint",
						Metadata:   Metadata{Name: "Test Project 4"},
						Include:    []IncludedBlueprintProcessed{},
						Variables:  []Variable{},
						TemplateConfigs: []TemplateConfig{
							{Path: "xld-app.yml.tmpl", FullPath: "aws/emptyparams/xld-app.yml.tmpl"},
							{Path: "xlr-pipeline.yml", FullPath: "aws/emptyparams/xlr-pipeline.yml"},
						},
					},
					DependsOn: []VarField{{}},
				},
			},
			false,
		},
		{
			"should compose the given blueprints together in after stage by default",
			args{
				"aws/test",
				&BlueprintConfig{
					ApiVersion: "xl/v2",
					Kind:       "Blueprint",
					Metadata:   Metadata{Name: "Test Project"},
					Include: []IncludedBlueprintProcessed{
						{
							Blueprint: "aws/datalake",
							ParameterOverrides: []Variable{
								{
									Name:  VarField{Value: "Foo"},
									Value: VarField{Value: "hello"},
								},
							},
							DependsOn: VarField{Tag: tagExpressionV2, Value: "Bar == 'testing'"},
							FileOverrides: []TemplateConfig{
								{
									Path:      "xlr-pipeline.yml",
									RenameTo:  VarField{Value: "xlr-pipeline2-new.yml"},
									DependsOn: VarField{Value: "TestDepends"},
								},
							},
						},
					},
					Variables: []Variable{
						{Name: VarField{Value: "Bar"}, Label: VarField{Value: "Bar"}, Value: VarField{Value: "testing"}},
					},
					TemplateConfigs: []TemplateConfig{
						{Path: "xld-environment.yml.tmpl", FullPath: "aws/compose/xld-environment.yml.tmpl"},
						{Path: "xld-infrastructure.yml.tmpl", FullPath: "aws/compose/xld-infrastructure.yml.tmpl"},
						{Path: "xlr-pipeline.yml", FullPath: "aws/compose/xlr-pipeline.yml"},
					},
				},
				repo,
				blueprints,
				VarField{},
				"bpParent",
			},
			ComposedBlueprintPtr{
				{
					Name:   "aws/test",
					Parent: "bpParent",
					BlueprintConfig: &BlueprintConfig{
						ApiVersion: "xl/v2",
						Kind:       "Blueprint",
						Metadata:   Metadata{Name: "Test Project"},
						Include: []IncludedBlueprintProcessed{
							{
								Blueprint: "aws/datalake",
								ParameterOverrides: []Variable{
									{
										Name:  VarField{Value: "Foo"},
										Value: VarField{Value: "hello"},
									},
								},
								DependsOn: VarField{Tag: tagExpressionV2, Value: "Bar == 'testing'"},
								FileOverrides: []TemplateConfig{
									{
										Path:      "xlr-pipeline.yml",
										RenameTo:  VarField{Value: "xlr-pipeline2-new.yml"},
										DependsOn: VarField{Value: "TestDepends"},
									},
								},
							},
						},
						Variables: []Variable{
							{Name: VarField{Value: "Bar"}, Label: VarField{Value: "Bar"}, Value: VarField{Value: "testing"}},
						},
						TemplateConfigs: []TemplateConfig{
							{Path: "xld-environment.yml.tmpl", FullPath: "aws/compose/xld-environment.yml.tmpl"},
							{Path: "xld-infrastructure.yml.tmpl", FullPath: "aws/compose/xld-infrastructure.yml.tmpl"},
							{Path: "xlr-pipeline.yml", FullPath: "aws/compose/xlr-pipeline.yml"},
						},
					},
					DependsOn: []VarField{{}},
				},
				{
					Name:   "aws/datalake",
					Parent: "aws/test",
					BlueprintConfig: &BlueprintConfig{
						ApiVersion: "xl/v2",
						Kind:       "Blueprint",
						Metadata:   Metadata{Name: "Test Project 2"},
						Include:    []IncludedBlueprintProcessed{},
						Variables: []Variable{
							{Name: VarField{Value: "Foo"}, Label: VarField{Value: "Foo"}, Value: VarField{Value: "hello"}},
						},
						TemplateConfigs: []TemplateConfig{
							{Path: "xld-app.yml.tmpl", FullPath: "aws/datalake/xld-app.yml.tmpl"},
							{Path: "xlr-pipeline.yml", FullPath: "aws/datalake/xlr-pipeline.yml", DependsOn: VarField{Value: "TestDepends"}, RenameTo: VarField{Value: "xlr-pipeline2-new.yml"}},
						},
					},
					DependsOn: []VarField{{}, {Tag: tagExpressionV2, Value: "Bar == 'testing'"}},
				},
			},
			false,
		},
		{
			"should compose the given blueprints together in before and after stage accordingly",
			args{
				"aws/test",
				&BlueprintConfig{
					ApiVersion: "xl/v2",
					Kind:       "Blueprint",
					Metadata:   Metadata{Name: "Test Project"},
					Include: []IncludedBlueprintProcessed{
						{
							Blueprint: "aws/monolith",
							Stage:     "before",
							ParameterOverrides: []Variable{
								{
									Name:      VarField{Value: "Test"},
									Value:     VarField{Value: "hello"},
									DependsOn: VarField{Tag: tagExpressionV2, Value: "Bar == 'testing'"},
								},
								{
									Name:  VarField{Value: "bar"},
									Value: VarField{Value: "true", Bool: true},
								},
							},
							FileOverrides: []TemplateConfig{
								{
									Path:      "xld-infrastructure.yml.tmpl",
									DependsOn: VarField{Value: "TestDepends"},
								},
							},
						},
						{
							Blueprint: "aws/datalake",
							Stage:     "after",
							ParameterOverrides: []Variable{
								{
									Name:  VarField{Value: "Foo"},
									Value: VarField{Value: "hello"},
								},
							},
							DependsOn: VarField{Tag: tagExpressionV2, Value: "Bar == 'testing'"},
							FileOverrides: []TemplateConfig{
								{
									Path:      "xlr-pipeline.yml",
									RenameTo:  VarField{Value: "xlr-pipeline2-new.yml"},
									DependsOn: VarField{Value: "TestDepends"},
								},
							},
						},
					},
					Variables: []Variable{
						{Name: VarField{Value: "Bar"}, Label: VarField{Value: "Bar"}, Value: VarField{Value: "testing"}},
					},
					TemplateConfigs: []TemplateConfig{
						{Path: "xld-environment.yml.tmpl", FullPath: "aws/compose/xld-environment.yml.tmpl"},
						{Path: "xld-infrastructure.yml.tmpl", FullPath: "aws/compose/xld-infrastructure.yml.tmpl"},
						{Path: "xlr-pipeline.yml", FullPath: "aws/compose/xlr-pipeline.yml"},
					},
				},
				repo,
				blueprints,
				VarField{},
				"bpParent",
			},
			ComposedBlueprintPtr{
				{
					Name:   "aws/monolith",
					Parent: "aws/test",
					BlueprintConfig: &BlueprintConfig{
						ApiVersion: "xl/v2",
						Kind:       "Blueprint",
						Metadata:   Metadata{Name: "Test Project"},
						Include:    []IncludedBlueprintProcessed{},
						Variables: []Variable{
							{Name: VarField{Value: "Test"}, Label: VarField{Value: "Test"}, Value: VarField{Value: "hello"}, SaveInXlvals: VarField{Value: "true", Bool: true}, DependsOn: VarField{Tag: tagExpressionV2, Value: "Bar == 'testing'"}},
						},
						TemplateConfigs: []TemplateConfig{
							{Path: "xld-environment.yml.tmpl", FullPath: "aws/monolith/xld-environment.yml.tmpl"},
							{Path: "xld-infrastructure.yml.tmpl", FullPath: "aws/monolith/xld-infrastructure.yml.tmpl", DependsOn: VarField{Value: "TestDepends"}},
							{Path: "xlr-pipeline.yml", FullPath: "aws/monolith/xlr-pipeline.yml"},
						},
					},
					DependsOn: []VarField{{}},
				},
				{
					Name:   "aws/test",
					Parent: "bpParent",
					BlueprintConfig: &BlueprintConfig{
						ApiVersion: "xl/v2",
						Kind:       "Blueprint",
						Metadata:   Metadata{Name: "Test Project"},
						Include: []IncludedBlueprintProcessed{
							{
								Blueprint: "aws/monolith",
								Stage:     "before",
								ParameterOverrides: []Variable{
									{
										Name:      VarField{Value: "Test"},
										Value:     VarField{Value: "hello"},
										DependsOn: VarField{Tag: tagExpressionV2, Value: "Bar == 'testing'"},
									},
									{
										Name:  VarField{Value: "bar"},
										Value: VarField{Value: "true", Bool: true},
									},
								},
								FileOverrides: []TemplateConfig{
									{
										Path:      "xld-infrastructure.yml.tmpl",
										DependsOn: VarField{Value: "TestDepends"},
									},
								},
							},
							{
								Blueprint: "aws/datalake",
								Stage:     "after",
								ParameterOverrides: []Variable{
									{
										Name:  VarField{Value: "Foo"},
										Value: VarField{Value: "hello"},
									},
								},
								DependsOn: VarField{Tag: tagExpressionV2, Value: "Bar == 'testing'"},
								FileOverrides: []TemplateConfig{
									{
										Path:      "xlr-pipeline.yml",
										RenameTo:  VarField{Value: "xlr-pipeline2-new.yml"},
										DependsOn: VarField{Value: "TestDepends"},
									},
								},
							},
						},
						Variables: []Variable{
							{Name: VarField{Value: "Bar"}, Label: VarField{Value: "Bar"}, Value: VarField{Value: "testing"}},
						},
						TemplateConfigs: []TemplateConfig{
							{Path: "xld-environment.yml.tmpl", FullPath: "aws/compose/xld-environment.yml.tmpl"},
							{Path: "xld-infrastructure.yml.tmpl", FullPath: "aws/compose/xld-infrastructure.yml.tmpl"},
							{Path: "xlr-pipeline.yml", FullPath: "aws/compose/xlr-pipeline.yml"},
						},
					},
					DependsOn: []VarField{{}},
				},
				{
					Name:   "aws/datalake",
					Parent: "aws/test",
					BlueprintConfig: &BlueprintConfig{
						ApiVersion: "xl/v2",
						Kind:       "Blueprint",
						Metadata:   Metadata{Name: "Test Project 2"},
						Include:    []IncludedBlueprintProcessed{},
						Variables: []Variable{
							{Name: VarField{Value: "Foo"}, Label: VarField{Value: "Foo"}, Value: VarField{Value: "hello"}},
						},
						TemplateConfigs: []TemplateConfig{
							{Path: "xld-app.yml.tmpl", FullPath: "aws/datalake/xld-app.yml.tmpl"},
							{Path: "xlr-pipeline.yml", FullPath: "aws/datalake/xlr-pipeline.yml", DependsOn: VarField{Value: "TestDepends"}, RenameTo: VarField{Value: "xlr-pipeline2-new.yml"}},
						},
					},
					DependsOn: []VarField{{}, {Tag: tagExpressionV2, Value: "Bar == 'testing'"}},
				},
			},
			false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := composeBlueprints(tt.args.blueprintName, tt.args.blueprintDoc, tt.args.blueprintContext, tt.args.blueprints, []VarField{tt.args.dependsOn}, tt.args.parentName)
			if (err != nil) != tt.wantErr {
				t.Errorf("composeBlueprints() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			assert.Equal(t, tt.want, got)
		})
	}
}

func Test_evaluateAndSkipIfDependsOnIsFalse(t *testing.T) {
	type args struct {
		dependsOn  VarField
		mergedData *PreparedData
	}
	tests := []struct {
		name    string
		args    args
		want    bool
		wantErr bool
	}{
		{
			"should return true when depends on is not defined",
			args{
				VarField{},
				&PreparedData{},
			},
			true,
			false,
		},
		{
			"should return false when there is error",
			args{
				VarField{Value: "Foo"},
				&PreparedData{},
			},
			false,
			true,
		},
		{
			"should return true when VarField evaluates to true",
			args{
				VarField{Value: "1 > 0", Tag: tagExpressionV2},
				&PreparedData{},
			},
			true,
			false,
		},
		{
			"should return false when VarField evaluates to false",
			args{
				VarField{Value: "1 > 2", Tag: tagExpressionV2},
				&PreparedData{},
			},
			false,
			false,
		},
		{
			"should return true when VarField evaluates to true based on variable lookup",
			args{
				VarField{Value: "Foo"},
				&PreparedData{
					TemplateData: map[string]interface{}{
						"Foo": true,
					},
				},
			},
			true,
			false,
		},
		{
			"should return true when VarField evaluates to true based on variable lookup in expression",
			args{
				VarField{Value: "Foo > 2", Tag: tagExpressionV2},
				&PreparedData{
					TemplateData: map[string]interface{}{
						"Foo": "3",
					},
				},
			},
			true,
			false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := evaluateAndSkipIfDependsOnIsFalse([]VarField{tt.args.dependsOn}, tt.args.mergedData)
			if (err != nil) != tt.wantErr {
				t.Errorf("evaluateAndCheckDependsOnIsTrue() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("evaluateAndCheckDependsOnIsTrue() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_prepareMergedTemplateData(t *testing.T) {
	defer httpmock.DeactivateAndReset()
	repo := getMockHttpBlueprintContext(t)
	blueprints, err := repo.initCurrentRepoClient()
	require.Nil(t, err)
	require.NotNil(t, blueprints)
	SkipFinalPrompt = true

	type args struct {
		blueprintContext *BlueprintContext
		blueprints       map[string]*models.BlueprintRemote
		params           BlueprintParams
		surveyOpts       []survey.AskOpt
	}
	tests := []struct {
		name    string
		args    args
		want    *PreparedData
		want1   *BlueprintConfig
		wantErr bool
	}{
		{
			"should return error when unable fetch blueprint",
			args{
				repo,
				blueprints,
				BlueprintParams{
					TemplatePath:       "foo",
					AnswersFile:        "",
					StrictAnswers:      false,
					UseDefaultsAsValue: false,
				},
				[]survey.AskOpt{},
			},
			nil,
			nil,
			true,
		},
		{
			"should return processed data for simple blueprint",
			args{
				repo,
				blueprints,
				BlueprintParams{
					TemplatePath:       "aws/monolith",
					AnswersFile:        "",
					StrictAnswers:      false,
					UseDefaultsAsValue: false,
				},
				[]survey.AskOpt{},
			},
			&PreparedData{
				TemplateData: map[string]interface{}{"Test": "testing"},
				SummaryData:  map[string]interface{}{"Test": "testing"},
				Secrets:      map[string]interface{}{},
				Values:       map[string]interface{}{"Test": "testing"},
			},
			&BlueprintConfig{
				ApiVersion: "xl/v2",
				Kind:       "Blueprint",
				Metadata:   Metadata{Name: "Test Project"},
				Include:    []IncludedBlueprintProcessed{},
				Variables: []Variable{
					{Name: VarField{Value: "Test"}, Label: VarField{Value: "Test"}, Value: VarField{Value: "testing"}, SaveInXlvals: VarField{Value: "true", Bool: true}},
				},
				TemplateConfigs: []TemplateConfig{
					{Path: "xld-environment.yml.tmpl", FullPath: "aws/monolith/xld-environment.yml.tmpl"},
					{Path: "xld-infrastructure.yml.tmpl", FullPath: "aws/monolith/xld-infrastructure.yml.tmpl"},
					{Path: "xlr-pipeline.yml", FullPath: "aws/monolith/xlr-pipeline.yml"},
				},
			},
			false,
		},
		{
			"should return processed data for blueprint with existing prepared data",
			args{
				repo,
				blueprints,
				BlueprintParams{
					TemplatePath:       "aws/compose",
					AnswersFile:        "",
					StrictAnswers:      false,
					UseDefaultsAsValue: false,
					ExistingPreparedData: &PreparedData{
						TemplateData: map[string]interface{}{"Bar": "original", "Foo2": "hello2", "Test2": "hello2"},
						SummaryData:  map[string]interface{}{"Bar": "original", "Foo2": "hello2", "Test2": "hello2"},
						Secrets:      map[string]interface{}{},
						Values:       map[string]interface{}{"Test2": "hello2"},
					},
				},
				[]survey.AskOpt{},
			},
			&PreparedData{
				TemplateData: map[string]interface{}{"Bar": "testing", "Foo": "hello", "Test": "hello", "Foo2": "hello2", "Test2": "hello2"},
				SummaryData:  map[string]interface{}{"Bar": "testing", "Foo": "hello", "Test": "hello", "Foo2": "hello2", "Test2": "hello2"},
				Secrets:      map[string]interface{}{},
				Values:       map[string]interface{}{"Test": "hello", "Test2": "hello2"},
			},
			&BlueprintConfig{
				ApiVersion: "xl/v2",
				Kind:       "Blueprint",
				Metadata:   Metadata{Name: "Test Project"},
				Include: []IncludedBlueprintProcessed{
					{
						Blueprint: "aws/monolith",
						Stage:     "before",
						ParameterOverrides: []Variable{
							{
								Name:      VarField{Value: "Test"},
								Value:     VarField{Value: "hello"},
								DependsOn: VarField{Tag: tagExpressionV2, Value: "2 > 1"},
							},
						},
						FileOverrides: []TemplateConfig{
							{
								Path:      "xld-infrastructure.yml.tmpl",
								DependsOn: VarField{Value: "false", Tag: tagExpressionV2},
							},
						},
					},
					{
						Blueprint: "aws/datalake",
						Stage:     "after",
						ParameterOverrides: []Variable{
							{
								Name:  VarField{Value: "Foo"},
								Value: VarField{Value: "hello"},
							},
						},
						DependsOn: VarField{Tag: tagExpressionV2, Value: "Bar == 'testing'"},
						FileOverrides: []TemplateConfig{
							{
								Path:      "xlr-pipeline.yml",
								RenameTo:  VarField{Value: "xlr-pipeline2-new.yml"},
								DependsOn: VarField{Value: "TestDepends"},
							},
						},
					},
				},
				Variables: []Variable{
					{Name: VarField{Value: "Test"}, Label: VarField{Value: "Test"}, Value: VarField{Value: "hello"}, SaveInXlvals: VarField{Value: "true", Bool: true}, DependsOn: VarField{Tag: tagExpressionV2, Value: "2 > 1"}},
					{Name: VarField{Value: "Bar"}, Label: VarField{Value: "Bar"}, Value: VarField{Value: "testing"}},
					{Name: VarField{Value: "Foo"}, Label: VarField{Value: "Foo"}, Value: VarField{Value: "hello"}},
				},
				TemplateConfigs: []TemplateConfig{
					{Path: "xld-environment.yml.tmpl", FullPath: "aws/monolith/xld-environment.yml.tmpl"},
					{Path: "xld-infrastructure.yml.tmpl", FullPath: "aws/monolith/xld-infrastructure.yml.tmpl", DependsOn: VarField{Value: "false", Tag: tagExpressionV2}},
					{Path: "xlr-pipeline.yml", FullPath: "aws/monolith/xlr-pipeline.yml"},
					{Path: "xld-environment.yml.tmpl", FullPath: "aws/compose/xld-environment.yml.tmpl"},
					{Path: "xld-infrastructure.yml.tmpl", FullPath: "aws/compose/xld-infrastructure.yml.tmpl"},
					{Path: "xlr-pipeline.yml", FullPath: "aws/compose/xlr-pipeline.yml"},
					{Path: "xld-app.yml.tmpl", FullPath: "aws/datalake/xld-app.yml.tmpl"},
					{Path: "xlr-pipeline.yml", FullPath: "aws/datalake/xlr-pipeline.yml", DependsOn: VarField{Value: "TestDepends"}, RenameTo: VarField{Value: "xlr-pipeline2-new.yml"}},
				},
			},
			false,
		},
		{
			"should return processed data for composed blueprint",
			args{
				repo,
				blueprints,
				BlueprintParams{
					TemplatePath:       "aws/compose",
					AnswersFile:        "",
					StrictAnswers:      false,
					UseDefaultsAsValue: false,
				},
				[]survey.AskOpt{},
			},
			&PreparedData{
				TemplateData: map[string]interface{}{"Bar": "testing", "Foo": "hello", "Test": "hello"},
				SummaryData:  map[string]interface{}{"Bar": "testing", "Foo": "hello", "Test": "hello"},
				Secrets:      map[string]interface{}{},
				Values:       map[string]interface{}{"Test": "hello"},
			},
			&BlueprintConfig{
				ApiVersion: "xl/v2",
				Kind:       "Blueprint",
				Metadata:   Metadata{Name: "Test Project"},
				Include: []IncludedBlueprintProcessed{
					{
						Blueprint: "aws/monolith",
						Stage:     "before",
						ParameterOverrides: []Variable{
							{
								Name:      VarField{Value: "Test"},
								Value:     VarField{Value: "hello"},
								DependsOn: VarField{Tag: tagExpressionV2, Value: "2 > 1"},
							},
						},
						FileOverrides: []TemplateConfig{
							{
								Path:      "xld-infrastructure.yml.tmpl",
								DependsOn: VarField{Value: "false", Tag: tagExpressionV2},
							},
						},
					},
					{
						Blueprint: "aws/datalake",
						Stage:     "after",
						ParameterOverrides: []Variable{
							{
								Name:  VarField{Value: "Foo"},
								Value: VarField{Value: "hello"},
							},
						},
						DependsOn: VarField{Tag: tagExpressionV2, Value: "Bar == 'testing'"},
						FileOverrides: []TemplateConfig{
							{
								Path:      "xlr-pipeline.yml",
								RenameTo:  VarField{Value: "xlr-pipeline2-new.yml"},
								DependsOn: VarField{Value: "TestDepends"},
							},
						},
					},
				},
				Variables: []Variable{
					{Name: VarField{Value: "Test"}, Label: VarField{Value: "Test"}, Value: VarField{Value: "hello"}, SaveInXlvals: VarField{Value: "true", Bool: true}, DependsOn: VarField{Tag: tagExpressionV2, Value: "2 > 1"}},
					{Name: VarField{Value: "Bar"}, Label: VarField{Value: "Bar"}, Value: VarField{Value: "testing"}},
					{Name: VarField{Value: "Foo"}, Label: VarField{Value: "Foo"}, Value: VarField{Value: "hello"}},
				},
				TemplateConfigs: []TemplateConfig{
					{Path: "xld-environment.yml.tmpl", FullPath: "aws/monolith/xld-environment.yml.tmpl"},
					{Path: "xld-infrastructure.yml.tmpl", FullPath: "aws/monolith/xld-infrastructure.yml.tmpl", DependsOn: VarField{Value: "false", Tag: tagExpressionV2}},
					{Path: "xlr-pipeline.yml", FullPath: "aws/monolith/xlr-pipeline.yml"},
					{Path: "xld-environment.yml.tmpl", FullPath: "aws/compose/xld-environment.yml.tmpl"},
					{Path: "xld-infrastructure.yml.tmpl", FullPath: "aws/compose/xld-infrastructure.yml.tmpl"},
					{Path: "xlr-pipeline.yml", FullPath: "aws/compose/xlr-pipeline.yml"},
					{Path: "xld-app.yml.tmpl", FullPath: "aws/datalake/xld-app.yml.tmpl"},
					{Path: "xlr-pipeline.yml", FullPath: "aws/datalake/xlr-pipeline.yml", DependsOn: VarField{Value: "TestDepends"}, RenameTo: VarField{Value: "xlr-pipeline2-new.yml"}},
				},
			},
			false,
		},
		{
			"should return processed data for composed blueprint with includeIf",
			args{
				repo,
				blueprints,
				BlueprintParams{
					TemplatePath:       "aws/compose-2",
					AnswersFile:        "",
					StrictAnswers:      false,
					UseDefaultsAsValue: false,
				},
				[]survey.AskOpt{},
			},
			&PreparedData{
				TemplateData: map[string]interface{}{"Bar": "testing"},
				SummaryData:  map[string]interface{}{"Bar": "testing"},
				Secrets:      map[string]interface{}{},
				Values:       map[string]interface{}{},
			},
			&BlueprintConfig{
				ApiVersion: "xl/v2",
				Kind:       "Blueprint",
				Metadata:   Metadata{Name: "Test Project"},
				Include: []IncludedBlueprintProcessed{
					{
						Blueprint: "aws/monolith",
						Stage:     "before",
						ParameterOverrides: []Variable{
							{
								Name:      VarField{Value: "Test"},
								Value:     VarField{Value: "hello"},
								DependsOn: VarField{Tag: tagExpressionV2, Value: "2 > 1"},
							},
							{
								Name:  VarField{Value: "bar"},
								Value: VarField{Value: "true", Bool: true},
							},
						},
						DependsOn: VarField{Tag: tagExpressionV2, Value: "2 < 1"},
						FileOverrides: []TemplateConfig{
							{
								Path:      "xld-infrastructure.yml.tmpl",
								DependsOn: VarField{Tag: tagExpressionV2, Value: "2 < 1"},
							},
						},
					},
					{
						Blueprint: "aws/datalake",
						Stage:     "after",
						ParameterOverrides: []Variable{
							{
								Name:  VarField{Value: "Foo"},
								Value: VarField{Value: "hello"},
							},
						},
						DependsOn: VarField{Tag: tagExpressionV2, Value: "Bar != 'testing'"},
						FileOverrides: []TemplateConfig{
							{
								Path:      "xlr-pipeline.yml",
								RenameTo:  VarField{Value: "xlr-pipeline2-new.yml"},
								DependsOn: VarField{Value: "TestDepends"},
							},
						},
					},
				},
				Variables: []Variable{
					{Name: VarField{Value: "Bar"}, Label: VarField{Value: "Bar"}, Value: VarField{Value: "testing"}},
				},
				TemplateConfigs: []TemplateConfig{
					{Path: "xld-environment.yml.tmpl", FullPath: "aws/compose-2/xld-environment.yml.tmpl"},
					{Path: "xld-infrastructure.yml.tmpl", FullPath: "aws/compose-2/xld-infrastructure.yml.tmpl"},
					{Path: "xlr-pipeline.yml", FullPath: "aws/compose-2/xlr-pipeline.yml"},
				},
			},
			false,
		},
		{
			"should return processed data for nested composed blueprint",
			args{
				repo,
				blueprints,
				BlueprintParams{
					TemplatePath:       "aws/compose-3",
					AnswersFile:        "",
					StrictAnswers:      false,
					UseDefaultsAsValue: false,
				},
				[]survey.AskOpt{},
			},
			&PreparedData{
				TemplateData: map[string]interface{}{"Bar": "hello", "Test": "hello"},
				SummaryData:  map[string]interface{}{"Bar": "hello", "Test": "hello"},
				Secrets:      map[string]interface{}{},
				Values:       map[string]interface{}{"Test": "hello"},
			},
			&BlueprintConfig{
				ApiVersion: "xl/v2",
				Kind:       "Blueprint",
				Metadata:   Metadata{Name: "Test Project"},
				Include: []IncludedBlueprintProcessed{
					{
						Blueprint: "aws/compose",
						Stage:     "before",
						ParameterOverrides: []Variable{
							{
								Name:      VarField{Value: "Bar"},
								Value:     VarField{Value: "hello"},
								DependsOn: VarField{Tag: tagExpressionV2, Value: "2 > 1"},
							},
						},
					},
					{
						Blueprint: "aws/compose-2",
						Stage:     "after",
						ParameterOverrides: []Variable{
							{
								Name:      VarField{Value: "Bar"},
								Value:     VarField{Value: "hello"},
								DependsOn: VarField{Tag: tagExpressionV2, Value: "2 > 1"},
							},
						},
						DependsOn: VarField{Tag: tagExpressionV2, Value: "2 < 1"},
					},
				},
				Variables: []Variable{
					{Name: VarField{Value: "Test"}, Label: VarField{Value: "Test"}, Value: VarField{Value: "hello"}, SaveInXlvals: VarField{Value: "true", Bool: true}, DependsOn: VarField{Tag: tagExpressionV2, Value: "2 > 1"}},
					{Name: VarField{Value: "Bar"}, Label: VarField{Value: "Bar"}, Value: VarField{Value: "hello"}, DependsOn: VarField{Tag: tagExpressionV2, Value: "2 > 1"}},
				},
				TemplateConfigs: []TemplateConfig{
					{Path: "xld-environment.yml.tmpl", FullPath: "aws/monolith/xld-environment.yml.tmpl"},
					{Path: "xld-infrastructure.yml.tmpl", FullPath: "aws/monolith/xld-infrastructure.yml.tmpl", DependsOn: VarField{Value: "false", Tag: tagExpressionV2}},
					{Path: "xlr-pipeline.yml", FullPath: "aws/monolith/xlr-pipeline.yml"},
					{Path: "xld-environment.yml.tmpl", FullPath: "aws/compose/xld-environment.yml.tmpl"},
					{Path: "xld-infrastructure.yml.tmpl", FullPath: "aws/compose/xld-infrastructure.yml.tmpl"},
					{Path: "xlr-pipeline.yml", FullPath: "aws/compose/xlr-pipeline.yml"},
				},
			},
			false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, got1, err := prepareMergedTemplateData(
				tt.args.blueprintContext,
				tt.args.blueprints,
				tt.args.params,
				tt.args.surveyOpts...,
			)
			if (err != nil) != tt.wantErr {
				t.Errorf("prepareMergedTemplateData() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			assert.Equal(t, tt.want, got)
			assert.Equal(t, tt.want1, got1)
		})
	}
}

func TestTemplateConfig_ProcessExpression(t *testing.T) {
	tests := []struct {
		name       string
		config     TemplateConfig
		parameters map[string]interface{}
		want       TemplateConfig
		wantErr    bool
	}{
		{
			"should throw error when expression is invalid",
			TemplateConfig{
				Path:      "Test.yaml",
				DependsOn: VarField{Value: "True ? 'A' : 'B'", Tag: tagExpressionV2},
			},
			map[string]interface{}{},
			TemplateConfig{
				Path:      "Test.yaml",
				DependsOn: VarField{Value: "True ? 'A' : 'B'", Tag: tagExpressionV2},
			},
			true,
		},
		{
			"should process expressions in variable fields",
			TemplateConfig{
				Path:      "Test.yaml",
				RenameTo:  VarField{Value: "true ? 'A' : 'B'", Tag: tagExpressionV2},
				DependsOn: VarField{Value: "Foo > 5 ? 'A' : 'B'", Tag: tagExpressionV2},
			},
			map[string]interface{}{
				"Foo": 10,
			},
			TemplateConfig{
				Path:      "Test.yaml",
				RenameTo:  VarField{Value: "A", Tag: tagExpressionV2},
				DependsOn: VarField{Value: "A", Tag: tagExpressionV2},
			},
			false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.ProcessExpression(tt.parameters)
			if tt.wantErr {
				assert.NotNil(t, err)
			} else {
				assert.Equal(t, tt.want, tt.config)
			}
		})
	}
}

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

func TestAdjustPathSeperatorIfNeeded(t *testing.T) {
	t.Run("should produce standard path using the host os seperator", func(t *testing.T) {
		assert.Equal(t, "", AdjustPathSeperatorIfNeeded(""))
		assert.Equal(t, "test", AdjustPathSeperatorIfNeeded("test"))
		assert.Equal(t, path.Join("..", "test"), AdjustPathSeperatorIfNeeded("../test"))
		assert.Equal(t, path.Join("..", "microservice", "blueprint.yaml"), AdjustPathSeperatorIfNeeded(`..\microservice\blueprint.yaml`))
		assert.Equal(t, path.Join("..", "microservice", "blueprint.yaml"), AdjustPathSeperatorIfNeeded(`../microservice\blueprint.yaml`))
		assert.Equal(t, path.Join("..", "microservice", "blueprint.yaml"), AdjustPathSeperatorIfNeeded(`../microservice/blueprint.yaml`))
		assert.Equal(t, path.Join("test", "test", "again"), AdjustPathSeperatorIfNeeded(`test/test\again`))
		assert.Equal(t, path.Join("test", "test", "again"), AdjustPathSeperatorIfNeeded(`test\test\again`))
		assert.Equal(t, path.Join("test", "test", "again"), AdjustPathSeperatorIfNeeded(`test/test/again`))
	})
}

func TestInstantiateBlueprint(t *testing.T) {
	SkipFinalPrompt = true

	t.Run("should error on unknown template", func(t *testing.T) {
		gb := &GeneratedBlueprint{OutputDir: "xebialabs"}
		defer gb.Cleanup()
		err := InstantiateBlueprint(
			false,
			"abc",
			getLocalTestBlueprintContext(t),
			gb,
			"",
			false,
			false,
			false,
		)

		require.NotNil(t, err)
		assert.Equal(t, "blueprint [abc] not found in repository Test", err.Error())
	})

	t.Run("should error on invalid test template", func(t *testing.T) {
		gb := &GeneratedBlueprint{OutputDir: "xebialabs"}
		defer gb.Cleanup()
		err := InstantiateBlueprint(
			false,
			"invalid",
			getLocalTestBlueprintContext(t),
			gb,
			"",
			false,
			false,
			false,
		)
		require.NotNil(t, err)
		assert.Equal(t, "parameter [Test] is missing required fields: [type]", err.Error())
	})

	t.Run("should create output files for valid test template with answers file", func(t *testing.T) {
		gb := &GeneratedBlueprint{OutputDir: "xebialabs"}
		defer gb.Cleanup()
		err := InstantiateBlueprint(
			false,
			"answer-input",
			getLocalTestBlueprintContext(t),
			gb,
			GetTestTemplateDir("answer-input.yaml"),
			true,
			false,
			false,
		)
		require.Nil(t, err)

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

	t.Run("should create output files for valid test template in use defaults as values mode", func(t *testing.T) {
		gb := &GeneratedBlueprint{OutputDir: "xebialabs"}
		defer gb.Cleanup()
		err := InstantiateBlueprint(
			false,
			"defaults-as-values",
			getLocalTestBlueprintContext(t),
			gb,
			"",
			false,
			true,
			false,
		)
		require.Nil(t, err)

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

	t.Run("should create output files for valid test template from local path when a registry is defined", func(t *testing.T) {
		gb := &GeneratedBlueprint{OutputDir: "xebialabs"}
		defer gb.Cleanup()
		err := InstantiateBlueprint(
			false,
			"valid-no-prompt",
			getLocalTestBlueprintContext(t),
			gb,
			"",
			false,
			false,
			false,
		)
		require.Nil(t, err)

		// assertions
		assert.FileExists(t, "xld-environment.yml")
		assert.FileExists(t, "xld-infrastructure.yml")
		assert.False(t, util.PathExists("xlr-pipeline.yml", false))
		assert.True(t, util.PathExists("xlr-pipeline-2.yml", false))
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

		// Check if only saveInXlVals marked fields are in values.xlvals
		valuesFileContent := GetFileContent(models.BlueprintOutputDir + string(os.PathSeparator) + valuesFile)
		assert.Contains(t, valuesFileContent, "Test = testing")
		assert.Contains(t, GetFileContent(path.Join(gb.OutputDir, valuesFile)), `FshYmQzRUNbYTA4Icc3V7JEgLXMNjcSLY9L1H4XQD79coMBRbbJFtOsp0Yk2btCKCAYLio0S8Jw85W5mgpLkasvCrXO5\nQJGxFvtQc2tHGLj0kNzM9KyAqbUJRe1l40TqfMdscEaWJimtd4oygqVc6y7zW1Wuj1EcDUvMD8qK8FEWfQgm5ilBIldQ\nomhDPbq8F84KRsRwCgT05mTrxhBtgqGuCHXcr115iUuUNW7dzzP5iXAgEp4Apa30NHzNsy5TUoIZGDJceO2BmAAmG4HS0cZ\notIXJ2BJEx95SGnqO4kZFoRJzghlZMWs50PkskI5JTM6tHFmtZIdYbo7ZvbA0LP71QtTbSDziqDzXoi5uwxpwaDO95fPYv0\nN1ajgotzn4czgX4hA8gFIipmUUA2AYfgQ5jZQ4I9zO5rxxj80lPWFNOnrHzD1jWZAhLgdpyWldWLt9NbcWegrgLpI\nhRA08PILJnV2z79aTfylL7Y3zJ2urSjr0XIbTWQlWwZ1VXBm13IbRffbku0qjFmSuxDrKFCwGEBtRZ4RnseholT8DA0yDIjPCsfY2jo\nCjljgZHYRoIe4E8WsMt0zzp9G0UP7It6jzJok3yk9Ril48yLthkPvyJ4qoH2PTLx8xBeGBJLKmHT9ojDbWQxOXpml72ati\n4jcxmZfSgDUqMPmTRHPqZ47k6f3XTrPxqIDJ8SzOj09OaKzjSYyZnxIEokm1JotTaqhZa64zptKlbuY0kblSbFAGFFQZnn7RjkU3ZKq872gTDh\nAdteR98sbMdmMGipaxgYbCfuomBEdxldjlApbwDiswJkOQIY0Vypwt95M3LAWha4zACRwrYz7rVqDBJqpo6hFh3V6zBRQR2C6GINUJZq3KWWz\nXAI0ncPo95GDraIFnaStGFHu6R1WC7oopSFS6kgbhJL6noGgMjxbmnPzDA8sXVo1GEtyq79oG2CTHBbrODI9KhsKYy3B0\n8Prpu561H6kDtwIyZqZQXHppVaeFbrGlWAsQpp5su5iHhfFllVaCsDI8kYmmy4JdtOEmPYNL3pF7Uf35X0LIdJKb54czjwBuc2rbbifX9mIn30I8tTgq\n9ldZFjj0SwtTxN1hjYh5pRRTdKZkuwNv6v9L0iPitR6YwuCQaIx1LlymGwfR1Zo6u4gLDCqBYjLz2s1jc7o5dhdmVXmMHKFjWrTaVbanLiwJuNWDQb1e14UikLg\nP4l6RiCx5nNF2wbSQ7uYrvDpYa6ToKysXVUTAPLxG3C4BirrQDaSnTThTzMC7GUAmxKAK3tnBHXEqOIsnYZ3rD92iUr2XI65oFIbIT\nXUrYNapiDWYsPEGTaQTX8L1ZkrFaQTL8wC1Zko8aZFfzqmYbNi5OvJydnWWoaRc0eyvnFmtNh0utLQZEME4DXCU3RxET3q6pwsid8DolT1FZtWBE0V3F0XM\nffWx27IYj63dyTtT4UoJwtTgdtXeHAG4a0AGvbfM9p462qEbV3rMNynLWyzQDc3sN6nI-`)
		assert.Contains(t, valuesFileContent, "DiskSizeWithBuffer = 125.1")

		// Check if only secret marked fields are in values.xlvals
		secretsFileContent := GetFileContent(models.BlueprintOutputDir + string(os.PathSeparator) + secretsFile)

		assert.Contains(t, secretsFileContent, "AWSAccessKey = accesskey")
		assert.Contains(t, secretsFileContent, "AWSAccessSecret = accesssecret")
		assert.NotContains(t, secretsFileContent, "SuperSecret = invisible")

	})

	t.Run("should create output files for valid test template composed from local path", func(t *testing.T) {
		gb := &GeneratedBlueprint{OutputDir: "xebialabs"}
		defer gb.Cleanup()
		err := InstantiateBlueprint(
			false,
			"composed",
			getLocalTestBlueprintContext(t),
			gb,
			"",
			false,
			true,
			false,
		)
		require.Nil(t, err)

		// assertions
		assert.FileExists(t, "xld-environment.yml")                  // this comes from composed blueprint 'valid-no-prompt'
		assert.FileExists(t, "xld-infrastructure.yml")               // this comes from composed blueprint 'valid-no-prompt'
		assert.True(t, util.PathExists("xlr-pipeline.yml", false))   // this comes from composed blueprint 'defaults-as-values'
		assert.True(t, util.PathExists("xlr-pipeline-4.yml", false)) // this comes from blueprint 'composed'

		// these files are from the main blueprint 'composed'
		assert.FileExists(t, path.Join(gb.OutputDir, valuesFile))
		assert.FileExists(t, path.Join(gb.OutputDir, secretsFile))
		assert.FileExists(t, path.Join(gb.OutputDir, gitignoreFile))

		envFile := GetFileContent("xld-environment.yml")
		assert.Contains(t, envFile, fmt.Sprintf("region: %s", "eu-central-1")) // the value is overridden by the last blueprint composed
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
			"Test":               "testing",
			"ClientCert":         "this is a multiline\\ntext\\n\\nwith escape chars\\n",
			"AppName":            "TestApp",
			"SuperSecret":        "supersecret",
			"AWSRegion":          "eu-central-1",
			"DiskSize":           "10",
			"DiskSizeWithBuffer": "125.6",
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
}

func TestShouldSkipFile(t *testing.T) {
	type args struct {
		templateConfig TemplateConfig
		variables      *[]Variable
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
				TemplateConfig{Path: "foo.yaml", DependsOn: VarField{Val: "foo"}},
				&[]Variable{
					{Name: VarField{Val: "foo"}, Value: VarField{Bool: false}},
				},
			},
			true,
			false,
		},
		{
			"should return true if dependsOnFalse is defined and its value is true",
			args{
				TemplateConfig{Path: "foo.yaml", DependsOnFalse: VarField{Val: "foo"}},
				&[]Variable{
					{Name: VarField{Val: "foo"}, Value: VarField{Bool: true}},
				},
			},
			true,
			false,
		},
		{
			"should return error if dependsOn value cannot be processed",
			args{
				TemplateConfig{Path: "foo.yaml", DependsOnFalse: VarField{Val: "foo"}},
				&[]Variable{},
			},
			false,
			true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := shouldSkipFile(tt.args.templateConfig, tt.args.variables, dummyData)
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
	require.Len(t, blueprints, 3)

	type args struct {
		blueprintContext   *BlueprintContext
		blueprintLocalMode bool
		blueprints         map[string]*models.BlueprintRemote
		templatePath       string
	}
	tests := []struct {
		name    string
		args    args
		want    *BlueprintConfig
		wantErr bool
	}{
		{
			"should error when invalid path is passed",
			args{
				repo,
				false,
				blueprints,
				"test",
			},
			nil,
			true,
		},
		{
			"should get blueprint config for a simple definition without composed includes",
			args{
				repo,
				false,
				blueprints,
				"aws/monolith",
			},
			&BlueprintConfig{
				ApiVersion: "xl/v1",
				Kind:       "Blueprint",
				Metadata:   Metadata{ProjectName: "Test Project"},
				TemplateConfigs: []TemplateConfig{
					{Path: "xld-environment.yml.tmpl", FullPath: "aws/monolith/xld-environment.yml.tmpl"},
					{Path: "xld-infrastructure.yml.tmpl", FullPath: "aws/monolith/xld-infrastructure.yml.tmpl"},
					{Path: "xlr-pipeline.yml", FullPath: "aws/monolith/xlr-pipeline.yml"},
				},
				Include: []IncludedBlueprintProcessed{},
				Variables: []Variable{
					{Name: VarField{Val: "Test", Bool: false, Tag: ""}, Type: VarField{Val: "Input", Bool: false, Tag: ""}, Value: VarField{Val: "testing", Bool: false, Tag: ""}},
				},
			},
			false,
		},
		{
			"should get blueprint config for a simple definition with composed includes",
			args{
				repo,
				false,
				blueprints,
				"aws/compose",
			},
			&BlueprintConfig{
				ApiVersion: "xl/v1",
				Kind:       "Blueprint",
				Metadata:   Metadata{ProjectName: "Test Project"},
				Include: []IncludedBlueprintProcessed{
					IncludedBlueprintProcessed{
						Blueprint: "aws/monolith",
						Stage:     "before",
						ParameterValues: []ParameterValuesProcessed{
							{
								Name:      "Foo",
								Value:     VarField{Val: "hello"},
								DependsOn: VarField{Tag: "!expression", Val: "ExpTest1 == 'us-west' && AppName != 'foo' && TestDepends"},
							},
							{
								Name:  "bar",
								Value: VarField{Val: "true", Bool: true},
							},
						},
						SkipFiles: []TemplateConfig{
							{
								Path:      "xld-infrastructure.yml.tmpl",
								DependsOn: VarField{Val: "TestDepends"},
							},
						},
					},
					IncludedBlueprintProcessed{
						Blueprint: "aws/datalake",
						Stage:     "after",
						ParameterValues: []ParameterValuesProcessed{
							{
								Name:  "Foo",
								Value: VarField{Val: "hello"},
							},
						},
						DependsOn: VarField{Tag: "!expression", Val: "ExpTest1 == 'us-west' && AppName != 'foo' && TestDepends"},
						RenameFiles: []TemplateConfig{
							{
								Path:      "xlr-pipeline.yml",
								RenameTo:  VarField{Val: "xlr-pipeline2.yml"},
								DependsOn: VarField{Val: "TestDepends"},
							},
						},
					},
				},
				Variables: []Variable{
					{Name: VarField{Val: "Test", Bool: false, Tag: ""}, Type: VarField{Val: "Input", Bool: false, Tag: ""}, Value: VarField{Val: "testing", Bool: false, Tag: ""}},
					{Name: VarField{Val: "Bar", Bool: false, Tag: ""}, Type: VarField{Val: "Input", Bool: false, Tag: ""}, Value: VarField{Val: "testing", Bool: false, Tag: ""}},
					{Name: VarField{Val: "Foo", Bool: false, Tag: ""}, Type: VarField{Val: "Input", Bool: false, Tag: ""}, Value: VarField{Val: "testing", Bool: false, Tag: ""}},
				},
				TemplateConfigs: []TemplateConfig{
					{Path: "xld-environment.yml.tmpl", FullPath: "aws/monolith/xld-environment.yml.tmpl"},
					{Path: "xld-infrastructure.yml.tmpl", FullPath: "aws/monolith/xld-infrastructure.yml.tmpl"},
					{Path: "xlr-pipeline.yml", FullPath: "aws/monolith/xlr-pipeline.yml"},
					{Path: "xld-environment.yml.tmpl", FullPath: "aws/compose/xld-environment.yml.tmpl"},
					{Path: "xld-infrastructure.yml.tmpl", FullPath: "aws/compose/xld-infrastructure.yml.tmpl"},
					{Path: "xlr-pipeline.yml", FullPath: "aws/compose/xlr-pipeline.yml"},
					{Path: "xld-app.yml.tmpl", FullPath: "aws/datalake/xld-app.yml.tmpl"},
					{Path: "xlr-pipeline.yml", FullPath: "aws/datalake/xlr-pipeline.yml"},
				},
			},
			false,
		},
		// TODO add test
		// {
		// 	"should get blueprint config for a nested blueprint compose scenario",
		// 	args{
		// 		repo,
		// 		false,
		// 		blueprints,
		// 		"aws/composenested",
		// 	},
		// 	&BlueprintConfig{},
		// 	false,
		// },
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := getBlueprintConfig(tt.args.blueprintContext, tt.args.blueprintLocalMode, tt.args.blueprints, tt.args.templatePath)
			if (err != nil) != tt.wantErr {
				t.Errorf("getBlueprintConfig() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			assert.Equal(t, tt.want, got)
		})
	}
}

func Test_composeBlueprints(t *testing.T) {
	defer httpmock.DeactivateAndReset()
	repo := getMockHttpBlueprintContext(t)
	blueprints, err := repo.initCurrentRepoClient()
	require.Nil(t, err)
	require.NotNil(t, blueprints)
	require.Len(t, blueprints, 3)

	type args struct {
		blueprintDoc       *BlueprintConfig
		blueprintContext   *BlueprintContext
		blueprintLocalMode bool
		blueprints         map[string]*models.BlueprintRemote
	}
	tests := []struct {
		name    string
		args    args
		want    *BlueprintConfig
		wantErr bool
	}{
		{
			"should error when invalid config is passed",
			args{
				&BlueprintConfig{
					Include: []IncludedBlueprintProcessed{
						IncludedBlueprintProcessed{
							Blueprint: "aws/nonexisting",
							Stage:     "after",
						},
					},
				},
				repo,
				false,
				blueprints,
			},
			&BlueprintConfig{
				Include: []IncludedBlueprintProcessed{
					IncludedBlueprintProcessed{
						Blueprint: "aws/nonexisting",
						Stage:     "after",
					},
				},
			},
			true,
		},
		{
			"should compose the given blueprints together in after stage by default",
			args{
				&BlueprintConfig{
					ApiVersion: "xl/v1",
					Kind:       "Blueprint",
					Metadata:   Metadata{ProjectName: "Test Project"},
					Include: []IncludedBlueprintProcessed{
						IncludedBlueprintProcessed{
							Blueprint: "aws/datalake",
							ParameterValues: []ParameterValuesProcessed{
								{
									Name:  "Foo",
									Value: VarField{Val: "hello"},
								},
							},
							DependsOn: VarField{Tag: "!expression", Val: "ExpTest1 == 'us-west' && AppName != 'foo' && TestDepends"},
							RenameFiles: []TemplateConfig{
								{
									Path:      "xlr-pipeline.yml",
									RenameTo:  VarField{Val: "xlr-pipeline2.yml"},
									DependsOn: VarField{Val: "TestDepends"},
								},
							},
						},
					},
					Variables: []Variable{
						{Name: VarField{Val: "Bar", Bool: false, Tag: ""}, Type: VarField{Val: "Input", Bool: false, Tag: ""}, Value: VarField{Val: "testing", Bool: false, Tag: ""}},
					},
					TemplateConfigs: []TemplateConfig{
						{Path: "xld-environment.yml.tmpl", FullPath: "aws/compose/xld-environment.yml.tmpl"},
						{Path: "xld-infrastructure.yml.tmpl", FullPath: "aws/compose/xld-infrastructure.yml.tmpl"},
						{Path: "xlr-pipeline.yml", FullPath: "aws/compose/xlr-pipeline.yml"},
					},
				},
				repo,
				false,
				blueprints,
			},
			&BlueprintConfig{
				ApiVersion: "xl/v1",
				Kind:       "Blueprint",
				Metadata:   Metadata{ProjectName: "Test Project"},
				Include: []IncludedBlueprintProcessed{
					IncludedBlueprintProcessed{
						Blueprint: "aws/datalake",
						ParameterValues: []ParameterValuesProcessed{
							{
								Name:  "Foo",
								Value: VarField{Val: "hello"},
							},
						},
						DependsOn: VarField{Tag: "!expression", Val: "ExpTest1 == 'us-west' && AppName != 'foo' && TestDepends"},
						RenameFiles: []TemplateConfig{
							{
								Path:      "xlr-pipeline.yml",
								RenameTo:  VarField{Val: "xlr-pipeline2.yml"},
								DependsOn: VarField{Val: "TestDepends"},
							},
						},
					},
				},
				Variables: []Variable{
					{Name: VarField{Val: "Bar", Bool: false, Tag: ""}, Type: VarField{Val: "Input", Bool: false, Tag: ""}, Value: VarField{Val: "testing", Bool: false, Tag: ""}},
					{Name: VarField{Val: "Foo", Bool: false, Tag: ""}, Type: VarField{Val: "Input", Bool: false, Tag: ""}, Value: VarField{Val: "testing", Bool: false, Tag: ""}},
				},
				TemplateConfigs: []TemplateConfig{
					{Path: "xld-environment.yml.tmpl", FullPath: "aws/compose/xld-environment.yml.tmpl"},
					{Path: "xld-infrastructure.yml.tmpl", FullPath: "aws/compose/xld-infrastructure.yml.tmpl"},
					{Path: "xlr-pipeline.yml", FullPath: "aws/compose/xlr-pipeline.yml"},
					{Path: "xld-app.yml.tmpl", FullPath: "aws/datalake/xld-app.yml.tmpl"},
					{Path: "xlr-pipeline.yml", FullPath: "aws/datalake/xlr-pipeline.yml"},
				},
			},
			false,
		},
		{
			"should compose the given blueprints together in before and after stage accordingly",
			args{
				&BlueprintConfig{
					ApiVersion: "xl/v1",
					Kind:       "Blueprint",
					Metadata:   Metadata{ProjectName: "Test Project"},
					Include: []IncludedBlueprintProcessed{
						IncludedBlueprintProcessed{
							Blueprint: "aws/monolith",
							Stage:     "before",
							ParameterValues: []ParameterValuesProcessed{
								{
									Name:      "Foo",
									Value:     VarField{Val: "hello"},
									DependsOn: VarField{Tag: "!expression", Val: "ExpTest1 == 'us-west' && AppName != 'foo' && TestDepends"},
								},
								{
									Name:  "bar",
									Value: VarField{Val: "true", Bool: true},
								},
							},
							SkipFiles: []TemplateConfig{
								{
									Path:      "xld-infrastructure.yml.tmpl",
									DependsOn: VarField{Val: "TestDepends"},
								},
							},
						},
						IncludedBlueprintProcessed{
							Blueprint: "aws/datalake",
							Stage:     "after",
							ParameterValues: []ParameterValuesProcessed{
								{
									Name:  "Foo",
									Value: VarField{Val: "hello"},
								},
							},
							DependsOn: VarField{Tag: "!expression", Val: "ExpTest1 == 'us-west' && AppName != 'foo' && TestDepends"},
							RenameFiles: []TemplateConfig{
								{
									Path:      "xlr-pipeline.yml",
									RenameTo:  VarField{Val: "xlr-pipeline2.yml"},
									DependsOn: VarField{Val: "TestDepends"},
								},
							},
						},
					},
					Variables: []Variable{
						{Name: VarField{Val: "Bar", Bool: false, Tag: ""}, Type: VarField{Val: "Input", Bool: false, Tag: ""}, Value: VarField{Val: "testing", Bool: false, Tag: ""}},
					},
					TemplateConfigs: []TemplateConfig{
						{Path: "xld-environment.yml.tmpl", FullPath: "aws/compose/xld-environment.yml.tmpl"},
						{Path: "xld-infrastructure.yml.tmpl", FullPath: "aws/compose/xld-infrastructure.yml.tmpl"},
						{Path: "xlr-pipeline.yml", FullPath: "aws/compose/xlr-pipeline.yml"},
					},
				},
				repo,
				false,
				blueprints,
			},
			&BlueprintConfig{
				ApiVersion: "xl/v1",
				Kind:       "Blueprint",
				Metadata:   Metadata{ProjectName: "Test Project"},
				Include: []IncludedBlueprintProcessed{
					IncludedBlueprintProcessed{
						Blueprint: "aws/monolith",
						Stage:     "before",
						ParameterValues: []ParameterValuesProcessed{
							{
								Name:      "Foo",
								Value:     VarField{Val: "hello"},
								DependsOn: VarField{Tag: "!expression", Val: "ExpTest1 == 'us-west' && AppName != 'foo' && TestDepends"},
							},
							{
								Name:  "bar",
								Value: VarField{Val: "true", Bool: true},
							},
						},
						SkipFiles: []TemplateConfig{
							{
								Path:      "xld-infrastructure.yml.tmpl",
								DependsOn: VarField{Val: "TestDepends"},
							},
						},
					},
					IncludedBlueprintProcessed{
						Blueprint: "aws/datalake",
						Stage:     "after",
						ParameterValues: []ParameterValuesProcessed{
							{
								Name:  "Foo",
								Value: VarField{Val: "hello"},
							},
						},
						DependsOn: VarField{Tag: "!expression", Val: "ExpTest1 == 'us-west' && AppName != 'foo' && TestDepends"},
						RenameFiles: []TemplateConfig{
							{
								Path:      "xlr-pipeline.yml",
								RenameTo:  VarField{Val: "xlr-pipeline2.yml"},
								DependsOn: VarField{Val: "TestDepends"},
							},
						},
					},
				},
				Variables: []Variable{
					{Name: VarField{Val: "Test", Bool: false, Tag: ""}, Type: VarField{Val: "Input", Bool: false, Tag: ""}, Value: VarField{Val: "testing", Bool: false, Tag: ""}},
					{Name: VarField{Val: "Bar", Bool: false, Tag: ""}, Type: VarField{Val: "Input", Bool: false, Tag: ""}, Value: VarField{Val: "testing", Bool: false, Tag: ""}},
					{Name: VarField{Val: "Foo", Bool: false, Tag: ""}, Type: VarField{Val: "Input", Bool: false, Tag: ""}, Value: VarField{Val: "testing", Bool: false, Tag: ""}},
				},
				TemplateConfigs: []TemplateConfig{
					{Path: "xld-environment.yml.tmpl", FullPath: "aws/monolith/xld-environment.yml.tmpl"},
					{Path: "xld-infrastructure.yml.tmpl", FullPath: "aws/monolith/xld-infrastructure.yml.tmpl"},
					{Path: "xlr-pipeline.yml", FullPath: "aws/monolith/xlr-pipeline.yml"},
					{Path: "xld-environment.yml.tmpl", FullPath: "aws/compose/xld-environment.yml.tmpl"},
					{Path: "xld-infrastructure.yml.tmpl", FullPath: "aws/compose/xld-infrastructure.yml.tmpl"},
					{Path: "xlr-pipeline.yml", FullPath: "aws/compose/xlr-pipeline.yml"},
					{Path: "xld-app.yml.tmpl", FullPath: "aws/datalake/xld-app.yml.tmpl"},
					{Path: "xlr-pipeline.yml", FullPath: "aws/datalake/xlr-pipeline.yml"},
				},
			},
			false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := composeBlueprints(tt.args.blueprintDoc, tt.args.blueprintContext, tt.args.blueprintLocalMode, tt.args.blueprints); (err != nil) != tt.wantErr {
				t.Errorf("composeBlueprints() error = %v, wantErr %v", err, tt.wantErr)
			}
			assert.Equal(t, tt.want, tt.args.blueprintDoc)
		})
	}
}

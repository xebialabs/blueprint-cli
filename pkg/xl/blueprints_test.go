package xl

import (
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Netflix/go-expect"
	"github.com/hinshun/vt10x"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/AlecAivazis/survey.v1/terminal"
)

type UserInput struct {
	inputType  string
	inputValue string
}

// run test in virtual console
func RunInVirtualConsole(t *testing.T, procedure func(*expect.Console), test func(terminal.Stdio) error) {
	c, state, err := vt10x.NewVT10XConsole()
	require.Nil(t, err)
	defer c.Close()

	donec := make(chan struct{})
	go func() {
		defer close(donec)
		procedure(c)
		c.ExpectEOF()
	}()
	stdio := terminal.Stdio{In: c.Tty(), Out: c.Tty(), Err: c.Tty()}
	err = test(stdio)
	require.Nil(t, err)

	// Close the slave end of the pty, and read the remaining bytes from the master end.
	c.Tty().Close()
	<-donec

	// Dump the terminal's screen.
	t.Log(expect.StripTrailingEmptyLines(state.String()))
}

// mimic sending prompt values to console
func SendPromptValues(values map[string]UserInput) func(c *expect.Console) {
	return func(console *expect.Console) {
		for k, input := range values {
			// TODO: If description field exists, expect desc text instead!
			_, err := console.Expect(expect.String(fmt.Sprintf("%s?", k)))
			if err != nil {
				panic(err)
			}
			_, err = console.SendLine(input.inputValue)
			if err != nil {
				panic(err)
			}
		}
	}
}

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
	return strings.Replace(pwd, path.Join("pkg", "xl"), path.Join("templates", "test", blueprint), -1)
}

func TestWriteDataToFile(t *testing.T) {
	t.Run("should write template data to output file", func(t *testing.T) {
		data := "test\ndata\n"
		filePath := "test.yml"
		err := writeDataToFile(filePath, &data)
		defer os.Remove("test.yml")
		require.Nil(t, err)
		assert.FileExists(t, filePath)
		assert.Equal(t, GetFileContent(filePath), data)
	})
	t.Run("should write template data to output file in a folder", func(t *testing.T) {
		data := "test\ndata\n"
		filePath := path.Join("test", "test.yml")
		err := writeDataToFile(filePath, &data)
		defer os.RemoveAll("test")
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
		err := writeConfigToFile("#comment", config, filePath)
		defer os.RemoveAll("test.xlvals")
		require.Nil(t, err)
		assert.FileExists(t, filePath)
		assert.Equal(t, "#comment\na = true\nd = 1\nz = test", strings.TrimSpace(GetFileContent(filePath)))
	})
	t.Run("should write config data to output file in folder", func(t *testing.T) {
		defer os.RemoveAll("test")
		config := make(map[string]interface{}, 3)
		config["d"] = 1
		config["a"] = true
		config["z"] = "test"
		filePath := path.Join("test", "test.xlvals")
		err := writeConfigToFile("#comment", config, filePath)
		require.Nil(t, err)
		assert.FileExists(t, filePath)
		assert.Equal(t, "#comment\na = true\nd = 1\nz = test", strings.TrimSpace(GetFileContent(filePath)))
	})
}

func TestAdjustPathSeperatorIfNeeded(t *testing.T) {
	t.Run("should produce standard path using the host os seperator", func(t *testing.T) {
		assert.Equal(t, "", adjustPathSeperatorIfNeeded(""))
		assert.Equal(t, "test", adjustPathSeperatorIfNeeded("test"))
		assert.Equal(t, path.Join("..", "test"), adjustPathSeperatorIfNeeded("../test"))
		assert.Equal(t, path.Join("..", "microservice", "blueprint.yaml"), adjustPathSeperatorIfNeeded(`..\microservice\blueprint.yaml`))
		assert.Equal(t, path.Join("..", "microservice", "blueprint.yaml"), adjustPathSeperatorIfNeeded(`../microservice\blueprint.yaml`))
		assert.Equal(t, path.Join("..", "microservice", "blueprint.yaml"), adjustPathSeperatorIfNeeded(`../microservice/blueprint.yaml`))
		assert.Equal(t, path.Join("test", "test", "again"), adjustPathSeperatorIfNeeded(`test/test\again`))
		assert.Equal(t, path.Join("test", "test", "again"), adjustPathSeperatorIfNeeded(`test\test\again`))
		assert.Equal(t, path.Join("test", "test", "again"), adjustPathSeperatorIfNeeded(`test/test/again`))
	})
}

func TestCreateBlueprint(t *testing.T) {
	SkipFinalPrompt = true
	t.Run("should error on unknown template", func(t *testing.T) {
		err := InstantiateBlueprint("abc", BlueprintRepository{}, "xebialabs")

		require.NotNil(t, err)
		assert.Equal(t, "template not found in path abc/blueprint.yml", err.Error())
	})
	t.Run("should error on invalid test template", func(t *testing.T) {
		err := InstantiateBlueprint(GetTestTemplateDir("invalid"), BlueprintRepository{}, "xebialabs")

		require.NotNil(t, err)
		assert.Equal(t, "parameter [Test] is missing required fields: [type]", err.Error())
	})
	t.Run("should create output files for valid test template without prompts when no registry is defined", func(t *testing.T) {
		outFolder := "xebialabs"
		defer RemoveFiles("xld-*.yml")
		defer RemoveFiles("xlr-*.yml")
		defer os.RemoveAll(outFolder)
		// create blueprint
		err := InstantiateBlueprint(GetTestTemplateDir("valid-no-prompt"), BlueprintRepository{}, outFolder)
		require.Nil(t, err)

		// assertions
		assert.FileExists(t, "xld-environment.yml")
		assert.FileExists(t, "xld-infrastructure.yml")
		assert.FileExists(t, "xlr-pipeline.yml")
		assert.False(t, PathExists("xlr-pipeline-2.yml", false))
		assert.FileExists(t, path.Join(outFolder, valuesFile))
		assert.FileExists(t, path.Join(outFolder, secretsFile))
		assert.FileExists(t, path.Join(outFolder, gitignoreFile))
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
	t.Run("should create output files for valid test template from local path when a registry is defined", func(t *testing.T) {
		outFolder := "xebialabs"
		defer RemoveFiles("xld-*.yml")
		defer RemoveFiles("xlr-*.yml")
		defer os.RemoveAll(outFolder)
		// create blueprint
		repository := BlueprintRepository{SimpleHTTPServer{Url: parseURIWithoutError("https://dist.xebialabs.com/public/blueprints/")}}
		err := InstantiateBlueprint(GetTestTemplateDir("valid-no-prompt"), repository, outFolder)
		require.Nil(t, err)

		// assertions
		assert.FileExists(t, "xld-environment.yml")
		assert.FileExists(t, "xld-infrastructure.yml")
		assert.FileExists(t, "xlr-pipeline.yml")
		assert.False(t, PathExists("xlr-pipeline-2.yml", false))
		assert.FileExists(t, path.Join(outFolder, valuesFile))
		assert.FileExists(t, path.Join(outFolder, secretsFile))
		assert.FileExists(t, path.Join(outFolder, gitignoreFile))
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
}

func TestCreateDirectoryIfNeeded(t *testing.T) {
	tmpDir, err := ioutil.TempDir("", "pathTest")
	require.Nil(t, err)
	defer os.RemoveAll(tmpDir)
	defer os.RemoveAll("test")
	type args struct {
		fileName string
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{"create directory if it doesn't exist", args{path.Join("test", "test.xlval")}, false},
		{"Do not create directory if it exists", args{path.Join(tmpDir, "test.xlval")}, false},
		{"Do not do anything if there is no directory", args{"test.xlval"}, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := createDirectoryIfNeeded(tt.args.fileName); (err != nil) != tt.wantErr {
				t.Errorf("createDirectoryIfNeeded() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
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
				TemplateConfig{File: "foo.yaml"},
				nil,
			},
			false,
			false,
		},
		{
			"should return true if dependsOnTrue is defined and its value is false",
			args{
				TemplateConfig{File: "foo.yaml", DependsOnTrue: VarField{Val: "foo"}},
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
				TemplateConfig{File: "foo.yaml", DependsOnFalse: VarField{Val: "foo"}},
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
				TemplateConfig{File: "foo.yaml", DependsOnFalse: VarField{Val: "foo"}},
				&[]Variable{},
			},
			false,
			true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := shouldSkipFile(tt.args.templateConfig, tt.args.variables)
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

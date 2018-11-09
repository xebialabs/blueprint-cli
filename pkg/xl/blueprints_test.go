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
	"gopkg.in/AlecAivazis/survey.v1"
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
			console.SendLine(input.inputValue)
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

func getValidTestBlueprintMetadata() (*BlueprintYaml, error) {
	metadata := []byte(
		`apiVersion: xl-cli/v1beta1
kind: Blueprint
metadata:
spec:
- name: pass
  type: Input
  secret: true
- name: test
  type: Input
  default: lala
  description: help text
- name: fn
  type: Input
  value: !fn aws.regions(ecs)[0]
- name: select
  type: Select
  options:
  - !fn aws.regions(ecs)[0]
  - b
  - c
  default: b
- name: isit
  type: Confirm
- name: isitnot
  type: Confirm
- name: dep
  type: Input
  dependsOnTrue: isit
  dependsOnFalse: isitnot`)
	return parseTemplateMetadata(&metadata)
}

func TestWriteDataToFile(t *testing.T) {
	t.Run("should write template data to output file", func(t *testing.T) {
		os.MkdirAll("test", os.ModePerm)
		defer os.RemoveAll("test")
		data := "test\ndata\n"
		filePath := path.Join("test", "test.yml")
		err := writeDataToFile(filePath, &data)
		require.Nil(t, err)
		assert.FileExists(t, filePath)
		assert.Equal(t, GetFileContent(filePath), data)
	})
}

func TestWriteConfigToFile(t *testing.T) {
	t.Run("should write config data to output file sorted", func(t *testing.T) {
		os.MkdirAll("test", os.ModePerm)
		defer os.RemoveAll("test")
		config := make(map[string]interface{}, 3)
		config["d"] = 1
		config["a"] = true
		config["z"] = "test"
		filePath := path.Join("test", "test.xlvals")
		err := writeConfigToFile(config, filePath)
		require.Nil(t, err)
		assert.FileExists(t, filePath)
		assert.Equal(t, strings.TrimSpace(GetFileContent(filePath)), "a = true\nd = 1\nz = test")
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
	t.Run("should error on unknown template", func(t *testing.T) {
		RunInVirtualConsole(t, func(c *expect.Console) {}, func(stdio terminal.Stdio) error {
			err := CreateBlueprint("abc", []TemplateRegistry{}, survey.WithStdio(stdio.In, stdio.Out, stdio.Err))

			require.NotNil(t, err)
			assert.Equal(t, "template configuration not found for path abc", err.Error())

			return nil
		})
	})
	t.Run("should error on invalid test template", func(t *testing.T) {
		RunInVirtualConsole(t, func(c *expect.Console) {}, func(stdio terminal.Stdio) error {
			err := CreateBlueprint(GetTestTemplateDir("invalid"), []TemplateRegistry{}, survey.WithStdio(stdio.In, stdio.Out, stdio.Err))

			require.NotNil(t, err)
			assert.Equal(t, "parameter [Test] is missing required fields: [type]", err.Error())

			return nil
		})
	})
	// todo: tests are hanging randomly!
	/*t.Run("should create output files for valid test template", func(t *testing.T) {
		userAnswers := make(map[string]UserInput)
		userAnswers["AppName"] = UserInput{inputType: TypeInput, inputValue: "test-project"}
		userAnswers["AWSRegion"] = UserInput{inputType: TypeSelect, inputValue: "eu-west-1"}

		RunInVirtualConsole(t, SendPromptValues(userAnswers), func(stdio terminal.Stdio) error {
			// create blueprint
			err := CreateBlueprint(GetTestTemplateDir("valid"), []TemplateRegistry{}, survey.WithStdio(stdio.In, stdio.Out, stdio.Err))
			require.Nil(t, err)

			// assertions
			assert.FileExists(t, "xld-environment.yml")
			assert.FileExists(t, "xld-infrastructure.yml")
			assert.FileExists(t, "xlr-pipeline.yml")
			assert.FileExists(t, valuesFile + xlvalsExt)
			assert.FileExists(t, secretsFile + xlvalsExt)
			assert.FileExists(t, gitignoreFile)
			envFile := GetFileContent("xld-environment.yml")
			assert.Contains(t, envFile, fmt.Sprintf("region: %s", userAnswers["AWSRegion"].inputValue))
			infraFile := GetFileContent("xld-infrastructure.yml")
			infraChecks := []string{
				fmt.Sprintf("- name: %s-ecs-fargate-cluster", userAnswers["AppName"].inputValue),
				fmt.Sprintf("- name: %s-ecs-vpc", userAnswers["AppName"].inputValue),
				fmt.Sprintf("- name: %s-ecs-subnet-ipv4-az-1a", userAnswers["AppName"].inputValue),
				fmt.Sprintf("- name: %s-ecs-route-table", userAnswers["AppName"].inputValue),
				fmt.Sprintf("- name: %s-ecs-security-group", userAnswers["AppName"].inputValue),
				fmt.Sprintf("- name: %s-targetgroup", userAnswers["AppName"].inputValue),
				fmt.Sprintf("- name: %s-ecs-alb", userAnswers["AppName"].inputValue),
				fmt.Sprintf("- name: %s-ecs-db-subnet-group", userAnswers["AppName"].inputValue),
				fmt.Sprintf("- name: %s-ecs-dictionary", userAnswers["AppName"].inputValue),
				"MYSQL_DB_ADDRESS: '{{%address%}}'",
			}
			for _, infraCheck := range infraChecks {
				assert.Contains(t, infraFile, infraCheck)
			}

			// cleanup any files created
			RemoveFiles("xld-*.yml")
			RemoveFiles("xlr-*.yml")
			RemoveFiles(valuesFile + xlvalsExt)
			RemoveFiles(secretsFile + xlvalsExt)
			RemoveFiles(gitignoreFile)
			return err
		})
	})*/
}

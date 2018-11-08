package xl

import (
	"fmt"
	"sort"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/xebialabs/xl-cli/pkg/cloud/aws"
)

func TestGetVariableDefaultVal(t *testing.T) {
	t.Run("should return empty string when default is not defined", func(t *testing.T) {
		v := Variable{
			Name: VarField{Val: "test"},
			Type: VarField{Val: TypeInput},
		}
		defaultVal := v.GetDefaultVal()
		assert.Equal(t, "", defaultVal)
	})

	t.Run("should return default value string when default is defined", func(t *testing.T) {
		v := Variable{
			Name:    VarField{Val: "test"},
			Type:    VarField{Val: TypeInput},
			Default: VarField{Val: "default_val"},
		}
		defaultVal := v.GetDefaultVal()
		assert.Equal(t, "default_val", defaultVal)
	})

	t.Run("should return false string when confirm field is not set", func(t *testing.T) {
		v := Variable{
			Name: VarField{Val: "test"},
			Type: VarField{Val: TypeConfirm},
		}
		defaultVal := v.GetDefaultVal()
		assert.Equal(t, "false", defaultVal)
	})

	t.Run("should return empty string when invalid function tag in default field", func(t *testing.T) {
		v := Variable{
			Name:    VarField{Val: "test"},
			Type:    VarField{Val: TypeInput},
			Default: VarField{Val: "aws.regs", Tag: tagFn},
		}
		defaultVal := v.GetDefaultVal()
		assert.Equal(t, "", defaultVal)
	})

	t.Run("should return function output on valid function tag in default field", func(t *testing.T) {
		v := Variable{
			Name:    VarField{Val: "test"},
			Type:    VarField{Val: TypeInput},
			Default: VarField{Val: "aws.regions(ecs)[0]", Tag: tagFn},
		}
		defaultVal := v.GetDefaultVal()
		regionsList, _ := aws.GetAvailableAWSRegionsForService("ecs")
		sort.Strings(regionsList)
		assert.Equal(t, regionsList[0], defaultVal)
	})
}

func TestGetValueFieldVal(t *testing.T) {
	t.Run("should return value field string value when defined", func(t *testing.T) {
		v := Variable{
			Name:  VarField{Val: "test"},
			Type:  VarField{Val: TypeInput},
			Value: VarField{Val: "testing"},
		}
		val := v.GetValueFieldVal()
		assert.Equal(t, "testing", val)
	})

	t.Run("should return empty on invalid function tag in value field", func(t *testing.T) {
		v := Variable{
			Name:  VarField{Val: "test"},
			Type:  VarField{Val: TypeInput},
			Value: VarField{Val: "aws.regs", Tag: tagFn},
		}
		val := v.GetValueFieldVal()
		assert.Equal(t, "", val)
	})

	t.Run("should return function output on valid function tag in value field", func(t *testing.T) {
		v := Variable{
			Name:  VarField{Val: "test"},
			Type:  VarField{Val: TypeInput},
			Value: VarField{Val: "aws.regions(ecs)[0]", Tag: tagFn},
		}
		val := v.GetValueFieldVal()
		regionsList, _ := aws.GetAvailableAWSRegionsForService("ecs")
		sort.Strings(regionsList)
		assert.Equal(t, regionsList[0], val)
	})
}

func TestGetOptions(t *testing.T) {
	t.Run("should return string values of options", func(t *testing.T) {
		v := Variable{
			Name:    VarField{Val: "test"},
			Type:    VarField{Val: TypeSelect},
			Options: []VarField{{Val: "a"}, {Val: "b"}, {Val: "c"}},
		}
		values := v.GetOptions()
		assert.Len(t, values, 3)
		assert.Equal(t, []string{"a", "b", "c"}, values)
	})

	t.Run("should return generated values for fn options tag", func(t *testing.T) {
		v := Variable{
			Name:    VarField{Val: "test"},
			Type:    VarField{Val: TypeSelect},
			Options: []VarField{{Val: "aws.regions(ecs)", Tag: "!fn"}},
		}
		values := v.GetOptions()
		assert.True(t, len(values) > 1)
	})

	t.Run("should return nil on invalid function tag for options", func(t *testing.T) {
		v := Variable{
			Name:    VarField{Val: "test"},
			Type:    VarField{Val: TypeSelect},
			Options: []VarField{{Val: "aws.regs", Tag: "!fn"}},
		}
		out := v.GetOptions()
		require.Nil(t, out)
	})
}

func TestSkipQuestionOnCondition(t *testing.T) {
	t.Run("should skip question (dependsOnFalse)", func(t *testing.T) {
		variables := make([]Variable, 2)
		variables[0] = Variable{
			Name:  VarField{Val: "confirm"},
			Type:  VarField{Val: TypeConfirm},
			Value: VarField{Bool: true},
		}
		variables[1] = Variable{
			Name:           VarField{Val: "test"},
			Type:           VarField{Val: TypeInput},
			DependsOnFalse: VarField{Val: "confirm"},
		}
		assert.True(t, skipQuestionOnCondition(&variables[1], variables[1].DependsOnFalse.Val, &variables[0], NewPreparedData(), "", true))
	})
	t.Run("should skip question (dependsOnTrue)", func(t *testing.T) {
		variables := make([]Variable, 2)
		variables[0] = Variable{
			Name:  VarField{Val: "confirm"},
			Type:  VarField{Val: TypeConfirm},
			Value: VarField{Bool: false},
		}
		variables[1] = Variable{
			Name:          VarField{Val: "test"},
			Type:          VarField{Val: TypeInput},
			DependsOnTrue: VarField{Val: "confirm"},
		}
		assert.True(t, skipQuestionOnCondition(&variables[1], variables[1].DependsOnTrue.Val, &variables[0], NewPreparedData(), "", false))
	})

	t.Run("should not skip question (dependsOnFalse)", func(t *testing.T) {
		variables := make([]Variable, 2)
		variables[0] = Variable{
			Name:  VarField{Val: "confirm"},
			Type:  VarField{Val: TypeConfirm},
			Value: VarField{Bool: false},
		}
		variables[1] = Variable{
			Name:           VarField{Val: "test"},
			Type:           VarField{Val: TypeInput},
			DependsOnFalse: VarField{Val: "confirm"},
		}
		assert.False(t, skipQuestionOnCondition(&variables[1], variables[1].DependsOnFalse.Val, &variables[0], NewPreparedData(), "", true))
	})
	t.Run("should not skip question (dependsOnTrue)", func(t *testing.T) {
		variables := make([]Variable, 2)
		variables[0] = Variable{
			Name:  VarField{Val: "confirm"},
			Type:  VarField{Val: TypeConfirm},
			Value: VarField{Bool: true},
		}
		variables[1] = Variable{
			Name:          VarField{Val: "test"},
			Type:          VarField{Val: TypeInput},
			DependsOnTrue: VarField{Val: "confirm"},
		}
		assert.False(t, skipQuestionOnCondition(&variables[1], variables[1].DependsOnTrue.Val, &variables[0], NewPreparedData(), "", false))
	})
}

func TestParseAndValidateTemplateMetadata(t *testing.T) {
	t.Run("should error on missing api version", func(t *testing.T) {
		metadata := []byte("")
		_, err := parseTemplateMetadata(&metadata)
		require.NotNil(t, err)
		assert.Equal(t, fmt.Sprintf("api version needs to be %s", apiVersion), err.Error())
	})

	t.Run("should error on missing doc kind", func(t *testing.T) {
		metadata := []byte("apiVersion: xl-cli/v1beta1")
		_, err := parseTemplateMetadata(&metadata)
		require.NotNil(t, err)
		assert.Equal(t, "yaml document kind needs to be Blueprint", err.Error())
	})

	t.Run("should error on unknown field type", func(t *testing.T) {
		metadata := []byte(
			`apiVersion: xl-cli/v1beta1
kind: Blueprint
metadata:
spec:
- name: Test
  type: Invalid
  value: testing`)
		_, err := parseTemplateMetadata(&metadata)
		require.NotNil(t, err)
		assert.Equal(t, "type [Invalid] is not valid for parameter [Test]", err.Error())
	})

	t.Run("should error on missing variable field", func(t *testing.T) {
		metadata := []byte(
			`apiVersion: xl-cli/v1beta1
kind: Blueprint
metadata:
spec:
- name: Test
  value: testing`)
		_, err := parseTemplateMetadata(&metadata)
		require.NotNil(t, err)
		assert.Equal(t, "parameter [Test] is missing required fields: [type]", err.Error())
	})

	t.Run("should error if default value is set for a field marked as secret", func(t *testing.T) {
		metadata := []byte(
			`apiVersion: xl-cli/v1beta1
kind: Blueprint
metadata:
spec:
- name: Test
  type: Input
  secret: true
  default: very_secret_pass`)
		_, err := parseTemplateMetadata(&metadata)
		require.NotNil(t, err)
		assert.Equal(t, "secret field [Test] is not allowed to have default value", err.Error())
	})

	t.Run("should error on missing options for variable", func(t *testing.T) {
		metadata := []byte(
			`apiVersion: xl-cli/v1beta1
kind: Blueprint
metadata:
spec:
- name: Test
  type: Select
  options:`)
		_, err := parseTemplateMetadata(&metadata)
		require.NotNil(t, err)
		assert.Equal(t, "at least one option field is need to be set for parameter [Test]", err.Error())
	})

	t.Run("should parse nested variables from valid metadata", func(t *testing.T) {
		doc, err := getValidTestBlueprintMetadata()
		require.Nil(t, err)
		assert.Len(t, doc.Variables, 7)
		assert.Equal(t, Variable{
			Name:   VarField{Val: "pass"},
			Type:   VarField{Val: TypeInput},
			Secret: VarField{Bool: true},
		}, doc.Variables[0])
		assert.Equal(t, Variable{
			Name:        VarField{Val: "test"},
			Type:        VarField{Val: TypeInput},
			Default:     VarField{Val: "lala"},
			Description: VarField{Val: "help text"},
		}, doc.Variables[1])
		assert.Equal(t, Variable{
			Name:  VarField{Val: "fn"},
			Type:  VarField{Val: TypeInput},
			Value: VarField{Val: "aws.regions(ecs)[0]", Tag: tagFn},
		}, doc.Variables[2])
		assert.Equal(t, Variable{
			Name: VarField{Val: "select"},
			Type: VarField{Val: TypeSelect},
			Options: []VarField{
				{Val: "aws.regions(ecs)[0]", Tag: tagFn},
				{Val: "b"},
				{Val: "c"},
			},
			Default: VarField{Val: "b"},
		}, doc.Variables[3])
		assert.Equal(t, Variable{
			Name: VarField{Val: "isit"},
			Type: VarField{Val: TypeConfirm},
		}, doc.Variables[4])
		assert.Equal(t, Variable{
			Name: VarField{Val: "isitnot"},
			Type: VarField{Val: TypeConfirm},
		}, doc.Variables[5])
		assert.Equal(t, Variable{
			Name:           VarField{Val: "dep"},
			Type:           VarField{Val: TypeInput},
			DependsOnTrue:  VarField{Val: "isit"},
			DependsOnFalse: VarField{Val: "isitnot"},
		}, doc.Variables[6])
	})
}

func TestPrepareTemplateData(t *testing.T) {
	// todo: tests are hanging randomly!
	/*t.Run("should not ask user for further input if confirm variable is false", func(t *testing.T) {
		doc, err := getValidTestBlueprintMetadata()
		require.Nil(t, err)
		userAnswers := make(map[string]UserInput)
		userAnswers["pass"] = UserInput{inputType: TypeInput, inputValue: "password"}
		userAnswers["test"] = UserInput{inputType: TypeInput, inputValue: "test"}
		userAnswers["select"] = UserInput{inputType: TypeSelect, inputValue: "c"}
		userAnswers["isit"] = UserInput{inputType: TypeConfirm, inputValue: "N"}
		userAnswers["isitnot"] = UserInput{inputType: TypeConfirm, inputValue: "y"}

		RunInVirtualConsole(t, SendPromptValues(userAnswers), func(stdio terminal.Stdio) error {
			preparedData, err := doc.prepareTemplateData(survey.WithStdio(stdio.In, stdio.Out, stdio.Err))
			require.Nil(t, err)
			require.NotNil(t, preparedData)

			return err
		})
	})
	t.Run("should ask user for further input if confirm variable is true", func(t *testing.T) {
		doc, err := getValidTestBlueprintMetadata()
		require.Nil(t, err)
		userAnswers := make(map[string]UserInput)
		userAnswers["pass"] = UserInput{inputType: TypeInput, inputValue: "password"}
		userAnswers["test"] = UserInput{inputType: TypeInput, inputValue: "test"}
		userAnswers["select"] = UserInput{inputType: TypeSelect, inputValue: "c"}
		userAnswers["isit"] = UserInput{inputType: TypeConfirm, inputValue: "y"}
		userAnswers["dep"] = UserInput{inputType: TypeInput, inputValue: "test2"}

		RunInVirtualConsole(t, SendPromptValues(userAnswers), func(stdio terminal.Stdio) error {
			preparedData, err := doc.prepareTemplateData(survey.WithStdio(stdio.In, stdio.Out, stdio.Err))
			require.Nil(t, err)
			require.NotNil(t, preparedData)
			require.NotNil(t, preparedData.TemplateData)
			assert.Len(t, preparedData.TemplateData, 6)
			require.NotNil(t, preparedData.Secrets)
			assert.Len(t, preparedData.Secrets, 1)
			assert.Equal(t, "password", *preparedData.Secrets["pass"].(*string))
			require.NotNil(t, preparedData.Values)
			assert.Len(t, preparedData.Values, 5)
			assert.Equal(t, "test", *preparedData.Values["test"].(*string))
			assert.Equal(t, "c", *preparedData.Values["select"].(*string))
			assert.Equal(t, "test2", *preparedData.Values["dep"].(*string))

			return err
		})
	})*/
}

func TestProcessCustomFunction(t *testing.T) {
	// Generic
	t.Run("should error on empty function string", func(t *testing.T) {
		_, err := processCustomFunction("")
		require.NotNil(t, err)
		assert.Contains(t, err.Error(), "invalid syntax in function reference:")
	})
	t.Run("should error on invalid function string", func(t *testing.T) {
		_, err := processCustomFunction("aws.regions.0")
		require.NotNil(t, err)
		assert.Equal(t, "invalid syntax in function reference: aws.regions.0", err.Error())
	})
	t.Run("should error on unknown function domain", func(t *testing.T) {
		_, err := processCustomFunction("test.module()")
		require.NotNil(t, err)
		assert.Equal(t, "unknown function type: test", err.Error())
	})

	//AWS
	t.Run("should error on unknown AWS module", func(t *testing.T) {
		_, err := processCustomFunction("aws.test()")
		require.NotNil(t, err)
		assert.Equal(t, "test is not a valid AWS module", err.Error())
	})
	t.Run("should error on missing service parameter for aws.regions function", func(t *testing.T) {
		_, err := processCustomFunction("aws.regions()")
		require.NotNil(t, err)
		assert.Equal(t, "service name parameter is required for AWS regions function", err.Error())
	})
	t.Run("should return list of AWS ECS regions", func(t *testing.T) {
		regions, err := processCustomFunction("aws.regions(ecs)")
		require.Nil(t, err)
		require.NotNil(t, regions)
		assert.NotEmpty(t, regions)
	})
	t.Run("should error on no attribute defined on AWS credentials", func(t *testing.T) {
		_, err := processCustomFunction("aws.credentials()")
		require.NotNil(t, err)
		assert.Equal(t, "requested credentials attribute is not set", err.Error())
	})
	t.Run("should return AWS credentials", func(t *testing.T) {
		vals, err := processCustomFunction("aws.credentials().AccessKeyID")
		require.Nil(t, err)
		require.NotNil(t, vals)
		require.Len(t, vals, 1)
		accessKey := vals[0]
		require.NotNil(t, accessKey)
	})
}

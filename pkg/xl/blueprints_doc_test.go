package xl

import (
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"sort"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/xebialabs/xl-cli/pkg/cloud/aws"
	"github.com/xebialabs/xl-cli/pkg/models"
	"github.com/xebialabs/yaml"
)

func getValidTestBlueprintMetadata(templatePath string, blueprintRepository BlueprintRepository) (*BlueprintYaml, error) {
	metadata := []byte(
		fmt.Sprintf(`
         apiVersion: %s
         kind: Blueprint
         metadata:
         spec:
           parameters:
           - name: pass
             type: Input
             description: password?
             secret: true
           - name: test
             type: Input
             default: lala
             saveInXlVals: true 
             description: help text
           - name: fn
             type: Input
             value: !fn aws.regions(ecs)[0]
           - name: select
             type: Select
             description: select region
             options:
             - !fn aws.regions(ecs)[0]
             - b
             - c
             default: b
           - name: isit
             description: is it?
             type: Confirm
             value: true
           - name: isitnot
             description: negative question?
             type: Confirm
           - name: dep
             description: depends on others
             type: Input
             dependsOnTrue: isit
             dependsOnFalse: isitnot
           files:
           - path: xebialabs/foo.yaml
           - path: readme.md
             dependsOnTrue: isit
           - path: bar.md
             dependsOnTrue: isitnot
           - path: foo.md
             dependsOnFalse: isitnot
`, models.YamlFormatVersion))
	return parseTemplateMetadata(&metadata, templatePath, blueprintRepository)
}

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

func TestParseDependsOnValue(t *testing.T) {
	t.Run("should error when unknown function in dependsOn", func(t *testing.T) {
		v := Variable{
			Name:          VarField{Val: "test"},
			Type:          VarField{Val: TypeInput},
			DependsOnTrue: VarField{Val: "aws.creds", Tag: "!fn"},
		}
		_, err := ParseDependsOnValue(v.DependsOnTrue, &[]Variable{})
		require.NotNil(t, err)
	})
	t.Run("should return parsed bool value for dependsOnFn field", func(t *testing.T) {
		v := Variable{
			Name:          VarField{Val: "test"},
			Type:          VarField{Val: TypeInput},
			DependsOnTrue: VarField{Val: "aws.credentials().IsAvailable", Tag: "!fn"},
		}
		_, err := ParseDependsOnValue(v.DependsOnTrue, &[]Variable{})
		require.Nil(t, err)
	})
	t.Run("should return bool value from referenced var for dependsOn field", func(t *testing.T) {
		vars := make([]Variable, 2)
		vars[0] = Variable{
			Name:  VarField{Val: "confirm"},
			Type:  VarField{Val: TypeConfirm},
			Value: VarField{Bool: true},
		}
		vars[1] = Variable{
			Name:          VarField{Val: "test"},
			Type:          VarField{Val: TypeInput},
			DependsOnTrue: VarField{Val: "confirm"},
		}
		val, err := ParseDependsOnValue(vars[1].DependsOnTrue, &vars)
		require.Nil(t, err)
		assert.Equal(t, vars[0].Value.Bool, val)
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
		assert.True(t, skipQuestionOnCondition(&variables[1], variables[1].DependsOnFalse.Val, variables[0].Value.Bool, NewPreparedData(), "", true))
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
		assert.True(t, skipQuestionOnCondition(&variables[1], variables[1].DependsOnTrue.Val, variables[0].Value.Bool, NewPreparedData(), "", false))
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
		assert.False(t, skipQuestionOnCondition(&variables[1], variables[1].DependsOnFalse.Val, variables[0].Value.Bool, NewPreparedData(), "", true))
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
		assert.False(t, skipQuestionOnCondition(&variables[1], variables[1].DependsOnTrue.Val, variables[0].Value.Bool, NewPreparedData(), "", false))
	})
}

func TestParseTemplateMetadata(t *testing.T) {
	templatePath := "test/blueprints"
	blueprintRepository := BlueprintRepository{Server: SimpleHTTPServer{Url: parseURIWithoutError("http://xebialabs.com/test/blueprints")}}
	tmpDir := path.Join("test", "blueprints")
	os.MkdirAll(tmpDir, os.ModePerm)
	defer os.RemoveAll("test")
	d1 := []byte("hello\ngo\n")
	ioutil.WriteFile(path.Join(tmpDir, "test.yaml.tmpl"), d1, os.ModePerm)

	t.Run("should error on invalid xl yaml", func(t *testing.T) {
		metadata := []byte("test: blueprint")
		_, err := parseTemplateMetadata(&metadata, templatePath, blueprintRepository)
		require.NotNil(t, err)
		assert.Equal(t, fmt.Sprintf("yaml: unmarshal errors:\n  line 1: field test not found in type xl.BlueprintYaml"), err.Error())
	})

	t.Run("should error on missing api version", func(t *testing.T) {
		metadata := []byte("kind: blueprint")
		_, err := parseTemplateMetadata(&metadata, templatePath, blueprintRepository)
		require.NotNil(t, err)
		assert.Equal(t, fmt.Sprintf("api version needs to be %s", models.YamlFormatVersion), err.Error())
	})

	t.Run("should error on missing doc kind", func(t *testing.T) {
		metadata := []byte("apiVersion: " + models.YamlFormatVersion)
		_, err := parseTemplateMetadata(&metadata, templatePath, blueprintRepository)
		require.NotNil(t, err)
		assert.Equal(t, "yaml document kind needs to be Blueprint", err.Error())
	})

	t.Run("should error on unknown field type", func(t *testing.T) {
		metadata := []byte(
			fmt.Sprintf(
				`
                apiVersion: %s
                kind: Blueprint
                metadata:
                spec:
                  parameters:
                  - name: Test
                    type: Invalid
                    value: testing`,
				models.YamlFormatVersion))
		_, err := parseTemplateMetadata(&metadata, templatePath, blueprintRepository)
		require.NotNil(t, err)
		assert.Equal(t, "type [Invalid] is not valid for parameter [Test]", err.Error())
	})

	t.Run("should error on missing variable field", func(t *testing.T) {
		metadata := []byte(
			fmt.Sprintf(`
              apiVersion: %s
              kind: Blueprint
              metadata:
              spec:
                parameters:
                - name: Test
                  value: testing`, models.YamlFormatVersion))
		_, err := parseTemplateMetadata(&metadata, templatePath, blueprintRepository)
		require.NotNil(t, err)
		assert.Equal(t, "parameter [Test] is missing required fields: [type]", err.Error())
	})

	t.Run("should error on missing options for variable", func(t *testing.T) {
		metadata := []byte(
			fmt.Sprintf(`
              apiVersion: %s
              kind: Blueprint
              metadata:
              spec:
                parameters:
                - name: Test
                  type: Select
                  options:`, models.YamlFormatVersion))
		_, err := parseTemplateMetadata(&metadata, templatePath, blueprintRepository)
		require.NotNil(t, err)
		assert.Equal(t, "at least one option field is need to be set for parameter [Test]", err.Error())
	})
	t.Run("should error on missing path for files", func(t *testing.T) {
		metadata := []byte(
			fmt.Sprintf(`
              apiVersion: %s
              kind: Blueprint
              metadata:
              spec:
                parameters:
                - name: Test
                  type: Confirm
                files:
                - dependsOnFalse: Test
                - path: xbc.yaml`, models.YamlFormatVersion))
		_, err := parseTemplateMetadata(&metadata, "aws/test", blueprintRepository)
		require.NotNil(t, err)
		assert.Equal(t, "path is missing for file specification in files", err.Error())
	})
	t.Run("should error on invalid path for files", func(t *testing.T) {
		metadata := []byte(
			fmt.Sprintf(`
              apiVersion: %s
              kind: Blueprint
              metadata:
              spec:
                parameters:
                - name: Test
                  type: Confirm
                files:
                - path: ../xbc.yaml`, models.YamlFormatVersion))
		_, err := parseTemplateMetadata(&metadata, "aws/test", blueprintRepository)
		require.NotNil(t, err)
		assert.Equal(t, "path for file specification cannot start with /, .. or ./", err.Error())
	})
	t.Run("should parse nested variables and files from valid legacy metadata", func(t *testing.T) {
		metadata := []byte(
			fmt.Sprintf(`
              apiVersion: %s
              kind: Blueprint
              metadata:
              parameters:
              - name: pass
                type: Input
                description: password?
                secret: true
              - name: test
                type: Input
                default: lala
                saveInXlVals: true 
                description: help text
              
              files:
              - path: xebialabs/foo.yaml
              - path: readme.md
                dependsOnTrue: isit`, models.YamlFormatVersion))
		doc, err := parseTemplateMetadata(&metadata, "aws/test", blueprintRepository)
		require.Nil(t, err)
		assert.Equal(t, Variable{
			Name:        VarField{Val: "pass"},
			Type:        VarField{Val: TypeInput},
			Description: VarField{Val: "password?"},
			Secret:      VarField{Bool: true},
		}, doc.Variables[0])
		assert.Equal(t, Variable{
			Name:         VarField{Val: "test"},
			Type:         VarField{Val: TypeInput},
			Default:      VarField{Val: "lala"},
			Description:  VarField{Val: "help text"},
			SaveInXlVals: VarField{Bool: true},
		}, doc.Variables[1])
		assert.Equal(t, TemplateConfig{
			File:       "xebialabs/foo.yaml",
			FullPath:   "http://xebialabs.com/test/blueprints/aws/test/xebialabs/foo.yaml",
			Repository: blueprintRepository,
		}, doc.TemplateConfigs[0])
		assert.Equal(t, TemplateConfig{
			File:          "readme.md",
			FullPath:      "http://xebialabs.com/test/blueprints/aws/test/readme.md",
			DependsOnTrue: VarField{Val: "isit"},
			Repository:    blueprintRepository,
		}, doc.TemplateConfigs[1])
	})

	t.Run("should parse nested variables from valid metadata", func(t *testing.T) {
		doc, err := getValidTestBlueprintMetadata(templatePath, blueprintRepository)
		require.Nil(t, err)
		assert.Len(t, doc.Variables, 7)
		assert.Equal(t, Variable{
			Name:        VarField{Val: "pass"},
			Type:        VarField{Val: TypeInput},
			Description: VarField{Val: "password?"},
			Secret:      VarField{Bool: true},
		}, doc.Variables[0])
		assert.Equal(t, Variable{
			Name:         VarField{Val: "test"},
			Type:         VarField{Val: TypeInput},
			Default:      VarField{Val: "lala"},
			Description:  VarField{Val: "help text"},
			SaveInXlVals: VarField{Bool: true},
		}, doc.Variables[1])
		assert.Equal(t, Variable{
			Name:  VarField{Val: "fn"},
			Type:  VarField{Val: TypeInput},
			Value: VarField{Val: "aws.regions(ecs)[0]", Tag: tagFn},
		}, doc.Variables[2])
		assert.Equal(t, Variable{
			Name:        VarField{Val: "select"},
			Type:        VarField{Val: TypeSelect},
			Description: VarField{Val: "select region"},
			Options: []VarField{
				{Val: "aws.regions(ecs)[0]", Tag: tagFn},
				{Val: "b"},
				{Val: "c"},
			},
			Default: VarField{Val: "b"},
		}, doc.Variables[3])
		assert.Equal(t, Variable{
			Name:        VarField{Val: "isit"},
			Type:        VarField{Val: TypeConfirm},
			Description: VarField{Val: "is it?"},
			Value:       VarField{Bool: true},
		}, doc.Variables[4])
		assert.Equal(t, Variable{
			Name:        VarField{Val: "isitnot"},
			Type:        VarField{Val: TypeConfirm},
			Description: VarField{Val: "negative question?"},
		}, doc.Variables[5])
		assert.Equal(t, Variable{
			Name:           VarField{Val: "dep"},
			Type:           VarField{Val: TypeInput},
			Description:    VarField{Val: "depends on others"},
			DependsOnTrue:  VarField{Val: "isit"},
			DependsOnFalse: VarField{Val: "isitnot"},
		}, doc.Variables[6])
	})
	t.Run("should parse files from valid metadata", func(t *testing.T) {
		doc, err := getValidTestBlueprintMetadata("templatePath/test", blueprintRepository)
		require.Nil(t, err)
		assert.Equal(t, 4, len(doc.TemplateConfigs))
		assert.Equal(t, TemplateConfig{
			File:       "xebialabs/foo.yaml",
			FullPath:   "http://xebialabs.com/test/blueprints/templatePath/test/xebialabs/foo.yaml",
			Repository: blueprintRepository,
		}, doc.TemplateConfigs[0])
		assert.Equal(t, TemplateConfig{
			File:          "readme.md",
			FullPath:      "http://xebialabs.com/test/blueprints/templatePath/test/readme.md",
			DependsOnTrue: VarField{Val: "isit"},
			Repository:    blueprintRepository,
		}, doc.TemplateConfigs[1])
		assert.Equal(t, TemplateConfig{
			File:          "bar.md",
			FullPath:      "http://xebialabs.com/test/blueprints/templatePath/test/bar.md",
			DependsOnTrue: VarField{Val: "isitnot"},
			Repository:    blueprintRepository,
		}, doc.TemplateConfigs[2])
		assert.Equal(t, TemplateConfig{
			File:           "foo.md",
			FullPath:       "http://xebialabs.com/test/blueprints/templatePath/test/foo.md",
			DependsOnFalse: VarField{Val: "isitnot"},
			Repository:     blueprintRepository,
		}, doc.TemplateConfigs[3])
	})
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

func TestValidatePrompt(t *testing.T) {
	type args struct {
		pattern string
		value   string
	}
	tests := []struct {
		name string
		args args
		want error
	}{
		{"should fail required validation on empty value", args{"", ""}, fmt.Errorf("Value is required")},
		{"should fail required validation on empty value with pattern", args{".", ""}, fmt.Errorf("Value is required")},
		{"should pass required validation on valid value", args{"", "test"}, nil},
		{"should fail pattern validation on invalid value", args{"[a-z]*", "123"}, fmt.Errorf("Value should match pattern [a-z]*")},
		{"should pass pattern validation on valid value", args{"[a-z]*", "abc"}, nil},
		{"should pass pattern validation on valid value with extra start/end tag on pattern", args{"^[a-z]*$", "abc"}, nil},
		{"should pass pattern validation on valid value with fixed pattern", args{"test", "test"}, nil},
		{"should fail pattern validation on invalid value with fixed pattern", args{"test", "abcd"}, fmt.Errorf("Value should match pattern test")},
		{
			"should fail pattern validation on valid value with complex pattern",
			args{`\b(?:(?:2(?:[0-4][0-9]|5[0-5])|[0-1]?[0-9]?[0-9])\.){3}(?:(?:2([0-4][0-9]|5[0-5])|[0-1]?[0-9]?[0-9]))\b`, "123.123.123.256"},
			fmt.Errorf(`Value should match pattern \b(?:(?:2(?:[0-4][0-9]|5[0-5])|[0-1]?[0-9]?[0-9])\.){3}(?:(?:2([0-4][0-9]|5[0-5])|[0-1]?[0-9]?[0-9]))\b`),
		},
		{
			"should pass pattern validation on valid value with complex pattern",
			args{`\b(?:(?:2(?:[0-4][0-9]|5[0-5])|[0-1]?[0-9]?[0-9])\.){3}(?:(?:2([0-4][0-9]|5[0-5])|[0-1]?[0-9]?[0-9]))\b`, "255.255.255.255"},
			nil,
		},
		{"should fail pattern validation on invalid pattern", args{"[[", "abcd"}, fmt.Errorf("error parsing regexp: missing closing ]: `[[$`")},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := validatePrompt(tt.args.pattern)(tt.args.value)
			if tt.want == nil || got == nil {
				assert.Equal(t, tt.want, got)
			} else {
				assert.Equal(t, tt.want.Error(), got.Error())
			}
		})
	}
}

func TestBlueprintYaml_parseFiles(t *testing.T) {
	templatePath := "aws/monolith"
	blueprintRepository := BlueprintRepository{Server: SimpleHTTPServer{Url: parseURIWithoutError("http://xebialabs.com/test/blueprints")}}
	type args struct {
		templatePath        string
		blueprintRepository BlueprintRepository
	}

	tests := []struct {
		name    string
		fields  BlueprintYaml
		args    args
		want    []TemplateConfig
		wantErr error
	}{
		{
			"parse a valid file declaration",
			BlueprintYaml{
				Spec: Spec{
					Files: []interface{}{
						map[interface{}]interface{}{"path": "test.yaml"},
						map[interface{}]interface{}{"path": "test2.yaml"},
					},
				},
			},
			args{templatePath, blueprintRepository},
			[]TemplateConfig{
				{File: "test.yaml", FullPath: "http://xebialabs.com/test/blueprints/aws/monolith/test.yaml", Repository: blueprintRepository},
				{File: "test2.yaml", FullPath: "http://xebialabs.com/test/blueprints/aws/monolith/test2.yaml", Repository: blueprintRepository},
			},
			nil,
		},
		{
			"parse a valid file declaration with dependsOn that refers to existing variables",
			BlueprintYaml{
				Spec: Spec{
					Parameters: []interface{}{
						map[interface{}]interface{}{"name": "foo", "type": "Confirm", "value": true},
						map[interface{}]interface{}{"name": "bar", "type": "Confirm", "value": false},
					},
					Files: []interface{}{
						map[interface{}]interface{}{"path": "test.yaml"},
						map[interface{}]interface{}{"path": "test2.yaml", "dependsOnTrue": "foo"},
						map[interface{}]interface{}{"path": "test3.yaml", "dependsOnFalse": "bar"},
						map[interface{}]interface{}{"path": "test4.yaml", "dependsOnTrue": "bar"},
						map[interface{}]interface{}{"path": "test5.yaml", "dependsOnFalse": "foo"},
					},
				},
				Variables: []Variable{
					{Name: VarField{Val: "foo"}, Type: VarField{Val: "Confirm"}, Value: VarField{Bool: true}},
					{Name: VarField{Val: "bar"}, Type: VarField{Val: "Confirm"}, Value: VarField{Bool: false}},
				},
			},
			args{templatePath, blueprintRepository},
			[]TemplateConfig{
				{File: "test.yaml", FullPath: "http://xebialabs.com/test/blueprints/aws/monolith/test.yaml", Repository: blueprintRepository},
				{File: "test2.yaml", FullPath: "http://xebialabs.com/test/blueprints/aws/monolith/test2.yaml", Repository: blueprintRepository, DependsOnTrue: VarField{Val: "foo", Bool: false, Tag: ""}},
				{File: "test3.yaml", FullPath: "http://xebialabs.com/test/blueprints/aws/monolith/test3.yaml", Repository: blueprintRepository, DependsOnFalse: VarField{Val: "bar", Bool: false, Tag: ""}},
				{File: "test4.yaml", FullPath: "http://xebialabs.com/test/blueprints/aws/monolith/test4.yaml", Repository: blueprintRepository, DependsOnTrue: VarField{Val: "bar", Bool: false, Tag: ""}},
				{File: "test5.yaml", FullPath: "http://xebialabs.com/test/blueprints/aws/monolith/test5.yaml", Repository: blueprintRepository, DependsOnFalse: VarField{Val: "foo", Bool: false, Tag: ""}},
			},
			nil,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			blueprintDoc := &BlueprintYaml{
				ApiVersion:      tt.fields.ApiVersion,
				Kind:            tt.fields.Kind,
				Metadata:        tt.fields.Metadata,
				Spec:            tt.fields.Spec,
				TemplateConfigs: tt.fields.TemplateConfigs,
				Variables:       tt.fields.Variables,
			}
			err := blueprintDoc.parseFiles(tt.args.templatePath, tt.args.blueprintRepository)
			if tt.wantErr == nil || err == nil {
				assert.Equal(t, tt.wantErr, err)
			} else {
				assert.Equal(t, tt.wantErr.Error(), err.Error())
			}
			assert.Equal(t, tt.want, blueprintDoc.TemplateConfigs)
		})
	}
}

func TestParseFileMap(t *testing.T) {
	type args struct {
		m *map[interface{}]interface{}
	}
	tests := []struct {
		name    string
		args    *map[interface{}]interface{}
		want    TemplateConfig
		wantErr error
	}{
		{
			"return empty for empty map",
			&map[interface{}]interface{}{},
			TemplateConfig{},
			nil,
		},
		{
			"return error on passing unknown key type",
			&map[interface{}]interface{}{
				true: "test.yaml",
			},
			TemplateConfig{},
			fmt.Errorf(`unknown variable key type in files [%s]`, "%!s(bool=true)"),
		},
		{
			"return error on passing unknown value type",
			&map[interface{}]interface{}{
				"path": true,
			},
			TemplateConfig{},
			fmt.Errorf(`unknown variable value type in files [%s]`, "%!s(bool=true)"),
		},
		{
			"parse a file declaration with only path",
			&map[interface{}]interface{}{
				"path": "test.yaml",
			},
			TemplateConfig{File: "test.yaml"},
			nil,
		},
		{
			"parse a file declaration with path and dependsOnTrue",
			&map[interface{}]interface{}{
				"path": "test.yaml", "dependsOnTrue": "foo",
			},
			TemplateConfig{File: "test.yaml", DependsOnTrue: VarField{Val: "foo"}},
			nil,
		},
		{
			"parse a file declaration with path dependsOnFalse and dependsOnTrue",
			&map[interface{}]interface{}{
				"path": "test.yaml", "dependsOnTrue": "foo", "dependsOnFalse": "bar",
			},
			TemplateConfig{File: "test.yaml", DependsOnTrue: VarField{Val: "foo"}, DependsOnFalse: VarField{Val: "bar"}},
			nil,
		},
		{
			"parse a file declaration with path and dependsOnTrue as !fn tag",
			&map[interface{}]interface{}{
				"path": "test.yaml", "dependsOnTrue": yaml.CustomTag{Tag: "!fn", Value: "aws.credentials().IsAvailable"},
			},
			TemplateConfig{File: "test.yaml", DependsOnTrue: VarField{Val: "aws.credentials().IsAvailable", Tag: "!fn"}},
			nil,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseFileMap(tt.args)
			if tt.wantErr == nil || err == nil {
				assert.Equal(t, tt.wantErr, err)
			} else {
				assert.Equal(t, tt.wantErr.Error(), err.Error())
			}
			assert.Equal(t, tt.want, got)
		})
	}
}

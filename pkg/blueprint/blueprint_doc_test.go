package blueprint

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"sort"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/xebialabs/xl-cli/pkg/cloud/aws"
	"github.com/xebialabs/xl-cli/pkg/models"
	"github.com/xebialabs/yaml"
)

var sampleKubeConfig = `apiVersion: v1
clusters:
- cluster:
    certificate-authority-data: REDACTED
    server: https://1A256A873510C6531DBC9D05142A309B.sk1.eu-west-1.eks.amazonaws.com
  name: elton-xl-platform-master
- cluster:
    certificate-authority-data: 123==
    server: https://test.hcp.eastus.azmk8s.io:443
  name: testCluster
- cluster:
    insecure-skip-tls-verify: true
    server: https://ocpm.test.com:8443
  name: ocpm-test-com:8443
- cluster:
    insecure-skip-tls-verify: true
    server: https://ocpm.test.com:8443
  name: testUserNotFound
contexts:
- context:
    cluster: elton-xl-platform-master
    namespace: xebialabs
    user: aws
  name: aws
- context:
    cluster: ocpm-test-com:8443
    namespace: default
    user: test/ocpm-test-com:8443
  name: default/ocpm-test-com:8443/test
- context:
    cluster: testCluster
    namespace: test
    user: clusterUser_testCluster_testCluster
  name: testCluster
- context:
    cluster: testClusterNotFound
    namespace: test
    user: testClusterNotFound
  name: testClusterNotFound
- context:
    cluster: testUserNotFound
    namespace: test
    user: testUserNotFound
  name: testUserNotFound
current-context: testCluster
kind: Config
preferences: {}
users:
- name: aws
  user:
    exec:
      apiVersion: client.authentication.k8s.io/v1alpha1
      args:
      - token
      - -i
      - elton-xl-platform-master
      command: aws-iam-authenticator
      env: null
- name: clusterUser_testCluster_testCluster
  user:
    client-certificate-data: 123==
    client-key-data: 123==
    token: 6555565666666666666
- name: test/ocpm-test-com:8443
  user:
    client-certificate-data: 123==
- name: testClusterNotFound
  user:
    client-certificate-data: 123==`

func Setupk8sConfig() {
	tmpDir := filepath.Join("test", "blueprints")
	os.MkdirAll(tmpDir, os.ModePerm)
	d1 := []byte(sampleKubeConfig)
	ioutil.WriteFile(filepath.Join(tmpDir, "config"), d1, os.ModePerm)
	os.Setenv("KUBECONFIG", filepath.Join(tmpDir, "config"))
}

func getValidTestBlueprintMetadata(templatePath string, blueprintRepository BlueprintContext) (*BlueprintConfig, error) {
	metadata := []byte(
		fmt.Sprintf(`
         apiVersion: %s
         kind: Blueprint
         metadata:
           projectName: Test Project
           description: Is just a test blueprint project used for manual testing of inputs
           author: XebiaLabs
           version: 1.0
           instructions: These are the instructions for executing this blueprint
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
             dependsOn: !expression "isit && true"
             dependsOnFalse: isitnot
           files:
           - path: xebialabs/foo.yaml
           - path: readme.md
             dependsOnTrue: isit
           - path: bar.md
             dependsOn: isitnot
           - path: foo.md
             dependsOnFalse: !expression "!!isitnot"
           include:
           - blueprint: kubernetes/gke-cluster
             stage: before
             parameterValues:
             - name: Foo
               value: hello
               dependsOn: !expression "ExpTest1 == 'us-west' && AppName != 'foo' && TestDepends" # do this later
             - name: bar
               value: true
             skipFiles:
             - path: xld-infrastructure.yml.tmpl
               dependsOnTrue: TestDepends # do this later
           - blueprint: kubernetes/namespace
             dependsOnTrue: !expression "ExpTest1 == 'us-west' && AppName != 'foo' && TestDepends"
             stage: after
             parameterValues:
             - name: Foo
               value: hello
`, models.YamlFormatVersion))
	return parseTemplateMetadata(&metadata, templatePath, &blueprintRepository, true)
}

var dummyData = make(map[string]interface{})

func TestGetVariableDefaultVal(t *testing.T) {
	t.Run("should return empty string when default is not defined", func(t *testing.T) {
		v := Variable{
			Name: VarField{Val: "test"},
			Type: VarField{Val: TypeInput},
		}
		defaultVal := v.GetDefaultVal(dummyData)
		assert.Equal(t, "", defaultVal)
	})

	t.Run("should return default value string when default is defined", func(t *testing.T) {
		v := Variable{
			Name:    VarField{Val: "test"},
			Type:    VarField{Val: TypeInput},
			Default: VarField{Val: "default_val"},
		}
		defaultVal := v.GetDefaultVal(dummyData)
		assert.Equal(t, "default_val", defaultVal)
	})

	t.Run("should return empty string when invalid function tag in default field", func(t *testing.T) {
		v := Variable{
			Name:    VarField{Val: "test"},
			Type:    VarField{Val: TypeInput},
			Default: VarField{Val: "aws.regs", Tag: tagFn},
		}
		defaultVal := v.GetDefaultVal(dummyData)
		assert.Equal(t, "", defaultVal)
	})

	t.Run("should return function output on valid function tag in default field", func(t *testing.T) {
		v := Variable{
			Name:    VarField{Val: "test"},
			Type:    VarField{Val: TypeInput},
			Default: VarField{Val: "aws.regions(ecs)[0]", Tag: tagFn},
		}
		defaultVal := v.GetDefaultVal(dummyData)
		regionsList, _ := aws.GetAvailableAWSRegionsForService("ecs")
		sort.Strings(regionsList)
		assert.Equal(t, regionsList[0], defaultVal)
	})

	t.Run("should return empty string when invalid expression tag in default field", func(t *testing.T) {
		v := Variable{
			Name:    VarField{Val: "test"},
			Type:    VarField{Val: TypeInput},
			Default: VarField{Val: "aws.regs", Tag: tagExpression},
		}
		defaultVal := v.GetDefaultVal(dummyData)
		assert.Equal(t, "", defaultVal)
	})

	t.Run("should return output on valid expression tag in default field", func(t *testing.T) {
		v := Variable{
			Name:    VarField{Val: "test"},
			Type:    VarField{Val: TypeInput},
			Default: VarField{Val: "'foo' + 'bar'", Tag: tagExpression},
		}
		defaultVal := v.GetDefaultVal(dummyData)
		assert.Equal(t, "foobar", defaultVal)
		v = Variable{
			Name:    VarField{Val: "test"},
			Type:    VarField{Val: TypeInput},
			Default: VarField{Val: "Foo > 10", Tag: tagExpression},
		}
		defaultVal = v.GetDefaultVal(map[string]interface{}{
			"Foo": 100,
		})
		assert.True(t, defaultVal.(bool))
	})
}

func TestParseDependsOnValue(t *testing.T) {
	t.Run("should error when unknown function in DependsOn", func(t *testing.T) {
		v := Variable{
			Name:      VarField{Val: "test"},
			Type:      VarField{Val: TypeInput},
			DependsOn: VarField{Val: "aws.creds", Tag: "!fn"},
		}
		_, err := ParseDependsOnValue(v.DependsOn, &[]Variable{}, dummyData)
		require.NotNil(t, err)
	})
	t.Run("should return parsed bool value for DependsOn field from function", func(t *testing.T) {
		v := Variable{
			Name:      VarField{Val: "test"},
			Type:      VarField{Val: TypeInput},
			DependsOn: VarField{Val: "aws.credentials().IsAvailable", Tag: "!fn"},
		}
		out, err := ParseDependsOnValue(v.DependsOn, &[]Variable{}, dummyData)
		require.Nil(t, err)
		assert.Equal(t, true, out)
	})
	t.Run("should error when invalid expression in DependsOn", func(t *testing.T) {
		v := Variable{
			Name:      VarField{Val: "test"},
			Type:      VarField{Val: TypeInput},
			DependsOn: VarField{Val: "aws.creds", Tag: tagExpression},
		}
		_, err := ParseDependsOnValue(v.DependsOn, &[]Variable{}, dummyData)
		require.NotNil(t, err)
	})
	t.Run("should return parsed bool value for DependsOn field from expression", func(t *testing.T) {
		v := Variable{
			Name:      VarField{Val: "test"},
			Type:      VarField{Val: TypeInput},
			DependsOn: VarField{Val: "Foo > 10", Tag: tagExpression},
		}

		val, err := ParseDependsOnValue(v.DependsOn, &[]Variable{}, map[string]interface{}{
			"Foo": 100,
		})
		require.Nil(t, err)
		require.True(t, val)
	})
	t.Run("should return bool value from referenced var for dependsOn field", func(t *testing.T) {
		vars := make([]Variable, 2)
		vars[0] = Variable{
			Name:  VarField{Val: "confirm"},
			Type:  VarField{Val: TypeConfirm},
			Value: VarField{Bool: true, Val: "true"},
		}
		vars[1] = Variable{
			Name:      VarField{Val: "test"},
			Type:      VarField{Val: TypeInput},
			DependsOn: VarField{Val: "confirm"},
		}
		val, err := ParseDependsOnValue(vars[1].DependsOn, &vars, dummyData)
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
		val := v.GetValueFieldVal(dummyData)
		assert.Equal(t, "testing", val)
	})

	t.Run("should return empty on invalid function tag in value field", func(t *testing.T) {
		v := Variable{
			Name:  VarField{Val: "test"},
			Type:  VarField{Val: TypeInput},
			Value: VarField{Val: "aws.regs", Tag: tagFn},
		}
		val := v.GetValueFieldVal(dummyData)
		assert.Equal(t, "", val)
	})

	t.Run("should return function output on valid function tag in value field", func(t *testing.T) {
		v := Variable{
			Name:  VarField{Val: "test"},
			Type:  VarField{Val: TypeInput},
			Value: VarField{Val: "aws.regions(ecs)[0]", Tag: tagFn},
		}
		val := v.GetValueFieldVal(dummyData)
		regionsList, _ := aws.GetAvailableAWSRegionsForService("ecs")
		sort.Strings(regionsList)
		assert.Equal(t, regionsList[0], val)
	})

	t.Run("should return empty on invalid expression tag in value field", func(t *testing.T) {
		v := Variable{
			Name:  VarField{Val: "test"},
			Type:  VarField{Val: TypeInput},
			Value: VarField{Val: "aws.regs()", Tag: tagExpression},
		}
		val := v.GetValueFieldVal(dummyData)
		assert.Equal(t, "", val)
	})

	t.Run("should return expression output on valid expression tag in value field", func(t *testing.T) {
		v := Variable{
			Name:  VarField{Val: "test"},
			Type:  VarField{Val: TypeInput},
			Value: VarField{Val: "'foo' + 'bar'", Tag: tagExpression},
		}
		defaultVal := v.GetValueFieldVal(dummyData)
		assert.Equal(t, "foobar", defaultVal)
		v = Variable{
			Name:  VarField{Val: "test"},
			Type:  VarField{Val: TypeInput},
			Value: VarField{Val: "Foo > 10", Tag: tagExpression},
		}
		defaultVal = v.GetValueFieldVal(map[string]interface{}{
			"Foo": 100,
		})
		assert.Equal(t, "true", defaultVal)
	})
}

func TestGetOptions(t *testing.T) {
	t.Run("should return string values of options", func(t *testing.T) {
		v := Variable{
			Name:    VarField{Val: "test"},
			Type:    VarField{Val: TypeSelect},
			Options: []VarField{{Val: "a"}, {Val: "b"}, {Val: "c"}},
		}
		values := v.GetOptions(dummyData)
		assert.Len(t, values, 3)
		assert.Equal(t, []string{"a", "b", "c"}, values)
	})

	t.Run("should return generated values for fn options tag", func(t *testing.T) {
		v := Variable{
			Name:    VarField{Val: "test"},
			Type:    VarField{Val: TypeSelect},
			Options: []VarField{{Val: "aws.regions(ecs)", Tag: "!fn"}},
		}
		values := v.GetOptions(dummyData)
		assert.True(t, len(values) > 1)
	})

	t.Run("should return nil on invalid function tag for options", func(t *testing.T) {
		v := Variable{
			Name:    VarField{Val: "test"},
			Type:    VarField{Val: TypeSelect},
			Options: []VarField{{Val: "aws.regs", Tag: "!fn"}},
		}
		out := v.GetOptions(dummyData)
		require.Nil(t, out)
	})

	t.Run("should return generated values for expression options tag", func(t *testing.T) {
		v := Variable{
			Name:    VarField{Val: "test"},
			Type:    VarField{Val: TypeSelect},
			Options: []VarField{{Val: "Foo ? Bar : (1, 2, 3)", Tag: tagExpression}},
		}
		values := v.GetOptions(map[string]interface{}{
			"Foo": true,
			"Bar": []string{"test", "foo"},
		})
		assert.True(t, len(values) == 2)
	})

	t.Run("should return generated string array values for expression options tag", func(t *testing.T) {
		v := Variable{
			Name:    VarField{Val: "test"},
			Type:    VarField{Val: TypeSelect},
			Options: []VarField{{Val: "Provider == 'GCP' ? ('GKE', 'CloudSore') : ('test')", Tag: tagExpression}},
		}
		values := v.GetOptions(map[string]interface{}{
			"Provider": "GCP",
		})
		assert.NotNil(t, values)
		assert.True(t, len(values) == 2)
	})

	t.Run("should return string values for param reference in expression options tag", func(t *testing.T) {
		v := Variable{
			Name:    VarField{Val: "test"},
			Type:    VarField{Val: TypeSelect},
			Options: []VarField{{Val: "Foo ? Bar : (Foo1, Foo2)", Tag: tagExpression}},
		}
		values := v.GetOptions(map[string]interface{}{
			"Foo":  false,
			"Foo1": "test",
			"Foo2": "foo",
			"Bar":  []string{"test", "foo"},
		})
		assert.NotNil(t, values)
		assert.True(t, len(values) == 2)
	})

	t.Run("should return string values for numeric type in expression options tag", func(t *testing.T) {
		v := Variable{
			Name:    VarField{Val: "test"},
			Type:    VarField{Val: TypeSelect},
			Options: []VarField{{Val: "Foo ? Bar : (1, 2, 3)", Tag: tagExpression}},
		}
		values := v.GetOptions(map[string]interface{}{
			"Foo": false,
			"Bar": []string{"test", "foo"},
		})
		assert.NotNil(t, values)
		assert.True(t, len(values) == 3)
	})

	t.Run("should return string values for boolean type in expression options tag", func(t *testing.T) {
		v := Variable{
			Name:    VarField{Val: "test"},
			Type:    VarField{Val: TypeSelect},
			Options: []VarField{{Val: "Foo ? Bar : (true, false)", Tag: tagExpression}},
		}
		values := v.GetOptions(map[string]interface{}{
			"Foo": false,
			"Bar": []string{"test", "foo"},
		})
		assert.NotNil(t, values)
		assert.True(t, len(values) == 2)
	})

	t.Run("should return nil values for invalid return type in expression options tag", func(t *testing.T) {
		v := Variable{
			Name:    VarField{Val: "test"},
			Type:    VarField{Val: TypeSelect},
			Options: []VarField{{Val: "Foo ? Bar : (Fooo, Foo)", Tag: tagExpression}},
		}
		values := v.GetOptions(map[string]interface{}{
			"Foo": false,
			"Bar": []string{"test", "foo"},
		})
		assert.Nil(t, values)
	})

	t.Run("should return nil on invalid expression tag for options", func(t *testing.T) {
		v := Variable{
			Name:    VarField{Val: "test"},
			Type:    VarField{Val: TypeSelect},
			Options: []VarField{{Val: "aws.regs()", Tag: tagExpression}},
		}
		out := v.GetOptions(dummyData)
		require.Nil(t, out)
	})
}

func TestSkipQuestionOnCondition(t *testing.T) {
	t.Run("should skip question (dependsOnFalse)", func(t *testing.T) {
		variables := make([]Variable, 2)
		variables[0] = Variable{
			Name:  VarField{Val: "confirm"},
			Type:  VarField{Val: TypeConfirm},
			Value: VarField{Bool: true, Val: "true"},
		}
		variables[1] = Variable{
			Name:           VarField{Val: "test"},
			Type:           VarField{Val: TypeInput},
			DependsOnFalse: VarField{Val: "confirm"},
		}
		assert.True(t, skipQuestionOnCondition(&variables[1], variables[1].DependsOnFalse.Val, variables[0].Value.Bool, NewPreparedData(), "", true))
	})
	t.Run("should skip question (dependsOn)", func(t *testing.T) {
		variables := make([]Variable, 2)
		variables[0] = Variable{
			Name:  VarField{Val: "confirm"},
			Type:  VarField{Val: TypeConfirm},
			Value: VarField{Bool: false, Val: "false"},
		}
		variables[1] = Variable{
			Name:      VarField{Val: "test"},
			Type:      VarField{Val: TypeInput},
			DependsOn: VarField{Val: "confirm"},
		}
		assert.True(t, skipQuestionOnCondition(&variables[1], variables[1].DependsOn.Val, variables[0].Value.Bool, NewPreparedData(), "", false))
	})
	t.Run("should skip question and default value should be false (dependsOn)", func(t *testing.T) {
		data := NewPreparedData()
		variables := make([]Variable, 2)
		variables[0] = Variable{
			Name:  VarField{Val: "confirm"},
			Type:  VarField{Val: TypeConfirm},
			Value: VarField{Bool: false, Val: "false"},
		}
		variables[1] = Variable{
			Name:      VarField{Val: "test"},
			Type:      VarField{Val: TypeConfirm},
			DependsOn: VarField{Val: "confirm"},
		}
		assert.True(t, skipQuestionOnCondition(&variables[1], variables[1].DependsOn.Val, variables[0].Value.Bool, data, "", false))
		assert.False(t, data.TemplateData[variables[1].Name.Val].(bool))
	})

	t.Run("should not skip question (dependsOnFalse)", func(t *testing.T) {
		variables := make([]Variable, 2)
		variables[0] = Variable{
			Name:  VarField{Val: "confirm"},
			Type:  VarField{Val: TypeConfirm},
			Value: VarField{Bool: false, Val: "false"},
		}
		variables[1] = Variable{
			Name:           VarField{Val: "test"},
			Type:           VarField{Val: TypeInput},
			DependsOnFalse: VarField{Val: "confirm"},
		}
		assert.False(t, skipQuestionOnCondition(&variables[1], variables[1].DependsOnFalse.Val, variables[0].Value.Bool, NewPreparedData(), "", true))
	})
	t.Run("should not skip question (dependsOn)", func(t *testing.T) {
		variables := make([]Variable, 2)
		variables[0] = Variable{
			Name:  VarField{Val: "confirm"},
			Type:  VarField{Val: TypeConfirm},
			Value: VarField{Bool: true, Val: "true"},
		}
		variables[1] = Variable{
			Name:      VarField{Val: "test"},
			Type:      VarField{Val: TypeInput},
			DependsOn: VarField{Val: "confirm"},
		}
		assert.False(t, skipQuestionOnCondition(&variables[1], variables[1].DependsOn.Val, variables[0].Value.Bool, NewPreparedData(), "", false))
	})
}

func TestParseTemplateMetadata(t *testing.T) {
	templatePath := "test/blueprints"
	blueprintRepository := BlueprintContext{}
	tmpDir := filepath.Join("test", "blueprints")
	os.MkdirAll(tmpDir, os.ModePerm)
	defer os.RemoveAll("test")
	d1 := []byte("hello\ngo\n")
	ioutil.WriteFile(filepath.Join(tmpDir, "test.yaml.tmpl"), d1, os.ModePerm)

	t.Run("should error on invalid xl yaml", func(t *testing.T) {
		metadata := []byte("test: blueprint")
		_, err := parseTemplateMetadata(&metadata, templatePath, &blueprintRepository, true)
		require.NotNil(t, err)
		assert.Equal(t, fmt.Sprintf("yaml: unmarshal errors:\n  line 1: field test not found in type blueprint.BlueprintYaml"), err.Error())
	})

	t.Run("should error on missing api version", func(t *testing.T) {
		metadata := []byte("kind: blueprint")
		_, err := parseTemplateMetadata(&metadata, templatePath, &blueprintRepository, true)
		require.NotNil(t, err)
		assert.Equal(t, fmt.Sprintf("api version needs to be %s", models.YamlFormatVersion), err.Error())
	})

	t.Run("should error on missing doc kind", func(t *testing.T) {
		metadata := []byte("apiVersion: " + models.YamlFormatVersion)
		_, err := parseTemplateMetadata(&metadata, templatePath, &blueprintRepository, true)
		require.NotNil(t, err)
		assert.Equal(t, "yaml document kind needs to be Blueprint", err.Error())
	})

	t.Run("should error on unknown field type", func(t *testing.T) {
		metadata := []byte(
			fmt.Sprintf(`
                apiVersion: %s
                kind: Blueprint
                metadata:
                spec:
                  parameters:
                  - name: Test
                    type: Invalid
                    value: testing`,
				models.YamlFormatVersion))
		_, err := parseTemplateMetadata(&metadata, templatePath, &blueprintRepository, true)
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
		_, err := parseTemplateMetadata(&metadata, templatePath, &blueprintRepository, true)
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
		_, err := parseTemplateMetadata(&metadata, templatePath, &blueprintRepository, true)
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
		_, err := parseTemplateMetadata(&metadata, "aws/test", &blueprintRepository, true)
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
		_, err := parseTemplateMetadata(&metadata, "aws/test", &blueprintRepository, true)
		require.NotNil(t, err)
		assert.Equal(t, "path for file specification cannot start with /, .. or ./", err.Error())
	})
	t.Run("should error on duplicate variable names", func(t *testing.T) {
		metadata := []byte(
			fmt.Sprintf(`
               apiVersion: %s
               kind: Blueprint
               metadata:
               spec:
                 parameters:
                 - name: Test
                   type: Input
                   default: 1
                 - name: Test
                   type: Input
                   default: 2
                 files:`, models.YamlFormatVersion))
		_, err := parseTemplateMetadata(&metadata, "aws/test", &blueprintRepository, true)
		require.NotNil(t, err)
		assert.Equal(t, "variable names must be unique within blueprint 'parameters' definition", err.Error())
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
                 useRawValue: true
               - name: test
                 type: Input
                 default: lala
                 saveInXlVals: true
                 description: help text
                 showValueOnSummary: true

               files:
               - path: xebialabs/foo.yaml
               - path: readme.md
                 dependsOn: isit`, models.YamlFormatVersion))
		doc, err := parseTemplateMetadata(&metadata, "aws/test", &blueprintRepository, true)
		require.Nil(t, err)
		assert.Equal(t, Variable{
			Name:        VarField{Val: "pass"},
			Type:        VarField{Val: TypeInput},
			Description: VarField{Val: "password?"},
			Secret:      VarField{Bool: true, Val: "true"},
			UseRawValue: VarField{Bool: true, Val: "true"},
		}, doc.Variables[0])
		assert.Equal(t, Variable{
			Name:               VarField{Val: "test"},
			Type:               VarField{Val: TypeInput},
			Default:            VarField{Val: "lala"},
			Description:        VarField{Val: "help text"},
			SaveInXlVals:       VarField{Bool: true, Val: "true"},
			ShowValueOnSummary: VarField{Bool: true, Val: "true"},
		}, doc.Variables[1])
		assert.Equal(t, TemplateConfig{
			Path:     "xebialabs/foo.yaml",
			FullPath: filepath.Join("aws/test", "xebialabs/foo.yaml"),
		}, doc.TemplateConfigs[0])
		assert.Equal(t, TemplateConfig{
			Path:      "readme.md",
			FullPath:  filepath.Join("aws/test", "readme.md"),
			DependsOn: VarField{Val: "isit"},
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
			Secret:      VarField{Bool: true, Val: "true"},
		}, doc.Variables[0])
		assert.Equal(t, Variable{
			Name:         VarField{Val: "test"},
			Type:         VarField{Val: TypeInput},
			Default:      VarField{Val: "lala"},
			Description:  VarField{Val: "help text"},
			SaveInXlVals: VarField{Bool: true, Val: "true"},
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
			Value:       VarField{Bool: true, Val: "true"},
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
			DependsOn:      VarField{Val: "isit && true", Tag: tagExpression},
			DependsOnFalse: VarField{Val: "isitnot"},
		}, doc.Variables[6])
	})
	t.Run("should parse files from valid metadata", func(t *testing.T) {
		doc, err := getValidTestBlueprintMetadata("templatePath/test", blueprintRepository)
		require.Nil(t, err)
		assert.Equal(t, 4, len(doc.TemplateConfigs))
		assert.Equal(t, TemplateConfig{
			Path:     "xebialabs/foo.yaml",
			FullPath: "templatePath/test/xebialabs/foo.yaml",
		}, doc.TemplateConfigs[0])
		assert.Equal(t, TemplateConfig{
			Path:      "readme.md",
			FullPath:  "templatePath/test/readme.md",
			DependsOn: VarField{Val: "isit"},
		}, doc.TemplateConfigs[1])
		assert.Equal(t, TemplateConfig{
			Path:      "bar.md",
			FullPath:  "templatePath/test/bar.md",
			DependsOn: VarField{Val: "isitnot"},
		}, doc.TemplateConfigs[2])
		assert.Equal(t, TemplateConfig{
			Path:           "foo.md",
			FullPath:       "templatePath/test/foo.md",
			DependsOnFalse: VarField{Val: "!!isitnot", Tag: tagExpression},
		}, doc.TemplateConfigs[3])
	})
	t.Run("should parse includes from valid metadata", func(t *testing.T) {
		doc, err := getValidTestBlueprintMetadata("templatePath/test", blueprintRepository)
		require.Nil(t, err)
		assert.Equal(t, 2, len(doc.Include))
		assert.Equal(t, IncludedBlueprintProcessed{
			Blueprint: "kubernetes/gke-cluster",
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
		}, doc.Include[0])
		assert.Equal(t, IncludedBlueprintProcessed{
			Blueprint: "kubernetes/namespace",
			Stage:     "after",
			ParameterValues: []ParameterValuesProcessed{
				{
					Name:  "Foo",
					Value: VarField{Val: "hello"},
				},
			},
			DependsOn: VarField{Tag: "!expression", Val: "ExpTest1 == 'us-west' && AppName != 'foo' && TestDepends"},
		}, doc.Include[1])
	})
	t.Run("should parse metadata fields", func(t *testing.T) {
		doc, err := getValidTestBlueprintMetadata("templatePath/test", blueprintRepository)
		require.Nil(t, err)
		assert.Equal(t, "Test Project", doc.Metadata.ProjectName)
		assert.Equal(t,
			"Is just a test blueprint project used for manual testing of inputs",
			doc.Metadata.Description)
		assert.Equal(t,
			"XebiaLabs",
			doc.Metadata.Author)
		assert.Equal(t,
			"1.0",
			doc.Metadata.Version)
		assert.Equal(t,
			"These are the instructions for executing this blueprint",
			doc.Metadata.Instructions)
	})
	t.Run("should parse multiline instructions", func(t *testing.T) {
		metadata := []byte(
			fmt.Sprintf(`
                apiVersion: %s
                kind: Blueprint
                metadata:
                  projectName: allala
                  instructions: |
                    This is a multiline instruction:

                    The instructions continue here:
                      1. First step
                      2. Second step
                spec:`, models.YamlFormatVersion))
		doc, err := parseTemplateMetadata(&metadata, "aws/test", &blueprintRepository, true)
		require.Nil(t, err)
		assert.Equal(t,
			"This is a multiline instruction:\n\nThe instructions continue here:\n  1. First step\n  2. Second step\n",
			doc.Metadata.Instructions)
	})
}

func TestVerifyTemplateDirAndPaths(t *testing.T) {
	t.Run("should get template config from relative paths", func(t *testing.T) {
		tmpDir := filepath.Join("test", "blueprints")
		os.MkdirAll(tmpDir, os.ModePerm)
		defer os.RemoveAll("test")
		d1 := []byte("hello\ngo\n")
		ioutil.WriteFile(filepath.Join(tmpDir, "test.yaml.tmpl"), d1, os.ModePerm)
		ioutil.WriteFile(filepath.Join(tmpDir, "test2.yaml.tmpl"), d1, os.ModePerm)

		blueprintDoc := BlueprintConfig{
			TemplateConfigs: []TemplateConfig{
				{Path: "test.yaml.tmpl", FullPath: filepath.Join(tmpDir, "test.yaml.tmpl")},
				{Path: "test2.yaml.tmpl", FullPath: filepath.Join(tmpDir, "test2.yaml.tmpl")},
			},
		}
		err := blueprintDoc.verifyTemplateDirAndPaths(tmpDir)
		require.Nil(t, err)
		require.NotNil(t, blueprintDoc.TemplateConfigs)
		assert.Equal(t, []TemplateConfig{
			{Path: "test.yaml.tmpl", FullPath: filepath.Join(tmpDir, "test.yaml.tmpl")},
			{Path: "test2.yaml.tmpl", FullPath: filepath.Join(tmpDir, "test2.yaml.tmpl")},
		}, blueprintDoc.TemplateConfigs)
	})
	t.Run("should get template config from relative nested paths", func(t *testing.T) {
		tmpDir := filepath.Join("test", "blueprints")
		os.MkdirAll(filepath.Join(tmpDir, "nested"), os.ModePerm)
		defer os.RemoveAll("test")
		d1 := []byte("hello\ngo\n")
		ioutil.WriteFile(filepath.Join(tmpDir, "test.yaml.tmpl"), d1, os.ModePerm)
		ioutil.WriteFile(filepath.Join(tmpDir, "nested", "test2.yaml.tmpl"), d1, os.ModePerm)

		blueprintDoc := BlueprintConfig{
			TemplateConfigs: []TemplateConfig{
				{Path: filepath.Join("nested", "test2.yaml.tmpl"), FullPath: filepath.Join(tmpDir, filepath.Join("nested", "test2.yaml.tmpl"))},
				{Path: "test.yaml.tmpl", FullPath: filepath.Join(tmpDir, "test.yaml.tmpl")},
			},
		}
		err := blueprintDoc.verifyTemplateDirAndPaths(tmpDir)
		require.Nil(t, err)
		require.NotNil(t, blueprintDoc.TemplateConfigs)
		assert.Equal(t, []TemplateConfig{
			{Path: filepath.Join("nested", "test2.yaml.tmpl"), FullPath: filepath.Join(tmpDir, filepath.Join("nested", "test2.yaml.tmpl"))},
			{Path: "test.yaml.tmpl", FullPath: filepath.Join(tmpDir, "test.yaml.tmpl")},
		}, blueprintDoc.TemplateConfigs)
	})

	t.Run("should get template config from absolute nested paths", func(t *testing.T) {
		tmpDir, err := ioutil.TempDir("", "blueprints")
		require.Nil(t, err)
		defer os.RemoveAll(tmpDir)
		os.MkdirAll(filepath.Join(tmpDir, "nested"), os.ModePerm)
		d1 := []byte("hello\ngo\n")
		ioutil.WriteFile(filepath.Join(tmpDir, "test.yaml.tmpl"), d1, os.ModePerm)
		ioutil.WriteFile(filepath.Join(tmpDir, "nested", "test2.yaml.tmpl"), d1, os.ModePerm)

		blueprintDoc := BlueprintConfig{
			TemplateConfigs: []TemplateConfig{
				{Path: filepath.Join("nested", "test2.yaml.tmpl"), FullPath: filepath.Join(tmpDir, filepath.Join("nested", "test2.yaml.tmpl"))},
				{Path: "test.yaml.tmpl", FullPath: filepath.Join(tmpDir, "test.yaml.tmpl")},
			},
		}
		err = blueprintDoc.verifyTemplateDirAndPaths(tmpDir)
		require.Nil(t, err)
		require.NotNil(t, blueprintDoc.TemplateConfigs)
		assert.Equal(t, []TemplateConfig{
			{Path: filepath.Join("nested", "test2.yaml.tmpl"), FullPath: filepath.Join(tmpDir, filepath.Join("nested", "test2.yaml.tmpl"))},
			{Path: "test.yaml.tmpl", FullPath: filepath.Join(tmpDir, "test.yaml.tmpl")},
		}, blueprintDoc.TemplateConfigs)
	})
	t.Run("should return error if directory is empty", func(t *testing.T) {
		tmpDir := filepath.Join("test", "blueprints")
		os.MkdirAll(tmpDir, os.ModePerm)
		defer os.RemoveAll("test")

		blueprintDoc := BlueprintConfig{}
		err := blueprintDoc.verifyTemplateDirAndPaths(tmpDir)
		require.Nil(t, blueprintDoc.TemplateConfigs)
		require.NotNil(t, err)
		require.Equal(t, "path [test/blueprints] doesn't include any valid files", err.Error())
	})
	t.Run("should return error if directory doesn't exist", func(t *testing.T) {
		blueprintDoc := BlueprintConfig{}
		err := blueprintDoc.verifyTemplateDirAndPaths(filepath.Join("test", "blueprints"))
		require.Nil(t, blueprintDoc.TemplateConfigs)
		require.NotNil(t, err)
		require.Equal(t, "path [test/blueprints] doesn't exist", err.Error())
	})
	t.Run("should return error if a file doesn't exist", func(t *testing.T) {
		tmpDir := filepath.Join("test", "blueprints")
		os.MkdirAll(tmpDir, os.ModePerm)
		defer os.RemoveAll("test")
		d1 := []byte("hello\ngo\n")
		ioutil.WriteFile(filepath.Join(tmpDir, "test2.yaml.tmpl"), d1, os.ModePerm)

		blueprintDoc := BlueprintConfig{
			TemplateConfigs: []TemplateConfig{
				{Path: "test.yaml.tmpl", FullPath: "test/blueprints/test.yaml.tmpl"},
			},
		}
		err := blueprintDoc.verifyTemplateDirAndPaths(filepath.Join("test", "blueprints"))
		require.NotNil(t, blueprintDoc.TemplateConfigs)
		require.NotNil(t, err)
		require.Equal(t, "path [test/blueprints/test.yaml.tmpl] doesn't exist", err.Error())
	})
}

func TestProcessCustomFunction_AWS(t *testing.T) {
	// Generic
	t.Run("should error on empty function string", func(t *testing.T) {
		_, err := ProcessCustomFunction("")
		require.NotNil(t, err)
		assert.Contains(t, err.Error(), "invalid syntax in function reference:")
	})
	t.Run("should error on invalid function string", func(t *testing.T) {
		_, err := ProcessCustomFunction("aws.regions.0")
		require.NotNil(t, err)
		assert.Equal(t, "invalid syntax in function reference: aws.regions.0", err.Error())
	})
	t.Run("should error on unknown function domain", func(t *testing.T) {
		_, err := ProcessCustomFunction("test.module()")
		require.NotNil(t, err)
		assert.Equal(t, "unknown function type: test", err.Error())
	})

	//AWS
	t.Run("should error on unknown AWS module", func(t *testing.T) {
		_, err := ProcessCustomFunction("aws.test()")
		require.NotNil(t, err)
		assert.Equal(t, "test is not a valid AWS module", err.Error())
	})
	t.Run("should error on missing service parameter for aws.regions function", func(t *testing.T) {
		_, err := ProcessCustomFunction("aws.regions()")
		require.NotNil(t, err)
		assert.Equal(t, "service name parameter is required for AWS regions function", err.Error())
	})
	t.Run("should return list of AWS ECS regions", func(t *testing.T) {
		regions, err := ProcessCustomFunction("aws.regions(ecs)")
		require.Nil(t, err)
		require.NotNil(t, regions)
		assert.NotEmpty(t, regions)
	})
	t.Run("should error on no attribute defined on AWS credentials", func(t *testing.T) {
		_, err := ProcessCustomFunction("aws.credentials()")
		require.NotNil(t, err)
		assert.Equal(t, "requested credentials attribute is not set", err.Error())
	})
	t.Run("should return AWS credentials", func(t *testing.T) {
		vals, err := ProcessCustomFunction("aws.credentials().AccessKeyID")
		require.Nil(t, err)
		require.NotNil(t, vals)
		require.Len(t, vals, 1)
		accessKey := vals[0]
		require.NotNil(t, accessKey)
	})
}

//OS
func TestProcessCustomFunction_OS(t *testing.T) {
	t.Run("should error on unknown OS module", func(t *testing.T) {
		_, err := ProcessCustomFunction("os.test()")
		require.NotNil(t, err)
		assert.Equal(t, "test is not a valid OS module", err.Error())
	})
	t.Run("should return an URL for os._defaultapiserverurl function", func(t *testing.T) {
		apiServerURL, err := ProcessCustomFunction("os._defaultapiserverurl()")
		require.Nil(t, err)
		assert.Len(t, apiServerURL, 1)
	})
}

// K8S
func TestProcessCustomFunction_K8S(t *testing.T) {
	defer os.RemoveAll("test")
	Setupk8sConfig()

	t.Run("should error on invalid function string", func(t *testing.T) {
		_, err := ProcessCustomFunction("k8s.IsAvailable.0")
		require.NotNil(t, err)
		assert.Equal(t, "invalid syntax in function reference: k8s.IsAvailable.0", err.Error())
	})

	t.Run("should error on unknown K8S module", func(t *testing.T) {
		_, err := ProcessCustomFunction("k8s.test()")
		require.NotNil(t, err)
		assert.Equal(t, "test is not a valid Kubernetes module", err.Error())
	})
	t.Run("should return empty on unknown parameter", func(t *testing.T) {
		out, err := ProcessCustomFunction("k8s.config().clusterParam")
		require.Nil(t, err)
		assert.Equal(t, []string{""}, out)
	})
	t.Run("should check if kubernetes config is available", func(t *testing.T) {
		out, err := ProcessCustomFunction("k8s.config().IsAvailable")
		require.Nil(t, err)
		assert.Equal(t, []string{"true"}, out)
	})
	t.Run("should check if kubernetes config is available when context doesn't exist", func(t *testing.T) {
		out, err := ProcessCustomFunction("k8s.config(dummy).IsAvailable")
		require.Nil(t, err)
		assert.Equal(t, []string{"false"}, out)
	})

	t.Run("should check if kubernetes config is available when user doesn't exist", func(t *testing.T) {
		out, err := ProcessCustomFunction("k8s.config(aws).IsAvailable")
		require.Nil(t, err)
		assert.Equal(t, []string{"true"}, out)
	})

	t.Run("should fetch a kubernetes config property from current context", func(t *testing.T) {
		out, err := ProcessCustomFunction("k8s.config().cluster_server")
		require.Nil(t, err)
		assert.Equal(t, []string{"https://test.hcp.eastus.azmk8s.io:443"}, out)
		out, err = ProcessCustomFunction("k8s.config().clusterServer")
		require.Nil(t, err)
		assert.Equal(t, []string{"https://test.hcp.eastus.azmk8s.io:443"}, out)
	})
	t.Run("should fetch a kubernetes config property from provided context", func(t *testing.T) {
		out, err := ProcessCustomFunction("k8s.config(default/ocpm-test-com:8443/test).cluster_server")
		require.Nil(t, err)
		assert.Equal(t, []string{"https://ocpm.test.com:8443"}, out)
		out, err = ProcessCustomFunction("k8s.config(default/ocpm-test-com:8443/test).clusterServer")
		require.Nil(t, err)
		assert.Equal(t, []string{"https://ocpm.test.com:8443"}, out)
	})
}
func TestProcessCustomFunction_K8S_noconfig(t *testing.T) {
	defer os.RemoveAll("test")
	tmpDir := filepath.Join("test", "blueprints")
	os.MkdirAll(tmpDir, os.ModePerm)
	os.Setenv("KUBECONFIG", filepath.Join(tmpDir, "config"))

	t.Run("should check if kubernetes config is available when file doesn't exist", func(t *testing.T) {
		out, err := ProcessCustomFunction("k8s.config().IsAvailable")
		require.Nil(t, err)
		assert.Equal(t, []string{"false"}, out)
	})

	t.Run("should check if kubernetes config is available when file doesn't exist", func(t *testing.T) {
		out, err := ProcessCustomFunction("k8s.config(test).IsAvailable")
		require.Nil(t, err)
		assert.Equal(t, []string{"false"}, out)
	})
}

func TestValidatePrompt(t *testing.T) {
	type args struct {
	    varName      string
	    validateExpr string
		pattern      string
		value        string
		emtpyAllowed bool
	    params       map[string]interface{}
	}
	tests := []struct {
		name string
		args args
		want error
	}{
		{
		    "should pass on empty value since empty values are allowed in secret fields",
		    args{"test", "", "", "", true, nil},
		    nil,
		},
		{
		    "should fail required validation on empty value",
		    args{"test", "", "", "", false, nil},
		    fmt.Errorf("Value is required"),
		},
		{
		    "should fail required validation on empty value with pattern",
		    args{"test", "", ".", "", false, nil},
		    fmt.Errorf("Value is required"),
		},
		{
		    "should pass required validation on valid value",
		    args{"test", "", "", "test", false, nil},
		    nil,
		},
		{
		    "should fail pattern validation on invalid value",
		    args{"test", "", "[a-z]*", "123", false, nil},
		    fmt.Errorf("Value should match pattern [a-z]*"),
		},
		{
		    "should pass pattern validation on valid value",
		    args{"test", "", "[a-z]*", "abc", false, nil},
		    nil,
		},
		{
		    "should pass pattern validation on valid value with extra start/end tag on pattern",
		    args{"test", "", "^[a-z]*$", "abc", false, nil},
		    nil,
		},
		{
		    "should pass pattern validation on valid value with fixed pattern",
		    args{"test", "", "test", "test", false, nil},
		    nil,
		},
		{
		    "should fail pattern validation on invalid value with fixed pattern",
		    args{"test", "", "test", "abcd", false, nil},
		    fmt.Errorf("Value should match pattern test"),
		},
		{
		    "should fail pattern validation on valid value with complex pattern",
		    args{"test", "", `\b(?:(?:2(?:[0-4][0-9]|5[0-5])|[0-1]?[0-9]?[0-9])\.){3}(?:(?:2([0-4][0-9]|5[0-5])|[0-1]?[0-9]?[0-9]))\b`, "123.123.123.256", false, nil},
			fmt.Errorf(`Value should match pattern \b(?:(?:2(?:[0-4][0-9]|5[0-5])|[0-1]?[0-9]?[0-9])\.){3}(?:(?:2([0-4][0-9]|5[0-5])|[0-1]?[0-9]?[0-9]))\b`),
		},
		{
			"should pass pattern validation on valid value with complex pattern",
			args{"test", "", `\b(?:(?:2(?:[0-4][0-9]|5[0-5])|[0-1]?[0-9]?[0-9])\.){3}(?:(?:2([0-4][0-9]|5[0-5])|[0-1]?[0-9]?[0-9]))\b`, "255.255.255.255", false, nil},
			nil,
		},
		{
		    "should fail pattern validation on invalid pattern",
		    args{"test", "", "[[", "abcd", false, nil},
		    fmt.Errorf("error parsing regexp: missing closing ]: `[[$`"),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := validatePrompt(tt.args.varName, tt.args.validateExpr, tt.args.pattern, tt.args.emtpyAllowed, tt.args.params)(tt.args.value)
			if tt.want == nil || got == nil {
				assert.Equal(t, tt.want, got)
			} else {
				assert.Equal(t, tt.want.Error(), got.Error())
			}
		})
	}
}

func TestValidateFilePath(t *testing.T) {
	tmpDir := filepath.Join("test", "file-input")
	type args struct {
		value      string
		fileExists bool
	}
	tests := []struct {
		name string
		args args
		want error
	}{
		{"should pass on existing valid file path input", args{filepath.Join(tmpDir, "valid.txt"), true}, nil},
		{"should fail on non-existing file path input", args{"not-valid.txt", false}, fmt.Errorf("file not found on path not-valid.txt")},
		{"should fail on directory path input", args{tmpDir, false}, fmt.Errorf("given path is a directory, file path is needed")},
		{"should fail on empty input", args{"", false}, fmt.Errorf("Value is required")},
	}
	for _, tt := range tests {
		// Create needed temporary directory for tests
		os.MkdirAll(tmpDir, os.ModePerm)
		defer os.RemoveAll("test")

		t.Run(tt.name, func(t *testing.T) {
			if tt.args.fileExists {
				contents := []byte("hello\ngo\n")
				ioutil.WriteFile(tt.args.value, contents, os.ModePerm)
			}

			got := validateFilePath()(tt.args.value)
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
	tests := []struct {
		name    string
		fields  BlueprintYaml
		want    []TemplateConfig
		wantErr error
	}{
		{
			"parse a valid file declaration",
			BlueprintYaml{
				Spec: Spec{
					Files: []File{
						{Path: "test.yaml"},
						{Path: "test2.yaml"},
					},
				},
			},
			[]TemplateConfig{
				{Path: "test.yaml", FullPath: filepath.Join(templatePath, "test.yaml")},
				{Path: "test2.yaml", FullPath: filepath.Join(templatePath, "test2.yaml")},
			},
			nil,
		},
		{
			"parse a valid file declaration with dependsOn that refers to existing variables",
			BlueprintYaml{
				Spec: Spec{
					Parameters: []Parameter{
						{Name: "foo", Type: "Confirm", Value: "true"},
						{Name: "bar", Type: "Confirm", Value: "false"},
					},
					Files: []File{
						{Path: "test.yaml"},
						{Path: "test2.yaml", DependsOn: "foo"},
						{Path: "test3.yaml", DependsOnFalse: "bar"},
						{Path: "test4.yaml", DependsOn: "bar"},
						{Path: "test5.yaml", DependsOnFalse: "foo"},
					},
				},
			},
			[]TemplateConfig{
				{Path: "test.yaml", FullPath: filepath.Join(templatePath, "test.yaml")},
				{Path: "test2.yaml", FullPath: filepath.Join(templatePath, "test2.yaml"), DependsOn: VarField{Val: "foo", Tag: ""}},
				{Path: "test3.yaml", FullPath: filepath.Join(templatePath, "test3.yaml"), DependsOnFalse: VarField{Val: "bar", Tag: ""}},
				{Path: "test4.yaml", FullPath: filepath.Join(templatePath, "test4.yaml"), DependsOn: VarField{Val: "bar", Tag: ""}},
				{Path: "test5.yaml", FullPath: filepath.Join(templatePath, "test5.yaml"), DependsOnFalse: VarField{Val: "foo", Tag: ""}},
			},
			nil,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			blueprintDoc := &BlueprintYaml{
				ApiVersion: tt.fields.ApiVersion,
				Kind:       tt.fields.Kind,
				Metadata:   tt.fields.Metadata,
				Spec:       tt.fields.Spec,
			}
			tconfigs, err := blueprintDoc.parseFiles(templatePath, true)
			if tt.wantErr == nil || err == nil {
				assert.Equal(t, tt.wantErr, err)
			} else {
				assert.Equal(t, tt.wantErr.Error(), err.Error())
			}
			assert.Equal(t, tt.want, tconfigs)
		})
	}
}

func TestParseFile(t *testing.T) {
	tests := []struct {
		name    string
		args    *File
		want    TemplateConfig
		wantErr error
	}{
		{
			"return empty for empty map",
			&File{},
			TemplateConfig{},
			nil,
		},
		{
			"parse a file declaration with only path",
			&File{
				Path: "test.yaml",
			},
			TemplateConfig{Path: "test.yaml"},
			nil,
		},
		{
			"parse a file declaration with only path and nil for dependsOn",
			&File{
				Path: "test.yaml", DependsOn: "",
			},
			TemplateConfig{Path: "test.yaml"},
			nil,
		},
		{
			"parse a file declaration with path and dependsOnTrue",
			&File{
				Path: "test.yaml", DependsOnTrue: "foo",
			},
			TemplateConfig{Path: "test.yaml", DependsOn: VarField{Val: "foo"}},
			nil,
		},
		{
			"parse a file declaration with path dependsOnFalse and dependsOn",
			&File{
				Path: "test.yaml", DependsOn: "foo", DependsOnFalse: "bar",
			},
			TemplateConfig{Path: "test.yaml", DependsOn: VarField{Val: "foo"}, DependsOnFalse: VarField{Val: "bar"}},
			nil,
		},
		{
			"parse a file declaration with path and dependsOn as !fn tag",
			&File{
				Path: "test.yaml", DependsOn: yaml.CustomTag{Tag: "!fn", Value: "aws.credentials().IsAvailable"},
			},
			TemplateConfig{Path: "test.yaml", DependsOn: VarField{Val: "aws.credentials().IsAvailable", Tag: "!fn"}},
			nil,
		},
		{
			"parse a file declaration with path and dependsOn as !expression tag",
			&File{
				Path: "test.yaml", DependsOnTrue: yaml.CustomTag{Tag: "!expression", Value: "1 > 2"},
			},
			TemplateConfig{Path: "test.yaml", DependsOn: VarField{Val: "1 > 2", Tag: "!expression"}},
			nil,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseFile(tt.args)
			if tt.wantErr == nil || err == nil {
				assert.Equal(t, tt.wantErr, err)
			} else {
				assert.Equal(t, tt.wantErr.Error(), err.Error())
			}
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestGetValuesFromAnswersFile(t *testing.T) {
	// Create needed temporary directory for tests
	os.MkdirAll("test", os.ModePerm)
	defer os.RemoveAll("test")
	validContent := []byte(`test: testing
sample: 5.45
confirm: true
`)
	badFormatContent := []byte(`test=testing
sample=5.45
confirm=true
`)
	validFilePath := filepath.Join("test", "answers.yaml")
	badFormatFilePath := filepath.Join("test", "badformat.yaml")
	ioutil.WriteFile(validFilePath, validContent, os.ModePerm)
	ioutil.WriteFile(badFormatFilePath, badFormatContent, os.ModePerm)

	tests := []struct {
		name            string
		answersFilePath string
		wantOut         map[string]interface{}
		errOut          bool
	}{
		{
			"answers file: error when file not found",
			"error.yaml",
			nil,
			true,
		},
		{
			"answers file: error when content is not proper yaml",
			badFormatFilePath,
			nil,
			true,
		},
		{
			"answers file: parse map of answers from valid file",
			validFilePath,
			map[string]interface{}{"test": "testing", "sample": 5.45, "confirm": true},
			false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := getValuesFromAnswersFile(tt.answersFilePath)
			if tt.errOut {
				require.NotNil(t, err)
			} else {
				require.Nil(t, err)
			}
			assert.Equal(t, tt.wantOut, got)
		})
	}
}

func TestVerifyVariableValue(t *testing.T) {
	// Create needed temporary directory for tests
	os.MkdirAll("test", os.ModePerm)
	defer os.RemoveAll("test")
	contents := []byte("hello\ngo\n")
	ioutil.WriteFile(filepath.Join("test", "sample.txt"), contents, os.ModePerm)

	tests := []struct {
		name       string
		variable   Variable
		answer     interface{}
		parameters map[string]interface{}
		wantOut    interface{}
		errOut     error
	}{
		{
			"answers from map: save string answer value to variable value with type Input",
			Variable{Name: VarField{Val: "Test"}, Type: VarField{Val: TypeInput}},
			"sample answer",
			map[string]interface{}{},
			"sample answer",
			nil,
		},
		{
			"answers from map: save float answer value to variable value with type Input",
			Variable{Name: VarField{Val: "Test"}, Type: VarField{Val: TypeInput}},
			5.45,
			map[string]interface{}{},
			5.45,
			nil,
		},
		{
			"answers from map: save boolean answer value to variable value with type Confirm",
			Variable{Name: VarField{Val: "Test"}, Type: VarField{Val: TypeConfirm}},
			true,
			map[string]interface{}{},
			true,
			nil,
		},
		{
			"answers from map: save boolean answer value (convert from string) to variable value with type Confirm",
			Variable{Name: VarField{Val: "Test"}, Type: VarField{Val: TypeConfirm}},
			"true",
			map[string]interface{}{},
			true,
			nil,
		},
		{
			"answers from map: save long text answer value to variable value with type Editor",
			Variable{Name: VarField{Val: "Test"}, Type: VarField{Val: TypeEditor}},
			"long text for testing..\nand the rest of it\n",
			map[string]interface{}{},
			"long text for testing..\nand the rest of it\n",
			nil,
		},
		{
			"answers from map: save long text answer value to variable value with type File",
			Variable{Name: VarField{Val: "Test"}, Type: VarField{Val: TypeFile}},
			filepath.Join("test", "sample.txt"),
			map[string]interface{}{},
			"hello\ngo\n",
			nil,
		},
		{
			"answers from map: give error on file not found (input type: File)",
			Variable{Name: VarField{Val: "Test"}, Type: VarField{Val: TypeFile}},
			filepath.Join("test", "error.txt"),
			map[string]interface{}{},
			"",
			fmt.Errorf(
				"error reading input file [%s]: open %s: no such file or directory",
				filepath.Join("test", "error.txt"),
				filepath.Join("test", "error.txt"),
			),
		},
		{
			"answers from map: save string answer value to variable value with type Select",
			Variable{Name: VarField{Val: "Test"}, Type: VarField{Val: TypeSelect}, Options: []VarField{{Val: "a"}, {Val: "b"}}},
			"b",
			map[string]interface{}{},
			"b",
			nil,
		},
		{
			"answers from map: give error on unknown select option value",
			Variable{Name: VarField{Val: "Test"}, Type: VarField{Val: TypeSelect}, Options: []VarField{{Val: "a"}, {Val: "b"}}},
			"c",
			map[string]interface{}{},
			"",
			fmt.Errorf("answer [c] is not one of the available options [a b] for variable [Test]"),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := tt.variable.VerifyVariableValue(tt.answer, tt.parameters)
			assert.Equal(t, tt.errOut, err)
			assert.Equal(t, tt.wantOut, got)
		})
	}
}

func TestBlueprintYaml_parseParameters(t *testing.T) {
	tests := []struct {
		name    string
		params  []Parameter
		spec    Spec
		want    []Variable
		wantErr bool
	}{
		{
			"should error on invalid tag in dependsOn ",
			nil,
			Spec{
				Parameters: []Parameter{
					{
						Name:           "test",
						Type:           "Input",
						Secret:         true,
						Value:          "string",
						Description:    "desc",
						Default:        "string2",
						DependsOn:      yaml.CustomTag{Tag: "!foo", Value: "1 > 2"},
						DependsOnFalse: "Var",
						Options: []interface{}{
							"test", "foo", 10, 13.4,
						},
						Pattern:      "pat",
						SaveInXlVals: true,
						UseRawValue:  false,
					},
				},
			},
			[]Variable{},
			true,
		},
		{
			"should error on invalid type in list ",
			nil,
			Spec{
				Parameters: []Parameter{
					{
						Name:           "test",
						Type:           "Input",
						Secret:         true,
						Value:          "string",
						Description:    "desc",
						Default:        "string2",
						DependsOnFalse: "Var",
						Options: []interface{}{
							"test", "foo", true,
						},
						Pattern:      "pat",
						SaveInXlVals: true,
						UseRawValue:  false,
					},
				},
			},
			[]Variable{},
			true,
		},
		{
			"should parse parameters under spec",
			nil,
			Spec{
				Parameters: []Parameter{
					{
						Name:           "test",
						Type:           "Input",
						Secret:         true,
						Value:          "string",
						Description:    "desc",
						Default:        "string2",
						DependsOn:      yaml.CustomTag{Tag: "!expression", Value: "1 > 2"},
						DependsOnFalse: "Var",
						Options: []interface{}{
							"test", "foo", 10, 13.4,
						},
						Pattern:      "pat",
						SaveInXlVals: true,
						UseRawValue:  false,
					},
					{
						Name:           "test",
						Type:           "Confirm",
						Secret:         false,
						Value:          true,
						Description:    "desc",
						Default:        false,
						DependsOnTrue:  yaml.CustomTag{Tag: "!expression", Value: "1 > 2"},
						DependsOnFalse: "Var",
						Options: []interface{}{
							"test", yaml.CustomTag{Tag: "!expression", Value: "1 > 2"},
						},
						Pattern:      "pat",
						SaveInXlVals: true,
						UseRawValue:  false,
					},
				},
			},
			[]Variable{
				{
					Name:           VarField{Val: "test"},
					Type:           VarField{Val: "Input"},
					Secret:         VarField{Bool: true, Val: "true"},
					Value:          VarField{Val: "string"},
					Description:    VarField{Val: "desc"},
					Default:        VarField{Val: "string2"},
					DependsOn:      VarField{Tag: "!expression", Val: "1 > 2"},
					DependsOnFalse: VarField{Val: "Var"},
					Options: []VarField{
						VarField{Val: "test"}, VarField{Val: "foo"}, VarField{Val: "10"}, VarField{Val: "13.400000"},
					},
					Pattern:      VarField{Val: "pat"},
					SaveInXlVals: VarField{Bool: true, Val: "true"},
					UseRawValue:  VarField{Bool: false, Val: "false"},
				},
				{
					Name:           VarField{Val: "test"},
					Type:           VarField{Val: "Confirm"},
					Secret:         VarField{Bool: false, Val: "false"},
					Value:          VarField{Bool: true, Val: "true"},
					Description:    VarField{Val: "desc"},
					Default:        VarField{Bool: false, Val: "false"},
					DependsOn:      VarField{Tag: "!expression", Val: "1 > 2"},
					DependsOnFalse: VarField{Val: "Var"},
					Options: []VarField{
						VarField{Val: "test"}, VarField{Tag: "!expression", Val: "1 > 2"},
					},
					Pattern:      VarField{Val: "pat"},
					SaveInXlVals: VarField{Bool: true, Val: "true"},
					UseRawValue:  VarField{Bool: false, Val: "false"},
				},
			},
			false,
		},
		{
			"should parse parameters",
			[]Parameter{
				{
					Name:           "test",
					Type:           "Input",
					Secret:         true,
					Value:          "string",
					Description:    "desc",
					Default:        "string2",
					DependsOnTrue:  yaml.CustomTag{Tag: "!expression", Value: "1 > 2"},
					DependsOnFalse: "Var",
					Options: []interface{}{
						"test", "foo", 10, 13.4,
					},
					Pattern:      "pat",
					SaveInXlVals: true,
					UseRawValue:  false,
				},
				{
					Name:           "test",
					Type:           "Confirm",
					Secret:         false,
					Value:          true,
					Description:    "desc",
					Default:        false,
					DependsOn:      yaml.CustomTag{Tag: "!expression", Value: "1 > 2"},
					DependsOnFalse: "Var",
					Options: []interface{}{
						"test", yaml.CustomTag{Tag: "!expression", Value: "1 > 2"},
					},
					Pattern:      "pat",
					SaveInXlVals: true,
					UseRawValue:  false,
				},
			},
			Spec{},
			[]Variable{
				{
					Name:           VarField{Val: "test"},
					Type:           VarField{Val: "Input"},
					Secret:         VarField{Bool: true, Val: "true"},
					Value:          VarField{Val: "string"},
					Description:    VarField{Val: "desc"},
					Default:        VarField{Val: "string2"},
					DependsOn:      VarField{Tag: "!expression", Val: "1 > 2"},
					DependsOnFalse: VarField{Val: "Var"},
					Options: []VarField{
						VarField{Val: "test"}, VarField{Val: "foo"}, VarField{Val: "10"}, VarField{Val: "13.400000"},
					},
					Pattern:      VarField{Val: "pat"},
					SaveInXlVals: VarField{Bool: true, Val: "true"},
					UseRawValue:  VarField{Bool: false, Val: "false"},
				},
				{
					Name:           VarField{Val: "test"},
					Type:           VarField{Val: "Confirm"},
					Secret:         VarField{Bool: false, Val: "false"},
					Value:          VarField{Bool: true, Val: "true"},
					Description:    VarField{Val: "desc"},
					Default:        VarField{Bool: false, Val: "false"},
					DependsOn:      VarField{Tag: "!expression", Val: "1 > 2"},
					DependsOnFalse: VarField{Val: "Var"},
					Options: []VarField{
						VarField{Val: "test"}, VarField{Tag: "!expression", Val: "1 > 2"},
					},
					Pattern:      VarField{Val: "pat"},
					SaveInXlVals: VarField{Bool: true, Val: "true"},
					UseRawValue:  VarField{Bool: false, Val: "false"},
				},
			},
			false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			blueprintDoc := &BlueprintYaml{
				ApiVersion: "",
				Kind:       "",
				Parameters: tt.params,
				Spec:       tt.spec,
			}
			got, err := blueprintDoc.parseParameters()
			if (err != nil) != tt.wantErr {
				t.Errorf("BlueprintYaml.parseParameters() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestBlueprintYaml_prepareTemplateData(t *testing.T) {
	SkipUserInput = true
	SkipFinalPrompt = true
	type args struct {
		answersFilePath    string
		strictAnswers      bool
		useDefaultsAsValue bool
	}
	tests := []struct {
		name    string
		fields  BlueprintConfig
		args    args
		want    *PreparedData
		wantErr bool
	}{
		{
			"prepare template data should show password hidden if ShowValueOnSummary is false",
			BlueprintConfig{
				Variables: []Variable{
					{
						Name:    VarField{Val: "input1"},
						Type:    VarField{Val: "Input"},
						Value:   VarField{Bool: false, Val: ""},
						Default: VarField{Bool: false, Val: "default1"},
					},
					{
						Name:    VarField{Val: "input2"},
						Type:    VarField{Val: "Input"},
						Value:   VarField{Bool: false, Val: ""},
						Default: VarField{Bool: false, Val: "default2"},
						Secret:  VarField{Bool: true, Val: "true"},
					},
					{
						Name:               VarField{Val: "input3"},
						Type:               VarField{Val: "Input"},
						Value:              VarField{Bool: false, Val: ""},
						Default:            VarField{Bool: false, Val: "default3"},
						Secret:             VarField{Bool: true, Val: "true"},
						ShowValueOnSummary: VarField{Bool: true, Val: "true"},
					},
				},
			},
			args{"", false, true},
			&PreparedData{
				TemplateData: map[string]interface{}{"input1": "default1", "input2": "!value input2", "input3": "!value input3"},
				DefaultData:  map[string]interface{}{"input1": "default1", "input2": "*****", "input3": "default3"},
				Secrets:      map[string]interface{}{"input2": "default2", "input3": "default3"},
				Values:       map[string]interface{}{},
			},
			false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			blueprintDoc := &BlueprintConfig{
				ApiVersion:      tt.fields.ApiVersion,
				Kind:            tt.fields.Kind,
				Metadata:        tt.fields.Metadata,
				TemplateConfigs: tt.fields.TemplateConfigs,
				Variables:       tt.fields.Variables,
			}
			got, err := blueprintDoc.prepareTemplateData(tt.args.answersFilePath, tt.args.strictAnswers, tt.args.useDefaultsAsValue, nil)
			if (err != nil) != tt.wantErr {
				t.Errorf("BlueprintYaml.prepareTemplateData() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestBlueprintYaml_parseIncludes(t *testing.T) {
	tests := []struct {
		name    string
		fields  BlueprintYaml
		want    []IncludedBlueprintProcessed
		wantErr error
	}{
		{
			"parse a valid include declaration with dependsOn that is an expression",
			BlueprintYaml{
				Spec: Spec{
					Include: []IncludedBlueprint{
						{
							Blueprint:      "bar",
							Stage:          "after",
							DependsOn:      yaml.CustomTag{Tag: "!expression", Value: "1 > 2"},
							DependsOnFalse: "Var",
						},
						{
							Blueprint: "foo",
							Stage:     "before",
							ParameterValues: []ParameterValue{
								{
									Name:      "foo",
									Value:     "bar",
									DependsOn: yaml.CustomTag{Tag: "!expression", Value: "1 > 2"},
								},
								{
									Name:          "bar",
									Value:         true,
									DependsOnTrue: yaml.CustomTag{Tag: "!fn", Value: "foo"},
								},
								{
									Name:           "barr",
									Value:          10.5,
									DependsOnFalse: yaml.CustomTag{Tag: "!fn", Value: "foo"},
								},
							},
							SkipFiles: []File{
								{
									Path:      "foo/bar.md",
									DependsOn: yaml.CustomTag{Tag: "!fn", Value: "foo"},
								},
								{
									Path:           "foo/bar2.md",
									DependsOnFalse: yaml.CustomTag{Tag: "!fn", Value: "foo"},
								},
							},
							DependsOnTrue:  yaml.CustomTag{Tag: "!expression", Value: "1 > 2"},
							DependsOnFalse: "Var",
						},
					},
				},
			},
			[]IncludedBlueprintProcessed{
				{
					Blueprint:      "bar",
					Stage:          "after",
					DependsOn:      VarField{Val: "1 > 2", Tag: "!expression"},
					DependsOnFalse: VarField{Val: "Var"},
				},
				{
					Blueprint: "foo",
					Stage:     "before",
					ParameterValues: []ParameterValuesProcessed{
						{
							Name:      "foo",
							Value:     VarField{Val: "bar"},
							DependsOn: VarField{Val: "1 > 2", Tag: "!expression"},
						},
						{
							Name:      "bar",
							Value:     VarField{Val: "true", Bool: true},
							DependsOn: VarField{Tag: "!fn", Val: "foo"},
						},
						{
							Name:           "barr",
							Value:          VarField{Val: "10.500000"},
							DependsOnFalse: VarField{Tag: "!fn", Val: "foo"},
						},
					},
					SkipFiles: []TemplateConfig{
						{
							Path:      "foo/bar.md",
							DependsOn: VarField{Tag: "!fn", Val: "foo"},
						},
						{
							Path:           "foo/bar2.md",
							DependsOnFalse: VarField{Tag: "!fn", Val: "foo"},
						},
					},
					DependsOn:      VarField{Val: "1 > 2", Tag: "!expression"},
					DependsOnFalse: VarField{Val: "Var"},
				},
			},
			nil,
		},
	}
	for _, tt := range tests {

		t.Run(tt.name, func(t *testing.T) {
			blueprintDoc := &BlueprintYaml{
				ApiVersion: tt.fields.ApiVersion,
				Kind:       tt.fields.Kind,
				Metadata:   tt.fields.Metadata,
				Spec:       tt.fields.Spec,
			}
			tconfigs, err := blueprintDoc.parseIncludes()
			if tt.wantErr == nil || err == nil {
				assert.Equal(t, tt.wantErr, err)
			} else {
				assert.Equal(t, tt.wantErr.Error(), err.Error())
			}
			assert.Equal(t, tt.want, tconfigs)
		})
	}
}

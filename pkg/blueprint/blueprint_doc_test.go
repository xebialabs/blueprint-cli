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
	"github.com/xebialabs/blueprint-cli/pkg/cloud/aws"
)

var SampleKubeConfig = `apiVersion: v1
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
	d1 := []byte(SampleKubeConfig)
	ioutil.WriteFile(filepath.Join(tmpDir, "config"), d1, os.ModePerm)
	os.Setenv("KUBECONFIG", filepath.Join(tmpDir, "config"))
}

var dummyData = make(map[string]interface{})

func TestGetVariableDefaultVal(t *testing.T) {
	t.Run("should return empty string when default is not defined", func(t *testing.T) {
		v := Variable{
			Name: VarField{Value: "test"},
			Type: VarField{Value: TypeInput},
		}
		defaultVal := v.GetDefaultVal()
		assert.Equal(t, "", defaultVal)
	})

	t.Run("should return default value string when default is defined", func(t *testing.T) {
		v := Variable{
			Name:    VarField{Value: "test"},
			Type:    VarField{Value: TypeInput},
			Default: VarField{Value: "default_val"},
		}
		defaultVal := v.GetDefaultVal()
		assert.Equal(t, "default_val", defaultVal)
	})

	t.Run("should return empty string when invalid function tag in default field", func(t *testing.T) {
		v := Variable{
			Name:    VarField{Value: "test"},
			Type:    VarField{Value: TypeInput},
			Default: VarField{Value: "aws.regs", Tag: tagFnV1},
		}
		defaultVal := v.GetDefaultVal()
		assert.Equal(t, "", defaultVal)
	})

	t.Run("should return function output on valid function tag in default field", func(t *testing.T) {
		v := Variable{
			Name:    VarField{Value: "test"},
			Type:    VarField{Value: TypeInput},
			Default: VarField{Value: "aws.regions(ecs)[0]", Tag: tagFnV1},
		}
		defaultVal := v.GetDefaultVal()
		regionsList, _ := aws.GetAvailableAWSRegionsForService("ecs")
		sort.Strings(regionsList)
		assert.Equal(t, regionsList[0], defaultVal)
	})

	t.Run("should return empty string when expression tag return nil", func(t *testing.T) {
		v := Variable{
			Name:    VarField{Value: "test"},
			Type:    VarField{Value: TypeInput},
			Default: VarField{Value: "Foo", Tag: tagExpressionV2},
		}
		v.ProcessExpression(map[string]interface{}{"Foo": nil}, nil)
		defaultVal := v.GetDefaultVal()
		assert.Equal(t, "", defaultVal)
	})

	t.Run("should return output on valid expression tag in default field", func(t *testing.T) {
		v := Variable{
			Name:    VarField{Value: "test"},
			Type:    VarField{Value: TypeInput},
			Default: VarField{Value: "'foo' + 'bar'", Tag: tagExpressionV2},
		}
		v.ProcessExpression(dummyData, nil)
		defaultVal := v.GetDefaultVal()
		assert.Equal(t, "foobar", defaultVal)
		v = Variable{
			Name:    VarField{Value: "test"},
			Type:    VarField{Value: TypeInput},
			Default: VarField{Value: "Foo > 10", Tag: tagExpressionV2},
		}
		v.ProcessExpression(map[string]interface{}{
			"Foo": 100,
		}, nil)
		defaultVal = v.GetDefaultVal()
		assert.Equal(t, "true", defaultVal)
	})
}

func TestGetValueFieldVal(t *testing.T) {
	t.Run("should return value field string value when defined", func(t *testing.T) {
		v := Variable{
			Name:  VarField{Value: "test"},
			Type:  VarField{Value: TypeInput},
			Value: VarField{Value: "testing"},
		}
		val := v.GetValueFieldVal()
		assert.Equal(t, "testing", val)
	})

	t.Run("should return empty on invalid function tag in value field", func(t *testing.T) {
		v := Variable{
			Name:  VarField{Value: "test"},
			Type:  VarField{Value: TypeInput},
			Value: VarField{Value: "aws.regs", Tag: tagFnV1},
		}
		val := v.GetValueFieldVal()
		assert.Equal(t, "", val)
	})

	t.Run("should return function output on valid function tag in value field", func(t *testing.T) {
		v := Variable{
			Name:  VarField{Value: "test"},
			Type:  VarField{Value: TypeInput},
			Value: VarField{Value: "aws.regions(ecs)[0]", Tag: tagFnV1},
		}
		val := v.GetValueFieldVal()
		regionsList, _ := aws.GetAvailableAWSRegionsForService("ecs")
		sort.Strings(regionsList)
		assert.Equal(t, regionsList[0], val)
	})

	t.Run("should return empty on invalid expression tag in value field", func(t *testing.T) {
		v := Variable{
			Name:  VarField{Value: "test"},
			Type:  VarField{Value: TypeInput},
			Value: VarField{Value: "Foo", Tag: tagExpressionV2},
		}
		v.ProcessExpression(map[string]interface{}{"Foo": nil}, nil)
		val := v.GetValueFieldVal()
		assert.Equal(t, "", val)
	})

	t.Run("should return expression output on valid expression tag in value field", func(t *testing.T) {
		v := Variable{
			Name:  VarField{Value: "test"},
			Type:  VarField{Value: TypeInput},
			Value: VarField{Value: "'foo' + 'bar'", Tag: tagExpressionV2},
		}
		v.ProcessExpression(dummyData, nil)
		defaultVal := v.GetValueFieldVal()
		assert.Equal(t, "foobar", defaultVal)
		v = Variable{
			Name:  VarField{Value: "test"},
			Type:  VarField{Value: TypeInput},
			Value: VarField{Value: "Foo > 10", Tag: tagExpressionV2},
		}
		v.ProcessExpression(map[string]interface{}{
			"Foo": 100,
		}, nil)
		defaultVal = v.GetValueFieldVal()
		assert.Equal(t, "true", defaultVal)
	})
}

func TestGetOptions(t *testing.T) {
	t.Run("should return string values of options", func(t *testing.T) {
		v := Variable{
			Name:    VarField{Value: "test"},
			Type:    VarField{Value: TypeSelect},
			Options: []VarField{{Value: "a"}, {Value: "b"}, {Value: "c"}},
		}
		values := v.GetOptions(dummyData, true, nil)
		assert.Len(t, values, 3)
		assert.Equal(t, []string{"a", "b", "c"}, values)
	})

	t.Run("should return string values of map options with label", func(t *testing.T) {
		v := Variable{
			Name:    VarField{Value: "test"},
			Type:    VarField{Value: TypeSelect},
			Options: []VarField{{Value: "aVal"}, {Label: "bLabel", Value: "bVal"}, {Label: "cLabel", Value: "cVal"}},
		}
		values := v.GetOptions(dummyData, true, nil)
		assert.Len(t, values, 3)
		assert.Equal(t, []string{"aVal", "bVal [bLabel]", "cVal [cLabel]"}, values)
	})

	t.Run("should return string values of map options without label", func(t *testing.T) {
		v := Variable{
			Name:    VarField{Value: "test"},
			Type:    VarField{Value: TypeSelect},
			Options: []VarField{{Value: "aVal"}, {Label: "bLabel", Value: "bVal"}, {Label: "cLabel", Value: "cVal"}},
		}
		values := v.GetOptions(dummyData, false, nil)
		assert.Len(t, values, 3)
		assert.Equal(t, []string{"aVal", "bVal", "cVal"}, values)
	})

	t.Run("should return generated values for fn options tag", func(t *testing.T) {
		v := Variable{
			Name:    VarField{Value: "test"},
			Type:    VarField{Value: TypeSelect},
			Options: []VarField{{Value: "aws.regions(ecs)", Tag: tagFnV1}},
		}
		values := v.GetOptions(dummyData, true, nil)
		assert.True(t, len(values) > 1)
	})

	t.Run("should return empty slice on invalid function tag for options", func(t *testing.T) {
		v := Variable{
			Name:    VarField{Value: "test"},
			Type:    VarField{Value: TypeSelect},
			Options: []VarField{{Value: "aws.regs", Tag: tagFnV1}},
		}
		out := v.GetOptions(dummyData, true, nil)
		require.Equal(t, []string{}, out)
	})

	t.Run("should return generated values for expression options tag", func(t *testing.T) {
		v := Variable{
			Name:    VarField{Value: "test"},
			Type:    VarField{Value: TypeSelect},
			Options: []VarField{{Value: "Foo ? Bar : (1, 2, 3)", Tag: tagExpressionV2}},
		}
		values := v.GetOptions(map[string]interface{}{
			"Foo": true,
			"Bar": []string{"test", "foo"},
		}, true, nil)
		assert.True(t, len(values) == 2)
	})

	t.Run("should return generated string array values for expression options tag", func(t *testing.T) {
		v := Variable{
			Name:    VarField{Value: "test"},
			Type:    VarField{Value: TypeSelect},
			Options: []VarField{{Value: "Provider == 'GCP' ? ('GKE', 'CloudSore') : ('test')", Tag: tagExpressionV2}},
		}
		values := v.GetOptions(map[string]interface{}{
			"Provider": "GCP",
		}, true, nil)
		assert.NotNil(t, values)
		assert.True(t, len(values) == 2)
	})

	t.Run("should return string values for param reference in expression options tag", func(t *testing.T) {
		v := Variable{
			Name:    VarField{Value: "test"},
			Type:    VarField{Value: TypeSelect},
			Options: []VarField{{Value: "Foo ? Bar : (Foo1, Foo2)", Tag: tagExpressionV2}},
		}
		values := v.GetOptions(map[string]interface{}{
			"Foo":  false,
			"Foo1": "test",
			"Foo2": "foo",
			"Bar":  []string{"test", "foo"},
		}, true, nil)
		assert.NotNil(t, values)
		assert.True(t, len(values) == 2)
	})

	t.Run("should return string values for numeric type in expression options tag", func(t *testing.T) {
		v := Variable{
			Name:    VarField{Value: "test"},
			Type:    VarField{Value: TypeSelect},
			Options: []VarField{{Value: "Foo ? Bar : (1, 2, 3)", Tag: tagExpressionV2}},
		}
		values := v.GetOptions(map[string]interface{}{
			"Foo": false,
			"Bar": []string{"test", "foo"},
		}, true, nil)
		assert.NotNil(t, values)
		assert.True(t, len(values) == 3)
	})

	t.Run("should return string values for boolean type in expression options tag", func(t *testing.T) {
		v := Variable{
			Name:    VarField{Value: "test"},
			Type:    VarField{Value: TypeSelect},
			Options: []VarField{{Value: "Foo ? Bar : (true, false)", Tag: tagExpressionV2}},
		}
		values := v.GetOptions(map[string]interface{}{
			"Foo": false,
			"Bar": []string{"test", "foo"},
		}, true, nil)
		assert.NotNil(t, values)
		assert.True(t, len(values) == 2)
	})

	t.Run("should return empty slices values for invalid return type in expression options tag", func(t *testing.T) {
		v := Variable{
			Name:    VarField{Value: "test"},
			Type:    VarField{Value: TypeSelect},
			Options: []VarField{{Value: "Foo ? Bar : (Fooo, Foo)", Tag: tagExpressionV2}},
		}
		values := v.GetOptions(map[string]interface{}{
			"Foo": false,
			"Bar": []string{"test", "foo"},
		}, true, nil)
		assert.Equal(t, []string{}, values)
	})

	t.Run("should return empty slice on invalid expression tag for options", func(t *testing.T) {
		v := Variable{
			Name:    VarField{Value: "test"},
			Type:    VarField{Value: TypeSelect},
			Options: []VarField{{Value: "aws.regs()", Tag: tagExpressionV2}},
		}
		out := v.GetOptions(dummyData, true, nil)
		assert.Equal(t, []string{}, out)
	})
}

func TestSkipQuestionOnCondition(t *testing.T) {
	t.Run("should skip question (promptIf)", func(t *testing.T) {
		variables := make([]Variable, 2)
		variables[0] = Variable{
			Name:  VarField{Value: "confirm"},
			Type:  VarField{Value: TypeConfirm},
			Value: VarField{Bool: true, Value: "true"},
		}
		variables[1] = Variable{
			Name:      VarField{Value: "test"},
			Type:      VarField{Value: TypeInput},
			DependsOn: VarField{Value: "confirm", InvertBool: true},
		}
		assert.True(t, skipQuestionOnCondition(&variables[1], variables[1].DependsOn.Value, variables[0].Value.Bool, NewPreparedData(), "", variables[1].DependsOn.InvertBool))
	})
	t.Run("should skip question (promptIf)", func(t *testing.T) {
		variables := make([]Variable, 2)
		variables[0] = Variable{
			Name:  VarField{Value: "confirm"},
			Type:  VarField{Value: TypeConfirm},
			Value: VarField{Bool: false, Value: "false"},
		}
		variables[1] = Variable{
			Name:      VarField{Value: "test"},
			Type:      VarField{Value: TypeInput},
			DependsOn: VarField{Value: "confirm"},
		}
		assert.True(t, skipQuestionOnCondition(&variables[1], variables[1].DependsOn.Value, variables[0].Value.Bool, NewPreparedData(), "", variables[1].DependsOn.InvertBool))
	})
	t.Run("should skip question and default value should be false (promptIf)", func(t *testing.T) {
		data := NewPreparedData()
		variables := make([]Variable, 2)
		variables[0] = Variable{
			Name:  VarField{Value: "confirm"},
			Type:  VarField{Value: TypeConfirm},
			Value: VarField{Bool: false, Value: "false"},
		}
		variables[1] = Variable{
			Name:      VarField{Value: "test"},
			Type:      VarField{Value: TypeConfirm},
			DependsOn: VarField{Value: "confirm"},
		}
		assert.True(t, skipQuestionOnCondition(&variables[1], variables[1].DependsOn.Value, variables[0].Value.Bool, data, "", variables[1].DependsOn.InvertBool))
		assert.False(t, data.TemplateData[variables[1].Name.Value].(bool))
	})

	t.Run("should not skip question (promptIf)", func(t *testing.T) {
		variables := make([]Variable, 2)
		variables[0] = Variable{
			Name:  VarField{Value: "confirm"},
			Type:  VarField{Value: TypeConfirm},
			Value: VarField{Bool: false, Value: "false"},
		}
		variables[1] = Variable{
			Name:      VarField{Value: "test"},
			Type:      VarField{Value: TypeInput},
			DependsOn: VarField{Value: "confirm", InvertBool: true},
		}
		assert.False(t, skipQuestionOnCondition(&variables[1], variables[1].DependsOn.Value, variables[0].Value.Bool, NewPreparedData(), "", variables[1].DependsOn.InvertBool))
	})
	t.Run("should not skip question (promptIf)", func(t *testing.T) {
		variables := make([]Variable, 2)
		variables[0] = Variable{
			Name:  VarField{Value: "confirm"},
			Type:  VarField{Value: TypeConfirm},
			Value: VarField{Bool: true, Value: "true"},
		}
		variables[1] = Variable{
			Name:      VarField{Value: "test"},
			Type:      VarField{Value: TypeInput},
			DependsOn: VarField{Value: "confirm"},
		}
		assert.False(t, skipQuestionOnCondition(&variables[1], variables[1].DependsOn.Value, variables[0].Value.Bool, NewPreparedData(), "", variables[1].DependsOn.InvertBool))
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

func TestGetValidateExpr(t *testing.T) {
	tests := []struct {
		name     string
		variable *Variable
		wantStr  string
		wantErr  error
	}{
		{
			"should error on empty tag for validate attribute",
			&Variable{Validate: VarField{Value: "test"}},
			"",
			fmt.Errorf("only '!expr' tag is supported for validate attribute"),
		},
		{
			"should error on non-expression tag for validate attribute",
			&Variable{Validate: VarField{Value: "test", Tag: tagFnV1}},
			"",
			fmt.Errorf("only '!expr' tag is supported for validate attribute"),
		},
		{
			"should return empty string for empty expression value with tag value",
			&Variable{Validate: VarField{Value: "", Tag: tagExpressionV2}},
			"",
			nil,
		},
		{
			"should return empty string for empty expression value without tag value",
			&Variable{Validate: VarField{Value: ""}},
			"",
			nil,
		},
		{
			"should return expression string for valid expression tag",
			&Variable{Validate: VarField{Value: "regex('*', TestVar)", Tag: tagExpressionV2}},
			"regex('*', TestVar)",
			nil,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := tt.variable.GetValidateExpr()
			if tt.wantErr == nil || err == nil {
				assert.Equal(t, tt.wantStr, got)
			} else {
				assert.Equal(t, tt.wantErr.Error(), err.Error())
			}
		})
	}
}

func TestValidatePrompt(t *testing.T) {
	type args struct {
		varName      string
		validateExpr string
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
			args{"test", "", "", true, nil},
			nil,
		},
		{
			"should pass on empty space since empty values are allowed in secret fields",
			args{"test", "", " ", true, nil},
			nil,
		},
		{
			"should fail required validation on empty value",
			args{"test", "", "", false, nil},
			fmt.Errorf("Value is required"),
		},
		{
			"should fail required validation on empty space value",
			args{"test", "", " ", false, nil},
			fmt.Errorf("Value is required"),
		},
		{
			"should fail required validation on empty value with pattern",
			args{"test", "regex('.', test)", "", false, make(map[string]interface{})},
			fmt.Errorf("Value is required"),
		},
		{
			"should fail required validation on empty space with pattern",
			args{"test", "regex('.', test)", " ", false, make(map[string]interface{})},
			fmt.Errorf("Value is required"),
		},
		{
			"should pass required validation on valid value",
			args{"test", "", "test", false, nil},
			nil,
		},
		{
			"should pass required validation on empty space with allowEmpty true",
			args{"test", "", "", true, make(map[string]interface{})},
			nil,
		},
		{
			"should fail pattern validation on invalid value",
			args{"test", "regex('[a-z]*', test)", "123", false, make(map[string]interface{})},
			fmt.Errorf("validation [regex('[a-z]*', test)] failed with value [123]"),
		},
		{
			"should pass pattern validation on valid value",
			args{"test", "regex('[a-z]*', test)", "abc", false, make(map[string]interface{})},
			nil,
		},
		{
			"should pass pattern validation on valid value with extra space",
			args{"test", "regex('[a-z]*', test)", "  abc  ", false, make(map[string]interface{})},
			nil,
		},
		{
			"should pass pattern validation on valid value with escape char in pattern",
			args{"test", "regex('(\\\\S)*', test)", "abc", false, make(map[string]interface{})},
			nil,
		},
		{
			"should pass pattern validation on valid value with extra start/end tag on pattern",
			args{"test", "regex('^[a-z]*$', test)", "abc", false, make(map[string]interface{})},
			nil,
		},
		{
			"should pass pattern validation on valid value with fixed pattern",
			args{"test", "regex('test', test)", "test", false, make(map[string]interface{})},
			nil,
		},
		{
			"should fail pattern validation on invalid value with fixed pattern",
			args{"test", "regex('test', test)", "abcd", false, make(map[string]interface{})},
			fmt.Errorf("validation [regex('test', test)] failed with value [abcd]"),
		},
		{
			"should fail pattern validation on valid value with complex pattern",
			args{"test", `regex('\\b(?:(?:2(?:[0-4][0-9]|5[0-5])|[0-1]?[0-9]?[0-9])\.){3}(?:(?:2([0-4][0-9]|5[0-5])|[0-1]?[0-9]?[0-9]))\\b', test)`, "123.123.123.256", false, make(map[string]interface{})},
			fmt.Errorf(`validation [regex('\\b(?:(?:2(?:[0-4][0-9]|5[0-5])|[0-1]?[0-9]?[0-9])\.){3}(?:(?:2([0-4][0-9]|5[0-5])|[0-1]?[0-9]?[0-9]))\\b', test)] failed with value [123.123.123.256]`),
		},
		{
			"should pass pattern validation on valid value with complex pattern",
			args{"test", `regex('\\b(?:(?:2(?:[0-4][0-9]|5[0-5])|[0-1]?[0-9]?[0-9])\.){3}(?:(?:2([0-4][0-9]|5[0-5])|[0-1]?[0-9]?[0-9]))\\b', test)`, "255.255.255.255 ", false, make(map[string]interface{})},
			nil,
		},
		{
			"should fail pattern validation on invalid pattern",
			args{"test", "regex('[[', test)", "abcd", false, make(map[string]interface{})},
			fmt.Errorf("invalid pattern in regex expression, error parsing regexp: unterminated [] set in `^[[$`"),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := validatePrompt(tt.args.varName, tt.args.validateExpr, tt.args.emtpyAllowed, tt.args.params, nil)(tt.args.value)
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
	tmpDir2 := filepath.Join("/tmp", "file-input")
	type args struct {
		value        string
		fileExists   bool
		validateExpr string
		varName      string
		emtpyAllowed bool
		params       map[string]interface{}
	}
	tests := []struct {
		name string
		args args
		want error
	}{
		{"should pass on existing valid file path input", args{filepath.Join(tmpDir, "valid.txt"), true, "", "", false, nil}, nil},
		{"should pass on existing valid file path input with extra space", args{" " + filepath.Join(tmpDir, "valid.txt") + " ", true, "", "", false, nil}, nil},
		{"should fail on non-existing file path input", args{"not-valid.txt", false, "", "", false, nil}, fmt.Errorf("file not found on path not-valid.txt")},
		{"should fail on directory path input", args{tmpDir, false, "", "", false, nil}, fmt.Errorf("given path is a directory, file path is needed")},
		{"should fail on empty input", args{"", false, "", "", false, nil}, fmt.Errorf("Value is required")},
		{
			"should pass on valid file path and validation",
			args{filepath.Join(tmpDir2, "valid.txt"), true, "isValidAbsPath(K8sClientCertFile)", "K8sClientCertFile", false, make(map[string]interface{})},
			nil,
		},
		{
			"should fail on invalid file path by validation",
			args{filepath.Join(tmpDir, "valid.txt"), false, "isValidAbsPath(K8sClientCertFile)", "K8sClientCertFile", false, make(map[string]interface{})},
			fmt.Errorf("validation error for answer value [test/file-input/valid.txt] for variable [K8sClientCertFile]: validation [isValidAbsPath(K8sClientCertFile)] failed with value [test/file-input/valid.txt]"),
		},
		{
			"should fail on invalid file path but pass validation",
			args{filepath.Join("/", tmpDir, "valid.txt"), false, "isValidAbsPath(K8sClientCertFile)", "K8sClientCertFile", false, make(map[string]interface{})},
			fmt.Errorf("file not found on path /test/file-input/valid.txt"),
		},
	}
	for _, tt := range tests {
		// Create needed temporary directory for tests
		os.MkdirAll(tmpDir, os.ModePerm)
		defer os.RemoveAll("test")
		os.MkdirAll(tmpDir2, os.ModePerm)
		defer os.RemoveAll(tmpDir2)

		t.Run(tt.name, func(t *testing.T) {
			if tt.args.fileExists {
				contents := []byte("hello\ngo\n")
				ioutil.WriteFile(tt.args.value, contents, os.ModePerm)
			}

			got := validateFilePath(tt.args.varName, tt.args.validateExpr, tt.args.emtpyAllowed, tt.args.params, nil)(tt.args.value)
			if tt.want == nil || got == nil {
				assert.Equal(t, tt.want, got)
			} else {
				assert.Equal(t, tt.want.Error(), got.Error())
			}
		})
	}
}

func TestGetValuesFromAnswersFile(t *testing.T) {
	// Create needed temporary directory for tests
	os.MkdirAll("test", os.ModePerm)
	defer os.RemoveAll("test")
	validContent := []byte(`
        test: testing
        test2: testing/path
        sample: 5.45
        sample2: 5
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
		wantOut         map[string]string
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
			map[string]string{
				"test":    "testing",
				"test2":   "testing/path",
				"sample":  "5.45",
				"sample2": "5",
				"confirm": "true",
			},
			false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := GetValuesFromAnswersFile(tt.answersFilePath)
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
			Variable{Name: VarField{Value: "Test"}, Type: VarField{Value: TypeInput}},
			"sample answer",
			map[string]interface{}{},
			"sample answer",
			nil,
		},
		{
			"answers from map: save float answer value to variable value with type Input",
			Variable{Name: VarField{Value: "Test"}, Type: VarField{Value: TypeInput}},
			5.45,
			map[string]interface{}{},
			5.45,
			nil,
		},
		{
			"answers from map: save boolean answer value to variable value with type Confirm",
			Variable{Name: VarField{Value: "Test"}, Type: VarField{Value: TypeConfirm}},
			true,
			map[string]interface{}{},
			true,
			nil,
		},
		{
			"answers from map: save boolean answer value (convert from string) to variable value with type Confirm",
			Variable{Name: VarField{Value: "Test"}, Type: VarField{Value: TypeConfirm}},
			"true",
			map[string]interface{}{},
			true,
			nil,
		},
		{
			"answers from map: save long text answer value to variable value with type Editor",
			Variable{Name: VarField{Value: "Test"}, Type: VarField{Value: TypeEditor}},
			"long text for testing..\nand the rest of it\n",
			map[string]interface{}{},
			"long text for testing..\nand the rest of it\n",
			nil,
		},
		{
			"answers from map: save long text answer value to variable value with type File",
			Variable{Name: VarField{Value: "Test"}, Type: VarField{Value: TypeFile}},
			filepath.Join("test", "sample.txt"),
			map[string]interface{}{},
			"hello\ngo\n",
			nil,
		},
		{
			"answers from map: give error on file not found (input type: File)",
			Variable{Name: VarField{Value: "Test"}, Type: VarField{Value: TypeFile}},
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
			Variable{Name: VarField{Value: "Test"}, Type: VarField{Value: TypeSelect}, Options: []VarField{{Value: "a"}, {Value: "b"}}},
			"b",
			map[string]interface{}{},
			"b",
			nil,
		},
		{
			"answers from map: give error on unknown select option value",
			Variable{Name: VarField{Value: "Test"}, Type: VarField{Value: TypeSelect}, Options: []VarField{{Value: "a"}, {Value: "b"}}},
			"c",
			map[string]interface{}{},
			"",
			fmt.Errorf("answer [c] is not one of the available options [a b] for variable [Test]"),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := tt.variable.VerifyVariableValue(tt.answer, tt.parameters, nil)
			assert.Equal(t, tt.errOut, err)
			assert.Equal(t, tt.wantOut, got)
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
		upMode             bool
		overideDefaults    map[string]string
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
						Name:    VarField{Value: "input1"},
						Label:   VarField{Value: "input1"},
						Type:    VarField{Value: "Input"},
						Value:   VarField{Bool: false, Value: ""},
						Default: VarField{Bool: false, Value: "default1"},
					},
					{
						Name:    VarField{Value: "input2"},
						Label:   VarField{Value: "input 2"},
						Type:    VarField{Value: "SecretInput"},
						Value:   VarField{Bool: false, Value: ""},
						Default: VarField{Bool: false, Value: "default2"},
					},
					{
						Name:            VarField{Value: "input3"},
						Label:           VarField{Value: "input 3"},
						Type:            VarField{Value: "SecretInput"},
						Value:           VarField{Bool: false, Value: ""},
						Default:         VarField{Bool: false, Value: "default3"},
						RevealOnSummary: VarField{Bool: true, Value: "true"},
					},
				},
			},
			args{"", false, true, false, nil},
			&PreparedData{
				TemplateData: map[string]interface{}{"input1": "default1", "input2": "!value input2", "input3": "!value input3"},
				SummaryData:  map[string]interface{}{"input1": "default1", "input 2": "*****", "input 3": "default3"},
				Secrets:      map[string]interface{}{"input2": "default2", "input3": "default3"},
				Values:       map[string]interface{}{},
			},
			false,
		},
		{
			"should fail when using incomplete answers file in strict mode",
			BlueprintConfig{
				Variables: []Variable{
					{
						Name:    VarField{Value: "input1"},
						Label:   VarField{Value: "input1"},
						Type:    VarField{Value: "Input"},
						Value:   VarField{Bool: false, Value: "val1"},
						Default: VarField{Bool: false, Value: "default1"},
					},
					{
						Name:    VarField{Value: "input2"},
						Label:   VarField{Value: "input 2"},
						Type:    VarField{Value: "SecretInput"},
						Default: VarField{Bool: false, Value: "default2"},
					},
					{
						Name:  VarField{Value: "input3"},
						Label: VarField{Value: "input 3"},
						Type:  VarField{Value: "Input"},
					},
					{
						Name:  VarField{Value: "input4"},
						Label: VarField{Value: "input 4"},
						Type:  VarField{Value: "Input"},
					},
					{
						Name:  VarField{Value: "input5"},
						Label: VarField{Value: "input 5"},
						Type:  VarField{Value: "Input"},
					},
				},
			},
			args{GetTestTemplateDir("answer-input-2.yaml"), true, false, false, nil},
			nil,
			true,
		},
		{
			"should not fail when using incomplete answers file in strict mode where the variable has ignoreIfSkipped true",
			BlueprintConfig{
				Variables: []Variable{
					{
						Name:    VarField{Value: "input1"},
						Label:   VarField{Value: "input1"},
						Type:    VarField{Value: "Input"},
						Value:   VarField{Bool: false, Value: "val1"},
						Default: VarField{Bool: false, Value: "default1"},
					},
					{
						Name:    VarField{Value: "input2"},
						Label:   VarField{Value: "input 2"},
						Type:    VarField{Value: "SecretInput"},
						Default: VarField{Bool: false, Value: "default2"},
					},
					{
						Name:  VarField{Value: "input3"},
						Label: VarField{Value: "input 3"},
						Type:  VarField{Value: "Input"},
					},
					{
						Name:  VarField{Value: "input4"},
						Label: VarField{Value: "input 4"},
						Type:  VarField{Value: "Input"},
					},
					{
						Name:            VarField{Value: "input5"},
						Label:           VarField{Value: "input 5"},
						Type:            VarField{Value: "Input"},
						IgnoreIfSkipped: VarField{Bool: true},
					},
				},
			},
			args{GetTestTemplateDir("answer-input-2.yaml"), true, false, false, nil},
			&PreparedData{
				TemplateData: map[string]interface{}{"input1": "val1", "input2": "!value input2", "input3": "ans3", "input4": "ans4", "input5": ""},
				SummaryData:  map[string]interface{}{"input1": "val1", "input 2": "*****", "input 3": "ans3", "input 4": "ans4"},
				Secrets:      map[string]interface{}{"input2": "ans2"},
				Values:       map[string]interface{}{},
			},
			false,
		},
		{
			"should use answers file when available",
			BlueprintConfig{
				Variables: []Variable{
					{
						Name:    VarField{Value: "input1"},
						Label:   VarField{Value: "input1"},
						Type:    VarField{Value: "Input"},
						Value:   VarField{Bool: false, Value: "val1"},
						Default: VarField{Bool: false, Value: "default1"},
					},
					{
						Name:    VarField{Value: "input2"},
						Label:   VarField{Value: "input 2"},
						Type:    VarField{Value: "SecretInput"},
						Default: VarField{Bool: false, Value: "default2"},
					},
					{
						Name:  VarField{Value: "input3"},
						Label: VarField{Value: "input 3"},
						Type:  VarField{Value: "Input"},
					},
				},
			},
			args{GetTestTemplateDir("answer-input-2.yaml"), true, false, false, nil},
			&PreparedData{
				TemplateData: map[string]interface{}{"input1": "val1", "input2": "!value input2", "input3": "ans3"},
				SummaryData:  map[string]interface{}{"input1": "val1", "input 2": "*****", "input 3": "ans3"},
				Secrets:      map[string]interface{}{"input2": "ans2"},
				Values:       map[string]interface{}{},
			},
			false,
		},
		{
			"should use answers file when available along with default data",
			BlueprintConfig{
				Variables: []Variable{
					{
						Name:    VarField{Value: "input1"},
						Label:   VarField{Value: "input1"},
						Type:    VarField{Value: "Input"},
						Value:   VarField{Bool: false, Value: "val1"},
						Default: VarField{Bool: false, Value: "default1"},
					},
					{
						Name:    VarField{Value: "input2"},
						Label:   VarField{Value: "input 2"},
						Type:    VarField{Value: "SecretInput"},
						Default: VarField{Bool: false, Value: "default2"},
					},
					{
						Name:  VarField{Value: "input3"},
						Label: VarField{Value: "input 3"},
						Type:  VarField{Value: "Input"},
					},
					{
						Name:    VarField{Value: "input5"},
						Label:   VarField{Value: "input 5"},
						Type:    VarField{Value: "Input"},
						Default: VarField{Bool: false, Value: "default5"},
					},
				},
			},
			args{GetTestTemplateDir("answer-input-2.yaml"), true, true, false, nil},
			&PreparedData{
				TemplateData: map[string]interface{}{"input1": "val1", "input2": "!value input2", "input3": "ans3", "input5": "default5"},
				SummaryData:  map[string]interface{}{"input1": "val1", "input 2": "*****", "input 3": "ans3", "input 5": "default5"},
				Secrets:      map[string]interface{}{"input2": "ans2"},
				Values:       map[string]interface{}{},
			},
			false,
		},
		{
			"should process variables correctly based on primitive or complex data",
			BlueprintConfig{
				Variables: []Variable{
					{
						Name:    VarField{Value: "input1"},
						Label:   VarField{Value: "input1"},
						Value:   VarField{Value: "true", Bool: true},
						Default: VarField{Value: "100"},
					},
					{
						Name:    VarField{Value: "input2"},
						Label:   VarField{Value: "1 < 2 ? 'input 2' : 'input 22'", Tag: tagExpressionV2},
						Type:    VarField{Value: "Input"},
						Value:   VarField{Value: "100"},
						Default: VarField{Value: "true", Bool: true},
					},
					{
						Name:    VarField{Value: "input3"},
						Label:   VarField{Value: "1 < 2 ? 'input 3' : 'input 33'", Tag: tagExpressionV2},
						Type:    VarField{Value: "Input"},
						Value:   VarField{Value: "1 < 2 ? true : false", Tag: tagExpressionV2},
						Default: VarField{Value: "false", Bool: false},
					},
					{
						Name:         VarField{Value: "input4"},
						Label:        VarField{Value: "input4"},
						Value:        VarField{Value: "50.885"},
						Default:      VarField{Value: "100"},
						SaveInXlvals: VarField{Value: "1 < 2", Tag: tagExpressionV2},
					},
					{
						Name:  VarField{Value: "input5"},
						Label: VarField{Value: "input 5"},
						Type:  VarField{Value: "SecretInput"},
						Value: VarField{Value: "50.58"},
					},
					{
						Name:         VarField{Value: "input6"},
						Label:        VarField{Value: "input6"},
						Type:         VarField{Value: "Input"},
						Default:      VarField{Value: "false", Bool: false},
						SaveInXlvals: VarField{Value: "true", Bool: true},
					},
				},
			},
			args{GetTestTemplateDir("answer-input-2.yaml"), true, true, false, nil},
			&PreparedData{
				TemplateData: map[string]interface{}{"input1": "true", "input2": "100", "input3": "true", "input4": "50.885", "input5": "!value input5", "input6": "false"},
				SummaryData:  map[string]interface{}{"input1": "true", "input 2": "100", "input 3": "true", "input4": "50.885", "input 5": "*****", "input6": "false"},
				Secrets:      map[string]interface{}{"input5": "50.58"},
				Values:       map[string]interface{}{"input4": "50.885", "input6": "false"},
			},
			false,
		},
		{
			"should overide defaults when provided but still use answer file",
			BlueprintConfig{
				Variables: []Variable{
					{
						Name:            VarField{Value: "input1"},
						Label:           VarField{Value: "input1"},
						Type:            VarField{Value: "Input"},
						Value:           VarField{Bool: false, Value: "val1"},
						Default:         VarField{Bool: false, Value: "default1"},
						OverrideDefault: VarField{Bool: true, Value: "true"},
					},
					{
						Name:            VarField{Value: "input2"},
						Label:           VarField{Value: "input 2"},
						Type:            VarField{Value: "SecretInput"},
						Default:         VarField{Bool: false, Value: "default2"},
						OverrideDefault: VarField{Bool: true, Value: "true"},
					},
					{
						Name:            VarField{Value: "input3"},
						Label:           VarField{Value: "input 3"},
						Type:            VarField{Value: "Input"},
						OverrideDefault: VarField{Bool: true, Value: "true"},
					},
					{
						Name:    VarField{Value: "input4"},
						Label:   VarField{Value: "input 4"},
						Type:    VarField{Value: "Input"},
						Default: VarField{Bool: false, Value: "default4"},
					},
					{
						Name:            VarField{Value: "input5"},
						Label:           VarField{Value: "input 5"},
						Type:            VarField{Value: "Input"},
						Default:         VarField{Bool: false, Value: "default5"},
						OverrideDefault: VarField{Bool: true, Value: "true"},
					},
				},
			},
			args{GetTestTemplateDir("answer-input-2.yaml"), true, true, false, map[string]string{
				"input1": "overdefault1",
				"input2": "overdefault2",
				"input3": "overdefault3",
				"input4": "overdefault4",
			}},
			&PreparedData{
				TemplateData: map[string]interface{}{"input1": "val1", "input2": "!value input2", "input3": "ans3", "input4": "ans4", "input5": "default5"},
				SummaryData:  map[string]interface{}{"input1": "val1", "input 2": "*****", "input 3": "ans3", "input 4": "ans4", "input 5": "default5"},
				Secrets:      map[string]interface{}{"input2": "ans2"},
				Values:       map[string]interface{}{},
			},
			false,
		},
		{
			"should overide defaults when provided but don't use answer form answer file in XL-UP mode",
			BlueprintConfig{
				Variables: []Variable{
					{
						Name:            VarField{Value: "input1"},
						Label:           VarField{Value: "input1"},
						Type:            VarField{Value: "Input"},
						Value:           VarField{Bool: false, Value: "val1"},
						Default:         VarField{Bool: false, Value: "default1"},
						OverrideDefault: VarField{Bool: true, Value: "true"},
					},
					{
						Name:            VarField{Value: "input2"},
						Label:           VarField{Value: "input 2"},
						Type:            VarField{Value: "SecretInput"},
						Default:         VarField{Bool: false, Value: "default2"},
						OverrideDefault: VarField{Bool: true, Value: "true"},
					},
					{
						Name:            VarField{Value: "input3"},
						Label:           VarField{Value: "input 3"},
						Type:            VarField{Value: "Input"},
						OverrideDefault: VarField{Bool: true, Value: "true"},
					},
					{
						Name:    VarField{Value: "input4"},
						Label:   VarField{Value: "input 4"},
						Type:    VarField{Value: "Input"},
						Default: VarField{Bool: false, Value: "default4"},
					},
					{
						Name:            VarField{Value: "input5"},
						Label:           VarField{Value: "input 5"},
						Type:            VarField{Value: "Input"},
						Default:         VarField{Bool: false, Value: "default5"},
						OverrideDefault: VarField{Bool: true, Value: "true"},
					},
					{
						Name:    VarField{Value: "input6"},
						Label:   VarField{Value: "input 6"},
						Type:    VarField{Value: "Input"},
						Default: VarField{Bool: false, Value: "default6"},
					},
				},
			},
			args{GetTestTemplateDir("answer-input-2.yaml"), true, true, true, map[string]string{
				"input1": "overdefault1",
				"input2": "overdefault2",
				"input3": "overdefault3",
				"input4": "overdefault4",
			}},
			&PreparedData{
				TemplateData: map[string]interface{}{"input1": "val1", "input2": "!value input2", "input3": "overdefault3", "input4": "ans4", "input5": "default5", "input6": "default6"},
				SummaryData:  map[string]interface{}{"input1": "val1", "input 2": "*****", "input 3": "overdefault3", "input 4": "ans4", "input 5": "default5", "input 6": "default6"},
				Secrets:      map[string]interface{}{"input2": "overdefault2"},
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
			got, err := blueprintDoc.prepareTemplateData(
				BlueprintParams{
					AnswersFile:        tt.args.answersFilePath,
					StrictAnswers:      tt.args.strictAnswers,
					UseDefaultsAsValue: tt.args.useDefaultsAsValue,
					OverrideDefaults:   tt.args.overideDefaults,
					FromUpCommand:      tt.args.upMode,
				},
				NewPreparedData(),
				nil,
			)
			if (err != nil) != tt.wantErr {
				t.Errorf("BlueprintYaml.prepareTemplateData() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			assert.Equal(t, tt.want, got)
		})
	}
}

func Test_findLabelValueFromOptions(t *testing.T) {
	tests := []struct {
		name    string
		val     string
		options []VarField
		want    string
	}{
		{
			"find correct values from different option types for string val",
			"yoyo",
			[]VarField{
				VarField{Value: "yoyo"},
				VarField{Value: "hiya", Label: "Hooya"},
				VarField{Value: "someFun()", Tag: "!expr"},
			},
			"yoyo",
		},
		{
			"find correct values from different option types for map val",
			"hiya [Hooya]",
			[]VarField{
				VarField{Value: "yoyo"},
				VarField{Value: "hiya", Label: "Hooya"},
				VarField{Value: "someFun()", Tag: "!expr"},
			},
			"hiya",
		},
		{
			"return given value for val from !expr",
			"foo",
			[]VarField{
				VarField{Value: "yoyo"},
				VarField{Value: "hiya", Label: "Hooya"},
				VarField{Value: "someFun()", Tag: "!expr"},
			},
			"foo",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := findLabelValueFromOptions(tt.val, tt.options); got != tt.want {
				t.Errorf("findLabelValueFromOptions() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_getDefaultTextWithLabel(t *testing.T) {
	tests := []struct {
		name         string
		defVal       string
		options      []string
		fieldOptions []VarField
		want         string
	}{
		{
			"should return default value with label",
			"hiya",
			[]string{
				"yoyo [Yoyo]",
				"hiya [Hiya]",
				"someFun()",
			},
			[]VarField{
				{
					Label:      "Yoyo",
					Value:      "yoyo",
					Bool:       false,
					Tag:        "",
					InvertBool: false,
				},
				{
					Label:      "Hiya",
					Value:      "hiya",
					Bool:       false,
					Tag:        "",
					InvertBool: false,
				},
				{
					Label:      "",
					Value:      "someFun()",
					Bool:       false,
					Tag:        "!expr",
					InvertBool: false,
				},
			},
			"hiya [Hiya]",
		},
		{
			"should return default value without label",
			"hiya",
			[]string{
				"yoyo",
				"hiya",
				"someFun()",
			},
			[]VarField{
				{
					Label:      "",
					Value:      "yoyo",
					Bool:       false,
					Tag:        "",
					InvertBool: false,
				},
				{
					Label:      "",
					Value:      "hiya",
					Bool:       false,
					Tag:        "",
					InvertBool: false,
				},
				{
					Label:      "",
					Value:      "someFun()",
					Bool:       false,
					Tag:        "!expr",
					InvertBool: false,
				},
			},
			"hiya",
		},
		{
			"should return default value without label(options evaluated from func & default is one of them)",
			"hiya",
			[]string{
				"yoyo",
				"hiya",
			},
			[]VarField{
				{
					Label:      "",
					Value:      "someFun()",
					Bool:       false,
					Tag:        "!expr",
					InvertBool: false,
				},
			},
			"hiya",
		},
		{
			"should return default value given when not found in options",
			"yaya",
			[]string{
				"yoyo",
				"hiya",
				"someFun()",
			},
			[]VarField{
				{
					Label:      "Yoyo",
					Value:      "yoyo",
					Bool:       false,
					Tag:        "",
					InvertBool: false,
				},
				{
					Label:      "Hiya",
					Value:      "hiya",
					Bool:       false,
					Tag:        "",
					InvertBool: false,
				},
				{
					Label:      "someFun()",
					Value:      "someFun()",
					Bool:       false,
					Tag:        "!expr",
					InvertBool: false,
				},
			},
			"yaya",
		},
		{
			"should return the first option as default if no default value given",
			"",
			[]string{
				"yoyo [Yoyo]",
				"hiya [Hiya]",
				"someResult1",
				"someResult2",
			},
			[]VarField{
				{
					Label:      "Yoyo",
					Value:      "yoyo",
					Bool:       false,
					Tag:        "",
					InvertBool: false,
				},
				{
					Label:      "Hiya",
					Value:      "hiya",
					Bool:       false,
					Tag:        "",
					InvertBool: false,
				},
				{
					Label:      "someFun()",
					Value:      "someFun()",
					Bool:       false,
					Tag:        "!expr",
					InvertBool: false,
				},
			},
			"yoyo [Yoyo]",
		},
		{
			"should return empty default value if no default value given and no options present",
			"",
			[]string{},
			[]VarField{
				{
					Label:      "",
					Value:      "someFun()",
					Bool:       false,
					Tag:        "!expr",
					InvertBool: false,
				},
			},
			"",
		},
		{
			"should return default value if no options present",
			"hiya",
			[]string{},
			[]VarField{
				{
					Label:      "",
					Value:      "someFun()",
					Bool:       false,
					Tag:        "!expr",
					InvertBool: false,
				},
			},
			"hiya",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := getDefaultTextWithLabel(tt.defVal, tt.fieldOptions, tt.options); got != tt.want {
				t.Errorf("getDefaultTextWithLabel() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_getOptionTextWithLabel(t *testing.T) {
	tests := []struct {
		name   string
		option VarField
		want   string
	}{
		{
			"should return value with label",
			VarField{Value: "hiya", Label: "Hooya"},
			"hiya [Hooya]",
		},
		{
			"should return default value without label",
			VarField{Value: "yoyo"},
			"yoyo",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := getOptionTextWithLabel(tt.option); got != tt.want {
				t.Errorf("getOptionTextWithLabel() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_saveItemToTemplateDataMap(t *testing.T) {
	type args struct {
		variable     *Variable
		preparedData *PreparedData
		data         interface{}
	}
	tests := []struct {
		name      string
		args      args
		exprected PreparedData
	}{
		{
			"should save secrets as *** in PreparedData",
			args{
				&Variable{
					Name:  VarField{Value: "Test"},
					Label: VarField{Value: "Test"},
					Type:  VarField{Value: "SecretInput"},
				},
				&PreparedData{
					TemplateData: map[string]interface{}{"input1": "val1"},
					SummaryData:  map[string]interface{}{"input1": "val1"},
					Secrets:      map[string]interface{}{},
					Values:       map[string]interface{}{},
				},
				"foo",
			},
			PreparedData{
				TemplateData: map[string]interface{}{"input1": "val1", "Test": "!value Test"},
				SummaryData:  map[string]interface{}{"input1": "val1", "Test": "*****"},
				Secrets:      map[string]interface{}{"Test": "foo"},
				Values:       map[string]interface{}{},
			},
		},
		{
			"should save secrets as as it is in PreparedData when RevealOnSummary & ReplaceAsIs is true",
			args{
				&Variable{
					Name:            VarField{Value: "Test"},
					Label:           VarField{Value: "Test"},
					RevealOnSummary: VarField{Bool: true},
					ReplaceAsIs:     VarField{Bool: true},
					Type:            VarField{Value: "SecretInput"},
				},
				&PreparedData{
					TemplateData: map[string]interface{}{"input1": "val1"},
					SummaryData:  map[string]interface{}{"input1": "val1"},
					Secrets:      map[string]interface{}{},
					Values:       map[string]interface{}{},
				},
				"foo",
			},
			PreparedData{
				TemplateData: map[string]interface{}{"input1": "val1", "Test": "foo"},
				SummaryData:  map[string]interface{}{"input1": "val1", "Test": "foo"},
				Secrets:      map[string]interface{}{"Test": "foo"},
				Values:       map[string]interface{}{},
			},
		},
		{
			"should not skip secret in SummaryData & Secrets when IgnoreIfSkipped is true & PromptIf is true",
			args{
				&Variable{
					Name:            VarField{Value: "Test"},
					Label:           VarField{Value: "Test"},
					IgnoreIfSkipped: VarField{Bool: true},
					Type:            VarField{Value: "SecretInput"},
					Meta: VariableMeta{
						PromptSkipped: false,
					},
				},
				&PreparedData{
					TemplateData: map[string]interface{}{"input1": "val1"},
					SummaryData:  map[string]interface{}{"input1": "val1"},
					Secrets:      map[string]interface{}{},
					Values:       map[string]interface{}{},
				},
				"foo",
			},
			PreparedData{
				TemplateData: map[string]interface{}{"input1": "val1", "Test": "!value Test"},
				SummaryData:  map[string]interface{}{"input1": "val1", "Test": "*****"},
				Secrets:      map[string]interface{}{"Test": "foo"},
				Values:       map[string]interface{}{},
			},
		},
		{
			"should skip secret in SummaryData & Secrets when IgnoreIfSkipped is true & PromptIf is false",
			args{
				&Variable{
					Name:            VarField{Value: "Test"},
					Label:           VarField{Value: "Test"},
					IgnoreIfSkipped: VarField{Bool: true},
					Type:            VarField{Value: "SecretInput"},
					Meta: VariableMeta{
						PromptSkipped: true,
					},
				},
				&PreparedData{
					TemplateData: map[string]interface{}{"input1": "val1"},
					SummaryData:  map[string]interface{}{"input1": "val1"},
					Secrets:      map[string]interface{}{},
					Values:       map[string]interface{}{},
				},
				"foo",
			},
			PreparedData{
				TemplateData: map[string]interface{}{"input1": "val1", "Test": "!value Test"},
				SummaryData:  map[string]interface{}{"input1": "val1"},
				Secrets:      map[string]interface{}{},
				Values:       map[string]interface{}{},
			},
		},
		{
			"should skip secret in SummaryData & Secrets when IgnoreIfSkipped is true & value is nil",
			args{
				&Variable{
					Name:            VarField{Value: "Test"},
					Label:           VarField{Value: "Test"},
					IgnoreIfSkipped: VarField{Bool: true},
					Type:            VarField{Value: "SecretInput"},
				},
				&PreparedData{
					TemplateData: map[string]interface{}{"input1": "val1"},
					SummaryData:  map[string]interface{}{"input1": "val1"},
					Secrets:      map[string]interface{}{},
					Values:       map[string]interface{}{},
				},
				"",
			},
			PreparedData{
				TemplateData: map[string]interface{}{"input1": "val1", "Test": "!value Test"},
				SummaryData:  map[string]interface{}{"input1": "val1"},
				Secrets:      map[string]interface{}{},
				Values:       map[string]interface{}{},
			},
		},
		// non secret variables
		{
			"should save variable in PreparedData",
			args{
				&Variable{
					Name:  VarField{Value: "Test"},
					Label: VarField{Value: "Test"},
					Type:  VarField{Value: "Input"},
				},
				&PreparedData{
					TemplateData: map[string]interface{}{"input1": "val1"},
					SummaryData:  map[string]interface{}{"input1": "val1"},
					Secrets:      map[string]interface{}{},
					Values:       map[string]interface{}{},
				},
				"foo",
			},
			PreparedData{
				TemplateData: map[string]interface{}{"input1": "val1", "Test": "foo"},
				SummaryData:  map[string]interface{}{"input1": "val1", "Test": "foo"},
				Secrets:      map[string]interface{}{},
				Values:       map[string]interface{}{},
			},
		},
		{
			"should save variable in PreparedData when SaveInXlvals is true",
			args{
				&Variable{
					Name:         VarField{Value: "Test"},
					Label:        VarField{Value: "Test"},
					Type:         VarField{Value: "Input"},
					SaveInXlvals: VarField{Bool: true},
				},
				&PreparedData{
					TemplateData: map[string]interface{}{"input1": "val1"},
					SummaryData:  map[string]interface{}{"input1": "val1"},
					Secrets:      map[string]interface{}{},
					Values:       map[string]interface{}{},
				},
				"foo",
			},
			PreparedData{
				TemplateData: map[string]interface{}{"input1": "val1", "Test": "foo"},
				SummaryData:  map[string]interface{}{"input1": "val1", "Test": "foo"},
				Secrets:      map[string]interface{}{},
				Values:       map[string]interface{}{"Test": "foo"},
			},
		},
		{
			"should save variable in PreparedData with correct type for Confirm types",
			args{
				&Variable{
					Name:         VarField{Value: "Test"},
					Label:        VarField{Value: "Test"},
					Type:         VarField{Value: "Confirm"},
					Default:      VarField{Value: "true", Bool: true},
					SaveInXlvals: VarField{Bool: true},
				},
				&PreparedData{
					TemplateData: map[string]interface{}{"input1": "val1"},
					SummaryData:  map[string]interface{}{"input1": "val1"},
					Secrets:      map[string]interface{}{},
					Values:       map[string]interface{}{},
				},
				"true",
			},
			PreparedData{
				TemplateData: map[string]interface{}{"input1": "val1", "Test": true},
				SummaryData:  map[string]interface{}{"input1": "val1", "Test": true},
				Secrets:      map[string]interface{}{},
				Values:       map[string]interface{}{"Test": true},
			},
		},
		{
			"should save variable in PreparedData with correct type for Confirm types when data is nil",
			args{
				&Variable{
					Name:         VarField{Value: "Test"},
					Label:        VarField{Value: "Test"},
					Type:         VarField{Value: "Confirm"},
					Default:      VarField{Value: "true", Bool: true},
					SaveInXlvals: VarField{Bool: true},
				},
				&PreparedData{
					TemplateData: map[string]interface{}{"input1": "val1"},
					SummaryData:  map[string]interface{}{"input1": "val1"},
					Secrets:      map[string]interface{}{},
					Values:       map[string]interface{}{},
				},
				nil,
			},
			PreparedData{
				TemplateData: map[string]interface{}{"input1": "val1", "Test": false},
				SummaryData:  map[string]interface{}{"input1": "val1", "Test": false},
				Secrets:      map[string]interface{}{},
				Values:       map[string]interface{}{"Test": false},
			},
		},
		{
			"should not skip variable in SummaryData & Values when IgnoreIfSkipped is true & PromptIf is true",
			args{
				&Variable{
					Name:            VarField{Value: "Test"},
					Label:           VarField{Value: "Test"},
					IgnoreIfSkipped: VarField{Bool: true},
					SaveInXlvals:    VarField{Bool: true},
					Type:            VarField{Value: "Input"},
					Meta: VariableMeta{
						PromptSkipped: false,
					},
				},
				&PreparedData{
					TemplateData: map[string]interface{}{"input1": "val1"},
					SummaryData:  map[string]interface{}{"input1": "val1"},
					Secrets:      map[string]interface{}{},
					Values:       map[string]interface{}{},
				},
				"foo",
			},
			PreparedData{
				TemplateData: map[string]interface{}{"input1": "val1", "Test": "foo"},
				SummaryData:  map[string]interface{}{"input1": "val1", "Test": "foo"},
				Secrets:      map[string]interface{}{},
				Values:       map[string]interface{}{"Test": "foo"},
			},
		},
		{
			"should skip secret in SummaryData & Secrets when IgnoreIfSkipped is true & PromptIf is false",
			args{
				&Variable{
					Name:            VarField{Value: "Test"},
					Label:           VarField{Value: "Test"},
					IgnoreIfSkipped: VarField{Bool: true},
					SaveInXlvals:    VarField{Bool: true},
					Type:            VarField{Value: "Input"},
					Meta: VariableMeta{
						PromptSkipped: true,
					},
				},
				&PreparedData{
					TemplateData: map[string]interface{}{"input1": "val1"},
					SummaryData:  map[string]interface{}{"input1": "val1"},
					Secrets:      map[string]interface{}{},
					Values:       map[string]interface{}{},
				},
				"foo",
			},
			PreparedData{
				TemplateData: map[string]interface{}{"input1": "val1", "Test": "foo"},
				SummaryData:  map[string]interface{}{"input1": "val1"},
				Secrets:      map[string]interface{}{},
				Values:       map[string]interface{}{},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			saveItemToTemplateDataMap(tt.args.variable, tt.args.preparedData, tt.args.data)
			assert.Equal(t, tt.exprected, *tt.args.preparedData)
		})
	}
}

func Test_shouldAskForInput(t *testing.T) {
	tests := []struct {
		name     string
		variable Variable
		want     bool
	}{
		{
			"should return false when SkipUserInput is set true",
			func() Variable {
				SkipUserInput = true
				return Variable{}
			}(),
			false,
		},
		{
			"should return false when prompt is empty",
			Variable{
				IgnoreIfSkipped: VarField{Bool: true},
			},
			false,
		},
		{
			"should return false when prompt is present and value is empty",
			Variable{
				IgnoreIfSkipped: VarField{Bool: true},
				Prompt:          VarField{Value: ""},
			},
			false,
		},
		{
			"should return true when prompt value is present",
			Variable{
				IgnoreIfSkipped: VarField{Bool: true},
				Prompt:          VarField{Value: "foo"},
			},
			true,
		},
		{
			"should return true when IgnoreIfSkipped is false",
			Variable{
				IgnoreIfSkipped: VarField{Bool: false},
			},
			true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := shouldAskForInput(tt.variable); got != tt.want {
				t.Errorf("shouldAskForInput() = %v, want %v", got, tt.want)
			}
			SkipUserInput = false // reset the field
		})
	}
}

func TestVariable_ProcessExpression(t *testing.T) {
	tests := []struct {
		name       string
		variable   Variable
		parameters map[string]interface{}
		want       Variable
		wantErr    bool
	}{
		{
			"should throw error when expression is invalid",
			Variable{
				Name:  VarField{Value: "Test"},
				Label: VarField{Value: "True ? 'A' : 'B'", Tag: tagExpressionV2},
			},
			map[string]interface{}{},
			Variable{
				Name:  VarField{Value: "Test"},
				Label: VarField{Value: "A", Tag: tagExpressionV2},
			},
			true,
		},
		{
			"should process expressions in variable fields",
			Variable{
				Name:        VarField{Value: "Test"},
				Label:       VarField{Value: "true ? 'A' : 'B'", Tag: tagExpressionV2},
				Description: VarField{Value: "Foo > 5 ? 5 : 'B'", Tag: tagExpressionV2},
				Value:       VarField{Value: "Foo > 5 ? 5.8 : 'B'", Tag: tagExpressionV2},
			},
			map[string]interface{}{
				"Foo": 10,
			},
			Variable{
				Name:        VarField{Value: "Test"},
				Label:       VarField{Value: "A", Tag: tagExpressionV2},
				Description: VarField{Value: "5.0", Tag: tagExpressionV2},
				Value:       VarField{Value: "5.8", Tag: tagExpressionV2},
			},
			false,
		},
		{
			"should process expressions in variable fields skipping validate & options",
			Variable{
				Name:      VarField{Value: "Test"},
				Label:     VarField{Value: "true ? 'A' : 'B'", Tag: tagExpressionV2},
				DependsOn: VarField{Value: "Foo > 5", Tag: tagExpressionV2},
				Validate:  VarField{Value: "Foo > 5", Tag: tagExpressionV2},
				Options: []VarField{
					{Value: "true ? 'A' : 'B'", Tag: tagExpressionV2},
					{Value: "Foo > 5 ? 'B' : 'C'", Tag: tagExpressionV2},
				},
			},
			map[string]interface{}{
				"Foo": 10,
			},
			Variable{
				Name:      VarField{Value: "Test"},
				Label:     VarField{Value: "A", Tag: tagExpressionV2},
				DependsOn: VarField{Value: "true", Bool: true, Tag: tagExpressionV2},
				Validate:  VarField{Value: "Foo > 5", Tag: tagExpressionV2},
				Options: []VarField{
					{Value: "true ? 'A' : 'B'", Tag: tagExpressionV2},
					{Value: "Foo > 5 ? 'B' : 'C'", Tag: tagExpressionV2},
				},
			},
			false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.variable.ProcessExpression(tt.parameters, nil)
			if tt.wantErr {
				assert.NotNil(t, err)
			} else {
				assert.Equal(t, tt.want, tt.variable)
			}
		})
	}
}

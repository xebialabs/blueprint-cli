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

var dummyData = make(map[string]interface{})

func TestGetVariableDefaultVal(t *testing.T) {
	t.Run("should return empty string when default is not defined", func(t *testing.T) {
		v := Variable{
			Name: VarField{Value: "test"},
			Type: VarField{Value: TypeInput},
		}
		defaultVal := v.GetDefaultVal(dummyData)
		assert.Equal(t, "", defaultVal)
	})

	t.Run("should return default value string when default is defined", func(t *testing.T) {
		v := Variable{
			Name:    VarField{Value: "test"},
			Type:    VarField{Value: TypeInput},
			Default: VarField{Value: "default_val"},
		}
		defaultVal := v.GetDefaultVal(dummyData)
		assert.Equal(t, "default_val", defaultVal)
	})

	t.Run("should return empty string when invalid function tag in default field", func(t *testing.T) {
		v := Variable{
			Name:    VarField{Value: "test"},
			Type:    VarField{Value: TypeInput},
			Default: VarField{Value: "aws.regs", Tag: tagFn},
		}
		defaultVal := v.GetDefaultVal(dummyData)
		assert.Equal(t, "", defaultVal)
	})

	t.Run("should return function output on valid function tag in default field", func(t *testing.T) {
		v := Variable{
			Name:    VarField{Value: "test"},
			Type:    VarField{Value: TypeInput},
			Default: VarField{Value: "aws.regions(ecs)[0]", Tag: tagFn},
		}
		defaultVal := v.GetDefaultVal(dummyData)
		regionsList, _ := aws.GetAvailableAWSRegionsForService("ecs")
		sort.Strings(regionsList)
		assert.Equal(t, regionsList[0], defaultVal)
	})

	t.Run("should return empty string when invalid expression tag in default field", func(t *testing.T) {
		v := Variable{
			Name:    VarField{Value: "test"},
			Type:    VarField{Value: TypeInput},
			Default: VarField{Value: "aws.regs", Tag: tagExpressionV2},
		}
		defaultVal := v.GetDefaultVal(dummyData)
		assert.Equal(t, "", defaultVal)
	})

	t.Run("should return output on valid expression tag in default field", func(t *testing.T) {
		v := Variable{
			Name:    VarField{Value: "test"},
			Type:    VarField{Value: TypeInput},
			Default: VarField{Value: "'foo' + 'bar'", Tag: tagExpressionV2},
		}
		defaultVal := v.GetDefaultVal(dummyData)
		assert.Equal(t, "foobar", defaultVal)
		v = Variable{
			Name:    VarField{Value: "test"},
			Type:    VarField{Value: TypeInput},
			Default: VarField{Value: "Foo > 10", Tag: tagExpressionV2},
		}
		defaultVal = v.GetDefaultVal(map[string]interface{}{
			"Foo": 100,
		})
		assert.True(t, defaultVal.(bool))
	})
}

func TestGetValueFieldVal(t *testing.T) {
	t.Run("should return value field string value when defined", func(t *testing.T) {
		v := Variable{
			Name:  VarField{Value: "test"},
			Type:  VarField{Value: TypeInput},
			Value: VarField{Value: "testing"},
		}
		val := v.GetValueFieldVal(dummyData)
		assert.Equal(t, "testing", val)
	})

	t.Run("should return empty on invalid function tag in value field", func(t *testing.T) {
		v := Variable{
			Name:  VarField{Value: "test"},
			Type:  VarField{Value: TypeInput},
			Value: VarField{Value: "aws.regs", Tag: tagFn},
		}
		val := v.GetValueFieldVal(dummyData)
		assert.Equal(t, "", val)
	})

	t.Run("should return function output on valid function tag in value field", func(t *testing.T) {
		v := Variable{
			Name:  VarField{Value: "test"},
			Type:  VarField{Value: TypeInput},
			Value: VarField{Value: "aws.regions(ecs)[0]", Tag: tagFn},
		}
		val := v.GetValueFieldVal(dummyData)
		regionsList, _ := aws.GetAvailableAWSRegionsForService("ecs")
		sort.Strings(regionsList)
		assert.Equal(t, regionsList[0], val)
	})

	t.Run("should return empty on invalid expression tag in value field", func(t *testing.T) {
		v := Variable{
			Name:  VarField{Value: "test"},
			Type:  VarField{Value: TypeInput},
			Value: VarField{Value: "aws.regs()", Tag: tagExpressionV2},
		}
		val := v.GetValueFieldVal(dummyData)
		assert.Equal(t, "", val)
	})

	t.Run("should return expression output on valid expression tag in value field", func(t *testing.T) {
		v := Variable{
			Name:  VarField{Value: "test"},
			Type:  VarField{Value: TypeInput},
			Value: VarField{Value: "'foo' + 'bar'", Tag: tagExpressionV2},
		}
		defaultVal := v.GetValueFieldVal(dummyData)
		assert.Equal(t, "foobar", defaultVal)
		v = Variable{
			Name:  VarField{Value: "test"},
			Type:  VarField{Value: TypeInput},
			Value: VarField{Value: "Foo > 10", Tag: tagExpressionV2},
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
			Name:    VarField{Value: "test"},
			Type:    VarField{Value: TypeSelect},
			Options: []VarField{{Value: "a"}, {Value: "b"}, {Value: "c"}},
		}
		values := v.GetOptions(dummyData)
		assert.Len(t, values, 3)
		assert.Equal(t, []string{"a", "b", "c"}, values)
	})

	t.Run("should return string values of map options", func(t *testing.T) {
		v := Variable{
			Name:    VarField{Value: "test"},
			Type:    VarField{Value: TypeSelect},
			Options: []VarField{{Value: "aVal"}, {Label: "bLabel", Value: "bVal"}, {Label: "cLabel", Value: "cVal"}},
		}
		values := v.GetOptions(dummyData)
		assert.Len(t, values, 3)
		assert.Equal(t, []string{"aVal", "bLabel (bVal)", "cLabel (cVal)"}, values)
	})

	t.Run("should return generated values for fn options tag", func(t *testing.T) {
		v := Variable{
			Name:    VarField{Value: "test"},
			Type:    VarField{Value: TypeSelect},
			Options: []VarField{{Value: "aws.regions(ecs)", Tag: tagFn}},
		}
		values := v.GetOptions(dummyData)
		assert.True(t, len(values) > 1)
	})

	t.Run("should return nil on invalid function tag for options", func(t *testing.T) {
		v := Variable{
			Name:    VarField{Value: "test"},
			Type:    VarField{Value: TypeSelect},
			Options: []VarField{{Value: "aws.regs", Tag: tagFn}},
		}
		out := v.GetOptions(dummyData)
		require.Nil(t, out)
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
		})
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
		})
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
		})
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
		})
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
		})
		assert.NotNil(t, values)
		assert.True(t, len(values) == 2)
	})

	t.Run("should return nil values for invalid return type in expression options tag", func(t *testing.T) {
		v := Variable{
			Name:    VarField{Value: "test"},
			Type:    VarField{Value: TypeSelect},
			Options: []VarField{{Value: "Foo ? Bar : (Fooo, Foo)", Tag: tagExpressionV2}},
		}
		values := v.GetOptions(map[string]interface{}{
			"Foo": false,
			"Bar": []string{"test", "foo"},
		})
		assert.Nil(t, values)
	})

	t.Run("should return nil on invalid expression tag for options", func(t *testing.T) {
		v := Variable{
			Name:    VarField{Value: "test"},
			Type:    VarField{Value: TypeSelect},
			Options: []VarField{{Value: "aws.regs()", Tag: tagExpressionV2}},
		}
		out := v.GetOptions(dummyData)
		require.Nil(t, out)
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
			&Variable{Validate: VarField{Value: "test", Tag: tagFn}},
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
			"should fail required validation on empty value",
			args{"test", "", "", false, nil},
			fmt.Errorf("Value is required"),
		},
		{
			"should fail required validation on empty value with pattern",
			args{"test", "regexMatch('.', test)", "", false, make(map[string]interface{})},
			fmt.Errorf("Value is required"),
		},
		{
			"should pass required validation on valid value",
			args{"test", "", "test", false, nil},
			nil,
		},
		{
			"should fail pattern validation on invalid value",
			args{"test", "regexMatch('[a-z]*', test)", "123", false, make(map[string]interface{})},
			fmt.Errorf("validation [regexMatch('[a-z]*', test)] failed with value [123]"),
		},
		{
			"should pass pattern validation on valid value",
			args{"test", "regexMatch('[a-z]*', test)", "abc", false, make(map[string]interface{})},
			nil,
		},
		{
			"should pass pattern validation on valid value with extra start/end tag on pattern",
			args{"test", "regexMatch('^[a-z]*$', test)", "abc", false, make(map[string]interface{})},
			nil,
		},
		{
			"should pass pattern validation on valid value with fixed pattern",
			args{"test", "regexMatch('test', test)", "test", false, make(map[string]interface{})},
			nil,
		},
		{
			"should fail pattern validation on invalid value with fixed pattern",
			args{"test", "regexMatch('test', test)", "abcd", false, make(map[string]interface{})},
			fmt.Errorf("validation [regexMatch('test', test)] failed with value [abcd]"),
		},
		{
			"should fail pattern validation on valid value with complex pattern",
			args{"test", `regexMatch('\\b(?:(?:2(?:[0-4][0-9]|5[0-5])|[0-1]?[0-9]?[0-9])\.){3}(?:(?:2([0-4][0-9]|5[0-5])|[0-1]?[0-9]?[0-9]))\\b', test)`, "123.123.123.256", false, make(map[string]interface{})},
			fmt.Errorf(`validation [regexMatch('\\b(?:(?:2(?:[0-4][0-9]|5[0-5])|[0-1]?[0-9]?[0-9])\.){3}(?:(?:2([0-4][0-9]|5[0-5])|[0-1]?[0-9]?[0-9]))\\b', test)] failed with value [123.123.123.256]`),
		},
		{
			"should pass pattern validation on valid value with complex pattern",
			args{"test", `regexMatch('\\b(?:(?:2(?:[0-4][0-9]|5[0-5])|[0-1]?[0-9]?[0-9])\.){3}(?:(?:2([0-4][0-9]|5[0-5])|[0-1]?[0-9]?[0-9]))\\b', test)`, "255.255.255.255", false, make(map[string]interface{})},
			nil,
		},
		{
			"should fail pattern validation on invalid pattern",
			args{"test", "regexMatch('[[', test)", "abcd", false, make(map[string]interface{})},
			fmt.Errorf("error parsing regexp: missing closing ]: `[[$`"),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := validatePrompt(tt.args.varName, tt.args.validateExpr, tt.args.emtpyAllowed, tt.args.params)(tt.args.value)
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
			map[string]string{"test": "testing", "sample": "5.45", "confirm": "true"},
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
			got, err := tt.variable.VerifyVariableValue(tt.answer, tt.parameters)
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
			args{"", false, true},
			&PreparedData{
				TemplateData: map[string]interface{}{"input1": "default1", "input2": "!value input2", "input3": "!value input3"},
				DefaultData:  map[string]interface{}{"input1": "default1", "input 2": "*****", "input 3": "default3"},
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

package blueprint

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/Knetic/govaluate"
	"github.com/stretchr/testify/assert"
)

func Test_processCustomExpression(t *testing.T) {
	// initialize temp dir for tests
	tmpDir, err := ioutil.TempDir("", "xltest")
	if err != nil {
		t.Error(err)
	}
	defer os.RemoveAll(tmpDir)

	// create test yaml file
	testFilePath := filepath.Join(tmpDir, "test.yaml")
	originalConfigBytes := []byte("testing")
	ioutil.WriteFile(testFilePath, originalConfigBytes, 0755)

	// create test k8s config file
	d1 := []byte(SampleKubeConfig)
	ioutil.WriteFile(filepath.Join(tmpDir, "config"), d1, os.ModePerm)
	os.Setenv("KUBECONFIG", filepath.Join(tmpDir, "config"))

	type args struct {
		exStr       string
		parameters  map[string]interface{}
		overrideFns ExpressionOverrideFn
	}
	tests := []struct {
		name       string
		onlyInUnix bool
		args       args
		want       interface{}
		wantFn     func(got interface{}) bool
		wantErr    bool
	}{
		{
			"should fail when using undefined parameter",
			false,
			args{
				"FooUndefined > 100",
				map[string]interface{}{
					"foo": "foo",
				},
				nil,
			},
			nil,
			nil,
			true,
		},
		{
			"should return a float64 value of the number as a string",
			false,
			args{
				"Foo",
				map[string]interface{}{
					"Foo": "100",
				},
				nil,
			},
			float64(100),
			nil,
			false,
		},
		{
			"should return true when parameter is evaluated",
			false,
			args{
				"Foo > 10",
				map[string]interface{}{
					"Foo": "100",
				},
				nil,
			},
			true,
			nil,
			false,
		},
		{
			"should return false when parameter is evaluated",
			false,
			args{
				"Foo && Bar",
				map[string]interface{}{
					"Foo": true,
					"Bar": false,
				},
				nil,
			},
			false,
			nil,
			false,
		},
		{
			"should return false when expression is evaluated",
			false,
			args{
				"Foo < 10",
				map[string]interface{}{
					"Foo": 100,
				},
				nil,
			},
			false,
			nil,
			false,
		},
		{
			"should return Bar as a float64 when ternary expression is evaluated",
			false,
			args{
				"Foo > 10 ? Foo : Bar",
				map[string]interface{}{
					"Foo": "10",
					"Bar": "200",
				},
				nil,
			},
			float64(200),
			nil,
			false,
		},
		{
			"should return Bar as a string when ternary expression is evaluated",
			false,
			args{
				"string(Foo > 10 ? Foo : Bar)",
				map[string]interface{}{
					"Foo": "10",
					"Bar": "200",
				},
				nil,
			},
			"200",
			nil,
			false,
		},
		{
			"should return an array when ternary expression is evaluated",
			false,
			args{
				"Foo ? Bar : (1, 2, 3)",
				map[string]interface{}{
					"Foo": true,
					"Bar": []string{"test", "foo"},
				},
				nil,
			},
			[]string{"test", "foo"},
			nil,
			false,
		},
		{
			"should return true when logical expression is evaluated",
			false,
			args{
				"Foo == 10 && Bar != 10",
				map[string]interface{}{
					"Foo": "10",
					"Bar": "200",
				},
				nil,
			},
			true,
			nil,
			false,
		},
		{
			"should return '100' when expression is evaluated",
			false,
			args{
				"Foo + Bar",
				map[string]interface{}{
					"Foo": "75",
					"Bar": 25,
				},
				nil,
			},
			float64(100),
			nil,
			false,
		},
		{
			"should return 'foo+bar' when expression is evaluated",
			false,
			args{
				"Foo + '+' + Bar",
				map[string]interface{}{
					"Foo": "foo",
					"Bar": "bar",
				},
				nil,
			},
			"foo+bar",
			nil,
			false,
		},
		{
			"should return length of Foo when expression is evaluated",
			false,
			args{
				"strlen(Foo)",
				map[string]interface{}{
					"Foo": "foo0",
				},
				nil,
			},
			float64(4),
			nil,
			false,
		},
		{
			"should return max of 2 variables when expression is evaluated",
			false,
			args{
				"max(arg1, arg2)",
				map[string]interface{}{
					"arg1": "2",
					"arg2": 1,
				},
				nil,
			},
			float64(2),
			nil,
			false,
		},
		{
			"should return length of a number as if it's a string when expression is evaluated",
			false,
			args{
				"strlen(string(arg))",
				map[string]interface{}{
					"arg": "1234",
				},
				nil,
			},
			float64(4),
			nil,
			false,
		},
		{
			"should return a number with two leading zeroes when expression is evaluated",
			false,
			args{
				"strlen(string(arg)) == 1 ? '00' + arg : (strlen(string(arg)) == 2 ? '0' + arg : arg)",
				map[string]interface{}{
					"arg": "9",
				},
				nil,
			},
			"009",
			nil,
			false,
		},
		{
			"should return a number with one leading zero when expression is evaluated",
			false,
			args{
				"strlen(string(arg)) == 1 ? '00' + arg : (strlen(string(arg)) == 2 ? '0' + arg : arg)",
				map[string]interface{}{
					"arg": "90",
				},
				nil,
			},
			"090",
			nil,
			false,
		},
		{
			"should return a number without a leading zero when expression is evaluated",
			false,
			args{
				"strlen(string(arg)) == 1 ? '00' + arg : (strlen(string(arg)) == 2 ? '0' + arg : arg)",
				map[string]interface{}{
					"arg": "100",
				},
				nil,
			},
			float64(100),
			nil,
			false,
		},
		{
			"should return max of 2 variables when expression is evaluated",
			false,
			args{
				"max(arg1, arg2)",
				map[string]interface{}{
					"arg1": "2",
					"arg2": "1",
				},
				nil,
			},
			float64(2),
			nil,
			false,
		},
		{
			"should return rounded value of a number when expression is evaluated",
			false,
			args{
				"round(arg)",
				map[string]interface{}{
					"arg": "2.12556",
				},
				nil,
			},
			float64(2),
			nil,
			false,
		},
		{
			"should error on invalid number of args for regex expression",
			false,
			args{
				"regex('[a-zA-Z-]*')",
				map[string]interface{}{},
				nil,
			},
			nil,
			nil,
			true,
		},
		{
			"should error on invalid pattern for regex",
			false,
			args{
				"regex('[a-zA-Z-*', TestVar)",
				map[string]interface{}{
					"TestVar": "SomeName",
				},
				nil,
			},
			false,
			nil,
			true,
		},
		{
			"should return success regex match for own valid value",
			false,
			args{
				"regex('[a-zA-Z-]*', TestVar)",
				map[string]interface{}{
					"TestVar": "SomeName",
				},
				nil,
			},
			true,
			nil,
			false,
		},
		{
			"should return fail regex match for own invalid value",
			false,
			args{
				"regex('[a-zA-Z-]*', TestVar)",
				map[string]interface{}{
					"TestVar": "SomeName123",
				},
				nil,
			},
			false,
			nil,
			false,
		},
		{
			"should return success regex match for own valid value with PCRE compatible regex",
			false,
			args{
				"regex('([a-c])x\\\\1x\\\\1', TestVar)",
				map[string]interface{}{
					"TestVar": "axaxa",
				},
				nil,
			},
			true,
			nil,
			false,
		},
		{
			"should return success regex match for own valid value with PCRE lookbehind regex",
			false,
			args{
				"regex('\\\\b\\\\w+(?<!s)\\\\b', TestVar)",
				map[string]interface{}{
					"TestVar": "john",
				},
				nil,
			},
			true,
			nil,
			false,
		},
		{
			"should return true for a pattern with escape char",
			false,
			args{
				"regex('(\\\\S){16,}', TestVar)",
				map[string]interface{}{
					"TestVar": "1234567890123456",
				},
				nil,
			},
			true,
			nil,
			false,
		},
		{
			"should use both own value and other parameter value in expression function",
			false,
			args{
				"min(TestVar1, TestVar2)",
				map[string]interface{}{
					"TestVar1": 123,
					"TestVar2": 100,
				},
				nil,
			},
			100.0,
			nil,
			false,
		},
		{
			"should return true when a complex logical expression is evaluated",
			false,
			args{
				"((Foo == 10 && Bar != 10) ? Bar: Foo) == 200 && (Fooz == 'test' || 'test' == Fooz) && (Fooz + Foo == 'test10') && Foo != 20",
				map[string]interface{}{
					"Foo":  "10",
					"Bar":  200,
					"Fooz": "test",
				},
				nil,
			},
			true,
			nil,
			false,
		},
		{
			"should return 3 when a complex math expression is evaluated",
			false,
			args{
				"ceil(min(Foo / Bar * Fooz, Foo * 0.5 ) * round(2.8956))",
				map[string]interface{}{
					"Foo":  "100",
					"Bar":  200,
					"Fooz": "1.88888",
				},
				nil,
			},
			float64(3),
			nil,
			false,
		},
		{
			"should return a random password when expression is evaluated",
			false,
			args{
				"strlen(randPassword())",
				map[string]interface{}{},
				nil,
			},
			float64(16),
			nil,
			false,
		},
		{
			"should return true when a valid file is tested with isFile",
			false,
			args{
				"isFile(Test)",
				map[string]interface{}{
					"Test": testFilePath,
				},
				nil,
			},
			true,
			nil,
			false,
		},
		{
			"should return false when a non-existing file is tested with isFile",
			false,
			args{
				"isFile(Test)",
				map[string]interface{}{
					"Test": filepath.Join(tmpDir, "not-valid.txt"),
				},
				nil,
			},
			false,
			nil,
			false,
		},
		{
			"should return true when a valid directory is tested with isDir",
			false,
			args{
				"isDir(Test)",
				map[string]interface{}{
					"Test": tmpDir,
				},
				nil,
			},
			true,
			nil,
			false,
		},
		{
			"should return true when user home directory is tested with isDir",
			true,
			args{
				"isDir(Test)",
				map[string]interface{}{
					"Test": "~/",
				},
				nil,
			},
			true,
			nil,
			false,
		},
		{
			"should return false when a non-existing directory is tested with isDir",
			false,
			args{
				"isDir(Test)",
				map[string]interface{}{
					"Test": filepath.Join("/", "not", "exists"),
				},
				nil,
			},
			false,
			nil,
			false,
		},
		{
			"should return false when an invalid URL is tested with isValidUrl",
			false,
			args{
				"isValidUrl(Test)",
				map[string]interface{}{
					"Test": "http//xebialabs.com",
				},
				nil,
			},
			false,
			nil,
			false,
		},
		{
			"should return true when a valid URL is tested with isValidUrl",
			false,
			args{
				"isValidUrl(Test)",
				map[string]interface{}{
					"Test": "http://xebialabs.com",
				},
				nil,
			},
			true,
			nil,
			false,
		},
		{
			"should return true when a valid unix path is tested with isValidAbsPath",
			false,
			args{
				"isValidAbsPath(Test)",
				map[string]interface{}{
					"Test": "/path/to/my folder/valid/unix-file",
				},
				nil,
			},
			true,
			nil,
			false,
		},
		{
			"should return true when a valid windows path is tested with isValidAbsPath",
			false,
			args{
				"isValidAbsPath(Test)",
				map[string]interface{}{
					"Test": "D:\\my folder\file.crt",
				},
				nil,
			},
			true,
			nil,
			false,
		},
		{
			"should return false when a valid windows path with space is tested with isValidAbsPath",
			false,
			args{
				"isValidAbsPath(Test, 'true')",
				map[string]interface{}{
					"Test": "D:\\my folder\file.crt",
				},
				nil,
			},
			false,
			nil,
			false,
		},
		{
			"should return false when a valid unix path with space is tested with isValidAbsPath",
			false,
			args{
				"isValidAbsPath(Test, 'true')",
				map[string]interface{}{
					"Test": "/path/to/my folder/valid/unix-file",
				},
				nil,
			},
			false,
			nil,
			false,
		},
		{
			"should return false when a invalid valid unix is tested with isValidAbsPath",
			false,
			args{
				"isValidAbsPath(Test)",
				map[string]interface{}{
					"Test": "../path/to/my-folder/valid/unix-file",
				},
				nil,
			},
			false,
			nil,
			false,
		},
		{
			"should return false when a invalid windows path is tested with isValidAbsPath",
			false,
			args{
				"isValidAbsPath(Test)",
				map[string]interface{}{
					"Test": "D://my folder/file.crt",
				},
				nil,
			},
			false,
			nil,
			false,
		},

		// aws helper functions
		{
			"should error when an invalid awsCredentials expression is requested",
			false,
			args{
				"awsCredentials()",
				map[string]interface{}{},
				nil,
			},
			nil,
			nil,
			true,
		},
		{
			"should error when an invalid attribute is sent for awsCredentials expression",
			false,
			args{
				"awsCredentials('unknown')",
				map[string]interface{}{},
				nil,
			},
			nil,
			nil,
			true,
		},
		{
			"should return boolean result for proper awsCredentials expression",
			false,
			args{
				"regex('^(true|false|1|0)$', awsCredentials('IsAvailable'))",
				map[string]interface{}{},
				nil,
			},
			true,
			nil,
			false,
		},
		{
			"should return string result for proper awsCredentials expression",
			false,
			args{
				"regex('^(true|false|1|0)$', awsCredentials('AccessKeyID'))",
				map[string]interface{}{},
				nil,
			},
			nil,
			func(result interface{}) bool {
				return result != ""
			},
			false,
		},
		{
			"should error when no aws service specified for awsRegions expression",
			false,
			args{
				"awsRegions()",
				map[string]interface{}{},
				nil,
			},
			nil,
			nil,
			true,
		},
		{
			"should return list of ECS regions for awsRegions expression",
			false,
			args{
				"awsRegions('ecs')",
				map[string]interface{}{},
				nil,
			},
			nil,
			func(result interface{}) bool {
				switch result.(type) {
				case []string:
					if len(result.([]string)) > 0 {
						return true
					}
				}

				return false
			},
			false,
		},
		{
			"should return first ECS region for awsRegions expression",
			false,
			args{
				"regex('[a-zA-Z0-9-]+', awsRegions('ecs', 0))",
				map[string]interface{}{},
				nil,
			},
			true,
			nil,
			false,
		},
		{
			"should error on invalid index for ECS regions list for awsRegions expression",
			false,
			args{
				"awsRegions('ecs', 10000)",
				map[string]interface{}{},
				nil,
			},
			nil,
			nil,
			true,
		},

		// k8s helper functions
		{
			"should error on invalid k8sConfig expression call",
			false,
			args{
				"k8sConfig()",
				map[string]interface{}{},
				nil,
			},
			nil,
			nil,
			true,
		},
		{
			"should error on unknown context for k8sConfig expression",
			false,
			args{
				"k8sConfig('ClusterServer', 'unknown')",
				map[string]interface{}{},
				nil,
			},
			nil,
			nil,
			true,
		},
		{
			"should return IsConfigAvailable true for valid config file",
			false,
			args{
				"k8sConfig('IsConfigAvailable')",
				map[string]interface{}{},
				nil,
			},
			true,
			nil,
			false,
		},
		{
			"should return isAvailable true for k8sConfig expression with default context",
			false,
			args{
				"k8sConfig('IsAvailable')",
				map[string]interface{}{},
				nil,
			},
			true,
			nil,
			false,
		},
		{
			"should return isAvailable false for k8sConfig expression with unknown context",
			false,
			args{
				"k8sConfig('IsAvailable', 'unknown')",
				map[string]interface{}{},
				nil,
			},
			false,
			nil,
			false,
		},
		{
			"should return cluster server url for k8sConfig expression",
			false,
			args{
				"k8sConfig('ClusterServer')",
				map[string]interface{}{},
				nil,
			},
			"https://test.hcp.eastus.azmk8s.io:443",
			nil,
			false,
		},

		// os helper functions
		{
			"should error on empty module for os expression",
			false,
			args{
				"os()",
				map[string]interface{}{},
				nil,
			},
			nil,
			nil,
			true,
		},
		{
			"should error on unknown module for os expression",
			false,
			args{
				"os('unknown')",
				map[string]interface{}{},
				nil,
			},
			nil,
			nil,
			true,
		},
		{
			"should return os name for valid os expression",
			false,
			args{
				"os('_operatingsystem') != ''",
				map[string]interface{}{},
				nil,
			},
			true,
			nil,
			false,
		},
		{
			"should return the same path if it's a UNIX-style path",
			false,
			args{
				"normalizePath('/home/test/some/path')",
				map[string]interface{}{},
				nil,
			},
			"/home/test/some/path",
			nil,
			false,
		},
		{
			"should return a normalised path if it's a Windows-style path",
			false,
			args{
				`normalizePath('C:\\Users\\Someone\\place')`,
				map[string]interface{}{},
				nil,
			},
			"/C/Users/Someone/place",
			nil,
			false,
		},
		{
			"should return an error if we provide no arguments",
			false,
			args{
				`normalizePath()`,
				map[string]interface{}{},
				nil,
			},
			nil,
			nil,
			true,
		},
		{
			"should return an error if we provide more than one argument",
			false,
			args{
				`normalizePath('/some/place', 'something')`,
				map[string]interface{}{},
				nil,
			},
			nil,
			nil,
			true,
		},
		{
			"should use functions defined in overrideFns instead of default",
			false,
			args{
				"strlen(Foo)",
				map[string]interface{}{
					"Foo": "foo0",
				},
				func(params map[string]interface{}) map[string]govaluate.ExpressionFunction {
					return map[string]govaluate.ExpressionFunction{
						"strlen": func(args ...interface{}) (interface{}, error) {
							length := len(args[0].(string)) + 1
							return (float64)(length), nil
						},
					}
				},
			},
			float64(5),
			nil,
			false,
		},
		{
			"should use functions defined in overrideFns",
			false,
			args{
				"strlenNew(Foo)",
				map[string]interface{}{
					"Foo": "foo0",
				},
				func(params map[string]interface{}) map[string]govaluate.ExpressionFunction {
					return map[string]govaluate.ExpressionFunction{
						"strlenNew": func(args ...interface{}) (interface{}, error) {
							length := len(args[0].(string)) - 1
							return (float64)(length), nil
						},
					}
				},
			},
			float64(3),
			nil,
			false,
		},
	}
	for _, tt := range tests {
		if tt.onlyInUnix && runtime.GOOS == "windows" {
			continue
		}
		t.Run(tt.name, func(t *testing.T) {
			got, err := ProcessCustomExpression(tt.args.exStr, tt.args.parameters, tt.args.overrideFns)
			if (err != nil) != tt.wantErr {
				t.Errorf("processCustomExpression() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.wantFn != nil {
				assert.True(t, tt.wantFn(got))
			} else {
				assert.Equal(t, tt.want, got)
			}
		})
	}
}

func Test_processCustomExpression_no_k8s(t *testing.T) {
	// initialize temp dir for tests
	tmpDir, err := ioutil.TempDir("", "xltest")
	if err != nil {
		t.Error(err)
	}
	defer os.RemoveAll(tmpDir)

	os.Setenv("KUBECONFIG", filepath.Join(tmpDir, "config"))

	type args struct {
		exStr       string
		parameters  map[string]interface{}
		overrideFns ExpressionOverrideFn
	}
	tests := []struct {
		name       string
		onlyInUnix bool
		args       args
		want       interface{}
		wantFn     func(got interface{}) bool
		wantErr    bool
	}{
		// k8s helper functions
		{
			"should return IsConfigAvailable false for invalid config file",
			false,
			args{
				"k8sConfig('IsConfigAvailable')",
				map[string]interface{}{},
				nil,
			},
			false,
			nil,
			false,
		},
		{
			"should return isAvailable true for k8sConfig expression with default context",
			false,
			args{
				"k8sConfig('IsAvailable')",
				map[string]interface{}{},
				nil,
			},
			false,
			nil,
			false,
		},
		{
			"should return cluster server url for k8sConfig expression",
			false,
			args{
				"k8sConfig('ClusterServer')",
				map[string]interface{}{},
				nil,
			},
			"",
			nil,
			false,
		},
	}
	for _, tt := range tests {
		if tt.onlyInUnix && runtime.GOOS == "windows" {
			continue
		}
		t.Run(tt.name, func(t *testing.T) {
			got, err := ProcessCustomExpression(tt.args.exStr, tt.args.parameters, tt.args.overrideFns)
			if (err != nil) != tt.wantErr {
				t.Errorf("processCustomExpression() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.wantFn != nil {
				assert.True(t, tt.wantFn(got))
			} else {
				assert.Equal(t, tt.want, got)
			}
		})
	}
}

func Test_fixValueTypes(t *testing.T) {
	tests := []struct {
		name       string
		parameters map[string]interface{}
		want       map[string]interface{}
	}{
		{
			"should convert to float",
			map[string]interface{}{
				"int":    "2",
				"float":  "2.5",
				"float2": "2.5548454545844",
				"float3": "098500",
			},
			map[string]interface{}{
				"int":    float64(2),
				"float":  float64(2.5),
				"float2": float64(2.5548454545844),
				"float3": float64(98500),
			},
		},
		{
			"should convert to bool",
			map[string]interface{}{
				"true":   "true",
				"false":  "false",
				"true1":  "True",
				"false1": "False",
			},
			map[string]interface{}{
				"true":   true,
				"false":  false,
				"true1":  true,
				"false1": false,
			},
		},
		{
			"should convert mixed map",
			map[string]interface{}{
				"float":  "2.5548454545844",
				"int":    "098500",
				"bool":   "true",
				"string": "hello",
				"float2": float64(2.5548454545844),
				"bool2":  true,
			},
			map[string]interface{}{
				"float":  float64(2.5548454545844),
				"int":    float64(98500),
				"bool":   true,
				"string": "hello",
				"float2": float64(2.5548454545844),
				"bool2":  true,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := FixValueTypes(tt.parameters)
			assert.Equal(t, tt.want, got)
		})
	}
}

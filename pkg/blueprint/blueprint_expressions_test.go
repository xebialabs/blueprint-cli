package blueprint

import (
    "io/ioutil"
    "os"
    "path/filepath"
    "runtime"
    "testing"

	"github.com/stretchr/testify/assert"
)

func Test_processCustomExpression(t *testing.T) {
    tmpDir, err := ioutil.TempDir("", "xltest")
    if err != nil {
        t.Error(err)
    }
    defer os.RemoveAll(tmpDir)
    testFilePath := filepath.Join(tmpDir, "test.yaml")
    originalConfigBytes := []byte("testing")
    ioutil.WriteFile(testFilePath, originalConfigBytes, 0755)

	type args struct {
		exStr      string
		parameters map[string]interface{}
		currentKey string
		currentVal interface{}
	}
	tests := []struct {
		name       string
		onlyInUnix bool
		args       args
		want       interface{}
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
                "",
                nil,
			},
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
                "",
                nil,
			},
			float64(100),
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
                "",
                nil,
			},
			true,
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
                "",
                nil,
			},
			false,
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
                "",
                nil,
			},
			false,
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
                "",
                nil,
			},
			float64(200),
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
                "",
                nil,
			},
			"200",
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
                "",
                nil,
			},
			[]string{"test", "foo"},
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
                "",
                nil,
			},
			true,
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
                "",
                nil,
			},
			float64(100),
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
                "",
                nil,
			},
			"foo+bar",
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
                "",
                nil,
			},
			float64(4),
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
                "",
                nil,
			},
			float64(2),
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
                "",
                nil,
			},
			float64(4),
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
                "",
                nil,
			},
			"009",
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
                "",
                nil,
			},
			"090",
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
                "",
                nil,
			},
			float64(100),
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
                "",
                nil,
			},
			float64(2),
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
                "",
                nil,
			},
			float64(2),
			false,
		},
        {
            "should error on invalid number of args for regex expression",
            false,
            args{
                "regexMatch('[a-zA-Z-]*')",
                map[string]interface{}{},
                "",
                nil,
            },
            nil,
            true,
        },
        {
            "should return success regex match for own valid value",
            false,
            args{
                "regexMatch('[a-zA-Z-]*', TestVar)",
                map[string]interface{}{},
                "TestVar",
                "SomeName",
            },
            true,
            false,
        },
        {
            "should return fail regex match for own invalid value",
            false,
            args{
                "regexMatch('[a-zA-Z-]*', TestVar)",
                map[string]interface{}{},
                "TestVar",
                "SomeName123",
            },
            false,
            false,
        },
        {
            "should use both own value and other parameter value in expression function",
            false,
            args{
                "min(TestVar1, TestVar2)",
                map[string]interface{}{"TestVar1": 123},
                "TestVar2",
                100,
            },
            100.0,
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
                "",
                nil,
			},
			true,
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
                "",
                nil,
			},
			float64(3),
			false,
		},
		{
			"should return a random password when expression is evaluated",
            false,
			args{
				"strlen(randPassword())",
				map[string]interface{}{},
                "",
                nil,
			},
			float64(16),
			false,
		},
        {
            "should return true when a valid file is tested with isFile",
            false,
            args{
                "isFile(Test)",
                map[string]interface{}{},
                "Test",
                testFilePath,
            },
            true,
            false,
        },
        {
            "should return false when a non-existing file is tested with isFile",
            false,
            args{
                "isFile(Test)",
                map[string]interface{}{},
                "Test",
                filepath.Join(tmpDir, "not-valid.txt"),
            },
            false,
            false,
        },
        {
            "should return true when a valid directory is tested with isDir",
            false,
            args{
                "isDir(Test)",
                map[string]interface{}{},
                "Test",
                tmpDir,
            },
            true,
            false,
        },
        {
            "should return true when user home directory is tested with isDir",
            true,
            args{
                "isDir(Test)",
                map[string]interface{}{},
                "Test",
                "~/",
            },
            true,
            false,
        },
        {
            "should return false when a non-existing directory is tested with isDir",
            false,
            args{
                "isDir(Test)",
                map[string]interface{}{},
                "Test",
                filepath.Join("/", "not", "exists"),
            },
            false,
            false,
        },
        {
            "should return false when an invalid URL is tested with isValidUrl",
            false,
            args{
                "isValidUrl(Test)",
                map[string]interface{}{},
                "Test",
                "http//xebialabs.com",
            },
            false,
            false,
        },
        {
            "should return true when a valid URL is tested with isValidUrl",
            false,
            args{
                "isValidUrl(Test)",
                map[string]interface{}{},
                "Test",
                "http://xebialabs.com",
            },
            true,
            false,
        },
	}
	for _, tt := range tests {
	    if tt.onlyInUnix && runtime.GOOS == "windows" {
	        continue
        }
        t.Run(tt.name, func(t *testing.T) {
            got, err := ProcessCustomExpression(tt.args.exStr, tt.args.parameters, tt.args.currentKey, tt.args.currentVal)
            if (err != nil) != tt.wantErr {
                t.Errorf("processCustomExpression() error = %v, wantErr %v", err, tt.wantErr)
                return
            }
            assert.Equal(t, tt.want, got)
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
			got := fixValueTypes(tt.parameters)
			assert.Equal(t, tt.want, got)
		})
	}
}

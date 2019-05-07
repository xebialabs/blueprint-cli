package util

import (
	"fmt"
	"io/ioutil"
	"os"
    "os/user"
    "path"
    "path/filepath"
    "runtime"
    "testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestUtil(t *testing.T) {
	t.Run("ValidateFilePath should prevent specifying absolute paths", func(t *testing.T) {
		absolute := path.Join(string(os.PathSeparator), "folder", "provision.yaml")
		assert.EqualError(t, ValidateFilePath(absolute, "test"), fmt.Sprintf("absolute path is not allowed in test: %s\n", absolute))
	})

	t.Run("ValidateFilePath happy flow", func(t *testing.T) {
		assert.Nil(t, ValidateFilePath("file.yaml", "test"))
		assert.Nil(t, ValidateFilePath(path.Join("folder", "provision.yaml"), "test"))
	})
}

func TestPathExists(t *testing.T) {
	tmpDir, err := ioutil.TempDir("", "pathTest")
	require.Nil(t, err)
	defer os.RemoveAll(tmpDir)
	os.MkdirAll(path.Join(tmpDir, "permitted"), os.ModePerm)
	d1 := []byte("hello\ngo\n")
	ioutil.WriteFile(path.Join(tmpDir, "test.yaml"), d1, os.ModePerm)
	t.Run("should result in true for an existing file", func(t *testing.T) {
		assert.True(t, PathExists(path.Join(tmpDir, "test.yaml"), false))
	})
	t.Run("should result in true for an existing folder", func(t *testing.T) {
		assert.True(t, PathExists(path.Join(tmpDir, "permitted"), false))
		assert.True(t, PathExists(path.Join(tmpDir, "permitted"), true))
	})
	t.Run("should result in false for an existing file when mustBeFolder is true", func(t *testing.T) {
		assert.False(t, PathExists(path.Join(tmpDir, "test.yaml"), true))
	})
	t.Run("should result in false for an existing file when there is no permission", func(t *testing.T) {
		os.MkdirAll(path.Join(tmpDir, "nopermission"), os.ModePerm)
		ioutil.WriteFile(path.Join(tmpDir, "nopermission", "test.yaml"), d1, 0000)
		os.Chmod(path.Join(tmpDir, "nopermission"), 0000)
		assert.True(t, PathExists(path.Join(tmpDir, "nopermission"), false))
		assert.False(t, PathExists(path.Join(tmpDir, "nopermission", "test.yaml"), false))
	})
}

func TestExpandHomeDirIfNeeded(t *testing.T) {
    // not to be tested on windows
    if runtime.GOOS != "windows" {
        currentUser, _ := user.Current()
        tests := []struct {
            name     string
            testPath string
            expected string
        }{
            {
                "should expand home path when given ~",
                "~",
                currentUser.HomeDir,
            },
            {
                "should expand home path when given ~/",
                "~",
                currentUser.HomeDir,
            },
            {
                "should expand home path when given relative path to ~",
                "~/some/dir",
                filepath.Join(currentUser.HomeDir, "some/dir"),
            },
            {
                "should not expand home path when given a path including ~ in between",
                "/tmp/~/some/dir",
                "/tmp/~/some/dir",
            },
            {
                "should return original path when a full path is given",
                "/tmp/path/to/some/local/dir/",
                "/tmp/path/to/some/local/dir/",
            },
            {
                "should return original path when a root path is given",
                "/",
                "/",
            },
        }
        for _, tt := range tests {
            t.Run(tt.name, func(t *testing.T) {
                got := ExpandHomeDirIfNeeded(tt.testPath, currentUser)
                assert.Equal(t, tt.expected, got)
            })
        }
    }
}

func TestMapContainsKeyWithVal(t *testing.T) {
	type args struct {
		dict map[string]string
		key  string
	}
	testMap := map[string]string{
		"foo": "foo",
		"bat": "bar",
		"bar": "",
	}
	tests := []struct {
		name string
		args args
		want bool
	}{
		{
			"should return false when map doesn't have key",
			args{
				testMap,
				"foooo",
			},
			false,
		},
		{
			"should return false when map doesn't have value for key",
			args{
				testMap,
				"bar",
			},
			false,
		},
		{
			"should return true when map has value for key",
			args{
				testMap,
				"foo",
			},
			true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := MapContainsKeyWithVal(tt.args.dict, tt.args.key); got != tt.want {
				t.Errorf("MapContainsKeyWithVal() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestMapContainsKeyWithValInterface(t *testing.T) {
	type args struct {
		dict map[string]interface{}
		key  string
	}
	testMap := map[string]interface{}{
		"foo":  "foo",
		"bat":  true,
		"bar":  5.6,
		"baz":  "",
		"baz2": nil,
	}
	tests := []struct {
		name string
		args args
		want bool
	}{
		{
			"should return false when map doesn't have key",
			args{
				testMap,
				"foooo",
			},
			false,
		},
		{
			"should return false when map have empty value for key",
			args{
				testMap,
				"baz",
			},
			false,
		},
		{
			"should return false when map have nil value for key",
			args{
				testMap,
				"baz2",
			},
			false,
		},
		{
			"should return true when map has float value for key",
			args{
				testMap,
				"bar",
			},
			true,
		},
		{
			"should return true when map has string value for key",
			args{
				testMap,
				"foo",
			},
			true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := MapContainsKeyWithValInterface(tt.args.dict, tt.args.key); got != tt.want {
				t.Errorf("MapContainsKeyWithVal() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestSortMapStringInterface(t *testing.T) {
	tests := []struct {
		name string
		m    map[string]interface{}
		want map[string]interface{}
	}{
		{
			"should sort the provided map",
			map[string]interface{}{
				"foo": "hello",
				"bar": map[string]interface{}{
					"foo": "hello",
					"xoo": map[string]interface{}{
						"foo": "hello",
						"bar": "bar",
					},
					"bar": "bar",
				},
				"aa": 1,
			},
			map[string]interface{}{
				"aa": 1,
				"bar": map[string]interface{}{
					"bar": "bar",
					"foo": "hello",
					"xoo": map[string]interface{}{
						"bar": "bar",
						"foo": "hello",
					},
				},
				"foo": "hello",
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := SortMapStringInterface(tt.m)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestDiffBetweenStringSlices(t *testing.T) {
	tests := []struct {
		name string
		s1   []string
		s2   []string
		want []string
	}{
		{
			"should get empty difference between two empty slices",
			[]string{},
			[]string{},
			[]string{},
		},
		{
			"should get difference between slice1 and slice2",
			[]string{"a", "b", "c", "d", "f"},
			[]string{"b", "c", "d", "e"},
			[]string{"a", "f"},
		},
		{
			"should get difference between slice1 and slice2 when second one is empty",
			[]string{"a", "b", "c", "d", "f"},
			[]string{},
			[]string{"a", "b", "c", "d", "f"},
		},
		{
			"should get difference between slice1 and slice2 when first one is empty",
			[]string{},
			[]string{"a", "b", "c", "d", "f"},
			[]string{},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := DiffBetweenStringSlices(tt.s1, tt.s2)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestExtractStringKeysFromMap(t *testing.T) {
	tests := []struct {
		name string
		m    map[string]interface{}
		want []string
	}{
		{
			"should extract empty slice from the empty map",
			map[string]interface{}{},
			[]string{},
		},
		{
			"should extract string keys from the provided map",
			map[string]interface{}{
				"foo": "hello",
				"bar": map[string]interface{}{
					"foo": "hello",
					"xoo": map[string]interface{}{
						"foo": "hello",
						"bar": "bar",
					},
					"bar": "bar",
				},
				"aa": 1,
			},
			[]string{"foo", "bar", "aa"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ExtractStringKeysFromMap(tt.m)
			assert.Empty(t, DiffBetweenStringSlices(tt.want, got))
		})
	}
}

func TestParseVersion(t *testing.T) {
	t.Run("should parse valid version into a number", func(t *testing.T) {
		actual := ParseVersion("1.0.0", 4)
		expected := int64(100000000)

		assert.Equal(t, actual, expected)

		actual = ParseVersion("8.5.1", 4)
		expected = int64(800050001)

		assert.Equal(t, actual, expected)

		actual = ParseVersion("9.9.1000", 4)
		expected = int64(900091000)

		assert.Equal(t, actual, expected)

		actual = ParseVersion("8.6.0", 4)
		expected = int64(800060000)

		assert.Equal(t, actual, expected)

		actual = ParseVersion("9999.9999.9999", 4)
		expected = int64(999999999999)

		assert.Equal(t, actual, expected)
	})
}

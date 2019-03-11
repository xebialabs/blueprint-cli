package util

import (
	"fmt"
	"io/ioutil"
	"os"
	"path"
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

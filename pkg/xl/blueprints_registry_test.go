package xl

import (
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func mockMakeHTTPCallForTemplateIndex(indexURL string, registry TemplateRegistry) ([]byte, int, error) {
	if indexURL == "http://test.registry/blueprint/index.json" {
		json := `
		  [
			"aws/test2",
			"aws/test"
		  ]
		  `
		bytes := []byte(json)
		return bytes, 200, nil
	} else if indexURL == "http://test.registry/blueprint2/index.json" {
		json := `
		  [
			"aws/test",
			"azure/test"
		  ]
		  `
		bytes := []byte(json)
		return bytes, 200, nil
	} else if indexURL == "http://test.registry/blueprint3/index.json" {
		json := `
			["gcp/test"]
		  `
		bytes := []byte(json)
		return bytes, 200, nil
	}
	return nil, 400, nil
}

func mockMakeHTTPCallForTemplatePathIndex(indexURL string, registry TemplateRegistry) ([]byte, int, error) {
	if indexURL == "http://test.registry/blueprint/aws/test/index.json" {
		json := `
		  [
			"test.yaml",
			"test.yml.tmpl"
		  ]
		  `
		bytes := []byte(json)
		return bytes, 200, nil
	}
	return nil, 400, nil
}

func mockMakeHTTPCallForTemplateFile(t *testing.T, expectedurl, output string) MakeHTTPCallForTemplateFn {
	return func(urlPath string, registry TemplateRegistry) ([]byte, int, error) {
		if expectedurl == "" {
			return nil, 0, fmt.Errorf("error")
		}
		require.NotNil(t, urlPath)
		assert.Equal(t, expectedurl, urlPath)
		bytes := []byte(output)
		return bytes, 200, nil
	}
}

func TestGetTemplateTypes(t *testing.T) {
	t.Run("should get sorted keys from a map", func(t *testing.T) {
		registry := TemplateRegistry{Name: "default", URL: parseURIWithoutError("http://test.registry/blueprint/")}

		out := getTemplateTypes(map[string]TemplateRegistry{
			"aws/test2":  registry,
			"aws/test":   registry,
			"azure/test": registry,
		})
		require.NotNil(t, out)
		assert.Equal(t, []string{
			"aws/test", "aws/test2", "azure/test",
		}, out)
	})
}

func TestMakeFullURLPath(t *testing.T) {
	t.Run("should modify the templates with full path", func(t *testing.T) {
		registry := TemplateRegistry{Name: "default", URL: parseURIWithoutError("http://test.registry/blueprint/")}
		out := makeFullURLPath([]string{
			"test1.yml", "test2.yaml",
		}, "aws/test", registry)
		require.NotNil(t, out)
		assert.Equal(t, []TemplateConfig{
			TemplateConfig{File: "test1.yml", FullPath: "http://test.registry/blueprint/aws/test/test1.yml", Registry: registry},
			TemplateConfig{File: "test2.yaml", FullPath: "http://test.registry/blueprint/aws/test/test2.yaml", Registry: registry},
		}, out)
	})
}
func TestGetTemplateConfigs(t *testing.T) {
	t.Run("should modify the templates with full path", func(t *testing.T) {
		registry := TemplateRegistry{Name: "default", URL: parseURIWithoutError("http://test.registry/blueprint/")}
		out, err := getTemplateConfigs("aws/test", registry, mockMakeHTTPCallForTemplatePathIndex)
		require.Nil(t, err)
		require.NotNil(t, out)
		assert.Equal(t, []TemplateConfig{
			TemplateConfig{File: "test.yaml", FullPath: "http://test.registry/blueprint/aws/test/test.yaml", Registry: registry},
			TemplateConfig{File: "test.yml.tmpl", FullPath: "http://test.registry/blueprint/aws/test/test.yml.tmpl", Registry: registry},
		}, out)
	})
}

func TestMergeRegistryIndex(t *testing.T) {
	t.Run("should merge multiple registry index", func(t *testing.T) {
		registry1 := TemplateRegistry{URL: parseURIWithoutError("http://test.registry/blueprint")}
		registry2 := TemplateRegistry{URL: parseURIWithoutError("http://test.registry/blueprint2/")}
		registry3 := TemplateRegistry{URL: parseURIWithoutError("http://test.registry/blueprint3")}
		out, err := mergeRegistryIndex([]TemplateRegistry{
			registry1, registry2, registry3,
		}, mockMakeHTTPCallForTemplateIndex)
		require.Nil(t, err)
		require.NotNil(t, out)
		assert.Equal(t, map[string]TemplateRegistry{
			"aws/test2":  registry1,
			"aws/test":   registry2,
			"azure/test": registry2,
			"gcp/test":   registry3,
		}, out)
	})
	t.Run("should produce error if http call fails", func(t *testing.T) {
		registry1 := TemplateRegistry{URL: parseURIWithoutError("http://test.registry/blueprint")}
		out, err := mergeRegistryIndex([]TemplateRegistry{
			registry1,
		}, func(indexURL string, registry TemplateRegistry) ([]byte, int, error) {
			json := `
			{
				"gcp/test": [
					"test1",
					"test2"
				],,,
			}
		  `
			bytes := []byte(json)
			return bytes, 200, nil
		})
		require.Nil(t, out)
		require.NotNil(t, err)
		assert.Equal(t, "invalid character ',' looking for beginning of object key string", err.Error())
	})
	t.Run("should produce error on invalid json", func(t *testing.T) {
		registry1 := TemplateRegistry{URL: parseURIWithoutError("http://test.registry/blueprint")}
		out, err := mergeRegistryIndex([]TemplateRegistry{
			registry1,
		}, func(indexURL string, registry TemplateRegistry) ([]byte, int, error) {
			return nil, 400, fmt.Errorf("http call error")
		})
		require.Nil(t, out)
		require.NotNil(t, err)
		assert.Equal(t, "http call error", err.Error())
	})
	t.Run("should produce error on invalid statuscode", func(t *testing.T) {
		registry1 := TemplateRegistry{URL: parseURIWithoutError("http://test.registry/blueprint")}
		out, err := mergeRegistryIndex([]TemplateRegistry{
			registry1,
		}, func(indexURL string, registry TemplateRegistry) ([]byte, int, error) {
			return nil, 401, nil
		})
		require.Nil(t, out)
		require.NotNil(t, err)
		assert.Equal(t, "401 Request unauthorized. Please check your credentials for http://test.registry/blueprint", err.Error())

		out, err = mergeRegistryIndex([]TemplateRegistry{
			registry1,
		}, func(indexURL string, registry TemplateRegistry) ([]byte, int, error) {
			return nil, 405, nil
		})
		require.Nil(t, out)
		require.NotNil(t, err)
		assert.Equal(t, "error: StatusCode 405 for URL http://test.registry/blueprint", err.Error())

	})
}

func TestGetFilePathRelativeToTemplatePath(t *testing.T) {
	t.Run("should get a relative file path based on templatepath", func(t *testing.T) {
		out := getFilePathRelativeToTemplatePath(path.Join("test", "blueprints", "test.yaml.tmpl"), path.Join("test", "blueprints"))
		require.NotNil(t, out)
		assert.Equal(t, "test.yaml.tmpl", out)
		out = getFilePathRelativeToTemplatePath(path.Join("test", "blueprints", "nested", "test.yaml.tmpl"), path.Join("test", "blueprints"))
		require.NotNil(t, out)
		assert.Equal(t, path.Join("nested", "test.yaml.tmpl"), out)
		out = getFilePathRelativeToTemplatePath(path.Join("test", "blueprints", "nested", "test.yaml.tmpl"), path.Join("test", "blueprints", ""))
		require.NotNil(t, out)
		assert.Equal(t, path.Join("nested", "test.yaml.tmpl"), out)
		out = getFilePathRelativeToTemplatePath(path.Join("test", "blueprints", "nested", "test.yaml.tmpl"), path.Join("test", "blueprintssss"))
		require.NotNil(t, out)
		assert.Equal(t, path.Join("test", "blueprints", "nested", "test.yaml.tmpl"), out)
	})
}

func TestGetFromRelativeFolder(t *testing.T) {
	t.Run("should get template config from relative paths", func(t *testing.T) {
		tmpDir := path.Join("test", "blueprints")
		os.MkdirAll(tmpDir, os.ModePerm)
		defer os.RemoveAll("test")
		d1 := []byte("hello\ngo\n")
		ioutil.WriteFile(path.Join(tmpDir, "test.yaml.tmpl"), d1, os.ModePerm)
		ioutil.WriteFile(path.Join(tmpDir, "test2.yaml.tmpl"), d1, os.ModePerm)
		out, err := getFromRelativeFolder(tmpDir)
		require.Nil(t, err)
		require.NotNil(t, out)
		assert.Equal(t, []TemplateConfig{
			TemplateConfig{File: "test.yaml.tmpl", FullPath: path.Join(tmpDir, "test.yaml.tmpl")},
			TemplateConfig{File: "test2.yaml.tmpl", FullPath: path.Join(tmpDir, "test2.yaml.tmpl")},
		}, out)
	})
	t.Run("should get template config from relative nested paths", func(t *testing.T) {
		tmpDir := path.Join("test", "blueprints")
		os.MkdirAll(path.Join(tmpDir, "nested"), os.ModePerm)
		defer os.RemoveAll("test")
		d1 := []byte("hello\ngo\n")
		ioutil.WriteFile(path.Join(tmpDir, "test.yaml.tmpl"), d1, os.ModePerm)
		ioutil.WriteFile(path.Join(tmpDir, "nested", "test2.yaml.tmpl"), d1, os.ModePerm)
		out, err := getFromRelativeFolder(tmpDir)
		require.Nil(t, err)
		require.NotNil(t, out)
		assert.Equal(t, []TemplateConfig{
			{File: path.Join("nested", "test2.yaml.tmpl"), FullPath: path.Join(tmpDir, "nested", "test2.yaml.tmpl")},
			{File: "test.yaml.tmpl", FullPath: path.Join(tmpDir, "test.yaml.tmpl")},
		}, out)
	})

	t.Run("should get template config from absolute nested paths", func(t *testing.T) {
		tmpDir, err := ioutil.TempDir("", "blueprints")
		require.Nil(t, err)
		defer os.RemoveAll(tmpDir)
		os.MkdirAll(path.Join(tmpDir, "nested"), os.ModePerm)
		d1 := []byte("hello\ngo\n")
		ioutil.WriteFile(path.Join(tmpDir, "test.yaml.tmpl"), d1, os.ModePerm)
		ioutil.WriteFile(path.Join(tmpDir, "nested", "test2.yaml.tmpl"), d1, os.ModePerm)
		out, err := getFromRelativeFolder(tmpDir)
		require.Nil(t, err)
		require.NotNil(t, out)
		assert.Equal(t, []TemplateConfig{
			{File: path.Join("nested", "test2.yaml.tmpl"), FullPath: path.Join(tmpDir, "nested", "test2.yaml.tmpl")},
			{File: "test.yaml.tmpl", FullPath: path.Join(tmpDir, "test.yaml.tmpl")},
		}, out)
	})
	t.Run("should return nil if directory is empty", func(t *testing.T) {
		tmpDir := path.Join("test", "blueprints")
		os.MkdirAll(tmpDir, os.ModePerm)
		defer os.RemoveAll("test")
		out, err := getFromRelativeFolder(tmpDir)
		require.Nil(t, err)
		require.Nil(t, out)
	})
	t.Run("should return nil if directory doesnt exist", func(t *testing.T) {
		out, err := getFromRelativeFolder(path.Join("test", "blueprints"))
		require.Nil(t, err)
		require.Nil(t, out)
	})
}

func TestCreateTemplateConfigForSingleFile(t *testing.T) {
	t.Run("should create template config for a file url", func(t *testing.T) {
		out, err := createTemplateConfigForSingleFile("http://xebialabs.com/test/blueprints/test.yaml.tmpl")
		require.Nil(t, err)
		require.NotNil(t, out)
		assert.Equal(t, []TemplateConfig{
			TemplateConfig{File: "test.yaml.tmpl", FullPath: "http://xebialabs.com/test/blueprints/test.yaml.tmpl"},
		}, out)
	})
	t.Run("should create template config for a relative file", func(t *testing.T) {
		tmpPath := path.Join("test", "blueprints", "test.yaml.tmpl")
		out, err := createTemplateConfigForSingleFile(tmpPath)
		require.Nil(t, err)
		require.NotNil(t, out)
		assert.Equal(t, []TemplateConfig{
			TemplateConfig{File: "test.yaml.tmpl", FullPath: tmpPath},
		}, out)
	})
	t.Run("should create template config for a file", func(t *testing.T) {
		out, err := createTemplateConfigForSingleFile("test.yaml.tmpl")
		require.Nil(t, err)
		require.NotNil(t, out)
		assert.Equal(t, []TemplateConfig{
			TemplateConfig{File: "test.yaml.tmpl", FullPath: "test.yaml.tmpl"},
		}, out)
	})
	t.Run("should return err if template is empty", func(t *testing.T) {
		out, err := createTemplateConfigForSingleFile("")
		require.NotNil(t, err)
		require.Nil(t, out)
	})
}

func TestFetchTemplateFromPath(t *testing.T) {
	stream := `
		###
		variables:
		- name: AccessKey
		type: Function
		value: !Fn aws.readCreds.AccessKey
		###
		---
		apiVersion: xl-deploy/v1beta1
		kind: Infrastructure
		spec:
		- name: aws
		type: aws.Cloud
		accesskey: {{.AccessKey}}
		accessSecret: {{.AccessSecret}}
		
		---	
		  `
	t.Run("should fetch a template from http url", func(t *testing.T) {
		out, err := fetchTemplateFromPath(TemplateConfig{
			File: "test.yaml", FullPath: "http://aws/monolith/test.yaml.tmpl", Registry: TemplateRegistry{},
		}, true, mockMakeHTTPCallForTemplateFile(t, "http://aws/monolith/test.yaml.tmpl", stream))
		require.Nil(t, err)
		assert.Equal(t, stream, string(out))
		// test without suffix for file
		out, err = fetchTemplateFromPath(TemplateConfig{
			File: "test.yaml", FullPath: "http://aws/monolith/test.yaml", Registry: TemplateRegistry{},
		}, true, mockMakeHTTPCallForTemplateFile(t, "http://aws/monolith/test.yaml.tmpl", stream))
		require.Nil(t, err)
		assert.Equal(t, stream, string(out))
	})
	t.Run("should fetch a template from local path", func(t *testing.T) {
		tmpDir := path.Join("test", "blueprints")
		os.MkdirAll(tmpDir, os.ModePerm)
		defer os.RemoveAll("test")
		d1 := []byte(stream)
		ioutil.WriteFile(path.Join(tmpDir, "test.yaml.tmpl"), d1, os.ModePerm)
		out, err := fetchTemplateFromPath(TemplateConfig{
			File: "test.yaml", FullPath: path.Join(tmpDir, "test.yaml.tmpl"),
		}, true, nil)
		require.Nil(t, err)
		assert.Equal(t, stream, string(out))
		// test without suffix for file
		out, err = fetchTemplateFromPath(TemplateConfig{
			File: "test.yaml", FullPath: path.Join(tmpDir, "test.yaml"),
		}, true, nil)
		require.Nil(t, err)
		assert.Equal(t, stream, string(out))
	})
	t.Run("should error on url not found", func(t *testing.T) {
		out, err := fetchTemplateFromPath(TemplateConfig{
			File: "test.yaml", FullPath: "http://aws/monolith/test.yaml.tmpl", Registry: TemplateRegistry{},
		}, true, mockMakeHTTPCallForTemplateFile(t, "", stream))
		require.NotNil(t, err)
		require.Nil(t, out)
		assert.Equal(t, "error", err.Error())
	})
	t.Run("should error on no file found", func(t *testing.T) {
		tmpPath := path.Join("aws", "monolith", "test.yaml.tmpl")
		out, err := fetchTemplateFromPath(TemplateConfig{
			File: "test.yaml", FullPath: tmpPath, Registry: TemplateRegistry{},
		}, true, nil)
		require.Nil(t, out)
		require.NotNil(t, err)
		assert.Equal(t, "template not found in path "+tmpPath, err.Error())
	})
}

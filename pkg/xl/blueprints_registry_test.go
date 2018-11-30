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

func mockMakeHTTPCallForTemplateIndex(indexURL string, blueprintRepository BlueprintRepository) ([]byte, int, error) {
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

func mockMakeHTTPCallForTemplateFile(t *testing.T, expectedurl, output string) MakeHTTPCallForBlueprintRepositoryFn {
	return func(urlPath string, blueprintRepository BlueprintRepository) ([]byte, int, error) {
		if expectedurl == "" {
			return nil, 0, fmt.Errorf("error")
		}
		require.NotNil(t, urlPath)
		assert.Equal(t, expectedurl, urlPath)
		bytes := []byte(output)
		return bytes, 200, nil
	}
}

func TestMakeFullURLPath(t *testing.T) {
	t.Run("should modify the template with full path and registry", func(t *testing.T) {
		blueprintRepository := BlueprintRepository{SimpleHTTPServer{Url: parseURIWithoutError("http://test.registry/blueprint/")}}
		templateConfig := TemplateConfig{
			File: "test1.yml",
		}
		templateConfig.generateFullURLPath("aws/test", blueprintRepository)
		assert.Equal(t, TemplateConfig{File: "test1.yml", FullPath: "http://test.registry/blueprint/aws/test/test1.yml", Repository: blueprintRepository}, templateConfig)
	})
	t.Run("should modify the template with full path only when its local file", func(t *testing.T) {
		tmpDir := path.Join("test", "blueprints")
		os.MkdirAll(tmpDir, os.ModePerm)
		defer os.RemoveAll("test")
		d1 := []byte("hello\ngo\n")
		ioutil.WriteFile(path.Join(tmpDir, "test1.yml"), d1, os.ModePerm)
		blueprintRepository := BlueprintRepository{SimpleHTTPServer{Url: parseURIWithoutError("http://test.registry/blueprint/")}}
		templateConfig := TemplateConfig{
			File: "test1.yml",
		}
		templateConfig.generateFullURLPath("test/blueprints", blueprintRepository)
		assert.Equal(t, TemplateConfig{File: "test1.yml", FullPath: "test/blueprints/test1.yml"}, templateConfig)
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
			{File: "test.yaml.tmpl", FullPath: path.Join(tmpDir, "test.yaml.tmpl")},
			{File: "test2.yaml.tmpl", FullPath: path.Join(tmpDir, "test2.yaml.tmpl")},
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
	t.Run("should return error if directory is empty", func(t *testing.T) {
		tmpDir := path.Join("test", "blueprints")
		os.MkdirAll(tmpDir, os.ModePerm)
		defer os.RemoveAll("test")
		out, err := getFromRelativeFolder(tmpDir)
		require.Nil(t, out)
		require.NotNil(t, err)
		require.Equal(t, "path [test/blueprints] doesn't include any valid files", err.Error())
	})
	t.Run("should return error if directory doesn't exist", func(t *testing.T) {
		out, err := getFromRelativeFolder(path.Join("test", "blueprints"))
		require.Nil(t, out)
		require.NotNil(t, err)
		require.Equal(t, "path [test/blueprints] doesn't exist", err.Error())
	})
}

func TestCreateTemplateConfigForSingleFile(t *testing.T) {
	t.Run("should create template config for a file url", func(t *testing.T) {
		out, err := createTemplateConfigForSingleFile("http://xebialabs.com/test/blueprints/test.yaml.tmpl")
		require.Nil(t, err)
		require.NotNil(t, out)
		assert.Equal(t, []TemplateConfig{
			{File: "test.yaml.tmpl", FullPath: "http://xebialabs.com/test/blueprints/test.yaml.tmpl"},
		}, out)
	})
	t.Run("should create template config for a relative file", func(t *testing.T) {
		tmpPath := path.Join("test", "blueprints", "test.yaml.tmpl")
		out, err := createTemplateConfigForSingleFile(tmpPath)
		require.Nil(t, err)
		require.NotNil(t, out)
		assert.Equal(t, []TemplateConfig{
			{File: "test.yaml.tmpl", FullPath: tmpPath},
		}, out)
	})
	t.Run("should create template config for a file", func(t *testing.T) {
		out, err := createTemplateConfigForSingleFile("test.yaml.tmpl")
		require.Nil(t, err)
		require.NotNil(t, out)
		assert.Equal(t, []TemplateConfig{
			{File: "test.yaml.tmpl", FullPath: "test.yaml.tmpl"},
		}, out)
	})
	t.Run("should return err if template is empty", func(t *testing.T) {
		out, err := createTemplateConfigForSingleFile("")
		require.NotNil(t, err)
		require.Nil(t, out)
	})
}

func TestFetchTemplateFromPath(t *testing.T) {
	stream := fmt.Sprintf(`
		apiVersion: %s
		kind: Infrastructure
		spec:
		- name: aws
		type: aws.Cloud
		accesskey: {{.AccessKey}}
		accessSecret: {{.AccessSecret}}
		
		---	
		  `, XldApiVersion)

	t.Run("should fetch a template from http url", func(t *testing.T) {
		templateConfig := TemplateConfig{
			File: "test.yaml.tmpl", FullPath: "http://aws/monolith/test.yaml.tmpl", Repository: BlueprintRepository{},
		}
		out, err := templateConfig.fetchBlueprintFromPath(true, mockMakeHTTPCallForTemplateFile(t, "http://aws/monolith/test.yaml.tmpl", stream))
		require.Nil(t, err)
		assert.Equal(t, stream, string(out))
	})

	t.Run("should fetch a template from local path", func(t *testing.T) {
		tmpDir := path.Join("test", "blueprints")
		templateConfig := TemplateConfig{
			File: "test.yaml", FullPath: path.Join(tmpDir, "test.yaml.tmpl"),
		}
		os.MkdirAll(tmpDir, os.ModePerm)
		defer os.RemoveAll("test")
		d1 := []byte(stream)
		ioutil.WriteFile(path.Join(tmpDir, "test.yaml.tmpl"), d1, os.ModePerm)
		out, err := templateConfig.fetchBlueprintFromPath(true, nil)
		require.Nil(t, err)
		assert.Equal(t, stream, string(out))
	})

	t.Run("should error on url not found", func(t *testing.T) {
		templateConfig := TemplateConfig{
			File: "test.yaml", FullPath: "http://aws/monolith/test.yaml.tmpl", Repository: BlueprintRepository{},
		}
		out, err := templateConfig.fetchBlueprintFromPath(true, mockMakeHTTPCallForTemplateFile(t, "", stream))
		require.NotNil(t, err)
		require.Nil(t, out)
		assert.Equal(t, "error", err.Error())
	})

	t.Run("should error on no file found", func(t *testing.T) {
		tmpPath := path.Join("aws", "monolith", "test.yaml.tmpl")
		templateConfig := TemplateConfig{
			File: "test.yaml", FullPath: tmpPath, Repository: BlueprintRepository{},
		}
		out, err := templateConfig.fetchBlueprintFromPath(true, nil)
		require.Nil(t, out)
		require.NotNil(t, err)
		assert.Equal(t, "template not found in path "+tmpPath, err.Error())
	})
}

func TestGetBlueprintVariableConfig(t *testing.T) {
	yaml := `
      apiVersion: xl/v1
      kind: Blueprint
      metadata:
        projectName: Test Project
      
      parameters:
      - name: Test
        type: Input
        value: testing
      
      files:
      - path: xld-environment.yml.tmpl
      - path: xld-infrastructure.yml.tmpl
      - path: xlr-pipeline.yml`
	repository := BlueprintRepository{SimpleHTTPServer{Url: parseURIWithoutError("http://xebialabs.com")}}

	tmpDir := path.Join("test", "blueprints")
	os.MkdirAll(tmpDir, os.ModePerm)
	defer os.RemoveAll("test")
	d1 := []byte(yaml)
	ioutil.WriteFile(path.Join(tmpDir, "blueprint.yaml"), d1, os.ModePerm)

	type args struct {
		templatePath      string
		repository        BlueprintRepository
		blueprintFileName string
		makeHTTPCallFn    MakeHTTPCallForBlueprintRepositoryFn
	}

	tests := []struct {
		name    string
		args    args
		want    string
		wantErr error
	}{
		{
			"read a given blueprint.yaml file from repository",
			args{"test/blueprints", repository, "blueprint.yaml", mockMakeHTTPCallForTemplateFile(t, "http://xebialabs.com/test/blueprints/blueprint.yaml", yaml)},
			yaml,
			nil,
		},
		{
			"read a given blueprint.yaml local file",
			args{"test/blueprints", BlueprintRepository{}, "blueprint.yaml", nil},
			yaml,
			nil,
		},
		{
			"read a given blueprint.yaml local file when repository exists",
			args{"test/blueprints", repository, "blueprint.yaml", nil},
			yaml,
			nil,
		},
		{
			"error on http error",
			args{"test2/blueprints", repository, "blueprint.yaml", mockMakeHTTPCallForTemplateFile(t, "", yaml)},
			"",
			fmt.Errorf("error"),
		},
		{
			"error on when remote or local file doesn't exist",
			args{"test/blueprints", BlueprintRepository{}, "blueprints.yml", nil},
			"",
			fmt.Errorf("template not found in path test/blueprints/blueprints.yml"),
		},
		{
			"error on local path doesn't exist",
			args{"test/blueprints2", BlueprintRepository{}, "blueprint.yaml", nil},
			"",
			fmt.Errorf("template not found in path test/blueprints2/blueprint.yaml"),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := getBlueprintVariableConfig(tt.args.templatePath, tt.args.repository, tt.args.blueprintFileName, tt.args.makeHTTPCallFn)
			if tt.wantErr == nil || err == nil {
				assert.Equal(t, tt.wantErr, err)
			} else {
				assert.Equal(t, tt.wantErr.Error(), err.Error())
			}
			if tt.want == "" {
				assert.Nil(t, got)
			} else {
				assert.Equal(t, tt.want, string(*got))
			}
		})
	}
}

func TestGetBlueprintConfig(t *testing.T) {
	yaml1 := `
      apiVersion: xl/v1
      kind: Blueprint
      metadata:
        projectName: Test Project
      parameters:
      - name: Test
        type: Input
        value: testing`
	yaml2 := `
      apiVersion: xl/v1
      kind: Blueprint
      metadata:
        projectName: Test Project
      parameters:
      - name: Test
        type: Input
        value: testing
      files:
      - path: xld-environment.yml.tmpl
      - path: xlr-pipeline.yml`
	templatePath := "test/blueprints"
	repository := BlueprintRepository{SimpleHTTPServer{Url: parseURIWithoutError("http://xebialabs.com")}}
	tmpDir, err := ioutil.TempDir("", "blueprints")
	require.Nil(t, err)
	defer os.RemoveAll(tmpDir)
	d1 := []byte(yaml1)
	ioutil.WriteFile(path.Join(tmpDir, "blueprint.yaml"), d1, os.ModePerm)
	os.MkdirAll(path.Join(tmpDir, "nested"), os.ModePerm)
	d1 = []byte("hello\ngo\n")
	ioutil.WriteFile(path.Join(tmpDir, "test.yaml.tmpl"), d1, os.ModePerm)
	ioutil.WriteFile(path.Join(tmpDir, "nested", "test2.yaml.tmpl"), d1, os.ModePerm)

	tmpDir2, err := ioutil.TempDir("", "blueprints2")
	require.Nil(t, err)
	defer os.RemoveAll(tmpDir2)
	d2 := []byte(yaml2)
	ioutil.WriteFile(path.Join(tmpDir2, "blueprint.yaml"), d2, os.ModePerm)
	ioutil.WriteFile(path.Join(tmpDir2, "test.yaml.tmpl"), d1, os.ModePerm)

	type args struct {
		templatePath string
		repository   BlueprintRepository
		makeHTTPCall []MakeHTTPCallForBlueprintRepositoryFn
	}
	tests := []struct {
		name    string
		args    args
		want    *BlueprintYaml
		wantErr error
	}{
		{
			"throw error when no configuration is found",
			args{templatePath, repository, []MakeHTTPCallForBlueprintRepositoryFn{mockMakeHTTPCallForTemplateFile(t, "http://xebialabs.com/test/blueprints/blueprint.yaml", yaml1)}},
			nil,
			fmt.Errorf("path [%s] doesn't exist", templatePath),
		},
		{
			"get config from files declaration",
			args{templatePath, repository, []MakeHTTPCallForBlueprintRepositoryFn{mockMakeHTTPCallForTemplateFile(t, "http://xebialabs.com/test/blueprints/blueprint.yaml", yaml2)}},
			&BlueprintYaml{
				ApiVersion: "xl/v1",
				Kind:       "Blueprint",
				Metadata:   map[interface{}]interface{}{"projectName": "Test Project"},
				Parameters: []interface{}{map[interface{}]interface{}{"name": "Test", "type": "Input", "value": "testing"}},
				Files: []interface{}{
					map[interface{}]interface{}{"path": "xld-environment.yml.tmpl"},
					map[interface{}]interface{}{"path": "xlr-pipeline.yml"},
				},
				TemplateConfigs: []TemplateConfig{
					{File: "xld-environment.yml.tmpl", FullPath: "http://xebialabs.com/test/blueprints/xld-environment.yml.tmpl", Repository: repository},
					{File: "xlr-pipeline.yml", FullPath: "http://xebialabs.com/test/blueprints/xlr-pipeline.yml", Repository: repository},
				},
				Variables: []Variable{
					{Name: VarField{Val: "Test"}, Type: VarField{Val: "Input"}, Value: VarField{Val: "testing"}},
				},
			},
			nil,
		},
		{
			"fallback to walking local file tree when declaration not found",
			args{tmpDir, BlueprintRepository{}, []MakeHTTPCallForBlueprintRepositoryFn{mockMakeHTTPCallForTemplateFile(t, "", "")}},
			&BlueprintYaml{
				ApiVersion: "xl/v1",
				Kind:       "Blueprint",
				Metadata:   map[interface{}]interface{}{"projectName": "Test Project"},
				Parameters: []interface{}{map[interface{}]interface{}{"name": "Test", "type": "Input", "value": "testing"}},
				TemplateConfigs: []TemplateConfig{
					{File: path.Join("nested", "test2.yaml.tmpl"), FullPath: tmpDir + "/nested/test2.yaml.tmpl"},
					{File: "test.yaml.tmpl", FullPath: tmpDir + "/test.yaml.tmpl"},
				},
				Variables: []Variable{
					{Name: VarField{Val: "Test"}, Type: VarField{Val: "Input"}, Value: VarField{Val: "testing"}},
				},
			},
			nil,
		},
		{
			"use files decalration when found in local path as well",
			args{tmpDir2, BlueprintRepository{}, []MakeHTTPCallForBlueprintRepositoryFn{mockMakeHTTPCallForTemplateFile(t, "", "")}},
			&BlueprintYaml{
				ApiVersion: "xl/v1",
				Kind:       "Blueprint",
				Metadata:   map[interface{}]interface{}{"projectName": "Test Project"},
				Parameters: []interface{}{map[interface{}]interface{}{"name": "Test", "type": "Input", "value": "testing"}},
				Files: []interface{}{
					map[interface{}]interface{}{"path": "xld-environment.yml.tmpl"},
					map[interface{}]interface{}{"path": "xlr-pipeline.yml"},
				},
				TemplateConfigs: []TemplateConfig{
					{File: "xld-environment.yml.tmpl", FullPath: tmpDir2 + "/xld-environment.yml.tmpl"},
					{File: "xlr-pipeline.yml", FullPath: tmpDir2 + "/xlr-pipeline.yml"},
				},
				Variables: []Variable{
					{Name: VarField{Val: "Test"}, Type: VarField{Val: "Input"}, Value: VarField{Val: "testing"}},
				},
			},
			nil,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := GetBlueprintConfig(tt.args.templatePath, tt.args.repository, tt.args.makeHTTPCall...)
			if tt.wantErr == nil || err == nil {
				assert.Equal(t, tt.wantErr, err)
			} else {
				assert.Equal(t, tt.wantErr.Error(), err.Error())
			}
			assert.Equal(t, tt.want, got)
		})
	}
}

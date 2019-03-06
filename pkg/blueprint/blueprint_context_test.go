package blueprint

import (
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/xebialabs/xl-cli/pkg/models"
)

var DefaultBlueprintContext = &BlueprintContext{
	/*Provider: models.ProviderMock,
	Name:     "blueprints",
	Owner:    "xebialabs",
	Branch:   "test",*/
}

// TODO
/*func TestBlueprintContextBuilder(t *testing.T) {
	t.Run("build simple context for Blueprint repository", func(t *testing.T) {
		v := viper.New()
		v.Set(ViperKeyBlueprintRepositoryProvider, models.ProviderGitHub)
		v.Set(ViperKeyBlueprintRepositoryName, "blueprints")

		c, err := ConstructBlueprintContext(v)

		assert.Nil(t, err)
		assert.NotNil(t, c)
		assert.Equal(t, models.ProviderGitHub, (*c.CurrentRepoContext).GetProvider())
		assert.Equal(t, "blueprints", (*c.CurrentRepoContext).GetName())
	})
}*/

// remote provider tests
/*func TestBlueprintContextFunctionsForRemote(t *testing.T) {
	// error cases
	t.Run("should return error when trying to init from an invalid blueprint context", func(t *testing.T) {
		context := &BlueprintContext{
			Provider: "false-provider",
			Name:     "blueprints",
			Owner:    "xebialabs",
			Branch:   "master",
			Token:    "",
		}
		err := (*context.ActiveRepo).Initialize()
		require.NotNil(t, err)
		assert.Equal(t, "no blueprint provider implementation found for false-provider", err.Error())
	})

	// mock success case
	t.Run("should init repo client with mock remote blueprint provider", func(t *testing.T) {
		blueprints, err := DefaultBlueprintContext.initRepoClient()
		require.Nil(t, err)
		require.NotNil(t, blueprints)
		require.Len(t, blueprints, 1)

		t.Run("should parse blueprint definition file", func(t *testing.T) {
			blueprintDefinition, err := DefaultBlueprintContext.parseDefinitionFile(false, blueprints, "xl/test")
			require.Nil(t, err)
			require.NotNil(t, blueprintDefinition)
			err = blueprintDefinition.validate()
			require.Nil(t, err)
			assert.Len(t, blueprintDefinition.Variables, 1)
			assert.Len(t, blueprintDefinition.TemplateConfigs, 2)
		})

		t.Run("should get file contents", func(t *testing.T) {
			contents, err := DefaultBlueprintContext.fetchFileContents(blueprints["xl/test"].Files[0].Path, false, false)
			require.Nil(t, err)
			require.NotNil(t, contents)
			assert.NotEmptyf(t, string(*contents), "mock blueprint file content is empty")
			assert.Equal(t, "template", string(*contents))
		})

		t.Run("should error on non-existing remote file path", func(t *testing.T) {
			_, err := DefaultBlueprintContext.fetchFileContents("non-existing-path/file.yaml", false, true)
			require.NotNil(t, err)
			assert.Equal(t, "file non-existing-path/file.yaml.tmpl not found in mock repo", err.Error())
		})
	})
}*/

// local provider tests
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
		  `, models.XldApiVersion)

	t.Run("should fetch a template from local path", func(t *testing.T) {
		blueprintContext := BlueprintContext{}
		tmpDir := path.Join("test", "blueprints")
		templateConfig := TemplateConfig{
			File: "test.yaml", FullPath: path.Join(tmpDir, "test.yaml"),
		}
		os.MkdirAll(tmpDir, os.ModePerm)
		defer os.RemoveAll("test")
		d1 := []byte(stream)
		ioutil.WriteFile(path.Join(tmpDir, "test.yaml.tmpl"), d1, os.ModePerm)
		out, err := blueprintContext.fetchFileContents(templateConfig.FullPath, true, true)
		require.Nil(t, err)
		assert.Equal(t, stream, string(*out))
	})

	t.Run("should error on no file found", func(t *testing.T) {
		blueprintContext := BlueprintContext{}
		tmpPath := path.Join("aws", "monolith", "test.yaml.tmpl")
		templateConfig := TemplateConfig{
			File: "test.yaml", FullPath: tmpPath,
		}
		out, err := blueprintContext.fetchFileContents(templateConfig.FullPath, true, true)
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

	tmpDir := path.Join("test", "blueprints")
	os.MkdirAll(tmpDir, os.ModePerm)
	defer os.RemoveAll("test")
	d1 := []byte(yaml)
	ioutil.WriteFile(path.Join(tmpDir, "blueprint.yaml"), d1, os.ModePerm)

	type args struct {
		templatePath      string
		blueprintFileName string
	}

	tests := []struct {
		name    string
		args    args
		want    string
		wantErr error
	}{
		{
			"read a given blueprint.yaml local file",
			args{"test/blueprints", "blueprint.yaml"},
			yaml,
			nil,
		},
		{
			"read a given blueprint.yaml local file when repository exists",
			args{"test/blueprints", "blueprint.yaml"},
			yaml,
			nil,
		},
		{
			"error on when local file doesn't exist",
			args{"test/blueprints", "blueprints.yml"},
			"",
			fmt.Errorf("template not found in path test/blueprints/blueprints.yml"),
		},
		{
			"error on local path doesn't exist",
			args{"test/blueprints2", "blueprint.yaml"},
			"",
			fmt.Errorf("template not found in path test/blueprints2/blueprint.yaml"),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			blueprintContext := BlueprintContext{}
			filePath := fmt.Sprintf("%s/%s", tt.args.templatePath, tt.args.blueprintFileName)
			got, err := blueprintContext.fetchFileContents(filePath, true, false)
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

// utility function tests
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

func TestCreateTemplateConfigForSingleFile(t *testing.T) {
	t.Run("should create template config for a relative file", func(t *testing.T) {
		tmpPath := path.Join("test", "blueprints")
		out, err := createTemplateConfigForSingleFile(path.Join(tmpPath, "test.yaml.tmpl"))
		require.Nil(t, err)
		require.NotNil(t, out)
		assert.Equal(t, []TemplateConfig{
			{File: "test.yaml.tmpl", FullPath: path.Join(tmpPath, "test.yaml.tmpl")},
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

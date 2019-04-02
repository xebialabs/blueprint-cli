package blueprint

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/jarcoal/httpmock"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/xebialabs/xl-cli/pkg/models"
	"github.com/xebialabs/xl-cli/pkg/util"
	"github.com/xebialabs/yaml"
)

var defaultContextYaml = `
blueprint:
  current-repository: XL Http
  repositories:
  - name: XL Http
    type: http
    url: http://mock.repo.server.com/
  - name: XL Github
    type: github
    owner: xebialabs
    repo-name: blueprints
    branch: master`

func GetViperConf(t *testing.T, yaml string) *viper.Viper {
	configdir, err := ioutil.TempDir("", "xebialabsconfig")
	if err != nil {
		t.Error(err)
	}
	defer os.RemoveAll(configdir)
	configfile := filepath.Join(configdir, "config.yaml")
	originalConfigBytes := []byte(yaml)
	ioutil.WriteFile(configfile, originalConfigBytes, 0755)

	v := viper.New()
	v.SetConfigFile(configfile)
	v.ReadInConfig()
	return v
}

func getDefaultBlueprintContext(t *testing.T) *BlueprintContext {
	configdir, _ := ioutil.TempDir("", "xebialabsconfig")

	v := GetViperConf(t, defaultContextYaml)
	c, err := ConstructBlueprintContext(v, configdir)
	if err != nil {
		t.Error(err)
	}
	const mockEndpoint = "http://mock.repo.server.com/"
	httpmock.Activate()
	httpmock.RegisterResponder(
		"GET",
		mockEndpoint+"index.json",
		httpmock.NewStringResponder(200, `["aws/monolith", "aws/datalake"]`),
	)

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

	httpmock.RegisterResponder(
		"GET",
		mockEndpoint+"aws/monolith/blueprint.yaml",
		httpmock.NewStringResponder(200, yaml),
	)
	httpmock.RegisterResponder(
		"GET",
		mockEndpoint+"aws/datalake/blueprint.yaml",
		httpmock.NewStringResponder(200, `sample test text
with a new line`),
	)
	httpmock.RegisterResponder(
		"GET",
		mockEndpoint+"aws/monolith/test.yaml",
		httpmock.NewStringResponder(200, `sample test text
with a new line`),
	)

	return c
}

func TestConstructBlueprintContext(t *testing.T) {
	configDir, _ := ioutil.TempDir("", "xebialabsconfig")
	defer os.RemoveAll(configDir)
	configFile := path.Join(configDir, "config.yaml")
	defer httpmock.DeactivateAndReset()
	t.Run("should error when config is invalid", func(t *testing.T) {
		v := GetViperConf(t, `
        blueprint:
          current-repository: XL Http
          repositories:
          - name: true
            type: 1`)
		c, err := ConstructBlueprintContext(v, configFile)

		require.NotNil(t, err)
		require.Nil(t, c)
	})
	t.Run("should return error for invalid provider", func(t *testing.T) {
		v := GetViperConf(t, `
        blueprint:
          current-repository: XL Http
          repositories:
          - name: XL Http
            type: https
            url: https://dist.xebialabs.com/public/blueprints/`)
		c, err := ConstructBlueprintContext(v, configFile)

		require.NotNil(t, err)
		require.Nil(t, c)
	})
	t.Run("should return error when there is no repo defined", func(t *testing.T) {
		v := GetViperConf(t, `
        blueprint:
          current-repository: XL Https
          repositories:
          - name: XL Http
            type: https
            url: https://dist.xebialabs.com/public/blueprints/`)
		c, err := ConstructBlueprintContext(v, configFile)

		require.NotNil(t, err)
		require.Nil(t, c)
	})
	t.Run("should add default config when current-repository is not set", func(t *testing.T) {
		v := GetViperConf(t, defaultContextYaml)
		v.Set(ViperKeyBlueprintCurrentRepository, "")
		c, err := ConstructBlueprintContext(v, configFile)

		require.Nil(t, err)
		require.NotNil(t, c)
		repo := *c.ActiveRepo

		assert.Equal(t, 3, len(c.DefinedRepos))
		assert.Equal(t, models.DefaultBlueprintRepositoryProvider, repo.GetProvider())
		assert.Equal(t, models.DefaultBlueprintRepositoryName, repo.GetName())
	})
	t.Run("build simple context for Blueprint repository with default current-repository", func(t *testing.T) {
		v := GetViperConf(t, defaultContextYaml)
		c, err := ConstructBlueprintContext(v, configFile)

		require.Nil(t, err)
		require.NotNil(t, c)
		repo := *c.ActiveRepo
		assert.Equal(t, models.ProviderHttp, repo.GetProvider())
		assert.Equal(t, "XL Http", repo.GetName())
	})
	t.Run("build simple context for Blueprint repository with provided current-repository", func(t *testing.T) {
		v := GetViperConf(t, defaultContextYaml)
		v.Set(ViperKeyBlueprintCurrentRepository, "XL Github")
		c, err := ConstructBlueprintContext(v, configFile)

		require.Nil(t, err)
		require.NotNil(t, c)
		repo := *c.ActiveRepo
		assert.Equal(t, models.ProviderGitHub, repo.GetProvider())
		assert.Equal(t, "XL Github", repo.GetName())
	})
}

// local provider tests
func TestBlueprintContext_fetchFileContents(t *testing.T) {
	defer httpmock.DeactivateAndReset()

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
		out, err := blueprintContext.fetchFileContents(templateConfig.FullPath, true, true, templateConfig.File)
		require.Nil(t, err)
		assert.Equal(t, stream, string(*out))
	})

	t.Run("should fetch a template from remote path", func(t *testing.T) {
		repo := getDefaultBlueprintContext(t)
		blueprints, err := repo.initCurrentRepoClient()
		require.Nil(t, err)
		require.NotNil(t, blueprints)
		require.Len(t, blueprints, 2)

		t.Run("should get file contents", func(t *testing.T) {
			contents, err := repo.fetchFileContents("aws/monolith/test.yaml", false, false, "test.yaml")
			require.Nil(t, err)
			require.NotNil(t, contents)
			assert.NotEmptyf(t, string(*contents), "mock blueprint file content is empty")
			assert.Equal(t, "sample test text\nwith a new line", string(*contents))
		})

		t.Run("should error on non-existing remote file path", func(t *testing.T) {
			_, err := repo.fetchFileContents("non-existing-path/file.yaml", false, true, "non-existing-path/file.yaml")
			require.NotNil(t, err)
			assert.Equal(t, "Get http://mock.repo.server.com/non-existing-path/file.yaml.tmpl: no responder found", err.Error())
		})
	})

	t.Run("should error on no file found", func(t *testing.T) {
		blueprintContext := BlueprintContext{}
        filePath := "test.yaml.tmpl"
        tmpPath := path.Join("aws", "monolith", filePath)
		templateConfig := TemplateConfig{
			File: filePath, FullPath: tmpPath,
		}
		out, err := blueprintContext.fetchFileContents(templateConfig.FullPath, true, true, templateConfig.File)
		require.Nil(t, out)
		require.NotNil(t, err)
		assert.Equal(t, "template not found in path " + filePath, err.Error())
	})
}

func TestBlueprintContext_fetchLocalFile(t *testing.T) {
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
			fmt.Errorf("template not found in path blueprints.yml"),
		},
		{
			"error on local path doesn't exist",
			args{"test/blueprints2", "blueprint.yaml"},
			"",
			fmt.Errorf("template not found in path blueprint.yaml"),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			blueprintContext := BlueprintContext{}
			filePath := fmt.Sprintf("%s/%s", tt.args.templatePath, tt.args.blueprintFileName)
			got, err := blueprintContext.fetchLocalFile(filePath)
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

func TestBlueprintContext_parseLocalDefinitionFile(t *testing.T) {
	defer httpmock.DeactivateAndReset()
	repo := getDefaultBlueprintContext(t)
	blueprints, err := repo.initCurrentRepoClient()
	require.Nil(t, err)
	require.NotNil(t, blueprints)
	require.Len(t, blueprints, 2)

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
	ioutil.WriteFile(path.Join(tmpDir, "xld-environment.yml.tmpl"), d1, os.ModePerm)
	ioutil.WriteFile(path.Join(tmpDir, "xld-infrastructure.yml.tmpl"), d1, os.ModePerm)
	ioutil.WriteFile(path.Join(tmpDir, "xlr-pipeline.yml"), d1, os.ModePerm)

	tests := []struct {
		name             string
		blueprintContext BlueprintContext
		templatePath     string
		want             []TemplateConfig
		wantErr          bool
	}{
		{
			"should error when non existing path is used",
			*repo,
			"aws/test",
			nil,
			true,
		},
		{
			"should parse blueprint definition file",
			*repo,
			tmpDir,
			[]TemplateConfig{
				{File: "xld-environment.yml.tmpl", FullPath: "test/blueprints/xld-environment.yml.tmpl"},
				{File: "xld-infrastructure.yml.tmpl", FullPath: "test/blueprints/xld-infrastructure.yml.tmpl"},
				{File: "xlr-pipeline.yml", FullPath: "test/blueprints/xlr-pipeline.yml"},
			},
			false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			blueprintContext := tt.blueprintContext
			got, err := blueprintContext.parseLocalDefinitionFile(tt.templatePath)
			if (err != nil) != tt.wantErr {
				t.Errorf("BlueprintContext.parseLocalDefinitionFile() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != nil && !reflect.DeepEqual(got.TemplateConfigs, tt.want) {
				t.Errorf("BlueprintContext.parseLocalDefinitionFile() = %v (length %d), want length %d", got, len(got.TemplateConfigs), len(tt.want))
			}
		})
	}
}

func TestBlueprintContext_parseRemoteDefinitionFile(t *testing.T) {
	defer httpmock.DeactivateAndReset()
	repo := getDefaultBlueprintContext(t)
	blueprints, err := repo.initCurrentRepoClient()
	require.Nil(t, err)
	require.NotNil(t, blueprints)
	require.Len(t, blueprints, 2)

	type args struct {
		blueprints   map[string]*models.BlueprintRemote
		templatePath string
	}
	tests := []struct {
		name             string
		blueprintContext BlueprintContext
		args             args
		want             []TemplateConfig
		wantErr          bool
	}{
		{
			"should error if path doesnt exist",
			*repo,
			args{
				blueprints,
				"test",
			},
			nil,
			true,
		},
		{
			"should parse blueprint definition file",
			*repo,
			args{
				blueprints,
				"aws/monolith",
			},
			[]TemplateConfig{
				{File: "xld-environment.yml.tmpl", FullPath: "aws/monolith/xld-environment.yml.tmpl"},
				{File: "xld-infrastructure.yml.tmpl", FullPath: "aws/monolith/xld-infrastructure.yml.tmpl"},
				{File: "xlr-pipeline.yml", FullPath: "aws/monolith/xlr-pipeline.yml"},
			},
			false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			blueprintContext := tt.blueprintContext
			got, err := blueprintContext.parseRemoteDefinitionFile(tt.args.blueprints, tt.args.templatePath)
			if (err != nil) != tt.wantErr {
				t.Errorf("BlueprintContext.parseRemoteDefinitionFile() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != nil && !reflect.DeepEqual(got.TemplateConfigs, tt.want) {
				t.Errorf("BlueprintContext.parseLocalDefinitionFile() = %v (length %d), want length %d", got, len(got.TemplateConfigs), len(tt.want))
			}
		})
	}
}

func TestBlueprintContext_initCurrentRepoClient(t *testing.T) {
	defer httpmock.DeactivateAndReset()
	t.Run("should init repo client with http blueprint provider", func(t *testing.T) {
		repo := getDefaultBlueprintContext(t)
		blueprints, err := repo.initCurrentRepoClient()
		require.Nil(t, err)
		require.NotNil(t, blueprints)
		require.Len(t, blueprints, 2)
	})
}

func TestBlueprintContext_parseRepositoryTree(t *testing.T) {
	defer httpmock.DeactivateAndReset()
	repo := getDefaultBlueprintContext(t)
	(*repo.ActiveRepo).Initialize()
	tests := []struct {
		name    string
		want    int
		wantErr bool
	}{
		{
			"should fetch 2 blueprints",
			2,
			false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			blueprintContext := repo
			got, err := blueprintContext.parseRepositoryTree()
			if (err != nil) != tt.wantErr {
				t.Errorf("BlueprintContext.parseRepositoryTree() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(len(got), tt.want) {
				t.Errorf("BlueprintContext.parseRepositoryTree() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestGetDefaultBlueprintViperConfig(t *testing.T) {
	tests := []struct {
		name  string
		vYaml string
		want  string
	}{
		{
			"should return unchanged config when default repo exists",
			`
xl-deploy:
  username: admin
  password: admin123
  url: http://localhost:4516/
  authmethod: http
xl-release:
  username: admin
  password: admin123
  url: http://localhost:5516/
  authmethod: http
blueprint:
  current-repository: XL Blueprints
  repositories:
  - name: XL Blueprints
    type: http
    url: http://mock.repo.server.com/
  - name: XL Github
    type: github
    owner: xebialabs
    repo-name: blueprints
    branch: master`,
			`
xl-deploy:
  username: admin
  password: admin123
  url: http://localhost:4516/
  authmethod: http
xl-release:
  username: admin
  password: admin123
  url: http://localhost:5516/
  authmethod: http
blueprint:
  current-repository: XL Blueprints
  repositories:
  - name: XL Blueprints
    type: http
    url: http://mock.repo.server.com/
  - name: XL Github
    type: github
    owner: xebialabs
    repo-name: blueprints
    branch: master`,
		},
		{
			"should return with default config created when no repo exists",
			`
xl-deploy:
  username: admin
  password: admin123
  url: http://localhost:4516/
  authmethod: http
xl-release:
  username: admin
  password: admin123
  url: http://localhost:5516/
  authmethod: http`,
			`
xl-deploy:
  username: admin
  password: admin123
  url: http://localhost:4516/
  authmethod: http
xl-release:
  username: admin
  password: admin123
  url: http://localhost:5516/
  authmethod: http
blueprint:
  current-repository: XL Blueprints
  repositories:
  - name: XL Blueprints
    type: http
    url: https://dist.xebialabs.com/public/blueprints/`,
		},
		{
			"should return with default config created when no config exists",
			``,
			`
blueprint:
  current-repository: XL Blueprints
  repositories:
  - name: XL Blueprints
    type: http
    url: https://dist.xebialabs.com/public/blueprints/`,
		},
		{
			"should return with default config added when default repo doesn't exist in repositories",
			`
xl-deploy:
  username: admin
  password: admin123
  url: http://localhost:4516/
  authmethod: http
xl-release:
  username: admin
  password: admin123
  url: http://localhost:5516/
  authmethod: http
blueprint:
  repositories:
  - name: XL Blueprints 2
    type: http
    url: https://dist.xebialabs.com/public/blueprints/`,
			`
xl-deploy:
  username: admin
  password: admin123
  url: http://localhost:4516/
  authmethod: http
xl-release:
  username: admin
  password: admin123
  url: http://localhost:5516/
  authmethod: http
blueprint:
  current-repository: XL Blueprints
  repositories:
  - name: XL Blueprints 2
    type: http
    url: https://dist.xebialabs.com/public/blueprints/
  - name: XL Blueprints
    type: http
    url: https://dist.xebialabs.com/public/blueprints/`,
		},
		{
			"should return with updated default config when default repo is incomplete",
			`
xl-deploy:
  username: admin
  password: admin123
  url: http://localhost:4516/
  authmethod: http
xl-release:
  username: admin
  password: admin123
  url: http://localhost:5516/
  authmethod: http
blueprint:
  repositories:
  - name: XL Blueprints
    type: http
    url: `,
			`
xl-deploy:
  username: admin
  password: admin123
  url: http://localhost:4516/
  authmethod: http
xl-release:
  username: admin
  password: admin123
  url: http://localhost:5516/
  authmethod: http
blueprint:
  current-repository: XL Blueprints
  repositories:
  - name: XL Blueprints
    type: http
    url: https://dist.xebialabs.com/public/blueprints/`,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			v := viper.New()
			v.SetConfigType("yaml")
			err := v.ReadConfig(bytes.NewBuffer([]byte(tt.vYaml)))
			require.Nil(t, err)

			got := GetDefaultBlueprintViperConfig(v)
			c := util.SortMapStringInterface(got.AllSettings())
			bs, err := yaml.Marshal(c)
			bss := string(bs)
			require.Nil(t, err)

			v2 := viper.New()
			v2.SetConfigType("yaml")
			err = v2.ReadConfig(bytes.NewBuffer([]byte(tt.want)))
			require.Nil(t, err)
			v2setsSorted := util.SortMapStringInterface(v2.AllSettings())
			bs1, err := yaml.Marshal(v2setsSorted)
			require.Nil(t, err)
			bs1s := string(bs1)

			assert.Equal(t, bs1s, bss)

		})
	}
}

func TestCreateOrUpdateBlueprintConfig(t *testing.T) {
	confPath, err := ioutil.TempDir("", "xebialabsconfig")
	if err != nil {
		t.Error(err)
	}
	defer os.RemoveAll(confPath)
	configfile := filepath.Join(confPath, "config.yaml")
	originalConfigBytes := []byte("")
	ioutil.WriteFile(configfile, originalConfigBytes, 0755)
	tests := []struct {
		name    string
		v       string
		want    string
		wantErr bool
	}{
		{
			"should return unchanged config when default repo exists",
			`
xl-deploy:
  username: admin
  password: admin123
  url: http://localhost:4516/
  authmethod: http
xl-release:
  username: admin
  password: admin123
  url: http://localhost:5516/
  authmethod: http
blueprint:
  current-repository: XL Blueprints
  repositories:
  - name: XL Blueprints
    type: http
    url: http://mock.repo.server.com/
  - name: XL Github
    type: github
    owner: xebialabs
    repo-name: blueprints
    branch: master`,
			`
xl-deploy:
  username: admin
  password: admin123
  url: http://localhost:4516/
  authmethod: http
xl-release:
  username: admin
  password: admin123
  url: http://localhost:5516/
  authmethod: http
blueprint:
  current-repository: XL Blueprints
  repositories:
  - name: XL Blueprints
    type: http
    url: http://mock.repo.server.com/
  - name: XL Github
    type: github
    owner: xebialabs
    repo-name: blueprints
    branch: master`,
			false,
		},
		{
			"should return with default config added when default repo doesn't exist in repositories",
			`
xl-deploy:
  username: admin
  password: admin123
  url: http://localhost:4516/
  authmethod: http
xl-release:
  username: admin
  password: admin123
  url: http://localhost:5516/
  authmethod: http
blueprint:
  repositories:
  - name: XL Blueprints 2
    type: http
    url: https://dist.xebialabs.com/public/blueprints/`,
			`
xl-deploy:
  username: admin
  password: admin123
  url: http://localhost:4516/
  authmethod: http
xl-release:
  username: admin
  password: admin123
  url: http://localhost:5516/
  authmethod: http
blueprint:
  current-repository: XL Blueprints
  repositories:
  - name: XL Blueprints 2
    type: http
    url: https://dist.xebialabs.com/public/blueprints/
  - name: XL Blueprints
    type: http
    url: https://dist.xebialabs.com/public/blueprints/`,
			false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			v := viper.New()
			v.SetConfigType("yaml")
			err := v.ReadConfig(bytes.NewBuffer([]byte(tt.v)))
			require.Nil(t, err)

			got, err := CreateOrUpdateBlueprintConfig(v, configfile)
			require.Nil(t, err)

			c := util.SortMapStringInterface(got.AllSettings())
			bs, err := yaml.Marshal(c)
			bss := string(bs)
			require.Nil(t, err)

			v2 := viper.New()
			v2.SetConfigType("yaml")
			err = v2.ReadConfig(bytes.NewBuffer([]byte(tt.want)))
			require.Nil(t, err)
			v2setsSorted := util.SortMapStringInterface(v2.AllSettings())
			bs1, err := yaml.Marshal(v2setsSorted)
			require.Nil(t, err)
			bs1s := string(bs1)

			assert.Equal(t, bs1s, bss)

			file, err := ioutil.ReadFile(configfile)
			require.Nil(t, err)

			assert.Equal(t, bs1s, string(file))
		})
	}
}

func Test_doesDefaultExist(t *testing.T) {
	tests := []struct {
		name         string
		repositories []ConfMap
		want         bool
		checkIndex   int
	}{
		{
			"should return false when repo doesn't exist",
			[]ConfMap{},
			false,
			-1,
		},
		{
			"should return true when repo exist",
			[]ConfMap{
				{
					"name": "Bar",
					"type": models.DefaultBlueprintRepositoryProvider,
					"url":  "",
				},
				defaultBlueprintRepo,
			},
			true,
			1,
		},
		{
			"should return true when repo exist with default url when its nil",
			[]ConfMap{
				{
					"name": "Foo",
					"type": models.DefaultBlueprintRepositoryProvider,
					"url":  "",
				},
				{
					"name": models.DefaultBlueprintRepositoryName,
					"type": models.DefaultBlueprintRepositoryProvider,
					"url":  "",
				},
				{
					"name": "Bar",
					"type": models.DefaultBlueprintRepositoryProvider,
					"url":  "",
				},
			},
			true,
			1,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := doesDefaultExist(tt.repositories)
			if got != tt.want {
				t.Errorf("doesDefaultExist() = %v, want %v", got, tt.want)
			}
			if tt.checkIndex > -1 {
				assert.Equal(t, models.DefaultBlueprintRepositoryUrl, tt.repositories[tt.checkIndex]["url"])
				assert.Equal(t, models.DefaultBlueprintRepositoryProvider, tt.repositories[tt.checkIndex]["type"])
			}
		})
	}
}

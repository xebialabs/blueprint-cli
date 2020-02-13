package blueprint

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/jarcoal/httpmock"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/xebialabs/xl-blueprint/pkg/models"
	"github.com/xebialabs/xl-blueprint/pkg/util"
	"github.com/xebialabs/yaml"
)

const DummyCLIVersion = "9.0.0-SNAPSHOT"

const defaultContextYaml = `
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

var BlueprintTestPath = ""

func getLocalTestBlueprintContext(t *testing.T) *BlueprintContext {
	configdir, _ := ioutil.TempDir("", "xebialabsconfig")
	configfile := filepath.Join(configdir, "config.yaml")
	if BlueprintTestPath == "" {
		pwd, _ := os.Getwd()
		BlueprintTestPath = strings.Replace(pwd, path.Join("pkg", "blueprint"), path.Join("templates", "test"), -1)
	}
	contextYaml := fmt.Sprintf(`
blueprint:
  current-repository: Test
  repositories:
  - name: Test
    type: local
    path: %s`, BlueprintTestPath)
	v := GetViperConf(t, contextYaml)
	c, err := ConstructBlueprintContext(v, configfile, DummyCLIVersion)
	if err != nil {
		t.Error(err)
	}
	return c
}

func getMockHttpBlueprintContext(t *testing.T) *BlueprintContext {
	configdir, _ := ioutil.TempDir("", "xebialabsconfig")
	configfile := filepath.Join(configdir, "config.yaml")

	v := GetViperConf(t, defaultContextYaml)
	c, err := ConstructBlueprintContext(v, configfile, DummyCLIVersion)
	if err != nil {
		t.Error(err)
	}
	const mockEndpoint = "http://mock.repo.server.com/"
	httpmock.Activate()
	httpmock.RegisterResponder(
		"GET",
		mockEndpoint+"index.json",
		httpmock.NewStringResponder(200, `["aws/monolith", "aws/datalake", "aws/compose", "aws/compose-2", "aws/compose-3", "aws/emptyfiles", "aws/emptyparams"]`),
	)
	// aws / monolith
	yaml := `
      apiVersion: xl/v2
      kind: Blueprint
      metadata:
        name: Test Project
      spec:
        parameters:
        - name: Test
          value: testing
          saveInXlvals: true

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
		mockEndpoint+"aws/monolith/test.yaml",
		httpmock.NewStringResponder(200, `sample test text
with a new line`),
	)
	// aws/datalake
	yaml = `
    apiVersion: xl/v2
    kind: Blueprint
    metadata:
      name: Test Project 2

    spec:
      parameters:
      - name: Foo
        value: testing

      files:
      - path: xld-app.yml.tmpl
      - path: xlr-pipeline.yml`

	httpmock.RegisterResponder(
		"GET",
		mockEndpoint+"aws/datalake/blueprint.yaml",
		httpmock.NewStringResponder(200, yaml),
	)
	// aws/emptyfiles
	yaml = `
    apiVersion: xl/v2
    kind: Blueprint
    metadata:
      name: Test Project 3

    spec:
      parameters:
      - name: Foo
        value: testing`

	httpmock.RegisterResponder(
		"GET",
		mockEndpoint+"aws/emptyfiles/blueprint.yaml",
		httpmock.NewStringResponder(200, yaml),
	)
	//aws/emptyparams
	yaml = `
    apiVersion: xl/v2
    kind: Blueprint
    metadata:
      name: Test Project 4
    spec:
      files:
      - path: xld-app.yml.tmpl
      - path: xlr-pipeline.yml`

	httpmock.RegisterResponder(
		"GET",
		mockEndpoint+"aws/emptyparams/blueprint.yaml",
		httpmock.NewStringResponder(200, yaml),
	)
	// aws/compose
	yaml = `
      apiVersion: xl/v2
      kind: Blueprint
      metadata:
        name: Test Project

      spec:
        parameters:
        - name: Bar
          value: testing
        includeBefore:
        - blueprint: aws/monolith
          parameterOverrides:
          - name: Test
            value: hello
            promptIf: !expr "2 > 1"
          fileOverrides:
          - path: xld-infrastructure.yml.tmpl
            writeIf: !expr "false"
        includeAfter:
        - blueprint: aws/datalake
          includeIf: !expr "Bar == 'testing'"
          parameterOverrides:
          - name: Foo
            value: hello
          fileOverrides:
          - path: xlr-pipeline.yml
            renameTo: xlr-pipeline2-new.yml
            writeIf: TestDepends

        files:
        - path: xld-environment.yml.tmpl
        - path: xld-infrastructure.yml.tmpl
        - path: xlr-pipeline.yml`

	httpmock.RegisterResponder(
		"GET",
		mockEndpoint+"aws/compose/blueprint.yaml",
		httpmock.NewStringResponder(200, yaml),
	)
	// aws/compose-2
	yaml = `
      apiVersion: xl/v2
      kind: Blueprint
      metadata:
        name: Test Project

      spec:
        parameters:
        - name: Bar
          value: testing
        includeBefore:
        - blueprint: aws/monolith
          includeIf: !expr "2 < 1"
          parameterOverrides:
          - name: Test
            value: hello
            promptIf: !expr "2 > 1"
          - name: bar
            value: true
          fileOverrides:
          - path: xld-infrastructure.yml.tmpl
            writeIf: !expr "2 < 1"
        includeAfter:
        - blueprint: aws/datalake
          includeIf: !expr "Bar != 'testing'"
          parameterOverrides:
          - name: Foo
            value: hello
          fileOverrides:
          - path: xlr-pipeline.yml
            renameTo: xlr-pipeline2-new.yml
            writeIf: TestDepends

        files:
        - path: xld-environment.yml.tmpl
        - path: xld-infrastructure.yml.tmpl
        - path: xlr-pipeline.yml`

	httpmock.RegisterResponder(
		"GET",
		mockEndpoint+"aws/compose-2/blueprint.yaml",
		httpmock.NewStringResponder(200, yaml),
	)

	// aws/compose-3
	yaml = `
      apiVersion: xl/v2
      kind: Blueprint
      metadata:
        name: Test Project

      spec:
        includeBefore:
        - blueprint: aws/compose
          parameterOverrides:
          - name: Bar
            value: hello
            promptIf: !expr "2 > 1"
        includeAfter:
        - blueprint: aws/compose-2
          includeIf: !expr "2 < 1"
          parameterOverrides:
          - name: Bar
            value: hello
            promptIf: !expr "2 > 1"`

	httpmock.RegisterResponder(
		"GET",
		mockEndpoint+"aws/compose-3/blueprint.yaml",
		httpmock.NewStringResponder(200, yaml),
	)

	return c
}

func TestConstructLocalBlueprintContext(t *testing.T) {
	pwd, _ := os.Getwd()
	localPath := strings.Replace(pwd, path.Join("pkg", "blueprint"), path.Join("templates", "test"), -1)
	defer httpmock.DeactivateAndReset()

	t.Run("should error when local blueprint path is invalid", func(t *testing.T) {
		c, err := ConstructLocalBlueprintContext(filepath.Join(localPath, "not-there"))
		require.NotNil(t, err)
		require.Nil(t, c)
	})
	t.Run("should return valid local blueprint test context", func(t *testing.T) {
		c, err := ConstructLocalBlueprintContext(localPath)
		require.Nil(t, err)
		require.NotNil(t, c)
		assert.Equal(t, "cmd-arg", (*c.ActiveRepo).GetName())
	})
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
		c, err := ConstructBlueprintContext(v, configFile, DummyCLIVersion)

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
		c, err := ConstructBlueprintContext(v, configFile, DummyCLIVersion)

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
		c, err := ConstructBlueprintContext(v, configFile, DummyCLIVersion)

		require.NotNil(t, err)
		require.Nil(t, c)
	})
	t.Run("should add default config when current-repository is not set", func(t *testing.T) {
		v := GetViperConf(t, defaultContextYaml)
		v.Set(ViperKeyBlueprintCurrentRepository, "")
		c, err := ConstructBlueprintContext(v, configFile, DummyCLIVersion)

		require.Nil(t, err)
		require.NotNil(t, c)
		repo := *c.ActiveRepo

		assert.Equal(t, 3, len(c.DefinedRepos))
		assert.Equal(t, models.DefaultBlueprintRepositoryProvider, repo.GetProvider())
		assert.Equal(t, models.DefaultBlueprintRepositoryName, repo.GetName())
	})
	t.Run("build simple context for Blueprint repository with default current-repository", func(t *testing.T) {
		v := GetViperConf(t, defaultContextYaml)
		c, err := ConstructBlueprintContext(v, configFile, DummyCLIVersion)

		require.Nil(t, err)
		require.NotNil(t, c)
		repo := *c.ActiveRepo
		assert.Equal(t, models.ProviderHttp, repo.GetProvider())
		assert.Equal(t, "XL Http", repo.GetName())
	})
	t.Run("build simple context for Blueprint repository with provided current-repository", func(t *testing.T) {
		v := GetViperConf(t, defaultContextYaml)
		v.Set(ViperKeyBlueprintCurrentRepository, "XL Github")
		c, err := ConstructBlueprintContext(v, configFile, DummyCLIVersion)

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

	t.Run("should fetch a template from remote path", func(t *testing.T) {
		repo := getMockHttpBlueprintContext(t)
		blueprints, err := repo.initCurrentRepoClient()
		require.Nil(t, err)
		require.NotNil(t, blueprints)

		t.Run("should get file contents", func(t *testing.T) {
			contents, err := repo.fetchFileContents("aws/monolith/test.yaml", false)
			require.Nil(t, err)
			require.NotNil(t, contents)
			assert.NotEmptyf(t, string(*contents), "mock blueprint file content is empty")
			assert.Equal(t, "sample test text\nwith a new line", string(*contents))
		})

		t.Run("should error on non-existing remote file path", func(t *testing.T) {
			_, err := repo.fetchFileContents("non-existing-path/file.yaml", true)
			require.NotNil(t, err)
			assert.Equal(t, "Get http://mock.repo.server.com/non-existing-path/file.yaml.tmpl: no responder found", err.Error())
		})
	})
}

// utility function tests

func TestBlueprintContext_parseDefinitionFile(t *testing.T) {
	defer httpmock.DeactivateAndReset()
	repo := getMockHttpBlueprintContext(t)
	blueprints, err := repo.initCurrentRepoClient()
	require.Nil(t, err)
	require.NotNil(t, blueprints)

	type args struct {
		blueprint    *models.BlueprintRemote
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
				blueprints["test"],
				"test",
			},
			nil,
			true,
		},
		{
			"should parse blueprint definition file",
			*repo,
			args{
				blueprints["aws/monolith"],
				"aws/monolith",
			},
			[]TemplateConfig{
				{Path: "xld-environment.yml.tmpl", FullPath: "aws/monolith/xld-environment.yml.tmpl"},
				{Path: "xld-infrastructure.yml.tmpl", FullPath: "aws/monolith/xld-infrastructure.yml.tmpl"},
				{Path: "xlr-pipeline.yml", FullPath: "aws/monolith/xlr-pipeline.yml"},
			},
			false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			blueprintContext := tt.blueprintContext
			got, err := blueprintContext.parseDefinitionFile(tt.args.blueprint, tt.args.templatePath)
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
		repo := getMockHttpBlueprintContext(t)
		blueprints, err := repo.initCurrentRepoClient()
		require.Nil(t, err)
		require.NotNil(t, blueprints)
		require.Len(t, blueprints, 7)
	})
}

func TestBlueprintContext_parseRepositoryTree(t *testing.T) {
	defer httpmock.DeactivateAndReset()
	repo := getMockHttpBlueprintContext(t)
	(*repo.ActiveRepo).Initialize()
	tests := []struct {
		name    string
		want    int
		wantErr bool
	}{
		{
			"should fetch 2 blueprints",
			7,
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
    url: https://dist.xebialabs.com/public/blueprints/${CLIVersion}/
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
    url: https://dist.xebialabs.com/public/blueprints/${CLIVersion}/
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
    url: https://dist.xebialabs.com/public/blueprints/${CLIVersion}/`,
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
    url: https://dist.xebialabs.com/public/blueprints/${CLIVersion}/`,
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
    url: https://dist.xebialabs.com/public/blueprints/${CLIVersion}/`,
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
    url: https://dist.xebialabs.com/public/blueprints/${CLIVersion}/`,
		},
	}

	confPath, err := ioutil.TempDir("", "xebialabsconfig")
	if err != nil {
		t.Error(err)
	}
	configfile := filepath.Join(confPath, "config.yaml")
	defer os.RemoveAll(confPath)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			v := viper.New()
			v.SetConfigType("yaml")
			err := v.ReadConfig(bytes.NewBuffer([]byte(tt.vYaml)))
			require.Nil(t, err)

			originalConfigBytes := []byte(tt.vYaml)
			ioutil.WriteFile(configfile, originalConfigBytes, 0755)

			gotTemp, got, repoName, err := GetDefaultBlueprintViperConfig(v, configfile)
			require.Nil(t, err)
			require.NotNil(t, repoName)
			require.NotNil(t, got)
			require.NotNil(t, gotTemp)

			c := util.SortMapStringInterface(got.AllSettings())
			bs, err := yaml.Marshal(c)
			require.Nil(t, err)
			bss := string(bs)

			c2 := util.SortMapStringInterface(gotTemp.AllSettings())
			bs2, err := yaml.Marshal(c2)
			require.Nil(t, err)
			bss2 := string(bs2)

			v2 := viper.New()
			v2.SetConfigType("yaml")
			err = v2.ReadConfig(bytes.NewBuffer([]byte(tt.want)))
			require.Nil(t, err)
			v2setsSorted := util.SortMapStringInterface(v2.AllSettings())
			bs1, err := yaml.Marshal(v2setsSorted)
			require.Nil(t, err)
			bs1s := string(bs1)

			assert.Equal(t, bs1s, bss)
			assert.Equal(t, bs1s, bss2)

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

	tests := []struct {
		name     string
		v        string
		vFile    string
		want     string
		wantFile string
		wantErr  bool
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
    url: https://dist.xebialabs.com/public/blueprints/${CLIVersion}/
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
    url: https://dist.xebialabs.com/public/blueprints/${CLIVersion}/
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
    url: https://dist.xebialabs.com/public/blueprints/${CLIVersion}/
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
    url: https://dist.xebialabs.com/public/blueprints/${CLIVersion}/
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
    url: https://dist.xebialabs.com/public/blueprints/${CLIVersion}/`,
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
    url: https://dist.xebialabs.com/public/blueprints/${CLIVersion}/`,
			false,
		},
		{
			"should return with default config added when default repo doesn't exist in repositories and when file config is different from current config",
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
  url: http://xl-deploy:4516/
  authmethod: http
xl-release:
  username: admin
  password: admin123
  url: http://xl-release:5516/
  authmethod: http
blueprint:
  current-repository: XL Blueprints 2
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
    url: https://dist.xebialabs.com/public/blueprints/${CLIVersion}/`,
			`
xl-deploy:
  username: admin
  password: admin123
  url: http://xl-deploy:4516/
  authmethod: http
xl-release:
  username: admin
  password: admin123
  url: http://xl-release:5516/
  authmethod: http
blueprint:
  current-repository: XL Blueprints 2
  repositories:
  - name: XL Blueprints 2
    type: http
    url: https://dist.xebialabs.com/public/blueprints/
  - name: XL Blueprints
    type: http
    url: https://dist.xebialabs.com/public/blueprints/${CLIVersion}/`,
			false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			v := viper.New()
			v.SetConfigType("yaml")
			err := v.ReadConfig(bytes.NewBuffer([]byte(tt.v)))
			require.Nil(t, err)

			originalConfigBytes := []byte(tt.vFile)
			ioutil.WriteFile(configfile, originalConfigBytes, 0755)

			got, repoName, err := CreateOrUpdateBlueprintConfig(v, configfile)
			require.Nil(t, err)
			require.NotNil(t, repoName)

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

			v3 := viper.New()
			v3.SetConfigType("yaml")
			err = v3.ReadConfig(bytes.NewBuffer([]byte(tt.wantFile)))
			require.Nil(t, err)
			v3setsSorted := util.SortMapStringInterface(v3.AllSettings())
			bs2, err := yaml.Marshal(v3setsSorted)
			require.Nil(t, err)
			bs2s := string(bs2)

			assert.Equal(t, bs2s, string(file))
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

func Test_parseTemplateMetadata(t *testing.T) {
	templatePath := "test/blueprints"
	blueprintRepository := BlueprintContext{}

	type args struct {
		ymlContent          string
		templatePath        string
		blueprintRepository *BlueprintContext
	}
	tests := []struct {
		name    string
		args    args
		want    *BlueprintConfig
		wantErr bool
	}{
		{
			"should error when invalid apiVersion is used",
			args{
				`
                  apiVersion: xl/v3
                  kind: Blueprint
                  metadata:
                    name: Test Project
                `,
				templatePath,
				&blueprintRepository,
			},
			nil,
			true,
		},
		{
			"should error when invalid fields are used for apiVersion",
			args{
				`
                  apiVersion: xl/v1
                  kind: Blueprint
                  metadata:
                    name: Test Project
                `,
				templatePath,
				&blueprintRepository,
			},
			nil,
			true,
		},
		{
			"should parse v1 when apiversion is xl/v1",
			args{
				`
                  apiVersion: xl/v1
                  kind: Blueprint
                  metadata:
                    projectName: Test Project
                `,
				templatePath,
				&blueprintRepository,
			},
			&BlueprintConfig{
				ApiVersion: "xl/v1",
				Kind:       "Blueprint",
				Metadata: Metadata{
					Name: "Test Project",
				},
				TemplateConfigs: []TemplateConfig{},
				Variables:       []Variable{},
			},
			false,
		},
		{
			"should parse v2 when apiversion is xl/v2",
			args{
				`
                  apiVersion: xl/v2
                  kind: Blueprint
                  metadata:
                    name: Test Project
                `,
				templatePath,
				&blueprintRepository,
			},
			&BlueprintConfig{
				ApiVersion: "xl/v2",
				Kind:       "Blueprint",
				Metadata: Metadata{
					Name: "Test Project",
				},
				TemplateConfigs: []TemplateConfig{},
				Variables:       []Variable{},
				Include:         []IncludedBlueprintProcessed{},
			},
			false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			yml := []byte(tt.args.ymlContent)
			got, err := parseTemplateMetadata(&yml, tt.args.templatePath, tt.args.blueprintRepository)
			if (err != nil) != tt.wantErr {
				t.Errorf("parseTemplateMetadata() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr {
				assert.Equal(t, tt.want, got)
			}
		})
	}
}

package xl

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"net/url"
	"os"
	"path/filepath"
	"testing"

	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v2"
)

func parseURIWithoutError(uri string) url.URL {
	url, err := url.ParseRequestURI(uri)
	if err != nil {
		panic(err)
	}
	return *url
}

func TestGetTemplateRegistries(t *testing.T) {
	t.Run("get Template Registries from valid config", func(t *testing.T) {
		yamlConfig := `
template-registries:
- name: default
  url: https://s3.amazonaws.com/xl-cli/blueprints
- name: custom
  url: https://s3.amazonaws.com/xl-cli/blueprints/
  username: admin
  password: admin
- url: http://test.com
`
		v := viper.New()
		v.SetConfigType("yaml")
		err := v.ReadConfig(bytes.NewBuffer([]byte(yamlConfig)))
		require.Nil(t, err)
		out, err := getTemplateRegistries(v)
		require.Nil(t, err)
		exp := []TemplateRegistry{
			TemplateRegistry{Name: "default", URL: parseURIWithoutError("https://s3.amazonaws.com/xl-cli/blueprints"), Username: "", Password: ""},
			TemplateRegistry{Name: "custom", URL: parseURIWithoutError("https://s3.amazonaws.com/xl-cli/blueprints/"), Username: "admin", Password: "admin"},
			TemplateRegistry{Name: "", URL: parseURIWithoutError("http://test.com"), Username: "", Password: ""},
		}
		assert.Equal(t, exp, out)
	})
	t.Run("should error on incomplete config", func(t *testing.T) {
		yamlConfig := `
template-registries:
- url: http://test.com
- name: default
`
		v := viper.New()
		v.SetConfigType("yaml")
		err := v.ReadConfig(bytes.NewBuffer([]byte(yamlConfig)))
		require.Nil(t, err)
		_, err = getTemplateRegistries(v)
		require.NotNil(t, err)
	})
	t.Run("throw error for invalid config", func(t *testing.T) {
		yamlConfig := `
template-registries:
- name: default
  url: invalidurl;
`
		v := viper.New()
		v.SetConfigType("yaml")
		err := v.ReadConfig(bytes.NewBuffer([]byte(yamlConfig)))
		require.Nil(t, err)
		_, err = getTemplateRegistries(v)
		require.NotNil(t, err)
	})
}

func TestContextBuilder(t *testing.T) {

	IsVerbose = true

	t.Run("build simple context for XL Deploy", func(t *testing.T) {
		v := viper.New()
		v.Set("xl-deploy.url", "http://testxld:6154")
		v.Set("xl-deploy.username", "deployer")
		v.Set("xl-deploy.password", "d3ploy1t")

		c, err := BuildContext(v, nil, []string{})

		assert.Nil(t, err)
		assert.NotNil(t, c)
		assert.NotNil(t, c.XLDeploy)
		assert.Equal(t, "http://testxld:6154", c.XLDeploy.(*XLDeployServer).Server.(*SimpleHTTPServer).Url.String())
		assert.Equal(t, "deployer", c.XLDeploy.(*XLDeployServer).Server.(*SimpleHTTPServer).Username)
		assert.Equal(t, "d3ploy1t", c.XLDeploy.(*XLDeployServer).Server.(*SimpleHTTPServer).Password)
		assert.Nil(t, c.XLRelease)
	})

	t.Run("build simple context for XL Release", func(t *testing.T) {
		v := viper.New()
		v.Set("xl-release.url", "http://masterxlr:6155")
		v.Set("xl-release.username", "releaser")
		v.Set("xl-release.password", "r3l34s3")

		c, err := BuildContext(v, nil, []string{})

		assert.Nil(t, err)
		assert.NotNil(t, c)
		assert.Nil(t, c.XLDeploy)
		assert.NotNil(t, c.XLRelease)
		assert.Equal(t, "http://masterxlr:6155", c.XLRelease.(*XLReleaseServer).Server.(*SimpleHTTPServer).Url.String())
		assert.Equal(t, "releaser", c.XLRelease.(*XLReleaseServer).Server.(*SimpleHTTPServer).Username)
		assert.Equal(t, "r3l34s3", c.XLRelease.(*XLReleaseServer).Server.(*SimpleHTTPServer).Password)
	})

	t.Run("build full context for XL Deploy and XL Release", func(t *testing.T) {
		v := viper.New()
		v.Set("xl-deploy.url", "http://testxld:6154")
		v.Set("xl-deploy.username", "deployer")
		v.Set("xl-deploy.password", "d3ploy1t")
		v.Set("xl-deploy.applications-home", "Applications/home/folder")
		v.Set("xl-deploy.configuration-home", "Configuration/home/folder")
		v.Set("xl-deploy.environments-home", "Environments/home/folder")
		v.Set("xl-deploy.infrastructure-home", "Infrastructure/home/folder")
		v.Set("xl-release.url", "http://masterxlr:6155")
		v.Set("xl-release.username", "releaser")
		v.Set("xl-release.password", "r3l34s3")
		v.Set("xl-release.home", "XLR/home/folder")

		c, err := BuildContext(v, nil, []string{})

		assert.Nil(t, err)
		assert.NotNil(t, c)
		assert.NotNil(t, c.XLDeploy)
		assert.Equal(t, "http://testxld:6154", c.XLDeploy.(*XLDeployServer).Server.(*SimpleHTTPServer).Url.String())
		assert.Equal(t, "deployer", c.XLDeploy.(*XLDeployServer).Server.(*SimpleHTTPServer).Username)
		assert.Equal(t, "d3ploy1t", c.XLDeploy.(*XLDeployServer).Server.(*SimpleHTTPServer).Password)
		assert.Equal(t, "Applications/home/folder", c.XLDeploy.(*XLDeployServer).ApplicationsHome)
		assert.Equal(t, "Configuration/home/folder", c.XLDeploy.(*XLDeployServer).ConfigurationHome)
		assert.Equal(t, "Environments/home/folder", c.XLDeploy.(*XLDeployServer).EnvironmentsHome)
		assert.Equal(t, "Infrastructure/home/folder", c.XLDeploy.(*XLDeployServer).InfrastructureHome)
		assert.NotNil(t, c.XLRelease)
		assert.Equal(t, "http://masterxlr:6155", c.XLRelease.(*XLReleaseServer).Server.(*SimpleHTTPServer).Url.String())
		assert.Equal(t, "releaser", c.XLRelease.(*XLReleaseServer).Server.(*SimpleHTTPServer).Username)
		assert.Equal(t, "r3l34s3", c.XLRelease.(*XLReleaseServer).Server.(*SimpleHTTPServer).Password)
		assert.Equal(t, "XLR/home/folder", c.XLRelease.(*XLReleaseServer).Home)
	})

	t.Run("build context without values", func(t *testing.T) {
		v := viper.New()

		c, err := BuildContext(v, nil, []string{})

		assert.Nil(t, err)
		assert.NotNil(t, c)
		assert.NotNil(t, c.values)
	})

	t.Run("build context from YAML", func(t *testing.T) {
		yamlConfig := `xl-deploy:
  url: http://xld.example.com:4516
  username: admin
  password: 3dm1n
`

		v := viper.New()
		v.SetConfigType("yaml")
		err := v.ReadConfig(bytes.NewBuffer([]byte(yamlConfig)))
		require.Nil(t, err)

		c, err := BuildContext(v, nil, []string{})

		require.Nil(t, err)
		require.NotNil(t, c)
		require.NotNil(t, c.XLDeploy)
		require.Equal(t, "http://xld.example.com:4516", c.XLDeploy.(*XLDeployServer).Server.(*SimpleHTTPServer).Url.String())
	})

	t.Run("do not write config file if xl-deploy.password was stored in the config file but was overridden", func(t *testing.T) {
		configdir, err := ioutil.TempDir("", "xebialabsconfig")
		if err != nil {
			t.Error(err)
			return
		}

		defer os.RemoveAll(configdir)
		configfile := filepath.Join(configdir, "config.yaml")
		originalConfigBytes := []byte(`xl-deploy:
  url: http://testxld:6154
  username: testuser
  password: t3stus3r
`)
		ioutil.WriteFile(configfile, originalConfigBytes, 0755)

		v := viper.New()
		v.SetConfigFile(configfile)
		v.ReadInConfig()
		v.Set("xl-deploy.password", "t3st")

		c, err := BuildContext(v, nil, []string{})

		assert.Nil(t, err)
		assert.NotNil(t, c)
		assert.Nil(t, c.XLRelease)
		assert.NotNil(t, c.XLDeploy)
		assert.Equal(t, "http://testxld:6154", c.XLDeploy.(*XLDeployServer).Server.(*SimpleHTTPServer).Url.String())
		assert.Equal(t, "testuser", c.XLDeploy.(*XLDeployServer).Server.(*SimpleHTTPServer).Username)
		assert.Equal(t, "t3st", c.XLDeploy.(*XLDeployServer).Server.(*SimpleHTTPServer).Password)

		configBytes, err := ioutil.ReadFile(configfile)
		assert.Equal(t, originalConfigBytes, configBytes)
	})

	t.Run("write config file and do not unfold values", func(t *testing.T) {
		configdir, err := ioutil.TempDir("", "xebialabsconfig")
		if err != nil {
			t.Error(err)
			return
		}

		defer os.RemoveAll(configdir)
		configfile := filepath.Join(configdir, "config.yaml")
		require.Nil(t, err)
		ioutil.WriteFile(configfile, []byte(`xl-deploy:
  url: http://testxld:6154
  username: testuser
  password: t3st
`), 0755)

		v := viper.New()
		v.SetConfigFile(configfile)
		v.ReadInConfig()

		_, err = BuildContext(v,  nil, []string{})
		require.Nil(t, err)

		configbytes, err := ioutil.ReadFile(configfile)
		require.Nil(t, err)

		parsed := make(map[interface{}]interface{})
		err = yaml.Unmarshal(configbytes, parsed)
		require.Nil(t, err)

	})

	t.Run("return error when password is missing", func(t *testing.T) {
		configdir, err := ioutil.TempDir("", "xebialabsconfig")
		if err != nil {
			t.Error(err)
			return
		}

		defer os.RemoveAll(configdir)
		configfile := filepath.Join(configdir, "config.yaml")
		originalConfigBytes := []byte(`xl-deploy:
  url: http://testxld:6154
`)
		ioutil.WriteFile(configfile, originalConfigBytes, 0755)

		v := viper.New()
		v.SetConfigFile(configfile)
		v.ReadInConfig()

		c, err := BuildContext(v, nil, []string{})

		assert.NotNil(t, err)
		assert.Nil(t, c)
		assert.Equal(t, "configuration property xl-deploy.username is required if xl-deploy.url is set", err.Error())
	})

	assertEnvKeyNotPresent := func(key string) {
		_, exists := os.LookupEnv(key)
		assert.False(t, exists)
	}

	assertEnvKeyEqual := func(key string, value string) {
		envValue, exists := os.LookupEnv(key)
		assert.True(t, exists)
		assert.Equal(t, value, envValue)
	}

	assertNoServerCredentials := func(serverKind string) {
		assertEnvKeyNotPresent(fmt.Sprintf("XL_%s_CREDENTIALS", serverKind))
		assertEnvKeyNotPresent(fmt.Sprintf("XL_%s_USERNAME", serverKind))
		assertEnvKeyNotPresent(fmt.Sprintf("XL_%s_PASSWORD", serverKind))
	}

	assertServerCredentials := func(serverKind string, credentials string, user string, password string) {
		assertEnvKeyEqual(fmt.Sprintf("XL_%s_CREDENTIALS", serverKind), credentials)
		assertEnvKeyEqual(fmt.Sprintf("XL_%s_USERNAME", serverKind), user)
		assertEnvKeyEqual(fmt.Sprintf("XL_%s_PASSWORD", serverKind), password)
	}

	cleanupServerCredentials := func(serverKind string) {
		os.Unsetenv(fmt.Sprintf("XL_%s_CREDENTIALS", serverKind))
		os.Unsetenv(fmt.Sprintf("XL_%s_USERNAME", serverKind))
		os.Unsetenv(fmt.Sprintf("XL_%s_PASSWORD", serverKind))
	}

	cleanupAllCredentials := func() {
		cleanupServerCredentials("DEPLOY")
		cleanupServerCredentials("RELEASE")
	}

	t.Run("parse credentials from XL_{SERVER_KIND}_CREDENTIALS env variable and fill in username and password", func(t *testing.T) {
		assertNoServerCredentials("DEPLOY")
		assertNoServerCredentials("RELEASE")

		os.Setenv("XL_DEPLOY_CREDENTIALS", "admin:qwerty")
		os.Setenv("XL_RELEASE_CREDENTIALS", "john:mat")
		assert.Nil(t, ProcessCredentials())

		assertServerCredentials("DEPLOY", "admin:qwerty", "admin", "qwerty")
		assertServerCredentials("RELEASE", "john:mat", "john", "mat")
		cleanupAllCredentials()
	})

	t.Run("parse credentials from XL_{SERVER_KIND}_CREDENTIALS env variable and set only fields that was not specified before", func(t *testing.T) {
		assertNoServerCredentials("DEPLOY")
		assertNoServerCredentials("RELEASE")

		os.Setenv("XL_DEPLOY_USERNAME", "user1")
		os.Setenv("XL_DEPLOY_CREDENTIALS", "admin:qwerty")
		os.Setenv("XL_RELEASE_PASSWORD", "password1")
		os.Setenv("XL_RELEASE_CREDENTIALS", "john:mat")
		assert.Nil(t, ProcessCredentials())

		assertServerCredentials("DEPLOY", "admin:qwerty", "user1", "qwerty")
		assertServerCredentials("RELEASE", "john:mat", "john", "password1")
		cleanupAllCredentials()
	})

	t.Run("validate format of XL_{SERVER_KIND}_CREDENTIALS", func(t *testing.T) {
		assertNoServerCredentials("DEPLOY")
		os.Setenv("XL_DEPLOY_CREDENTIALS", "admin")
		assert.Equal(t, "environment variable XL_DEPLOY_CREDENTIALS has an invalid format. It must container a username and a password separated by a colon", ProcessCredentials().Error())
		cleanupAllCredentials()
	})

	t.Run("validate that names of values are correct", func(t *testing.T) {
		v := viper.New()

		values := make(map[string]string)
		values["!incorrectKey"] = "test value"

		_, err := BuildContext(v, &values, []string{})

		assert.NotNil(t, err)
		assert.Equal(t, "the name of the value !incorrectKey is invalid. It must start with an alphabetical character or an underscore and be followed by zero or more alphanumerical characters or underscores", err.Error())
	})


	t.Run("Should read values into context", func(t *testing.T) {
		v := viper.New()

		propfile1 := writePropFile("file1", `
test=test
test2=test2
`)
		defer os.Remove(propfile1.Name())

		valsFiles := []string{propfile1.Name()}

		context, err2 := BuildContext(v, nil, valsFiles)
		assert.Nil(t, err2)

		assert.Equal(t, "test", context.values["test"])
		assert.Equal(t, "test2", context.values["test2"])
	})

	t.Run("Should read case sensitive values", func(t *testing.T) {
		v := viper.New()

		propfile1 := writePropFile("file1", `
test=test1
TEST=test2
Test=test3
`)
		defer os.Remove(propfile1.Name())

		valsFiles := []string{propfile1.Name()}

		context, err2 := BuildContext(v, nil, valsFiles)
		assert.Nil(t, err2)

		assert.Equal(t, "test1", context.values["test"])
		assert.Equal(t, "test2", context.values["TEST"])
		assert.Equal(t, "test3", context.values["Test"])
	})

	t.Run("Should override values from value files in right order (only value files)", func(t *testing.T) {
		v := viper.New()

		propfile1 := writePropFile("file1", `
test=test
test2=test2
verifythisfilegetsread=ok
`)
		defer os.Remove(propfile1.Name())

		propfile2 := writePropFile("file2", `
test=override
test2=override2
`)
		defer os.Remove(propfile2.Name())

		valsFiles := []string{propfile1.Name(),propfile2.Name()}

		context, err2 := BuildContext(v, nil, valsFiles)
		assert.Nil(t, err2)

		assert.Equal(t, 3, len(context.values))
		assert.Equal(t, "override", context.values["test"])
		assert.Equal(t, "override2", context.values["test2"])
		assert.Equal(t, "ok", context.values["verifythisfilegetsread"])
	})

	t.Run("Should command line parameter value should override value files", func(t *testing.T) {
		v := viper.New()

		propfile1 := writePropFile("file1", `
test=test
test2=test2
verifythisfilegetsread=ok
`)
		defer os.Remove(propfile1.Name())

		valsFiles := []string{propfile1.Name()}

		values := make(map[string]string)
		values["test"] = "override"
		values["test2"] = "override2"

		context, err2 := BuildContext(v, &values, valsFiles)
		assert.Nil(t, err2)

		assert.Equal(t, 3, len(context.values))
		assert.Equal(t, "override", context.values["test"])
		assert.Equal(t, "override2", context.values["test2"])
		assert.Equal(t, "ok", context.values["verifythisfilegetsread"])
	})

	t.Run("Environment variables should override value files", func(t *testing.T) {
		v := viper.New()

		propfile1 := writePropFile("file1", `
test=test
test2=test2
verifythisfilegetsread=ok
`)
		defer os.Remove(propfile1.Name())

		valsFiles := []string{propfile1.Name()}

		os.Setenv("XL_VALUE_test", "override")
		os.Setenv("XL_VALUE_test2", "override2")

		context, err2 := BuildContext(v, nil, valsFiles)
		assert.Nil(t, err2)

		assert.Equal(t, 3, len(context.values))
		assert.Equal(t, "override", context.values["test"])
		assert.Equal(t, "override2", context.values["test2"])
		assert.Equal(t, "ok", context.values["verifythisfilegetsread"])
	})

}

func writePropFile(prefix string, content3 string) (f *os.File) {
	tmpfile, err := ioutil.TempFile("", prefix)
	if err != nil {
		panic(err)
	}
	ioutil.WriteFile(tmpfile.Name(), []byte(content3), 0755)
	return tmpfile
}

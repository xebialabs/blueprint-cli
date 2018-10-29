package xl

import (
	"bytes"
	"fmt"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v2"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"
)

func TestContextBuilder(t *testing.T) {
	t.Run("build simple context for XL Deploy", func(t *testing.T) {
		v := viper.New()
		v.Set("xl-deploy.url", "http://testxld:6154")
		v.Set("xl-deploy.username", "deployer")
		v.Set("xl-deploy.password", "d3ploy1t")

		c, err := BuildContext(v, nil)

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

		c, err := BuildContext(v, nil)

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

		c, err := BuildContext(v, nil)

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

		c, err := BuildContext(v, nil)

		assert.Nil(t, err)
		assert.NotNil(t, c)
		assert.NotNil(t, c.values)
	})

	t.Run("build context with values", func(t *testing.T) {
		v := viper.New()
		v.Set("values", map[string]string{"server_address": "server.example.com"})

		c, err := BuildContext(v, nil)

		assert.Nil(t, err)
		assert.NotNil(t, c)
		assert.NotNil(t, c.values)
		assert.Equal(t, "server.example.com", c.values["server_address"])
	})

	t.Run("build context from YAML", func(t *testing.T) {
		yamlConfig := `xl-deploy:
  url: http://xld.example.com:4516
  username: admin
  password: 3dm1n
values:
  server_address: server.example.com
`

		v := viper.New()
		v.SetConfigType("yaml")
		err := v.ReadConfig(bytes.NewBuffer([]byte(yamlConfig)))
		require.Nil(t, err)

		c, err := BuildContext(v, nil)

		require.Nil(t, err)
		require.NotNil(t, c)
		require.NotNil(t, c.XLDeploy)
		require.Equal(t, "http://xld.example.com:4516", c.XLDeploy.(*XLDeployServer).Server.(*SimpleHTTPServer).Url.String())
		require.NotNil(t, c.values)
		require.Equal(t, "server.example.com", c.values["server_address"])
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

		c, err := BuildContext(v, nil)

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
values:
  server_username: root
  server_hostname: server.example.com
`), 0755)

		v := viper.New()
		v.SetConfigFile(configfile)
		v.ReadInConfig()

		_, err = BuildContext(v,  nil)
		require.Nil(t, err)

		configbytes, err := ioutil.ReadFile(configfile)
		require.Nil(t, err)

		parsed := make(map[interface{}]interface{})
		err = yaml.Unmarshal(configbytes, parsed)
		require.Nil(t, err)

		values := parsed["values"].(map[interface{}]interface{})
		require.NotNil(t, values)
		serverUsername := values["server_username"].(string)
		require.Equal(t, "root", serverUsername)
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

		c, err := BuildContext(v, nil)

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
		v.Set("values", map[string]string{"!incorrectKey": "test value"})

		_, err := BuildContext(v, nil)

		assert.NotNil(t, err)
		assert.Equal(t, "the name of the value !incorrectKey is invalid. It must start with an alphabetical character or an underscore and be followed by zero or more alphanumerical characters or underscores", err.Error())
	})

}

package xl

import (
	"bytes"
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

		c, err := BuildContext(v, nil, nil)

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

		c, err := BuildContext(v, nil, nil)

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

		c, err := BuildContext(v, nil, nil)

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

	t.Run("build context without values or secrets", func(t *testing.T) {
		v := viper.New()

		c, err := BuildContext(v, nil, nil)

		assert.Nil(t, err)
		assert.NotNil(t, c)
		assert.NotNil(t, c.values)
		assert.NotNil(t, c.secrets)
	})

	t.Run("build context with values", func(t *testing.T) {
		v := viper.New()
		v.Set("values", map[string]string{"server.address": "server.example.com"})

		c, err := BuildContext(v, nil, nil)

		assert.Nil(t, err)
		assert.NotNil(t, c)
		assert.NotNil(t, c.values)
		assert.Equal(t, "server.example.com", c.values["server.address"])
	})

	t.Run("build context with secrets", func(t *testing.T) {
		v := viper.New()
		v.Set("secrets", map[string]string{"server.password": "r00t"})

		c, err := BuildContext(v, nil, nil)

		assert.Nil(t, err)
		assert.NotNil(t, c)
		assert.NotNil(t, c.values)
		assert.Equal(t, "r00t", c.secrets["server.password"])
	})

	t.Run("build context from YAML", func(t *testing.T) {
		yamlConfig := `xl-deploy:
  url: http://xld.example.com:4516
  username: admin
  password: 3dm1n
values:
  server.address: server.example.com
secrets:
  server.password: r00t`

		v := viper.New()
		v.SetConfigType("yaml")
		err := v.ReadConfig(bytes.NewBuffer([]byte(yamlConfig)))
		require.Nil(t, err)

		c, err := BuildContext(v, nil, nil)

		require.Nil(t, err)
		require.NotNil(t, c)
		require.NotNil(t, c.XLDeploy)
		require.Equal(t, "http://xld.example.com:4516", c.XLDeploy.(*XLDeployServer).Server.(*SimpleHTTPServer).Url.String())
		require.NotNil(t, c.values)
		require.Equal(t, "server.example.com", c.values["server.address"])
		require.Equal(t, "r00t", c.secrets["server.password"])
	})

	t.Run("write config file if xl-deploy.password was not obfuscrypted", func(t *testing.T) {
		configdir, err := ioutil.TempDir("", "xebialabsconfig")
		if err != nil {
			t.Error(err)
			return
		}

		defer os.RemoveAll(configdir)
		configfile := filepath.Join(configdir, "config.yaml")
		ioutil.WriteFile(configfile, []byte(`xl-deploy:
  url: http://testxld:6154
  username: testuser
  password: t3st
`), 0755)

		v := viper.New()
		v.SetConfigFile(configfile)
		v.ReadInConfig()

		c, err := BuildContext(v, nil, nil)

		assert.Nil(t, err)
		assert.NotNil(t, c)
		assert.Nil(t, c.XLRelease)
		assert.NotNil(t, c.XLDeploy)
		assert.Equal(t, "http://testxld:6154", c.XLDeploy.(*XLDeployServer).Server.(*SimpleHTTPServer).Url.String())
		assert.Equal(t, "testuser", c.XLDeploy.(*XLDeployServer).Server.(*SimpleHTTPServer).Username)
		assert.Equal(t, "t3st", c.XLDeploy.(*XLDeployServer).Server.(*SimpleHTTPServer).Password)

		configbytes, err := ioutil.ReadFile(configfile)
		assert.Nil(t, err)
		parsed := make(map[interface{}]interface{})
		err = yaml.Unmarshal(configbytes, parsed)
		assert.Nil(t, err)
		xldeployConfig := parsed["xl-deploy"].(map[interface{}]interface{})
		assert.Equal(t, "http://testxld:6154", xldeployConfig["url"].(string))
		assert.Equal(t, "testuser", xldeployConfig["username"].(string))
		obfuscryptedPassword := xldeployConfig["password"].(string)
		deobfuscryptedPassword, err := Deobfuscrypt(obfuscryptedPassword)
		assert.Nil(t, err)
		assert.Equal(t, "t3st", deobfuscryptedPassword)
	})

	t.Run("do not write config file if xl-deploy.password was already obfuscrypted", func(t *testing.T) {
		configdir, err := ioutil.TempDir("", "xebialabsconfig")
		if err != nil {
			t.Error(err)
			return
		}

		defer os.RemoveAll(configdir)
		configfile := filepath.Join(configdir, "config.yaml")
		obfuscryptedXLDeployPassword, err := Obfuscrypt("t3st")
		originalConfigBytes := []byte(`xl-deploy:
  url: http://testxld:6154
  username: testuser
  password: ` + obfuscryptedXLDeployPassword + `
`)
		ioutil.WriteFile(configfile, originalConfigBytes, 0755)

		v := viper.New()
		v.SetConfigFile(configfile)
		v.ReadInConfig()

		c, err := BuildContext(v, nil, nil)

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

	t.Run("do not write config file if xl-deploy.password was not obfuscrupted but was not stored in the config file", func(t *testing.T) {
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
`)
		ioutil.WriteFile(configfile, originalConfigBytes, 0755)

		v := viper.New()
		v.SetConfigFile(configfile)
		v.ReadInConfig()
		v.Set("xl-deploy.password", "t3st")

		c, err := BuildContext(v, nil, nil)

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

	t.Run("do not write config file if xl-deploy.password was stored in the config file but was overridden", func(t *testing.T) {
		configdir, err := ioutil.TempDir("", "xebialabsconfig")
		if err != nil {
			t.Error(err)
			return
		}

		obfuscryptedXLDeployPassword, err := Obfuscrypt("t3stus3r")
		defer os.RemoveAll(configdir)
		configfile := filepath.Join(configdir, "config.yaml")
		originalConfigBytes := []byte(`xl-deploy:
  url: http://testxld:6154
  username: testuser
  password: ` + obfuscryptedXLDeployPassword + `
`)
		ioutil.WriteFile(configfile, originalConfigBytes, 0755)

		v := viper.New()
		v.SetConfigFile(configfile)
		v.ReadInConfig()
		v.Set("xl-deploy.password", "t3st")

		c, err := BuildContext(v, nil, nil)

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

	t.Run("do not write config file if xl-deploy.password was already obfuscrypted and xl-release.password was not obfuscrypted but not stored in the config file", func(t *testing.T) {
		configdir, err := ioutil.TempDir("", "xebialabsconfig")
		if err != nil {
			t.Error(err)
			return
		}

		defer os.RemoveAll(configdir)
		configfile := filepath.Join(configdir, "config.yaml")
		obfuscryptedXLDeployPassword, err := Obfuscrypt("t3st")
		originalConfigBytes := []byte(`xl-release:
  url: http://testxlr:6155
  username: releaseuser
xl-deploy:
  url: http://testxld:6154
  username: testuser
  password: ` + obfuscryptedXLDeployPassword + `
`)
		ioutil.WriteFile(configfile, originalConfigBytes, 0755)

		v := viper.New()
		v.SetConfigFile(configfile)
		v.ReadInConfig()
		v.Set("xl-release.password", "r3l34s3")

		c, err := BuildContext(v, nil, nil)

		assert.Nil(t, err)
		assert.NotNil(t, c)
		assert.NotNil(t, c.XLDeploy)
		assert.Equal(t, "http://testxld:6154", c.XLDeploy.(*XLDeployServer).Server.(*SimpleHTTPServer).Url.String())
		assert.Equal(t, "testuser", c.XLDeploy.(*XLDeployServer).Server.(*SimpleHTTPServer).Username)
		assert.Equal(t, "t3st", c.XLDeploy.(*XLDeployServer).Server.(*SimpleHTTPServer).Password)
		assert.NotNil(t, c.XLRelease)
		assert.Equal(t, "http://testxlr:6155", c.XLRelease.(*XLReleaseServer).Server.(*SimpleHTTPServer).Url.String())
		assert.Equal(t, "releaseuser", c.XLRelease.(*XLReleaseServer).Server.(*SimpleHTTPServer).Username)
		assert.Equal(t, "r3l34s3", c.XLRelease.(*XLReleaseServer).Server.(*SimpleHTTPServer).Password)

		configBytes, err := ioutil.ReadFile(configfile)
		assert.Equal(t, originalConfigBytes, configBytes)
	})

	t.Run("write config file if a secret was not obfuscrypted", func(t *testing.T) {
		configdir, err := ioutil.TempDir("", "xebialabsconfig")
		if err != nil {
			t.Error(err)
			return
		}

		defer os.RemoveAll(configdir)
		configfile := filepath.Join(configdir, "config.yaml")
		obfuscryptedXLDeployPassword, err := Obfuscrypt("t3st")
		require.Nil(t, err)
		ioutil.WriteFile(configfile, []byte(`xl-deploy:
  url: http://testxld:6154
  username: testuser
  password: ` + obfuscryptedXLDeployPassword + `
secrets:
  server.password: r00t
`), 0755)

		v := viper.New()
		v.SetConfigFile(configfile)
		v.ReadInConfig()

		c, err := BuildContext(v, nil, nil)
		require.Nil(t, err)
		require.NotNil(t, c)
		require.NotNil(t, c.secrets)
		require.Equal(t, "r00t", c.secrets["server.password"])

		configbytes, err := ioutil.ReadFile(configfile)
		require.Nil(t, err)

		parsed := make(map[interface{}]interface{})
		err = yaml.Unmarshal(configbytes, parsed)
		require.Nil(t, err)

		secrets := parsed["secrets"].(map[interface{}]interface{})
		require.NotNil(t, secrets)

		obfuscryptedSecret := secrets["server.password"].(string)
		require.NotNil(t, obfuscryptedSecret)

		deobfuscryptedSecret, err := Deobfuscrypt(obfuscryptedSecret)
		require.Nil(t, err)
		require.Equal(t, "r00t", deobfuscryptedSecret)

		_, containsValues := parsed["values"]
		require.False(t, containsValues)
	})

	t.Run("write config file and do not unfold values", func(t *testing.T) {
		configdir, err := ioutil.TempDir("", "xebialabsconfig")
		if err != nil {
			t.Error(err)
			return
		}

		defer os.RemoveAll(configdir)
		configfile := filepath.Join(configdir, "config.yaml")
		obfuscryptedXLDeployPassword, err := Obfuscrypt("t3st")
		require.Nil(t, err)
		ioutil.WriteFile(configfile, []byte(`xl-deploy:
  url: http://testxld:6154
  username: testuser
  password: ` + obfuscryptedXLDeployPassword + `
secrets:
  server.password: r00t
values:
  server.username: root
  server.hostname: server.example.com
`), 0755)

		v := viper.New()
		v.SetConfigFile(configfile)
		v.ReadInConfig()

		_, err = BuildContext(v, nil, nil)
		require.Nil(t, err)

		configbytes, err := ioutil.ReadFile(configfile)
		require.Nil(t, err)

		parsed := make(map[interface{}]interface{})
		err = yaml.Unmarshal(configbytes, parsed)
		require.Nil(t, err)

		values := parsed["values"].(map[interface{}]interface{})
		require.NotNil(t, values)
		serverUsername := values["server.username"].(string)
		require.Equal(t, "root", serverUsername)
	})

	t.Run("do not write config file if a secret was not obfuscrypted", func(t *testing.T) {
		configdir, err := ioutil.TempDir("", "xebialabsconfig")
		if err != nil {
			t.Error(err)
			return
		}

		defer os.RemoveAll(configdir)
		configfile := filepath.Join(configdir, "config.yaml")
		obfuscryptedXLDeployPassword, err := Obfuscrypt("t3st")
		obfusctyptedSecret, err := Obfuscrypt("r00t")
		require.Nil(t, err)
		originalConfigBytes := []byte(`xl-deploy:
  url: http://testxld:6154
  username: testuser
  password: ` + obfuscryptedXLDeployPassword + `
secrets:
  server.password: ` + obfusctyptedSecret + `
`)
		ioutil.WriteFile(configfile, originalConfigBytes, 0755)

		v := viper.New()
		v.SetConfigFile(configfile)
		v.ReadInConfig()

		c, err := BuildContext(v, nil, nil)

		require.Nil(t, err)
		require.NotNil(t, c)
		require.NotNil(t, c.secrets)
		require.Equal(t, "r00t", c.secrets["server.password"])

		configBytes, err := ioutil.ReadFile(configfile)
		assert.Equal(t, originalConfigBytes, configBytes)
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

		c, err := BuildContext(v, nil, nil)

		assert.NotNil(t, err)
		assert.Nil(t, c)
		assert.Equal(t, "configuration property xl-deploy.username is required if xl-deploy.url is set", err.Error())
	})


}

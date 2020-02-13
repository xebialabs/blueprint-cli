package xl

import (
	"bytes"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	"github.com/spf13/cobra"
	"github.com/xebialabs/xl-blueprint/pkg/blueprint"
	"github.com/xebialabs/xl-blueprint/pkg/util"

	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v2"
)

var TestCmd = &cobra.Command{
	Use:   "xl",
	Short: "Test command",
}

const DummyCLIVersion = "9.0.0-SNAPSHOT"

func TestContextBuilder(t *testing.T) {
	util.IsVerbose = true
	blueprint.WriteConfigFile = false

	t.Run("build context without values", func(t *testing.T) {
		v, _, _, _ := blueprint.GetDefaultBlueprintViperConfig(viper.New(), "")

		c, err := BuildContext(v, DummyCLIVersion)

		assert.Nil(t, err)
		assert.NotNil(t, c)
	})

	t.Run("build context from YAML", func(t *testing.T) {
		yamlConfig := `
blueprint:
  current-repository: XL Blueprints
  repositories:
  - name: XL Blueprints
    type: http
    url: https://dist.xebialabs.com/public/blueprints/
`
		v := viper.New()
		v.SetConfigType("yaml")
		err := v.ReadConfig(bytes.NewBuffer([]byte(yamlConfig)))
		require.Nil(t, err)

		c, err := BuildContext(v, DummyCLIVersion)

		require.Nil(t, err)
		require.NotNil(t, c)
		// TODO add blueprint assertion
		// require.NotNil(t, c.XLDeploy)
		// require.Equal(t, "http://xld.example.com:4516", c.XLDeploy.(*XLDeployServer).Server.(*SimpleHTTPServer).Url.String())
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
		ioutil.WriteFile(configfile, []byte(`
blueprint:
  current-repository: XL Blueprints
  repositories:
  - name: XL Blueprints
    type: http
    url: https://dist.xebialabs.com/public/blueprints/
`), 0755)

		v := viper.New()
		v.SetConfigFile(configfile)
		v.ReadInConfig()

		_, err = BuildContext(v, DummyCLIVersion)
		require.Nil(t, err)

		configbytes, err := ioutil.ReadFile(configfile)
		require.Nil(t, err)

		parsed := make(map[interface{}]interface{})
		err = yaml.Unmarshal(configbytes, parsed)
		require.Nil(t, err)

	})

	t.Run("Should get default flag value for server values when there's no override", func(t *testing.T) {
		v, _, _, _ := blueprint.GetDefaultBlueprintViperConfig(viper.GetViper(), "")
		cfgFile := ""
		PrepareRootCmdFlags(TestCmd, &cfgFile)
		context, err2 := BuildContext(v, DummyCLIVersion)
		assert.Nil(t, err2)
		assert.NotNil(t, context)

		// TODO
		// assert.Equal(t, "http://localhost:4516/", context.values["XL_DEPLOY_URL"])
		// assert.Equal(t, "admin", context.values["XL_DEPLOY_USERNAME"])
		// assert.Equal(t, "admin", context.values["XL_DEPLOY_PASSWORD"])
		// assert.Equal(t, "http://localhost:5516/", context.values["XL_RELEASE_URL"])
		// assert.Equal(t, "admin", context.values["XL_RELEASE_USERNAME"])
		// assert.Equal(t, "admin", context.values["XL_RELEASE_PASSWORD"])
	})

	t.Run("Should override defaults from viper", func(t *testing.T) {
		v, _, _, _ := blueprint.GetDefaultBlueprintViperConfig(viper.New(), "")

		// v.Set("xl-deploy.url", "http://testxld:6154")
		// v.Set("xl-deploy.username", "deployer")
		// v.Set("xl-deploy.password", "d3ploy1t")
		// v.Set(models.ViperKeyXLDAuthMethod, "basicAuth")

		context, err2 := BuildContext(v, DummyCLIVersion)
		assert.Nil(t, err2)
		assert.NotNil(t, context)

		// TODO
		// assert.Equal(t, "http://testxld:6154", context.values["XL_DEPLOY_URL"])
		// assert.Equal(t, "deployer", context.values["XL_DEPLOY_USERNAME"])
		// assert.Equal(t, "d3ploy1t", context.values["XL_DEPLOY_PASSWORD"])
		// assert.Equal(t, "basicAuth", context.values["XL_DEPLOY_AUTHMETHOD"])
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

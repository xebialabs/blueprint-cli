package xl

import (
	"bytes"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	"github.com/spf13/cobra"
	"github.com/xebialabs/xl-cli/pkg/blueprint"
	"github.com/xebialabs/xl-cli/pkg/util"

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

		c, err := BuildContext(v, nil, []string{}, nil, DummyCLIVersion)

		assert.Nil(t, err)
		assert.NotNil(t, c)
		assert.NotNil(t, c.values)
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

		c, err := BuildContext(v, nil, []string{}, nil, DummyCLIVersion)

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

		_, err = BuildContext(v, nil, []string{}, nil, DummyCLIVersion)
		require.Nil(t, err)

		configbytes, err := ioutil.ReadFile(configfile)
		require.Nil(t, err)

		parsed := make(map[interface{}]interface{})
		err = yaml.Unmarshal(configbytes, parsed)
		require.Nil(t, err)

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

	t.Run("validate that names of values are correct", func(t *testing.T) {
		v, _, _, _ := blueprint.GetDefaultBlueprintViperConfig(viper.New(), "")

		values := make(map[string]string)
		values["!incorrectKey"] = "test value"

		_, err := BuildContext(v, &values, []string{}, nil, DummyCLIVersion)

		assert.NotNil(t, err)
		assert.Equal(t, "the name of the value !incorrectKey is invalid. It must start with an alphabetical character or an underscore and be followed by zero or more alphanumerical characters or underscores", err.Error())
	})

	t.Run("Should read values into context", func(t *testing.T) {
		v, _, _, _ := blueprint.GetDefaultBlueprintViperConfig(viper.New(), "")

		propfile1 := writePropFile("file1", `
test=test
test2=test2
`)
		defer os.Remove(propfile1.Name())

		valsFiles := []string{propfile1.Name()}

		context, err2 := BuildContext(v, nil, valsFiles, nil, DummyCLIVersion)
		assert.Nil(t, err2)

		assert.Equal(t, "test", context.values["test"])
		assert.Equal(t, "test2", context.values["test2"])
	})

	t.Run("Should read case sensitive values", func(t *testing.T) {
		v, _, _, _ := blueprint.GetDefaultBlueprintViperConfig(viper.New(), "")

		propfile1 := writePropFile("file1", `
test=test1
TEST=test2
Test=test3
`)
		defer os.Remove(propfile1.Name())

		valsFiles := []string{propfile1.Name()}

		context, err2 := BuildContext(v, nil, valsFiles, nil, DummyCLIVersion)
		assert.Nil(t, err2)

		assert.Equal(t, "test1", context.values["test"])
		assert.Equal(t, "test2", context.values["TEST"])
		assert.Equal(t, "test3", context.values["Test"])
	})

	t.Run("Should override values from value files in right order (only value files)", func(t *testing.T) {
		v, _, _, _ := blueprint.GetDefaultBlueprintViperConfig(viper.New(), "")

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

		valsFiles := []string{propfile1.Name(), propfile2.Name()}

		context, err2 := BuildContext(v, nil, valsFiles, nil, DummyCLIVersion)
		assert.Nil(t, err2)

		assert.Equal(t, 11, len(context.values))
		assert.Equal(t, "override", context.values["test"])
		assert.Equal(t, "override2", context.values["test2"])
		assert.Equal(t, "ok", context.values["verifythisfilegetsread"])
	})

	t.Run("Should command line parameter value should override value files", func(t *testing.T) {
		v, _, _, _ := blueprint.GetDefaultBlueprintViperConfig(viper.New(), "")

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

		context, err2 := BuildContext(v, &values, valsFiles, nil, DummyCLIVersion)
		assert.Nil(t, err2)

		assert.Equal(t, 11, len(context.values))
		assert.Equal(t, "override", context.values["test"])
		assert.Equal(t, "override2", context.values["test2"])
		assert.Equal(t, "ok", context.values["verifythisfilegetsread"])
	})

	t.Run("Environment variables should override value files", func(t *testing.T) {
		v, _, _, _ := blueprint.GetDefaultBlueprintViperConfig(viper.New(), "")

		propfile1 := writePropFile("file1", `
test=test
test2=test2
verifythisfilegetsread=ok
`)
		defer os.Remove(propfile1.Name())

		valsFiles := []string{propfile1.Name()}

		os.Setenv("XL_VALUE_test", "override")
		os.Setenv("XL_VALUE_test2", "override2")

		context, err2 := BuildContext(v, nil, valsFiles, nil, DummyCLIVersion)
		assert.Nil(t, err2)

		assert.Equal(t, 11, len(context.values))
		assert.Equal(t, "override", context.values["test"])
		assert.Equal(t, "override2", context.values["test2"])
		assert.Equal(t, "ok", context.values["verifythisfilegetsread"])
	})

	t.Run("Should get default flag value for server values when there's no override", func(t *testing.T) {
		v, _, _, _ := blueprint.GetDefaultBlueprintViperConfig(viper.GetViper(), "")
		cfgFile := ""
		PrepareRootCmdFlags(TestCmd, &cfgFile)
		context, err2 := BuildContext(v, nil, []string{}, nil, DummyCLIVersion)
		assert.Nil(t, err2)

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

		context, err2 := BuildContext(v, nil, []string{}, nil, DummyCLIVersion)
		assert.Nil(t, err2)

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

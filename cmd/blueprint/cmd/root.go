package cmd

import (
	"os"
	"strings"

	"io/ioutil"
	"path/filepath"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/xebialabs/blueprint-cli/pkg/blueprint"
	"github.com/xebialabs/blueprint-cli/pkg/util"
	"github.com/xebialabs/blueprint-cli/pkg/xl"
	"github.com/xebialabs/yaml"
)

var initCfgTry = 0
var CliVersion = "undefined"
var cfgFile string

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "xl",
	Short: "The xl command line tool interacts with XL Release and XL Deploy",
	Long: "XL Cli " + CliVersion + "\n" + `The xl command line tool provides a fast and straightforward method for provisioning
XL Release and XL Deploy with YAML files. The files can include items like
releases, pipelines, applications and target environments.`,
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		util.Fatal("Error occurred when running command: %s\n", err)
	}
}

func init() {
	cobra.OnInitialize(initConfig)
	xl.PrepareRootCmdFlags(rootCmd, &cfgFile)
}

// initConfig reads in config file and ENV variables if set.
func initConfig() {
	if util.IsQuiet && util.IsVerbose {
		util.Fatal("Cannot use --quiet (-q) and --verbose (-v) flags together\n")
	}

	if cfgFile != "" {
		// Use config file from the flag.
		viper.SetConfigFile(cfgFile)
	} else if envCfgFile := os.Getenv("XL_CONFIG"); envCfgFile != "" {
		viper.SetConfigFile(envCfgFile)
	} else {
		configfilePath, err := util.DefaultConfigfilePath()
		if err != nil {
			util.Fatal("Could not get config file location:\n%s", err)
		} else {
			if _, err := os.Stat(configfilePath); !os.IsNotExist(err) {
				viper.SetConfigFile(configfilePath)
			}
		}
	}

	viper.SetEnvKeyReplacer(strings.NewReplacer("-", "_", ".", "_"))
	viper.AutomaticEnv() // read in environment variables that match

	viper.SetConfigType("yaml")
	// If a config file is found, read it in.

	err := viper.ReadInConfig()
	if err == nil {
		util.Verbose("Using configuration file: %s\n", viper.ConfigFileUsed())
	} else {
		switch err.(type) {
		case viper.ConfigFileNotFoundError:
			util.Verbose("No configuration file found. Using default configuration and command line options\n")
			err := writeDefaultConfigurationFile()
			if err != nil {
				util.Info("Could not write default configuration: %s/n", err.Error())
			}

			initCfgTry += 1
			if initCfgTry == 5 {
				util.Fatal("Cannot read generated config (%s) after 5 attempts: %s\n", viper.ConfigFileUsed(), err)
			}
			initConfig()
		default:
			util.Fatal("Cannot read config file %s: %s\n", viper.ConfigFileUsed(), err)
		}
	}
}

func writeDefaultConfigurationFile() error {
	configFileUsed, err := util.DefaultConfigfilePath()
	if err != nil {
		return err
	}

	if _, err := os.Stat(filepath.Dir(configFileUsed)); os.IsNotExist(err) {
		os.Mkdir(filepath.Dir(configFileUsed), 0750)
	}

	util.Verbose("Writing default configuration to %s\n", configFileUsed)

	blueprintConfData := blueprint.GetDefaultBlueprintConfData()

	// using MapSlice to maintain order of keys
	slices := yaml.MapSlice{
		{"blueprint", blueprintConfData},
	}

	d, err := yaml.Marshal(&slices)
	if err != nil {
		return err
	}
	err2 := ioutil.WriteFile(configFileUsed, d, 0640)
	if err2 != nil {
		return err2
	} else {
		return nil
	}
}

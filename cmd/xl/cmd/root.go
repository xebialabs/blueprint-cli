package cmd

import (
	"os"
	"strings"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/xebialabs/xl-cli/pkg/xl"
	"io/ioutil"
	"github.com/mitchellh/go-homedir"
	"path"
	"path/filepath"
	"github.com/xebialabs/yaml"
)

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
		xl.Fatal("Error occurred when running command: %s\n", err)
	}
}

func init() {
	cobra.OnInitialize(initConfig)
	xl.PrepareRootCmdFlags(rootCmd, &cfgFile)
}

// initConfig reads in config file and ENV variables if set.
func initConfig() {
	err := xl.ProcessCredentials()
	if err != nil {
		xl.Fatal("Error processing server credentials:\n%s", err)
	}

	if xl.IsQuiet && xl.IsVerbose {
		xl.Fatal("Cannot use --quiet (-q) and --verbose (-v) flags together\n")
	}

	if cfgFile != "" {
		// Use config file from the flag.
		viper.SetConfigFile(cfgFile)
	} else if envCfgFile := os.Getenv("XL_CONFIG"); envCfgFile != "" {
		viper.SetConfigFile(envCfgFile)
	} else {
		configfilePath, err := defaultConfigfilePath()
		if err != nil {
			xl.Fatal("Could not get config file location:\n%s", err)
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

	err = viper.ReadInConfig()
	if err == nil {
		xl.Verbose("Using configuration file: %s\n", viper.ConfigFileUsed())
	} else {
		switch err.(type) {
		case viper.ConfigFileNotFoundError:
			xl.Verbose("No configuration file found. Using default configuration and command line options\n")
			err := writeDefaultConfigurationFile()
			if err != nil {
				xl.Info("Could not write default configuration: %s/n", err.Error())
			}
		default:
			xl.Fatal("Cannot read config file %s: %s\n", viper.ConfigFileUsed(), err)
		}
	}
}
func writeDefaultConfigurationFile() error {
	configFileUsed, err := defaultConfigfilePath()
	if err != nil {
		return err
	}

	if _, err := os.Stat(filepath.Dir(configFileUsed)); os.IsNotExist(err) {
		os.Mkdir(filepath.Dir(configFileUsed), 0750)
	}

	xl.Verbose("Writing default configuration to %s\n", configFileUsed)

	// using MapSlice to maintain order of keys
	slices := yaml.MapSlice{{"xl-deploy", yaml.MapSlice{{"username", xl.DefaultXlDeployUsername},
																    {"password", xl.DefaultXlDeployPassword},
																    {"url",  xl.DefaultXlDeployUrl}}},
							{"xl-release", yaml.MapSlice{{"username", xl.DefaultXlReleaseUsername},
																	{"password", xl.DefaultXlReleasePassword},
																	{"url",  xl.DefaultXlReleaseUrl}}},
							{"template-registries", []yaml.MapSlice{{{"name", "default"},
														                	     {"url", xl.DefaultTemplateRegistry}}}}}

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

func defaultConfigfilePath() (string, error) {
    home, err := homedir.Dir()
	if err != nil {
		return "", err
	}
	xebialabsFolder := path.Join(home, ".xebialabs", "config.yaml")
	return xebialabsFolder, nil
}
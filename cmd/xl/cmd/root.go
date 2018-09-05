package cmd

import (
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"os"
	"strings"
	"github.com/xebialabs/xl-cli/pkg/xl"
)

var cfgFile string

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "xl",
	Short: "The xl command line tool interacts with XL Release and XL Deploy",
	Long: `The xl command line tool provides a fast and straightforward method for provisioning
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

	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default: $HOME/.xebialabs/config.yaml)")
	rootCmd.PersistentFlags().BoolVarP(&xl.IsVerbose, "verbose", "v", false, "verbose output")
}

// initConfig reads in config file and ENV variables if set.
func initConfig() {
	if cfgFile != "" {
		// Use config file from the flag.
		viper.SetConfigFile(cfgFile)
	} else if envCfgFile := os.Getenv("XL_CONFIG"); envCfgFile != "" {
		viper.SetConfigFile(envCfgFile)
	} else {
		viper.AddConfigPath("$HOME/.xebialabs")
		viper.SetConfigName("config")
	}

	viper.SetEnvPrefix("CLI")
	replacer := strings.NewReplacer("-", "_", ".", "_")
	viper.SetEnvKeyReplacer(replacer)
	viper.AutomaticEnv() // read in environment variables that match

	viper.SetConfigType("yaml")
	// If a config file is found, read it in.

	err := viper.ReadInConfig()
	if err == nil {
		xl.Info("Using config file: %s\n", viper.ConfigFileUsed())
	} else {
		switch err.(type) {
		case viper.ConfigFileNotFoundError:
			xl.Info("No config file used\n")
		default:
			xl.Fatal("Cannot read config file %s: %s\n", viper.ConfigFileUsed(), err)
		}
	}
}

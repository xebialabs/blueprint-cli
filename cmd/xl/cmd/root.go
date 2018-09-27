package cmd

import (
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/xebialabs/xl-cli/pkg/xl"
	"os"
	"strings"
)

var cfgFile string

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use: "xl",
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

	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default: $HOME/.xebialabs/config.yaml)")
	rootCmd.PersistentFlags().BoolVarP(&xl.IsQuiet, "quiet", "q", false, "suppress all output, except for errors")
	rootCmd.PersistentFlags().BoolVarP(&xl.IsVerbose, "verbose", "v", false, "verbose output")
	rootCmd.PersistentFlags().String("xl-deploy-url", "http://localhost:4516/", "URL to access the XL Deploy server")
	rootCmd.PersistentFlags().String("xl-deploy-username", "admin", "Username to access the XL Deploy server")
	rootCmd.PersistentFlags().String("xl-deploy-password", "admin", "Password to access the XL Deploy server")
	viper.BindPFlag("xl-deploy.url", rootCmd.PersistentFlags().Lookup("xl-deploy-url"))
	viper.BindPFlag("xl-deploy.username", rootCmd.PersistentFlags().Lookup("xl-deploy-username"))
	viper.BindPFlag("xl-deploy.password", rootCmd.PersistentFlags().Lookup("xl-deploy-password"))

	rootCmd.PersistentFlags().String("xl-release-url", "http://localhost:5516/", "URL to access the XL Release server")
	rootCmd.PersistentFlags().String("xl-release-username", "admin", "Username to access the XL Release server")
	rootCmd.PersistentFlags().String("xl-release-password", "admin", "Password to access the XL Release server")
	viper.BindPFlag("xl-release.url", rootCmd.PersistentFlags().Lookup("xl-release-url"))
	viper.BindPFlag("xl-release.username", rootCmd.PersistentFlags().Lookup("xl-release-username"))
	viper.BindPFlag("xl-release.password", rootCmd.PersistentFlags().Lookup("xl-release-password"))
}

// initConfig reads in config file and ENV variables if set.
func initConfig() {
	if xl.IsQuiet && xl.IsVerbose {
		xl.Fatal("Cannot use --quiet (-q) and --verbose (-v) flags together\n")
	}

	if cfgFile != "" {
		// Use config file from the flag.
		viper.SetConfigFile(cfgFile)
	} else if envCfgFile := os.Getenv("XL_CONFIG"); envCfgFile != "" {
		viper.SetConfigFile(envCfgFile)
	} else {
		viper.AddConfigPath("$HOME/.xebialabs")
		viper.SetConfigName("config")
	}

	viper.SetEnvKeyReplacer(strings.NewReplacer("-", "_", ".", "_"))
	viper.AutomaticEnv() // read in environment variables that match

	viper.SetConfigType("yaml")
	// If a config file is found, read it in.

	err := viper.ReadInConfig()
	if err == nil {
		xl.Verbose("Using configuration file: %s\n", viper.ConfigFileUsed())
	} else {
		switch err.(type) {
		case viper.ConfigFileNotFoundError:
			xl.Verbose("No configuration file found. Using default configuration and command line options\n")
		default:
			xl.Fatal("Cannot read config file %s: %s\n", viper.ConfigFileUsed(), err)
		}
	}
}

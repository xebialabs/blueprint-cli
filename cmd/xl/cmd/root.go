package cmd

import (
	"os"
	"strings"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/xebialabs/xl-cli/pkg/xl"
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

	rootFlags := rootCmd.PersistentFlags()

	rootFlags.StringVar(&cfgFile, "config", "", "config file (default: $HOME/.xebialabs/config.yaml)")
	rootFlags.BoolVarP(&xl.IsQuiet, "quiet", "q", false, "suppress all output, except for errors")
	rootFlags.BoolVarP(&xl.IsVerbose, "verbose", "v", false, "verbose output")
	rootFlags.String("xl-deploy-url", "http://localhost:4516/", "URL to access the XL Deploy server")
	rootFlags.String("xl-deploy-username", "admin", "Username to access the XL Deploy server")
	rootFlags.String("xl-deploy-password", "admin", "Password to access the XL Deploy server")
	viper.BindPFlag("xl-deploy.url", rootFlags.Lookup("xl-deploy-url"))
	viper.BindPFlag("xl-deploy.username", rootFlags.Lookup("xl-deploy-username"))
	viper.BindPFlag("xl-deploy.password", rootFlags.Lookup("xl-deploy-password"))

	rootFlags.String("xl-release-url", "http://localhost:5516/", "URL to access the XL Release server")
	rootFlags.String("xl-release-username", "admin", "Username to access the XL Release server")
	rootFlags.String("xl-release-password", "admin", "Password to access the XL Release server")
	viper.BindPFlag("xl-release.url", rootFlags.Lookup("xl-release-url"))
	viper.BindPFlag("xl-release.username", rootFlags.Lookup("xl-release-username"))
	viper.BindPFlag("xl-release.password", rootFlags.Lookup("xl-release-password"))
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
		viper.AddConfigPath("$HOME/.xebialabs")
		viper.SetConfigName("config")
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
		default:
			xl.Fatal("Cannot read config file %s: %s\n", viper.ConfigFileUsed(), err)
		}
	}
}

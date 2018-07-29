package cmd

import (
	"fmt"
	"github.com/mitchellh/go-homedir"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"log"
	"os"
	"path/filepath"
)

const defCfgFile = "config"

//default subdirectories of $HOME where the config is located
var defCfgFilePath = []string{".xebialabs"}

// default config path
var defCfgPath = defCfgDir()

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "xl",
	Short: "This tool interacts with XL Release and XL Deploy",
	Long: `This CLI provides a fast and straightforward method for provisioning
XL Release and XL Deploy with YAML files. The files can include items like
releases, pipelines, applications and target environments.`,
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func init() {
	cobra.OnInitialize(initConfig)

	// Here you will define your flags and configuration settings.
	// Cobra supports persistent flags, which, if defined here,
	// will be global for your application.
	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", fmt.Sprintf("config file (default: %s)", defCfg()))
}

// initConfig reads in config file and ENV variables if set.
func initConfig() {
	if cfgFile != "" {
		// Use config file from the flag.
		viper.SetConfigFile(cfgFile)
	} else {
		//Use default config file
		viper.SetConfigFile(defCfg())
	}

	viper.AutomaticEnv() // read in environment variables that match

	// If a config file is found, read it in.
	// Otherwise, create the directory for it.
	// The directory needs to exists in order to create the config file later on.
	if _, err := os.Stat(viper.ConfigFileUsed()); os.IsNotExist(err) {
		if dirErr := os.MkdirAll(filepath.Dir(viper.ConfigFileUsed()), 0755); dirErr != nil {
			log.Fatalf("%v", dirErr)
		}
	} else {
		viper.ReadInConfig()
	}
}

func defCfg() string {
	return fmt.Sprintf("%s%s%s%s", defCfgPath, string(filepath.Separator), defCfgFile, ".yaml")
}

func defCfgDir() string {
	// Find home directory.
	home, err := homedir.Dir()

	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	hs := []string{home}
	ps := append(hs, defCfgFilePath...)
	return filepath.Join(ps...)
}

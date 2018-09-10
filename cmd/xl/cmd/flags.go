package cmd

import (
	"github.com/spf13/viper"
	"github.com/spf13/pflag"
)

func AddServerFlags(flagSet *pflag.FlagSet) {
	flagSet.String("xl-deploy-url", "http://localhost:4516/", "URL to access the XL Deploy server")
	flagSet.String("xl-deploy-username", "admin", "Username to access the XL Deploy server")
	flagSet.String("xl-deploy-password", "admin", "Password to access the XL Deploy server")
	viper.BindPFlag("xl-deploy.url", flagSet.Lookup("xl-deploy-url"))
	viper.BindPFlag("xl-deploy.username", flagSet.Lookup("xl-deploy-username"))
	viper.BindPFlag("xl-deploy.password", flagSet.Lookup("xl-deploy-password"))

	flagSet.String("xl-release-url", "http://localhost:5516/", "URL to access the XL Release server")
	flagSet.String("xl-release-username", "admin", "Username to access the XL Release server")
	flagSet.String("xl-release-password", "admin", "Password to access the XL Release server")
	viper.BindPFlag("xl-release.url", flagSet.Lookup("xl-release-url"))
	viper.BindPFlag("xl-release.username", flagSet.Lookup("xl-release-username"))
	viper.BindPFlag("xl-release.password", flagSet.Lookup("xl-release-password"))
}

package cmd

import (
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/xebialabs/xl-cli/pkg/xl"
)

var upCmd = &cobra.Command{
	Use:   "up",
	Short: "Install XLR, XLD via XL-Seed",
	Long:  `Pulls and runs XL-Seed to deploy XLR and XLD`,
	Run: func(cmd *cobra.Command, args []string) {
		context, err := xl.BuildContext(viper.GetViper(), nil, nil)
		if err != nil {
			xl.Fatal("Error while reading configuration: %s\n", err)
		}
		if xl.IsVerbose {
			context.PrintConfiguration()
		}

		DoUp(context)
	},
}

func DoUp(context *xl.Context) {
	xl.Verbose("Running XL Seed")
	xl.RunXlSeed(context)
}

func init() {
	rootCmd.AddCommand(upCmd)
}

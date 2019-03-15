package cmd

import (
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/xebialabs/xl-cli/pkg/util"
	"github.com/xebialabs/xl-cli/pkg/xl"
)

var upCmd = &cobra.Command{
	Use:   "up",
	Short: "Install XLR, XLD via XL-Seed",
	Long:  `Pulls and runs XL-Seed to deploy XLR and XLD`,
	Run: func(cmd *cobra.Command, args []string) {
		context, err := xl.BuildContext(viper.GetViper(), nil, nil)
		if err != nil {
			util.Fatal("Error while reading configuration: %s\n", err)
		}
		if util.IsVerbose {
			context.PrintConfiguration()
		}

		DoUp(context)
	},
}

var upLocalMode bool
var upQuickSetup bool
var upAdvancedSetup bool
var upBlueprintTemplate string

func DoUp(context *xl.Context) {
	util.Verbose("Running XL Seed")
	xl.InvokeBlueprintAndSeed(context, upLocalMode, upQuickSetup, upAdvancedSetup, upBlueprintTemplate)
}

func init() {
	rootCmd.AddCommand(upCmd)

	upFlags := upCmd.Flags()
	upFlags.BoolVarP(&upLocalMode, "local", "l", false, "Enable local file mode, by default remote file mode is used")
	upFlags.StringVarP(&upBlueprintTemplate, "blueprint", "b", "", "The folder containing the blueprint to use; this can be a folder path relative to the remote blueprint repository or a local folder path")
	upFlags.BoolVarP(&upQuickSetup, "quick-setup", "", false, "Quickly run setup with all default values")
	upFlags.BoolVarP(&upAdvancedSetup, "advanced-setup", "", false, "Advanced setup")
}

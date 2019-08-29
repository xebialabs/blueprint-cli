// DONT REMOVE THIS COMMENT BLOCK ITS USED TO CONTROL THE INCLUSION OF THIS FEATURE
// BUILD THE PROJECT WITH -PincludeXlUp TO GET A VERSION OF THE CLI WITH THE UP COMMAND
//
// +build includeXlUp

package cmd

import (
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/xebialabs/xl-cli/pkg/up"
	"github.com/xebialabs/xl-cli/pkg/util"
	"github.com/xebialabs/xl-cli/pkg/xl"
)

var upCmd = &cobra.Command{
	Use:   "up",
	Short: "Install XLR, XLD via XL-Seed",
	Long:  `Pulls and runs XL-Seed to deploy XLR and XLD`,
	Run: func(cmd *cobra.Command, args []string) {
		context, err := xl.BuildContext(viper.GetViper(), nil, nil, nil, CliVersion)
		if err != nil {
			util.Fatal("Error while reading configuration: %s\n", err)
		}
		if util.IsVerbose {
			context.PrintConfiguration()
		}

		DoUp(context, CliVersion)
	},
}

var upParams = up.UpParams{}

// DoUp executes the up command
func DoUp(context *xl.Context, branchVersion string) {
	util.Verbose("Running XL Seed\n")
	up.InvokeBlueprintAndSeed(context, upParams, branchVersion)
}

func init() {
	rootCmd.AddCommand(upCmd)

	upFlags := upCmd.Flags()
	upFlags.StringVarP(&upParams.localMode, "local", "l", "", "Enable local file mode, by default remote file mode is used")
	upFlags.StringVarP(&upParams.blueprintTemplate, "blueprint", "b", "", "The folder containing the blueprint to use; this can be a folder path relative to the remote blueprint repository or a local folder path")
	upFlags.BoolVarP(&upParams.quickSetup, "quick-setup", "", false, "Quickly run setup with all default values")
	upFlags.BoolVarP(&upParams.advancedSetup, "advanced-setup", "", false, "Advanced setup")
	upFlags.StringVarP(&upParams.answerFile, "answers", "a", "", "The file containing answers for the questions")
	upFlags.BoolVarP(&upParams.cfgOverridden, "dev", "d", false, "Enable dev mode, uses repository config from your local config instead")
	upFlags.BoolVar(&upParams.noCleanup, "no-cleanup", false, "Leave generated files on the filesystem")
	upFlags.BoolVar(&upParams.destroy, "destroy", false, "Undeploy the deployed resources")
	err := upFlags.MarkHidden("dev")
	if err != nil {
		util.Error("error setting up cmd flags: %s\n", err.Error())
	}
}

// DONT REMOVE THIS COMMENT BLOCK ITS USED TO CONTROL THE INCLUSION OF THIS FEATURE
// BUILD THE PROJECT WITH -PincludeXlUp TO GET A VERSION OF THE CLI WITH THE UP COMMAND
//
// +build includeXlUp

package cmd

import (
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/xebialabs/xl-cli/pkg/blueprint"
	"github.com/xebialabs/xl-cli/pkg/models"
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
func DoUp(context *xl.Context, gitBranch string) {
	util.Verbose("Running XL Seed\n")
	gb := &blueprint.GeneratedBlueprint{OutputDir: models.BlueprintOutputDir}

	err := up.InvokeBlueprintAndSeed(context.BlueprintContext, upParams, gitBranch, gb)
	if err != nil {
		util.Fatal("Error while running xl-up: %s\n", err)
	}
	if !upParams.NoCleanup {
		defer gb.Cleanup(up.GeneratedFinalAnswerFile)
	}
}

func init() {
	rootCmd.AddCommand(upCmd)

	upFlags := upCmd.Flags()
	upFlags.StringVarP(&upParams.LocalPath, "local", "l", "", "Provide local file path where blueprints are located, by default a remote repository is used")
	upFlags.StringVarP(&upParams.BlueprintTemplate, "blueprint", "b", "", "The folder containing the xl-infra blueprint; this can be a folder path relative to the remote blueprint repository or a local folder path provided using -l flag")
	upFlags.BoolVarP(&upParams.QuickSetup, "quick-setup", "", false, "Quickly run setup with all default values")
	upFlags.BoolVarP(&upParams.AdvancedSetup, "advanced-setup", "", false, "Advanced setup")
	upFlags.StringVarP(&upParams.AnswerFile, "answers", "a", "", "The file containing answers for the questions")
	upFlags.BoolVarP(&upParams.CfgOverridden, "dev", "d", false, "Enable dev mode, uses repository config from your local config instead")
	upFlags.BoolVar(&upParams.NoCleanup, "no-cleanup", false, "Leave generated files on the filesystem")
	upFlags.BoolVar(&upParams.Undeploy, "undeploy", false, "Undeploy the deployed resources")
	upFlags.BoolVar(&upParams.DryRun, "dry-run", false, "Create files only, nothing will be deployed")
	err := upFlags.MarkHidden("dev")
	if err != nil {
		util.Error("error setting up cmd flags: %s\n", err.Error())
	}
}

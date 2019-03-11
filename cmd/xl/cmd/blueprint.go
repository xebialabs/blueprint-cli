package cmd

import (
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/xebialabs/xl-cli/pkg/blueprint"
	"github.com/xebialabs/xl-cli/pkg/models"
	"github.com/xebialabs/xl-cli/pkg/util"
	"github.com/xebialabs/xl-cli/pkg/xl"
)

var blueprintCmd = &cobra.Command{
	Use:   "blueprint",
	Short: "Create a Blueprint",
	Long:  `Create a Blueprint for XL Platform Releases and Deployments`,
	Run: func(cmd *cobra.Command, args []string) {
		context, err := xl.BuildContext(viper.GetViper(), nil, nil)
		if err != nil {
			util.Fatal("Error while reading configuration: %s\n", err)
		}
		if util.IsVerbose {
			context.PrintConfiguration()
		}

		DoBlueprint(context)
	},
}

var blueprintLocalMode bool
var blueprintTemplate string

// DoBlueprint creates blueprint templates
func DoBlueprint(context *xl.Context) {
	blueprintLocalMode = util.PathExists(blueprintTemplate, true)
	err := blueprint.InstantiateBlueprint(blueprintLocalMode, blueprintTemplate, context.BlueprintContext, models.BlueprintOutputDir)
	if err != nil {
		util.Fatal("Error while creating Blueprint: %s\n", err)
	}
}

func init() {
	rootCmd.AddCommand(blueprintCmd)

	blueprintFlags := blueprintCmd.Flags()
	blueprintFlags.StringVarP(&blueprintTemplate, "blueprint", "b", "", "The folder containing the blueprint to use; this can be a folder path relative to the remote blueprint repository or a local absolute/relative folder path")
}

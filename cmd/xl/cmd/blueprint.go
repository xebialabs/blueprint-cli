package cmd

import (
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
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
	err := xl.InstantiateBlueprint(blueprintLocalMode, blueprintTemplate, context.BlueprintContext, models.BlueprintOutputDir)
	if err != nil {
		util.Fatal("Error while creating Blueprint: %s\n", err)
	}
}

func init() {
	rootCmd.AddCommand(blueprintCmd)

	blueprintFlags := blueprintCmd.Flags()
	blueprintFlags.BoolVarP(&blueprintLocalMode, "local", "l", false, "Enable local blueprint mode, by default remote mode is enabled")
	blueprintFlags.StringVarP(&blueprintTemplate, "blueprint", "b", "", "The blueprint to use, a path relative to the blueprint repository or a local path to a blueprint")
}

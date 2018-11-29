package cmd

import (
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/xebialabs/xl-cli/pkg/xl"
)

var blueprintCmd = &cobra.Command{
	Use:   "blueprint",
	Short: "Create a Blueprint",
	Long:  `Create a Blueprint for XL Platform Releases and Deployments`,
	Run: func(cmd *cobra.Command, args []string) {
		context, err := xl.BuildContext(viper.GetViper(), nil, nil)
		if err != nil {
			xl.Fatal("Error while reading configuration: %s\n", err)
		}
		if xl.IsVerbose {
			context.PrintConfiguration()
		}

		DoBlueprint(context)
	},
}

var blueprintTemplate string

const outputDir = "xebialabs"

// DoBlueprint creates blueprint templates
func DoBlueprint(context *xl.Context) {
	err := xl.InstantiateBlueprint(blueprintTemplate, context.BlueprintRepository, outputDir)
	if err != nil {
		xl.Fatal("Error while creating Blueprint: %s\n", err)
	}
}

func init() {
	rootCmd.AddCommand(blueprintCmd)

	blueprintFlags := blueprintCmd.Flags()
	blueprintFlags.StringVarP(&blueprintTemplate, "blueprint", "b", "", "The blueprint path/url to use")
}

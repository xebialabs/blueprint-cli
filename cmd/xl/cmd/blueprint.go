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
		context, err := xl.BuildContext(viper.GetViper(), nil, nil, nil)
		if err != nil {
			util.Fatal("Error while reading configuration: %s\n", err)
		}
		if util.IsVerbose {
			context.PrintConfiguration()
		}

		DoBlueprint(context)
	},
}

var blueprintTemplate string
var answersFile string
var localRepoPath string
var strictAnswers bool
var useDefaultsAsValue bool

// DoBlueprint creates blueprint templates
func DoBlueprint(context *xl.Context) {
	// if in dev local repo mode, recreate context
	var err error
	blueprintContext := context.BlueprintContext
	if localRepoPath != "" {
		blueprintContext, err = blueprint.ConstructLocalBlueprintContext(localRepoPath)
		if err != nil {
			util.Fatal("Error creating local blueprint context: %s\n", err)
		}
	}

	generatedBlueprint := &blueprint.GeneratedBlueprint{OutputDir: models.BlueprintOutputDir}
	err = blueprint.InstantiateBlueprint(blueprintTemplate, blueprintContext, generatedBlueprint, answersFile, strictAnswers, useDefaultsAsValue, false)
	if err != nil {
		generatedBlueprint.Cleanup() // Cleanup the partially generated blueprint
		util.Fatal("Error while creating Blueprint: %s\n", err)
	}
}

func init() {
	rootCmd.AddCommand(blueprintCmd)

	blueprintFlags := blueprintCmd.Flags()
	blueprintFlags.StringVarP(&blueprintTemplate, "blueprint", "b", "", "Blueprint path to use, relative to the active repository")
	blueprintFlags.StringVarP(&localRepoPath, "local-repo", "l", "", "Local repository directory to use (bypasses active repository)")
	blueprintFlags.StringVarP(&answersFile, "answers", "a", "", "The file containing answers for blueprint questions")
	blueprintFlags.BoolVarP(&strictAnswers, "strict-answers", "s", false, "If flag is set, answers file will be expected to have all the variable values")
	blueprintFlags.BoolVarP(&useDefaultsAsValue, "use-defaults", "d", false, "If flag is set, default values for variables will be treated as value fields")
}

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
		context, err := xl.BuildContext(viper.GetViper(), CliVersion)
		if err != nil {
			util.Fatal("Error while reading configuration: %s\n", err)
		}
		if util.IsVerbose {
			context.PrintConfiguration()
		}

		DoBlueprint(context)
	},
}

var localRepoPath string
var params = blueprint.BlueprintParams{}

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
	_, _, err = blueprint.InstantiateBlueprint(params, blueprintContext, generatedBlueprint)
	if err != nil {
		generatedBlueprint.Cleanup() // Cleanup the partially generated blueprint
		util.Fatal("Error while creating Blueprint: %s\n", err)
	}
}

func init() {
	rootCmd.AddCommand(blueprintCmd)

	blueprintFlags := blueprintCmd.Flags()
	blueprintFlags.StringVarP(&params.TemplatePath, "blueprint", "b", "", "Blueprint path to use, relative to the active repository")
	blueprintFlags.StringVarP(&localRepoPath, "local-repo", "l", "", "Local repository directory to use (bypasses active repository)")
	blueprintFlags.StringVarP(&params.AnswersFile, "answers", "a", "", "The file containing answers for blueprint questions")
	blueprintFlags.BoolVarP(&params.StrictAnswers, "strict-answers", "s", false, "If flag is set, answers file will be expected to have all the variable values")
	blueprintFlags.BoolVarP(&params.UseDefaultsAsValue, "use-defaults", "d", false, "If flag is set, default values for variables will be treated as value fields")
}

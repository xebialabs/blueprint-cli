package up

import (
	"github.com/xebialabs/xl-cli/pkg/xl"
	"time"

	"github.com/briandowns/spinner"
	"github.com/xebialabs/xl-cli/pkg/blueprint"
	"github.com/xebialabs/xl-cli/pkg/models"
	"github.com/xebialabs/xl-cli/pkg/util"
)

var s = spinner.New(spinner.CharSets[9], 100*time.Millisecond)
var applyValues map[string]string

// InvokeBlueprintAndSeed will invoke blueprint and then call XL Seed
func InvokeBlueprintAndSeed(context *xl.Context, upLocalMode bool, quickSetup bool, advancedSetup bool, blueprintTemplate string, cfgOverridden bool, upAnswerFile string, noCleanup bool, branchVersion string) {

	if upAnswerFile == "" {
		if !(quickSetup || advancedSetup) {
			// ask for setup mode.
			mode := askSetupMode()

			if mode == "quick" {
				quickSetup = true
			} else {
				advancedSetup = true
			}
		}
	} else {
		advancedSetup = true
	}

	// Skip Generate blueprint file
	blueprint.SkipFinalPrompt = true
	util.IsQuiet = true

	if !upLocalMode && !cfgOverridden {
		blueprintTemplate = DefaultBlueprintTemplate
		repo := getRepo(branchVersion)
		context.BlueprintContext.ActiveRepo = &repo
	}

	gb := &blueprint.GeneratedBlueprint{OutputDir: models.BlueprintOutputDir}
	if !noCleanup {
		defer gb.Cleanup()
	}

	err := blueprint.InstantiateBlueprint(upLocalMode, blueprintTemplate, context.BlueprintContext, gb, upAnswerFile, false, quickSetup, true)
	if err != nil {
		util.Fatal("Error while creating Blueprint: %s \n", err)
	}

	util.IsQuiet = false

	applyFilesAndSave()

	util.Info("Generated files for deployment successfully! \nSpinning up xl seed! \n")

	runAndCaptureResponse(pullSeedImage)
	runAndCaptureResponse(runSeed())
}

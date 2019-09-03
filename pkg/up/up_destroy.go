package up

import (
	"github.com/xebialabs/xl-cli/pkg/blueprint"
	"github.com/xebialabs/xl-cli/pkg/util"
)

// InvokeDestroy un-deploys the resources deployed by the up command.
func InvokeDestroy(blueprintContext *blueprint.BlueprintContext, upParams UpParams, branchVersion string, configMap string, gb *blueprint.GeneratedBlueprint) {

	if configMap != "" {
		util.Verbose("Destroy workflow started.... \n")

		answerMapFromConfigMap := parseConfigMap(configMap)

		generateLicenseAndKeystore(answerMapFromConfigMap, gb)

		convertMapToAnswerFile(answerMapFromConfigMap, GeneratedAnswerFile)

		util.IsQuiet = true
		runApplicationBlueprint(&upParams, blueprintContext, gb, branchVersion)
		util.IsQuiet = false

		applyFilesAndSave()

		util.Info("Generated files for un-deployment successfully! \nSpinning up xl seed! \n")

		runAndCaptureResponse(pullSeedImage)
		runAndCaptureResponse(runSeed(true))
	} else {
		util.Fatal("No resources found. Nothing to un-deploy!\n")
	}

}

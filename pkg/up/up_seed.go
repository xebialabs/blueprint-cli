package up

import (
	"io/ioutil"
	"strings"
	"time"

	"github.com/xebialabs/xl-cli/pkg/cloud/k8s"

	"github.com/xebialabs/xl-cli/pkg/xl"
	"gopkg.in/yaml.v2"

	"github.com/briandowns/spinner"
	"github.com/xebialabs/xl-cli/pkg/blueprint"
	"github.com/xebialabs/xl-cli/pkg/models"
	"github.com/xebialabs/xl-cli/pkg/util"
)

var s = spinner.New(spinner.CharSets[9], 100*time.Millisecond)
var applyValues map[string]string

// InvokeBlueprintAndSeed will invoke blueprint and then call XL Seed
func InvokeBlueprintAndSeed(context *xl.Context, upParams UpParams, branchVersion string) {

	defer util.StopAndRemoveContainer(s)

	if upParams.answerFile == "" {
		if !(upParams.quickSetup || upParams.advancedSetup) && !upParams.destroy {
			// ask for setup mode.
			mode := askSetupMode()

			if mode == "quick" {
				upParams.quickSetup = true
			}
		}
	}

	blueprint.SkipFinalPrompt = true
	util.IsQuiet = true

	var err error
	blueprintContext := context.BlueprintContext

	if upParams.localMode != "" {
		blueprintContext, err = blueprint.ConstructLocalBlueprintContext(upParams.localMode)
		if err != nil {
			util.Fatal("Error while creating local blueprint context: %s \n", err)
		}
	} else if upParams.localMode == "" && !upParams.cfgOverridden {
		upParams.blueprintTemplate = DefaultInfraBlueprintTemplate
		repo := getRepo(branchVersion)
		blueprintContext.ActiveRepo = &repo
	}

	gb := &blueprint.GeneratedBlueprint{OutputDir: models.BlueprintOutputDir}

	if !upParams.noCleanup {
		defer gb.Cleanup()
	}

	answerFileToBlueprint := upParams.answerFile

	if answerFileToBlueprint != "" {
		generateAnswerFile(answerFileToBlueprint, gb)
		answerFileToBlueprint = TempAnswerFile
	}

	// Infra blueprint
	err = blueprint.InstantiateBlueprint(upParams.blueprintTemplate, blueprintContext, gb, answerFileToBlueprint, false, upParams.quickSetup, true, false)
	if err != nil {
		util.Fatal("Error while creating Infrastructure Blueprint: %s \n", err)
	}
	util.IsQuiet = false

	configMap := getKubeConfigMap()

	if upParams.destroy {
		InvokeDestroy(blueprintContext, upParams, branchVersion, configMap, gb)
		return
	}

	if configMap != "" {
		util.Verbose("Update workflow started.... \n")

		answerMapFromConfigMap := parseConfigMap(configMap)

		// Strip the version information
		models.AvailableVersion = getVersion(answerMapFromConfigMap, "xlVersion", "prevVersion")
		if models.AvailableVersion != "" {
			answerMapFromConfigMap["xlVersion"] = ""
			answerMapFromConfigMap["prevVersion"] = models.AvailableVersion
		}

		models.AvailableXlrVersion = getVersion(answerMapFromConfigMap, "xlrVersion", "prevXlrVersion")
		if models.AvailableXlrVersion != "" {
			answerMapFromConfigMap["xlrVersion"] = ""
			answerMapFromConfigMap["prevXlrVersion"] = models.AvailableXlrVersion
		}

		models.AvailableXldVersion = getVersion(answerMapFromConfigMap, "xldVersion", "prevXldVersion")
		if models.AvailableXldVersion != "" {
			answerMapFromConfigMap["xldVersion"] = ""
			answerMapFromConfigMap["prevXldVersion"] = models.AvailableXldVersion
		}

		generateLicenseAndKeystore(answerMapFromConfigMap, gb)
		convertMapToAnswerFile(answerMapFromConfigMap, GeneratedAnswerFile)
	} else {
		util.Verbose("Install workflow started")
	}

	util.IsQuiet = true
	runApplicationBlueprint(&upParams, blueprintContext, gb, branchVersion)
	util.IsQuiet = false

	applyFilesAndSave()

	util.Info("Generated files for deployment successfully! \nSpinning up xl seed! \n")

	runAndCaptureResponse(pullSeedImage)
	runAndCaptureResponse(runSeed(false))
}

func parseConfigMap(configMap string) map[string]string {
	util.Verbose("%s", configMap)
	answerMapFromConfigMap := make(map[string]string)
	if err := yaml.Unmarshal([]byte(configMap), &answerMapFromConfigMap); err != nil {
		util.Fatal("Error parsing configMap: %s \n", err)
	}
	return answerMapFromConfigMap
}

func runApplicationBlueprint(upParams *UpParams, blueprintContext *blueprint.BlueprintContext, gb *blueprint.GeneratedBlueprint, branchVersion string) {
	// Switch blueprint once the infrastructure is done.
	if upParams.blueprintTemplate != "" {
		upParams.blueprintTemplate = strings.Replace(upParams.blueprintTemplate, DefaultInfraBlueprintTemplate, DefaultBlueprintTemplate, 1)
	} else {
		upParams.blueprintTemplate = DefaultBlueprintTemplate
		repo := getRepo(branchVersion)
		blueprintContext.ActiveRepo = &repo
	}

	if upParams.answerFile != "" {
		upParams.answerFile = getAnswerFile(TempAnswerFile)
		gb.GeneratedFiles = append(gb.GeneratedFiles, TempAnswerFile)
		gb.GeneratedFiles = append(gb.GeneratedFiles, MergedAnswerFile)
	} else {
		upParams.answerFile = getAnswerFile(upParams.answerFile)
	}

	err := blueprint.InstantiateBlueprint(upParams.blueprintTemplate, blueprintContext, gb, upParams.answerFile, false, upParams.quickSetup, true, true)
	if err != nil {
		util.Fatal("Error while creating Blueprint: %s \n", err)
	}
}

func generateAnswerFile(upAnswerFile string, gb *blueprint.GeneratedBlueprint) {
	answerMap := convertAnswerFileToMap(upAnswerFile)
	generateLicenseAndKeystore(answerMap, gb)
	convertMapToAnswerFile(answerMap, TempAnswerFile)
	gb.GeneratedFiles = append(gb.GeneratedFiles, TempAnswerFile)
}

func convertAnswerFileToMap(answerFilePath string) map[string]string {
	answerMap := make(map[string]string)

	contents, err := ioutil.ReadFile(answerFilePath)

	if err != nil {
		util.Fatal("Error reading answer file %s: %s", answerFilePath, err)
	}

	if err := yaml.Unmarshal(contents, &answerMap); err != nil {
		util.Fatal("Error converting answer file %s", err)
	}

	return answerMap
}

func convertMapToAnswerFile(contents map[string]string, filename string) {
	yamlBytes, err := yaml.Marshal(contents)
	if err != nil {
		util.Fatal("Error when marshalling the answer map to yaml %s", err.Error())
	}

	err = ioutil.WriteFile(filename, yamlBytes, 0640)
	if err != nil {
		util.Fatal("Error when creating an answer file %s", err.Error())
	}
}

func getVersion(answerMapFromConfigMap map[string]string, key, prevKey string) string {
	var version string
	if k8s.IsPropertyPresent(key, answerMapFromConfigMap) {
		version = k8s.GetRequiredPropertyFromMap(key, answerMapFromConfigMap)
		util.Verbose("Version %s is existing.\n", version)
	} else if k8s.IsPropertyPresent(prevKey, answerMapFromConfigMap) {
		version = k8s.GetRequiredPropertyFromMap(prevKey, answerMapFromConfigMap)
	}
	return version
}

func getAnswerFile(answerFile string) string {
	// If the answer file is provided merge them and use the merged file as the answer file
	if answerFile != "" {

		newAnswerMap, isConflict := mergeAnswerFiles(answerFile)

		if isConflict {
			isAnswerFileClash := askOverrideAnswerFile()
			if !isAnswerFileClash {
				util.Fatal("Quitting deployment due to conflict in files.")
			}
		}
		answerFile = MergedAnswerFile

		convertMapToAnswerFile(newAnswerMap, answerFile)

	} else {
		answerFile = GeneratedAnswerFile
	}
	return answerFile
}

func mergeAnswerFiles(answerFile string) (map[string]string, bool) {
	autoAnswerFile, err := blueprint.GetValuesFromAnswersFile(GeneratedAnswerFile)
	if err != nil {
		util.Fatal(err.Error())
	}

	providedAnswerFile, err := blueprint.GetValuesFromAnswersFile(answerFile)
	if err != nil {
		util.Fatal(err.Error())
	}

	msg, err := VersionCheck(autoAnswerFile, providedAnswerFile)

	if err != nil {
		util.Fatal(err.Error())
	}

	if msg != "" {
		util.Info(msg)
	} else {
		util.Verbose("No version provided, will ask the version in the application blueprint")
	}

	return mergeMaps(autoAnswerFile, providedAnswerFile)
}

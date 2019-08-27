package up

import (
	"io/ioutil"
	"log"
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
func InvokeBlueprintAndSeed(context *xl.Context, upLocalMode string, quickSetup bool, advancedSetup bool, blueprintTemplate string, cfgOverridden bool, upAnswerFile string, noCleanup bool, branchVersion string) {
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

	blueprint.SkipFinalPrompt = true
	util.IsQuiet = true

	var err error
	blueprintContext := context.BlueprintContext

	if upLocalMode != "" {
		blueprintContext, err = blueprint.ConstructLocalBlueprintContext(upLocalMode)
		if err != nil {
			util.Fatal("Error while creating local blueprint context: %s \n", err)
		}
	} else if upLocalMode == "" && !cfgOverridden {
		blueprintTemplate = DefaultInfraBlueprintTemplate
		repo := getRepo(branchVersion)
		blueprintContext.ActiveRepo = &repo
	}

	gb := &blueprint.GeneratedBlueprint{OutputDir: models.BlueprintOutputDir}

	if upAnswerFile != "" {
		generateAnswerFile(upAnswerFile, gb)
		upAnswerFile = GeneratedAnswerFile
	}

	// Infra blueprint
	err = blueprint.InstantiateBlueprint(blueprintTemplate, blueprintContext, gb, upAnswerFile, false, quickSetup, true, false)
	if err != nil {
		util.Fatal("Error while creating Infrastructure Blueprint: %s \n", err)
	}
	util.IsQuiet = false

	configMap := connectToKube()

	if configMap != "" {
		util.Verbose("Update workflow started.... \n")
		util.Verbose("%s", configMap)

		answerMapFromConfigMap := make(map[string]string)
		if err := yaml.Unmarshal([]byte(configMap), &answerMapFromConfigMap); err != nil {
			log.Fatal(err)
		}

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

	// Switch blueprint once the infrastructure is done.
	if blueprintTemplate != "" {
		blueprintTemplate = strings.Replace(blueprintTemplate, DefaultInfraBlueprintTemplate, DefaultBlueprintTemplate, 1)
	} else {
		blueprintTemplate = DefaultBlueprintTemplate
		repo := getRepo(branchVersion)
		blueprintContext.ActiveRepo = &repo
	}

	if !noCleanup {
		defer gb.Cleanup()
	}

	defer util.StopAndRemoveContainer(s)

	upAnswerFile = getAnswerFile(upAnswerFile)

	err = blueprint.InstantiateBlueprint(blueprintTemplate, blueprintContext, gb, upAnswerFile, false, quickSetup, true, true)
	if err != nil {
		util.Fatal("Error while creating Blueprint: %s \n", err)
	}

	util.IsQuiet = false

	applyFilesAndSave()

	util.Info("Generated files for deployment successfully! \nSpinning up xl seed! \n")

	runAndCaptureResponse(pullSeedImage)
	runAndCaptureResponse(runSeed())
}

func generateAnswerFile(upAnswerFile string, gb *blueprint.GeneratedBlueprint) {
	answerMap := convertAnswerFileToMap(upAnswerFile)
	generateLicenseAndKeystore(answerMap, gb)
	convertMapToAnswerFile(answerMap, GeneratedAnswerFile)
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

func getAnswerFile(upAnswerFile string) string {
	// If the answer file is provided merge them and use the merged file as the answer file
	if upAnswerFile != "" {
		newAnswerMap, isConflict := mergeAnswerFiles(upAnswerFile)
		if isConflict {
			isAnswerFileClash := askOverrideAnswerFile()
			if !isAnswerFileClash {
				util.Fatal("Quitting deployment due to conflict in files.")
			}
		}
		upAnswerFile = MergedAnswerFile
		convertMapToAnswerFile(newAnswerMap, upAnswerFile)
	} else {
		upAnswerFile = GeneratedAnswerFile
	}
	return upAnswerFile
}

func mergeAnswerFiles(upAnswerFile string) (map[string]string, bool) {

	autoAnswerFile, err := blueprint.GetValuesFromAnswersFile(GeneratedAnswerFile)
	if err != nil {
		util.Fatal(err.Error())
	}

	providedAnswerFile, err := blueprint.GetValuesFromAnswersFile(upAnswerFile)
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

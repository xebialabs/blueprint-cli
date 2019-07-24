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

	// Infra blueprint
	err = blueprint.InstantiateBlueprint(blueprintTemplate, blueprintContext, gb, upAnswerFile, false, quickSetup, true)
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
		if k8s.IsPropertyPresent("xlVersion", answerMapFromConfigMap) {
			models.AvailableVersion = k8s.GetRequiredPropertyFromMap("xlVersion", answerMapFromConfigMap)
			answerMapFromConfigMap["xlVersion"] = ""
			answerMapFromConfigMap["prevVersion"] = models.AvailableVersion
			util.Verbose("Version %s is existing.\n", models.AvailableVersion)
		} else if k8s.IsPropertyPresent("prevVersion", answerMapFromConfigMap) {
			models.AvailableVersion = k8s.GetRequiredPropertyFromMap("prevVersion", answerMapFromConfigMap)
		}

		createLicenseAndKeystore(answerMapFromConfigMap, gb)

		createYamlFileFromMap(answerMapFromConfigMap, AnswerFileFromKubernetes)
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

	upAnswerFile = getAnswerFile(upAnswerFile)

	err = blueprint.InstantiateBlueprint(blueprintTemplate, blueprintContext, gb, upAnswerFile, false, quickSetup, true)
	if err != nil {
		util.Fatal("Error while creating Blueprint: %s \n", err)
	}

	util.IsQuiet = false

	applyFilesAndSave()

	util.Info("Generated files for deployment successfully! \nSpinning up xl seed! \n")

	runAndCaptureResponse(pullSeedImage)
	runAndCaptureResponse(runSeed())
}

func getAnswerFile(upAnswerFile string) string {
	// If the answer file is provided merge them and use the merged file as the answer file
	if upAnswerFile != "" {

		newAnswerMap, isConflict := mergeAnswerFiles(upAnswerFile)

		if isConflict {
			isAnswerFileClash := askOverrideAnswerFile()
			if !isAnswerFileClash {
				util.Fatal("Quitting deployment due to conflicting files")
			}
		}
		upAnswerFile = "merged_answer_file.yaml"

		createYamlFileFromMap(newAnswerMap, upAnswerFile)

	} else {
		upAnswerFile = AnswerFileFromKubernetes
	}
	return upAnswerFile
}

func createYamlFileFromMap(contents map[string]string, filename string) {
	yamlBytes, err := yaml.Marshal(contents)
	if err != nil {
		util.Fatal("Error when marshalling the answer map to yaml %s", err.Error())
	}

	err = ioutil.WriteFile(filename, yamlBytes, 0640)
	if err != nil {
		util.Fatal("Error when creating an answer file %s", err.Error())
	}
}

func mergeAnswerFiles(upAnswerFile string) (map[string]string, bool) {

	autoAnswerFile, err := blueprint.GetValuesFromAnswersFile(AnswerFileFromKubernetes)
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

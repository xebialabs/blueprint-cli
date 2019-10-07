package up

import (
	"fmt"
	"io/ioutil"
	"strings"
	"time"

	"gopkg.in/AlecAivazis/survey.v1"

	"gopkg.in/yaml.v2"

	"github.com/briandowns/spinner"
	"github.com/xebialabs/xl-cli/pkg/blueprint"
	"github.com/xebialabs/xl-cli/pkg/models"
	"github.com/xebialabs/xl-cli/pkg/util"
)

var s = spinner.New(spinner.CharSets[9], 100*time.Millisecond)
var applyValues map[string]string

// SkipPrompts can be set to true to skip asking prompts
var SkipPrompts = false

// InvokeBlueprintAndSeed will invoke blueprint and then call XL Seed
func InvokeBlueprintAndSeed(blueprintContext *blueprint.BlueprintContext, upParams UpParams, gitBranch string, gb *blueprint.GeneratedBlueprint) error {

	if !upParams.DryRun {
		defer util.StopAndRemoveContainer(s)
	}

	if upParams.AnswerFile == "" {
		if !(upParams.QuickSetup || upParams.AdvancedSetup) && !upParams.Undeploy {
			// ask for setup mode.
			mode, err := askSetupMode()

			if err != nil {
				return err
			}

			if mode == "quick" {
				upParams.QuickSetup = true
			}
		}
	}

	blueprint.SkipFinalPrompt = true
	util.IsQuiet = true

	var err error

	if upParams.LocalPath != "" {
		blueprintContext, err = blueprint.ConstructLocalBlueprintContext(upParams.LocalPath)
		if err != nil {
			return fmt.Errorf("error while creating local blueprint context: %s", err)
		}
	} else if upParams.LocalPath == "" && !upParams.CfgOverridden {
		upParams.BlueprintTemplate = DefaultInfraBlueprintTemplate
		repo, err := getRepo(gitBranch)
		if err != nil {
			return err
		}
		blueprintContext.ActiveRepo = &repo
	}

	answerFileToBlueprint := upParams.AnswerFile

	if answerFileToBlueprint != "" {
		if err = generateAnswerFile(answerFileToBlueprint, gb); err != nil {
			return err
		}
		answerFileToBlueprint = TempAnswerFile // TODO this can be kept in memory
	}

	// Infra blueprint, // This also generates an answer file for next blueprint
	preparedData, _, err := blueprint.InstantiateBlueprint(
		blueprint.BlueprintParams{
			TemplatePath:       upParams.BlueprintTemplate,
			AnswersFile:        answerFileToBlueprint,
			StrictAnswers:      false,
			UseDefaultsAsValue: upParams.QuickSetup,
			FromUpCommand:      true,
			PrintSummaryTable:  false,
		},
		blueprintContext, gb,
	)
	if err != nil {
		return fmt.Errorf("error while creating Infrastructure Blueprint: %s", err)
	}
	util.IsQuiet = false

	if upParams.Undeploy {
		if !SkipPrompts {
			shouldUndeploy := false
			err := survey.AskOne(&survey.Confirm{Message: models.UndeployConfirmationPrompt, Default: false}, &shouldUndeploy, nil)

			if err != nil {
				return err
			} else if shouldUndeploy == false {
				return fmt.Errorf("undeployment cancelled")
			}
		}

		kubeClient, err := getKubeClient()

		if err != nil {
			return err
		}

		if err = undeployAll(kubeClient); err != nil {
			return fmt.Errorf("an error occurred while undeploying - %s", err)
		}

		util.Info("Everything has been undeployed!\n")

		return nil
	}

	configMap := ""
	if !upParams.SkipK8sConnection {
		if configMap, err = getKubeConfigMap(); err != nil {
			return err
		}
	}

	if configMap != "" {
		util.Verbose("Update workflow started.... \n")

		answerMapFromConfigMap, err := parseConfigMap(configMap)
		if err != nil {
			return err
		}

		// Strip the version information
		models.AvailableOfficialXlrVersion, err = getVersion(answerMapFromConfigMap, "XlrOfficialVersion", "PrevXlrOfficialVersion")
		if err != nil {
			return err
		}
		if models.AvailableOfficialXlrVersion != "" {
			answerMapFromConfigMap["XlrOfficialVersion"] = ""
			answerMapFromConfigMap["PrevXlrOfficialVersion"] = models.AvailableOfficialXlrVersion
		}

		models.AvailableOfficialXldVersion, err = getVersion(answerMapFromConfigMap, "XldOfficialVersion", "PrevXldOfficialVersion")
		if err != nil {
			return err
		}
		if models.AvailableOfficialXldVersion != "" {
			answerMapFromConfigMap["XldOfficialVersion"] = ""
			answerMapFromConfigMap["PrevXldOfficialVersion"] = models.AvailableOfficialXldVersion
		}

		models.AvailableXlrVersion, err = getVersion(answerMapFromConfigMap, "XlrVersion", "PrevXlrVersion")
		if err != nil {
			return err
		}
		if models.AvailableXlrVersion != "" {
			answerMapFromConfigMap["XlrVersion"] = ""
			answerMapFromConfigMap["PrevXlrVersion"] = models.AvailableXlrVersion
		}

		models.AvailableXldVersion, err = getVersion(answerMapFromConfigMap, "XldVersion", "PrevXldVersion")
		if err != nil {
			return err
		}
		if models.AvailableXldVersion != "" {
			answerMapFromConfigMap["XldVersion"] = ""
			answerMapFromConfigMap["PrevXldVersion"] = models.AvailableXldVersion
		}

		if err = generateLicenseAndKeystore(answerMapFromConfigMap, gb); err != nil {
			return err
		}
		if err = convertMapToAnswerFile(answerMapFromConfigMap, AnswerFileFromConfigMap); err != nil { // can be in memory
			return err
		}
	} else {
		util.Verbose("Install workflow started")
	}

	util.IsQuiet = true
	if err = runApplicationBlueprint(&upParams, blueprintContext, gb, gitBranch, preparedData); err != nil {
		return err
	}
	util.IsQuiet = false

	if err = applyFilesAndSave(); err != nil {
		return err
	}

	util.Info("Generated files successfully! \n")

	if !upParams.DryRun {
		util.Info("Spinning up xl seed! \n")

		if err = runAndCaptureResponse(pullSeedImage); err != nil {
			return err
		}
		seed, err := runSeed()
		if err != nil {
			return err
		}

		if err = runAndCaptureResponse(seed); err != nil {
			return err
		}
	}
	return nil
}

func parseConfigMap(configMap string) (map[string]string, error) {
	util.Verbose("%s", configMap)
	answerMapFromConfigMap := make(map[string]string)

	if err := yaml.Unmarshal([]byte(configMap), &answerMapFromConfigMap); err != nil {
		return nil, fmt.Errorf("error parsing configMap: %s", err)
	}
	return answerMapFromConfigMap, nil
}

func runApplicationBlueprint(upParams *UpParams, blueprintContext *blueprint.BlueprintContext, gb *blueprint.GeneratedBlueprint, gitBranch string, preparedData *blueprint.PreparedData) error {
	var err error
	// Switch blueprint once the infrastructure is done.
	if upParams.BlueprintTemplate != "" {
		upParams.BlueprintTemplate = strings.Replace(upParams.BlueprintTemplate, DefaultInfraBlueprintTemplate, DefaultBlueprintTemplate, 1)
	} else {
		upParams.BlueprintTemplate = DefaultBlueprintTemplate
		repo, err := getRepo(gitBranch)
		if err != nil {
			return err
		}
		blueprintContext.ActiveRepo = &repo
	}
	// TODO don't read from answer files
	if upParams.AnswerFile != "" {
		upParams.AnswerFile, err = getAnswerFile(TempAnswerFile)
		if err != nil {
			return err
		}
		gb.GeneratedFiles = append(gb.GeneratedFiles, TempAnswerFile)
		gb.GeneratedFiles = append(gb.GeneratedFiles, MergedAnswerFile)
	} else {
		upParams.AnswerFile, err = getAnswerFile(upParams.AnswerFile)
		if err != nil {
			return err
		}
		gb.GeneratedFiles = append(gb.GeneratedFiles, AnswerFileFromConfigMap)
		gb.GeneratedFiles = append(gb.GeneratedFiles, MergedAnswerFile)
	}

	_, _, err = blueprint.InstantiateBlueprint(
		blueprint.BlueprintParams{
			TemplatePath:         upParams.BlueprintTemplate,
			AnswersFile:          upParams.AnswerFile,
			StrictAnswers:        false,
			UseDefaultsAsValue:   upParams.QuickSetup,
			FromUpCommand:        true,
			PrintSummaryTable:    true,
			ExistingPreparedData: preparedData,
		},
		blueprintContext, gb,
	)
	if err != nil {
		return fmt.Errorf("error while creating Blueprint: %s", err)
	}
	return nil
}

func generateAnswerFile(upAnswerFile string, gb *blueprint.GeneratedBlueprint) error {
	answerMap, err := convertAnswerFileToMap(upAnswerFile)
	if err != nil {
		return err
	}

	if err = generateLicenseAndKeystore(answerMap, gb); err != nil {
		return err
	}

	if err = convertMapToAnswerFile(answerMap, TempAnswerFile); err != nil {
		return err
	}
	gb.GeneratedFiles = append(gb.GeneratedFiles, TempAnswerFile)
	return nil
}

func convertAnswerFileToMap(answerFilePath string) (map[string]string, error) {
	answerMap := make(map[string]string)

	contents, err := ioutil.ReadFile(answerFilePath)

	if err != nil {
		return nil, fmt.Errorf("error reading answer file %s: %s", answerFilePath, err)
	}

	if err := yaml.Unmarshal(contents, &answerMap); err != nil {
		return nil, fmt.Errorf("error converting answer file %s", err)
	}

	return answerMap, nil
}

func convertMapToAnswerFile(contents map[string]string, filename string) error {
	var contentsInterface = map[string]interface{}{}
	for k, v := range contents {
		contentsInterface[k] = v
	}
	contentsInterface = blueprint.FixValueTypes(contentsInterface)

	yamlBytes, err := yaml.Marshal(contentsInterface)
	if err != nil {
		fmt.Errorf("error when marshalling the answer map to yaml %s", err.Error())
	}

	if err = ioutil.WriteFile(filename, yamlBytes, 0640); err != nil {
		fmt.Errorf("error when creating an answer file %s", err.Error())
	}
	return nil
}

func getVersion(answerMapFromConfigMap map[string]string, key, prevKey string) (string, error) {
	var version string
	if util.MapContainsKeyWithVal(answerMapFromConfigMap, key) {
		version = answerMapFromConfigMap[key]
		util.Verbose("Version %s is existing.\n", version)
	} else if util.MapContainsKeyWithVal(answerMapFromConfigMap, prevKey) {
		version = answerMapFromConfigMap[prevKey]
	}
	return version, nil
}

func getAnswerFile(answerFile string) (string, error) {
	// If the answer file is provided merge them and use the merged file as the answer file
	var err error
	if answerFile != "" {
		answerFile, err = mergeAndGetAnswerFile(answerFile)
		if err != nil {
			return "", err
		}
	} else {
		if util.PathExists(AnswerFileFromConfigMap, false) {
			answerFile, err = mergeAndGetAnswerFile(AnswerFileFromConfigMap)
			if err != nil {
				return "", err
			}
		} else {
			answerFile = GeneratedAnswerFile
		}
	}
	return answerFile, nil
}

func mergeAndGetAnswerFile(answerFile string) (string, error) {
	newAnswerMap, isConflict, err := mergeAnswerFiles(answerFile)
	if err != nil {
		return "", err
	}
	if isConflict && util.PathExists(AnswerFileFromConfigMap, false) {
		isAnswerFileClash, err := askOverrideAnswerFile()
		if err != nil {
			return "", err
		}
		if !isAnswerFileClash {
			return "", fmt.Errorf("quitting deployment due to conflict in files")
		}
	}
	answerFile = MergedAnswerFile
	if err = convertMapToAnswerFile(newAnswerMap, answerFile); err != nil {
		return "", err
	}

	return answerFile, nil
}

func mergeAnswerFiles(answerFile string) (map[string]string, bool, error) {

	autoAnswerFile, err := blueprint.GetValuesFromAnswersFile(GeneratedAnswerFile)
	if err != nil {
		return nil, false, err
	}

	providedAnswerFile, err := blueprint.GetValuesFromAnswersFile(answerFile)
	if err != nil {
		return nil, false, err
	}

	msg, err := VersionCheck(autoAnswerFile, providedAnswerFile)

	if err != nil {
		return nil, false, err
	}

	if msg != "" {
		util.Info(msg)
	} else {
		util.Verbose("No version provided, will ask the version in the application blueprint")
	}
	mergedAnswerFile, isConflict := mergeMaps(autoAnswerFile, providedAnswerFile)
	return mergedAnswerFile, isConflict, nil
}

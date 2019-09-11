package up

import (
	"fmt"
	"io/ioutil"
	"strings"
	"time"

	"github.com/xebialabs/xl-cli/pkg/cloud/k8s"

	"gopkg.in/yaml.v2"

	"github.com/briandowns/spinner"
	"github.com/xebialabs/xl-cli/pkg/blueprint"
	"github.com/xebialabs/xl-cli/pkg/models"
	"github.com/xebialabs/xl-cli/pkg/util"
)

var s = spinner.New(spinner.CharSets[9], 100*time.Millisecond)
var applyValues map[string]string

// SkipSeed when set to true will skip running xl-seed docker images
var SkipSeed = false

// SkipKube can be set to true to skip kubernetes connection activities
var SkipKube = false

// InvokeBlueprintAndSeed will invoke blueprint and then call XL Seed
func InvokeBlueprintAndSeed(blueprintContext *blueprint.BlueprintContext, upParams UpParams, branchVersion string) error {

	if !SkipSeed {
		defer util.StopAndRemoveContainer(s)
	}

	if upParams.AnswerFile == "" {
		if !(upParams.QuickSetup || upParams.AdvancedSetup) && !upParams.Destroy {
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
		repo, err := getRepo(branchVersion)
		if err != nil {
			return err
		}
		blueprintContext.ActiveRepo = &repo
	}

	gb := &blueprint.GeneratedBlueprint{OutputDir: models.BlueprintOutputDir}

	if !upParams.NoCleanup {
		defer gb.Cleanup()
	}

	answerFileToBlueprint := upParams.AnswerFile

	if answerFileToBlueprint != "" {
		err = generateAnswerFile(answerFileToBlueprint, gb)
		if err != nil {
			return err
		}
		answerFileToBlueprint = TempAnswerFile
	}

	// Infra blueprint
	err = blueprint.InstantiateBlueprint(upParams.BlueprintTemplate, blueprintContext, gb, answerFileToBlueprint, false, upParams.QuickSetup, true, false)
	if err != nil {
		return fmt.Errorf("error while creating Infrastructure Blueprint: %s", err)
	}
	util.IsQuiet = false

	configMap := ""
	if !SkipKube {
		configMap, err = getKubeConfigMap()
		if err != nil {
			return err
		}
	}

	if upParams.Destroy {
		// InvokeDestroy(blueprintContext, upParams, branchVersion, configMap, gb)
		return nil
	}

	if configMap != "" {
		util.Verbose("Update workflow started.... \n")

		answerMapFromConfigMap, err := parseConfigMap(configMap)
		if err != nil {
			return err
		}

		// Strip the version information
		models.AvailableVersion, err = getVersion(answerMapFromConfigMap, "xlVersion", "prevVersion")
		if err != nil {
			return err
		}
		if models.AvailableVersion != "" {
			answerMapFromConfigMap["xlVersion"] = ""
			answerMapFromConfigMap["prevVersion"] = models.AvailableVersion
		}

		models.AvailableXlrVersion, err = getVersion(answerMapFromConfigMap, "xlrVersion", "prevXlrVersion")
		if err != nil {
			return err
		}
		if models.AvailableXlrVersion != "" {
			answerMapFromConfigMap["xlrVersion"] = ""
			answerMapFromConfigMap["prevXlrVersion"] = models.AvailableXlrVersion
		}

		models.AvailableXldVersion, err = getVersion(answerMapFromConfigMap, "xldVersion", "prevXldVersion")
		if err != nil {
			return err
		}
		if models.AvailableXldVersion != "" {
			answerMapFromConfigMap["xldVersion"] = ""
			answerMapFromConfigMap["prevXldVersion"] = models.AvailableXldVersion
		}

		err = generateLicenseAndKeystore(answerMapFromConfigMap, gb)
		if err != nil {
			return err
		}
		err = convertMapToAnswerFile(answerMapFromConfigMap, GeneratedAnswerFile)
		if err != nil {
			return err
		}
	} else {
		util.Verbose("Install workflow started")
	}

	util.IsQuiet = true
	err = runApplicationBlueprint(&upParams, blueprintContext, gb, branchVersion)
	if err != nil {
		return err
	}
	util.IsQuiet = false

	err = applyFilesAndSave()
	if err != nil {
		return err
	}

	util.Info("Generated files for deployment successfully! \nSpinning up xl seed! \n")

	if !SkipSeed {
		err = runAndCaptureResponse(pullSeedImage)
		if err != nil {
			return err
		}
		seed, err := runSeed(false)
		if err != nil {
			return err
		}
		err = runAndCaptureResponse(seed)
		if err != nil {
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

func runApplicationBlueprint(upParams *UpParams, blueprintContext *blueprint.BlueprintContext, gb *blueprint.GeneratedBlueprint, branchVersion string) error {
	var err error
	// Switch blueprint once the infrastructure is done.
	if upParams.BlueprintTemplate != "" {
		upParams.BlueprintTemplate = strings.Replace(upParams.BlueprintTemplate, DefaultInfraBlueprintTemplate, DefaultBlueprintTemplate, 1)
	} else {
		upParams.BlueprintTemplate = DefaultBlueprintTemplate
		repo, err := getRepo(branchVersion)
		if err != nil {
			return err
		}
		blueprintContext.ActiveRepo = &repo
	}

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
	}

	err = blueprint.InstantiateBlueprint(upParams.BlueprintTemplate, blueprintContext, gb, upParams.AnswerFile, false, upParams.QuickSetup, true, true)
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
	err = generateLicenseAndKeystore(answerMap, gb)
	if err != nil {
		return err
	}
	err = convertMapToAnswerFile(answerMap, TempAnswerFile)
	if err != nil {
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
	yamlBytes, err := yaml.Marshal(contents)
	if err != nil {
		fmt.Errorf("error when marshalling the answer map to yaml %s", err.Error())
	}

	err = ioutil.WriteFile(filename, yamlBytes, 0640)
	if err != nil {
		fmt.Errorf("error when creating an answer file %s", err.Error())
	}
	return nil
}

func getVersion(answerMapFromConfigMap map[string]string, key, prevKey string) (string, error) {
	var version string
	var err error
	if k8s.IsPropertyPresent(key, answerMapFromConfigMap) {
		version, err = k8s.GetRequiredPropertyFromMap(key, answerMapFromConfigMap)
		if err != nil {
			return "", err
		}
		util.Verbose("Version %s is existing.\n", version)
	} else if k8s.IsPropertyPresent(prevKey, answerMapFromConfigMap) {
		version, err = k8s.GetRequiredPropertyFromMap(prevKey, answerMapFromConfigMap)
		if err != nil {
			return "", err
		}
	}
	return version, nil
}

func getAnswerFile(answerFile string) (string, error) {
	// If the answer file is provided merge them and use the merged file as the answer file
	if answerFile != "" {
		newAnswerMap, isConflict, err := mergeAnswerFiles(answerFile)
		if err != nil {
			return "", err
		}
		if isConflict {
			isAnswerFileClash, err := askOverrideAnswerFile()
			if err != nil {
				return "", err
			}
			if !isAnswerFileClash {
				fmt.Errorf("quitting deployment due to conflict in files")
			}
		}
		answerFile = MergedAnswerFile
		err = convertMapToAnswerFile(newAnswerMap, answerFile)
		if err != nil {
			return "", err
		}
	} else {
		answerFile = GeneratedAnswerFile
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

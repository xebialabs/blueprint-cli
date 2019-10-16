package up

import (
	b64 "encoding/base64"
	"fmt"
	"strings"
	"time"

	"gopkg.in/AlecAivazis/survey.v1"

	"gopkg.in/yaml.v2"

	"github.com/briandowns/spinner"
	"github.com/xebialabs/xl-cli/pkg/blueprint"
	"github.com/xebialabs/xl-cli/pkg/blueprint/repository"
	"github.com/xebialabs/xl-cli/pkg/models"
	"github.com/xebialabs/xl-cli/pkg/util"
)

var s = spinner.New(spinner.CharSets[9], 100*time.Millisecond)
var applyValues map[string]string

// SkipPrompts can be set to true to skip asking prompts
var SkipPrompts = false

// InvokeBlueprintAndSeed will invoke blueprint and then call XL Seed
func InvokeBlueprintAndSeed(blueprintContext *blueprint.BlueprintContext, upParams UpParams, CliVersion string, gb *blueprint.GeneratedBlueprint) error {

	if !upParams.DryRun {
		defer util.StopAndRemoveContainer(s)
	}

	if upParams.SkipPrompts {
		SkipPrompts = true
		blueprint.SkipUpFinalPrompt = true
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
	} else if !upParams.CfgOverridden {
		var repo repository.BlueprintRepository
		if upParams.GITBranch != "" {
			repo, err = getGitRepo(upParams.GITBranch)
		} else {
			repo, err = getHttpRepo(CliVersion)
		}
		if err != nil {
			return err
		}
		blueprintContext.ActiveRepo = &repo
	}

	if upParams.BlueprintTemplate == "" {
		upParams.BlueprintTemplate = DefaultInfraBlueprintTemplate
	}

	var answers map[string]string

	if upParams.AnswerFile != "" {
		// Update the user provided answerfile with License and Keystore information if needed
		answers, err = generateAnswerFile(upParams.AnswerFile, gb)
		if err != nil {
			return err
		}
	}

	// xl-infra blueprint, This generates answer data for next blueprint
	preparedData, _, err := blueprint.InstantiateBlueprint(
		blueprint.BlueprintParams{
			TemplatePath:       upParams.BlueprintTemplate,
			AnswersMap:         answers,
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

	// adjust the generated values from xl-infra blueprint
	answersFromInfra := processAnswerMapFromPreparedData(preparedData)

	if upParams.Undeploy {
		if !SkipPrompts {
			shouldUndeploy := false
			err := survey.AskOne(&survey.Confirm{Message: models.UndeployConfirmationPrompt, Default: false}, &shouldUndeploy, nil)

			if err != nil {
				return err
			}
			if shouldUndeploy == false {
				return fmt.Errorf("undeployment cancelled by user")
			}
		}

		kubeClient, err := getKubeClient(answersFromInfra)

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
		if configMap, err = getKubeConfigMap(answersFromInfra); err != nil {
			return err
		}
	}

	if configMap != "" {
		if !SkipPrompts {
			shouldUpdate := false
			err := survey.AskOne(&survey.Confirm{Message: models.UpdateConfirmationPrompt, Default: false}, &shouldUpdate, nil)

			if err != nil {
				return err
			}
			if shouldUpdate == false {
				return fmt.Errorf("Update cancelled by user")
			}
		}
		util.Verbose("Update workflow started.... \n")

		answersFromConfigMap, err := parseConfigMap(configMap)
		if err != nil {
			return err
		}

		// Strip the version information
		models.AvailableOfficialXlrVersion = getVersion(answersFromConfigMap, "XlrOfficialVersion", "PrevXlrOfficialVersion")
		if models.AvailableOfficialXlrVersion != "" {
			answersFromConfigMap["XlrOfficialVersion"] = ""
			answersFromConfigMap["PrevXlrOfficialVersion"] = models.AvailableOfficialXlrVersion
		}

		models.AvailableOfficialXldVersion = getVersion(answersFromConfigMap, "XldOfficialVersion", "PrevXldOfficialVersion")
		if models.AvailableOfficialXldVersion != "" {
			answersFromConfigMap["XldOfficialVersion"] = ""
			answersFromConfigMap["PrevXldOfficialVersion"] = models.AvailableOfficialXldVersion
		}

		models.AvailableXlrVersion = getVersion(answersFromConfigMap, "XlrVersion", "PrevXlrVersion")
		if models.AvailableXlrVersion != "" {
			answersFromConfigMap["XlrVersion"] = ""
			answersFromConfigMap["PrevXlrVersion"] = models.AvailableXlrVersion
		}

		models.AvailableXldVersion = getVersion(answersFromConfigMap, "XldVersion", "PrevXldVersion")
		if models.AvailableXldVersion != "" {
			answersFromConfigMap["XldVersion"] = ""
			answersFromConfigMap["PrevXldVersion"] = models.AvailableXldVersion
		}

		if err = generateLicenseAndKeystore(answersFromConfigMap, gb); err != nil {
			return err
		}
		if upParams.AnswerFile == "" {
			// answerfile is not present, see if there is an answerfile from k8s configmap
			answers = answersFromConfigMap
			answers["FromConfigMap"] = "true"
		}
	} else {
		util.Verbose("Install workflow started")
	}

	util.IsQuiet = true
	if err = runApplicationBlueprint(&upParams, blueprintContext, gb, CliVersion, preparedData, answers, answersFromInfra); err != nil {
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

func processAnswerMapFromPreparedData(preparedData *blueprint.PreparedData) map[string]string {
	if !util.MapContainsKeyWithValInterface(preparedData.TemplateData, "K8sToken") {
		if !util.MapContainsKeyWithValInterface(preparedData.TemplateData, "K8sClientCert") {
			preparedData.TemplateData["K8sClientCertFile"] = preparedData.TemplateData["CertFile"]
			preparedData.TemplateData["K8sClientKeyFile"] = preparedData.TemplateData["KeyFile"]
		} else {
			preparedData.TemplateData["K8sClientCert"] = b64.StdEncoding.EncodeToString([]byte(preparedData.TemplateData["K8sClientCert"].(string)))
			preparedData.TemplateData["K8sClientKey"] = b64.StdEncoding.EncodeToString([]byte(preparedData.TemplateData["K8sClientKey"].(string)))
		}
	}

	answers := map[string]string{}

	for k, v := range preparedData.TemplateData {
		answers[k] = fmt.Sprintf("%v", v)
	}

	return answers
}

func parseConfigMap(configMap string) (map[string]string, error) {
	util.Verbose("%s", configMap)
	answerMapFromConfigMap := make(map[string]string)

	if err := yaml.Unmarshal([]byte(configMap), &answerMapFromConfigMap); err != nil {
		return nil, fmt.Errorf("error parsing configMap: %s", err)
	}
	return answerMapFromConfigMap, nil
}

func runApplicationBlueprint(upParams *UpParams, blueprintContext *blueprint.BlueprintContext, gb *blueprint.GeneratedBlueprint, CliVersion string, preparedData *blueprint.PreparedData, answers, answersFromInfra map[string]string) error {
	var err error
	// Switch blueprint once the infrastructure is done.
	if upParams.BlueprintTemplate != "" && strings.Contains(upParams.BlueprintTemplate, DefaultInfraBlueprintTemplate) {
		upParams.BlueprintTemplate = strings.Replace(upParams.BlueprintTemplate, DefaultInfraBlueprintTemplate, DefaultBlueprintTemplate, 1)
	} else {
		upParams.BlueprintTemplate = DefaultBlueprintTemplate
	}

	if answers != nil {
		answers, err = mergeAndGetAnswers(answers, answersFromInfra)
		if err != nil {
			return err
		}
	}

	_, _, err = blueprint.InstantiateBlueprint(
		blueprint.BlueprintParams{
			TemplatePath:         upParams.BlueprintTemplate,
			AnswersMap:           answers,
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

func getVersion(answerMapFromConfigMap map[string]string, key, prevKey string) string {
	var version string
	if util.MapContainsKeyWithVal(answerMapFromConfigMap, key) {
		version = answerMapFromConfigMap[key]
		util.Verbose("Version %s is existing.\n", version)
	} else if util.MapContainsKeyWithVal(answerMapFromConfigMap, prevKey) {
		version = answerMapFromConfigMap[prevKey]
	}
	return version
}

func generateAnswerFile(upAnswerFile string, gb *blueprint.GeneratedBlueprint) (map[string]string, error) {
	answerMap, err := blueprint.GetValuesFromAnswersFile(upAnswerFile)
	if err != nil {
		return nil, err
	}

	if err = generateLicenseAndKeystore(answerMap, gb); err != nil {
		return nil, err
	}

	return answerMap, nil
}

func mergeAndGetAnswers(answers, answersFromInfra map[string]string) (map[string]string, error) {
	newAnswers, isConflict, err := mergeAnswerFiles(answers, answersFromInfra)
	if err != nil {
		return nil, err
	}
	if isConflict && answers["FromConfigMap"] != "" {
		isAnswerFileClash, err := askOverrideAnswerFile()
		if err != nil {
			return nil, err
		}
		if !isAnswerFileClash {
			return nil, fmt.Errorf("quitting deployment due to conflict in files")
		}
	}

	return newAnswers, nil
}

func mergeAnswerFiles(answers, answersFromInfra map[string]string) (map[string]string, bool, error) {
	msg, err := VersionCheck(answersFromInfra, answers)

	if err != nil {
		return nil, false, err
	}

	if msg != "" {
		util.Info(msg)
	} else {
		util.Verbose("No version provided, will ask the version in the application blueprint")
	}
	mergedAnswers, isConflict := mergeMaps(answersFromInfra, answers)
	return mergedAnswers, isConflict, nil
}

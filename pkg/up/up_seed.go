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
	versionHelper "github.com/xebialabs/xl-cli/pkg/version"
)

var s = spinner.New(spinner.CharSets[9], 100*time.Millisecond)
var applyValues map[string]string

const (
	CurrentXLDVersionSupported = "9.5.1"
	CurrentXLRVersionSupported = "9.5.2"
)

// SkipPrompts can be set to true to skip asking prompts
var SkipPrompts = false
var MockConfigMap = ""
var ForceConfigMap = false
var EmptyVersion = ""

// InvokeBlueprintAndSeed will invoke blueprint and then call XL Seed
func InvokeBlueprintAndSeed(blueprintContext *blueprint.BlueprintContext, upParams UpParams, CliVersion string, gb *blueprint.GeneratedBlueprint) error {

	versionHelper.AvailableXldVersions = getAvailableVersions(upParams.XLDVersions, []string{CurrentXLDVersionSupported})
	versionHelper.AvailableXlrVersions = getAvailableVersions(upParams.XLRVersions, []string{CurrentXLRVersionSupported})

	if !upParams.DryRun {
		defer StopAndRemoveContainer(s)
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

	util.Verbose("[xl up] Passed parameters: \n%+v\n", upParams)

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
	var defaultFromValues map[string]string

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
	} else {
		// Used only for testing purposes
		configMap = MockConfigMap
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
		util.Verbose("[xl up] Update workflow started.... \n")

		answersFromConfigMap, err := parseConfigMap(configMap)
		if err != nil {
			return err
		}

		// Strip the version information
		models.AvailableOfficialXlrVersion = getVersion(answersFromConfigMap, "XlrOfficialVersion", "PrevXlrOfficialVersion")
		if models.AvailableOfficialXlrVersion != "" {
			answersFromConfigMap["XlrOfficialVersion"] = EmptyVersion
			answersFromConfigMap["PrevXlrOfficialVersion"] = models.AvailableOfficialXlrVersion
		}

		models.AvailableOfficialXldVersion = getVersion(answersFromConfigMap, "XldOfficialVersion", "PrevXldOfficialVersion")
		if models.AvailableOfficialXldVersion != "" {
			answersFromConfigMap["XldOfficialVersion"] = EmptyVersion
			answersFromConfigMap["PrevXldOfficialVersion"] = models.AvailableOfficialXldVersion
		}

		models.AvailableXlrVersion = getVersion(answersFromConfigMap, "XlrVersion", "PrevXlrVersion")
		if models.AvailableXlrVersion != "" {
			answersFromConfigMap["PrevXlrVersion"] = models.AvailableXlrVersion
		}

		models.AvailableXldVersion = getVersion(answersFromConfigMap, "XldVersion", "PrevXldVersion")
		if models.AvailableXldVersion != "" {
			answersFromConfigMap["PrevXldVersion"] = models.AvailableXldVersion
		}

		if err = generateLicenseAndKeystore(answersFromConfigMap, gb); err != nil {
			return err
		}

		if upParams.AnswerFile == "" || ForceConfigMap {
			// answerfile is not present, see if there is an answerfile from k8s configmap
			answers = answersFromConfigMap
			answers["FromConfigMap"] = "true"
			util.Verbose("[xl up] config map is used as answers\n")
			// Prepare parameters that can be overriden
			if !SkipPrompts {
				shouldUpdate := false
				err := survey.AskOne(&survey.Confirm{Message: models.UpdateParamsConfirmationPrompt, Default: false}, &shouldUpdate, nil)

				if err != nil {
					return err
				}
				if shouldUpdate {
					util.Verbose("[xl up] config map is used as defaults\n")
					upParams.QuickSetup = false
					defaultFromValues = util.CopyIntoStringStringMap(answersFromConfigMap, make(map[string]string))
					if val, ok := defaultFromValues["InstallXLD"]; ok && val == "true" {
						delete(defaultFromValues, "InstallXLD")
					}
					if val, ok := defaultFromValues["InstallXLR"]; ok && val == "true" {
						delete(defaultFromValues, "InstallXLR")
					}
					if val, ok := defaultFromValues["MonitoringInstall"]; ok && val == "true" {
						delete(defaultFromValues, "MonitoringInstall")
					}
				}
			}
		}

	} else {
		util.Verbose("[xl up] Install workflow started\n")
	}

	util.IsQuiet = true
	if err = runApplicationBlueprint(&upParams, blueprintContext, gb, CliVersion, preparedData, answers, answersFromInfra, defaultFromValues); err != nil {
		return err
	}
	util.IsQuiet = false

	if err = applyFilesAndSave(); err != nil {
		return err
	}

	util.Info("Generated files successfully! \n")

	if !upParams.DryRun {
		util.Info("Spinning up xl seed! \n")

		if err = runAndCaptureResponse(pullSeedImage(upParams.SeedVersion)); err != nil {
			return err
		}
		seed, err := runSeed(upParams.SeedVersion, upParams.RollingUpdate)
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

	answers := make(map[string]string)

	for k, v := range preparedData.TemplateData {
		answers[k] = fmt.Sprintf("%v", v)
	}

	return answers
}

func parseConfigMap(configMap string) (map[string]string, error) {
	util.Verbose("[xl up] Config map: %s\n", configMap)
	answerMapFromConfigMap := make(map[string]string)

	if err := yaml.Unmarshal([]byte(configMap), &answerMapFromConfigMap); err != nil {
		return nil, fmt.Errorf("error parsing configMap: %s", err)
	}
	return answerMapFromConfigMap, nil
}

func runApplicationBlueprint(
	upParams *UpParams,
	blueprintContext *blueprint.BlueprintContext,
	gb *blueprint.GeneratedBlueprint,
	CliVersion string,
	preparedData *blueprint.PreparedData,
	answers, answersFromInfra, defaultFromValues map[string]string,
) error {
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
			OverrideDefaults:     defaultFromValues,
		},
		blueprintContext, gb,
	)
	if err != nil {
		return fmt.Errorf("error while creating Blueprint: %s", err)
	}
	return nil
}

func getAvailableVersions(versions string, defaultVersions []string) []string {
	if versions != "" && versions != "undefined" {
		versionSlice := []string{}
		for _, ver := range strings.Split(versions, ",") {
			versionSlice = append(versionSlice, strings.TrimSpace(ver))
		}
		return versionSlice
	}
	return defaultVersions
}

func getVersion(answerMapFromConfigMap map[string]string, key, prevKey string) string {
	var version string
	if util.MapContainsKeyWithVal(answerMapFromConfigMap, key) {
		version = answerMapFromConfigMap[key]
		util.Verbose("[xl up] Version %s is existing.\n", version)
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
		util.Verbose("[xl up] No version provided, will ask the version in the application blueprint\n")
	}
	mergedAnswers, isConflict := mergeMaps(answersFromInfra, answers)
	return mergedAnswers, isConflict, nil
}

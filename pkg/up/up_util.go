package up

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"

	"github.com/xebialabs/xl-cli/pkg/cloud/k8s"

	"github.com/xebialabs/xl-cli/pkg/blueprint"
	"github.com/xebialabs/xl-cli/pkg/blueprint/repository"
	"github.com/xebialabs/xl-cli/pkg/blueprint/repository/github"
	"github.com/xebialabs/xl-cli/pkg/models"
	"github.com/xebialabs/xl-cli/pkg/util"
)

const (
	Docker                        = "docker"
	SeedImage                     = "xl-docker.xebialabs.com/xl-seed:demo"
	Kubernetes                    = "kubernetes"
	Xebialabs                     = "xebialabs"
	XlUpBlueprint                 = "xl-up-blueprint"
	DefaultInfraBlueprintTemplate = "xl-infra"
	DefaultBlueprintTemplate      = "xl-up"
	GeneratedAnswerFile           = "cm_answer_file_auto.yaml"
	TempAnswerFile                = "temp_answer_file_auto.yaml"
	MergedAnswerFile              = "merged_answer_file.yaml"
	ConfigMapName                 = "answers-config-map"
	DataFile                      = "answers.yaml"
)

var pullSeedImage = models.Command{
	Name: Docker,
	Args: []string{"pull", SeedImage},
}

func runSeed() models.Command {
	dir, err := os.Getwd()

	if err != nil {
		util.Fatal("Error while getting current work directory")
	}

	return models.Command{
		Name: Docker,
		Args: []string{"run", "--name", "xl-seed", "-v", dir + ":/data", SeedImage, "--init", "xebialabs/common.yaml", "xebialabs.yaml"},
	}
}

func getLocalContext(templatePath string) (*blueprint.BlueprintContext, string, error) {
	if len(templatePath) < 1 {
		return nil, "", fmt.Errorf("template path cannot be empty")
	}

	// add leading slash if not there
	if templatePath[len(templatePath)-1:] != string(os.PathSeparator) {
		templatePath += string(os.PathSeparator)
	}

	// prepare local context from provided template path
	blueprintDir := filepath.Dir(templatePath)
	paths := strings.Split(blueprintDir, string(os.PathSeparator))
	if runtime.GOOS != "windows" {
		paths = append([]string{string(os.PathSeparator)}, paths[:len(paths)-1]...)
	}
	parentDir := filepath.Join(paths[:len(paths)-1]...)
	blueprintContext, err := blueprint.ConstructLocalBlueprintContext(parentDir)
	if err != nil {
		return nil, "", err
	}

	// adjust relative template path from provided path
	blueprintTemplate := paths[len(paths)-1]
	return blueprintContext, blueprintTemplate, nil
}

func getRepo(branchVersion string) repository.BlueprintRepository {

	repo, err := github.NewGitHubBlueprintRepository(map[string]string{
		"name":      XlUpBlueprint,
		"repo-name": XlUpBlueprint,
		"owner":     Xebialabs,
		"branch":    branchVersion,
	})

	if err != nil {
		util.Fatal("Error while creating Blueprint: %s \n", err)
	}

	return repo
}

func generateLicenseAndKeystore(answerMapFromConfigMap map[string]string, gb *blueprint.GeneratedBlueprint) {
	GenerateFileAndUpdateProperty("xlrLic", "xl-release.lic", answerMapFromConfigMap, gb)
	GenerateFileAndUpdateProperty("xldLic", "deploy-it.lic", answerMapFromConfigMap, gb)
	GenerateFileAndUpdateProperty("xlKeyStore", "keystore.jceks", answerMapFromConfigMap, gb)
}

func isBase64Encoded(content string) bool {
	re := regexp.MustCompile(`^([A-Za-z0-9+/]{4})*([A-Za-z0-9+/]{3}=|[A-Za-z0-9+/]{2}==)?$`)
	return re.Match([]byte(content))
}

func GenerateFileAndUpdateProperty(propertyName, newPropertyValue string, answerMapFromConfigMap map[string]string, gb *blueprint.GeneratedBlueprint) {
	if k8s.IsPropertyPresent(propertyName, answerMapFromConfigMap) {
		propertyValue := k8s.GetRequiredPropertyFromMap(propertyName, answerMapFromConfigMap)

		isBase64 := isBase64Encoded(propertyValue)

		if !isBase64 {
			f, err := ioutil.ReadFile(propertyValue)
			if err != nil {
				util.Fatal("Error reading the value of %s - %s", propertyName, err)
			}
			propertyValue = string(f)
		}

		util.Verbose("writing %s", newPropertyValue)

        if _, err := os.Stat(models.BlueprintOutputDir); os.IsNotExist(err) {
            err := os.Mkdir(models.BlueprintOutputDir, os.ModePerm)
            if err != nil {
                util.Fatal("Error creating %s folder", models.BlueprintOutputDir, err)
            }
        }

		location := filepath.Join(models.BlueprintOutputDir, newPropertyValue)

		var err error

		if isBase64 {
            err = ioutil.WriteFile(location, k8s.DecodeBase64(propertyValue), 0640)
        } else {
            err = ioutil.WriteFile(location, []byte(propertyValue), 0640)
        }

		if err != nil {
			util.Fatal("Error creating file %s - %s", newPropertyValue, err)
		}
		answerMapFromConfigMap[propertyName] = location
		gb.GeneratedFiles = append(gb.GeneratedFiles, location)
	}
}

func mergeMaps(autoAnswerFile, providedAnswerFile map[string]string) (map[string]string, bool) {

	mergedAnswerFile := make(map[string]string)

	isConflict := false

	for autoKey, autoValue := range autoAnswerFile {
		askQuestion := false
		for providedKey, providedValue := range providedAnswerFile {
			if autoKey == providedKey {
				askQuestion = true
				if autoValue != providedValue {
					isConflict = true
				}
			}
		}
		if askQuestion {
			delete(providedAnswerFile, autoKey)
		}
		mergedAnswerFile[autoKey] = autoValue
	}

	for providedKey, providedValue := range providedAnswerFile {
		mergedAnswerFile[providedKey] = providedValue
	}

	return mergedAnswerFile, isConflict
}

func VersionCheck(autoAnswerFile map[string]string, providedAnswerFile map[string]string) (string, error) {
	// Strip the version information - if the value is provided to the up command.
	if k8s.IsPropertyPresent("xlVersion", providedAnswerFile) {
		var versionFromKubernetesConfigMap string
		versionFromAnswerFileProvided := k8s.GetRequiredPropertyFromMap("xlVersion", providedAnswerFile)

		if k8s.IsPropertyPresent("prevVersion", autoAnswerFile) {
			versionFromKubernetesConfigMap = k8s.GetRequiredPropertyFromMap("prevVersion", autoAnswerFile)
		}

		return decideVersionMatch(versionFromKubernetesConfigMap, versionFromAnswerFileProvided)
	}

	if k8s.IsPropertyPresent("xlrVersion", providedAnswerFile) {
		var versionFromKubernetesConfigMap string
		versionFromAnswerFileProvided := k8s.GetRequiredPropertyFromMap("xlrVersion", providedAnswerFile)

		if k8s.IsPropertyPresent("prevXlrVersion", autoAnswerFile) {
			versionFromKubernetesConfigMap = k8s.GetRequiredPropertyFromMap("prevXlrVersion", autoAnswerFile)
		}

		return decideVersionMatch(versionFromKubernetesConfigMap, versionFromAnswerFileProvided)
	}

	if k8s.IsPropertyPresent("xldVersion", providedAnswerFile) {
		var versionFromKubernetesConfigMap string
		versionFromAnswerFileProvided := k8s.GetRequiredPropertyFromMap("xldVersion", providedAnswerFile)

		if k8s.IsPropertyPresent("prevXldVersion", autoAnswerFile) {
			versionFromKubernetesConfigMap = k8s.GetRequiredPropertyFromMap("prevXldVersion", autoAnswerFile)
		}

		return decideVersionMatch(versionFromKubernetesConfigMap, versionFromAnswerFileProvided)
	}

	return "", nil
}

func decideVersionMatch(installedVersion string, newVersion string) (string, error) {
	installed := util.ParseVersion(installedVersion, 4)
	versionToInstall := util.ParseVersion(newVersion, 4)

	if installed != 0 {
		switch {
		case installed > versionToInstall:
			return "", fmt.Errorf("cannot downgrade the deployment from %s to %s", installedVersion, newVersion)
		case installed < versionToInstall:
			return fmt.Sprintf("upgrading from %s to %s", installedVersion, newVersion), nil
		case installed == versionToInstall:
			return "", fmt.Errorf("the given version %s already exists", installedVersion)
		}
	}

	return "", nil
}

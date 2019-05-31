package up

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
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
	AnswerFileFromKubernetes      = "cm_answer_file_auto.yaml"
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
		Args: []string{"run", "-v", dir + ":/data", SeedImage, "--init", "xebialabs/common.yaml", "xebialabs.yaml"},
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

func createFileFromConfigMap(answerMapFromConfigMap map[string]string) {
	createFile("xlrLic", "xl-release.lic", answerMapFromConfigMap)
	createFile("xldLic", "deploy-it.lic", answerMapFromConfigMap)
	createFile("xlKeyStore", "keystore.jceks", answerMapFromConfigMap)
}

func createFile(propertyName, fileName string, answerMapFromConfigMap map[string]string) {
	if k8s.IsPropertyPresent(propertyName, answerMapFromConfigMap) {
		util.Verbose("writing %s", fileName)
		content := k8s.DecodeBase64(k8s.GetRequiredPropertyFromMap(propertyName, answerMapFromConfigMap))
		ioutil.WriteFile(fileName, []byte(content), 0640)
		answerMapFromConfigMap[propertyName] = fileName
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

	return "", nil
}

func decideVersionMatch(installedVersion string, newVersion string) (string, error) {
	installed := util.ParseVersion(installedVersion, 4)
	versionToInstall := util.ParseVersion(newVersion, 4)

	switch {
	case installed > versionToInstall:
		return "", fmt.Errorf("cannot downgrade the deployment from %s to %s", installedVersion, newVersion)
	case installed < versionToInstall:
		return fmt.Sprintf("upgrading from %s to %s", installedVersion, newVersion), nil
	case installed == versionToInstall:
		return "", fmt.Errorf("the given version %s already exists", installedVersion)
	}

	return "", nil
}

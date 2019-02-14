package blueprint

import (
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strings"

	"gopkg.in/AlecAivazis/survey.v1"

	"github.com/xebialabs/xl-cli/pkg/models"
	"github.com/xebialabs/xl-cli/pkg/repository"
	"github.com/xebialabs/xl-cli/pkg/repository/github"
	"github.com/xebialabs/xl-cli/pkg/repository/mock"
	"github.com/xebialabs/xl-cli/pkg/util"
)

// TemplateConfig holds the merged template file definitions with repository info
type TemplateConfig struct {
	File           string
	FullPath       string
	DependsOnTrue  VarField
	DependsOnFalse VarField
	Repository     BlueprintContext
}

const templateExtension = ".tmpl"

var blueprintRepository repository.BlueprintRepository

type BlueprintContext struct {
	Provider string
	Name     string
	Owner    string
	Token    string
	Branch   string
}

/*
 * ---------------------------
 * Blueprint Context Functions
 * ---------------------------
 */
func (blueprintContext *BlueprintContext) initRepoClient() (map[string]*models.BlueprintRemote, error) {
	switch blueprintContext.Provider {
	case models.ProviderMock: // only used for testing purposes
		blueprintRepository = mock.NewMockBlueprintRepository(
			blueprintContext.Name,
			blueprintContext.Owner,
			blueprintContext.Branch,
		)
	case models.ProviderGitHub:
		blueprintRepository = github.NewGitHubBlueprintRepository(
			blueprintContext.Name,
			blueprintContext.Owner,
			blueprintContext.Branch,
			blueprintContext.Token,
		)
	default:
		return nil, fmt.Errorf("no blueprint provider implementation found for %s", blueprintContext.Provider)
	}
	return blueprintContext.parseRepositoryTree()
}

func (blueprintContext *BlueprintContext) parseRepositoryTree() (map[string]*models.BlueprintRemote, error) {
	var blueprints map[string]*models.BlueprintRemote
	var blueprintDirs []string
	var err error

	// Parse GIT tree from provider
	blueprints, blueprintDirs, err = blueprintRepository.ListBlueprintsFromRepo()
	if err != nil {
		return nil, err
	}

	// Clear non-blueprint items in result map
	for path := range blueprints {
		if !util.IsStringInSlice(path, blueprintDirs) {
			delete(blueprints, path)
		}
	}
	return blueprints, nil
}

func (blueprintContext *BlueprintContext) askUserToChooseBlueprint(blueprints map[string]*models.BlueprintRemote, blueprintTemplate string, surveyOpts ...survey.AskOpt) (string, error) {
	if blueprintTemplate == "" {
		var blueprintKeys []string
		for k := range blueprints {
			blueprintKeys = append(blueprintKeys, k)
		}
		if len(blueprintKeys) == 0 {
			return "", fmt.Errorf("no blueprints found in repository [%s - branch: %s]", blueprintContext.Name, blueprintContext.Branch)
		}
		sort.Strings(blueprintKeys)

		_ = survey.AskOne(
			&survey.Select{
				Message: "Choose a blueprint:",
				Options: blueprintKeys,
				Default: blueprintKeys[0],
			},
			&blueprintTemplate,
			survey.Required,
			surveyOpts...,
		)
	}

	return blueprintTemplate, nil
}

func (blueprintContext *BlueprintContext) fetchFileContents(filePath string, blueprintLocalMode bool, addSuffix bool) (*[]byte, error) {
	if addSuffix {
		filePath = util.AddSuffixIfNeeded(filePath, templateExtension)
	}

	// local/remote
	if !blueprintLocalMode {
		return blueprintContext.fetchRemoteFile(filePath)
	} else if util.PathExists(filePath, false) {
		// fetch templates from local path
		content, err := ioutil.ReadFile(filePath)
		if err != nil {
			return nil, err
		}
		return &content, nil
	}
	return nil, fmt.Errorf("template not found in path %s", filePath)
}

func (blueprintContext *BlueprintContext) fetchRemoteFile(filePath string) (*[]byte, error) {
	return blueprintRepository.GetFileContents(filePath)
}

func (blueprintContext *BlueprintContext) fetchLocalFile(filePath string) (*[]byte, error) {
	variableConfigs, err := createTemplateConfigForSingleFile(filePath)
	if err != nil {
		return nil, err
	}
	return blueprintContext.fetchFileContents(variableConfigs[0].FullPath, true, false)
}

func (blueprintContext *BlueprintContext) parseDefinitionFile(blueprintLocalMode bool, blueprints map[string]*models.BlueprintRemote, templatePath string) (*BlueprintYaml, error) {
	// local/remote
	if blueprintLocalMode {
		return blueprintContext.parseLocalDefinitionFile(templatePath)
	} else {
		return blueprintContext.parseRemoteDefinitionFile(blueprints, templatePath)
	}
}

func (blueprintContext *BlueprintContext) parseRemoteDefinitionFile(blueprints map[string]*models.BlueprintRemote, templatePath string) (*BlueprintYaml, error) {
	// Check if user provided/selected template path is in available blueprints map
	if _, ok := blueprints[templatePath]; !ok {
		return nil, fmt.Errorf("blueprint [%s] not found in repository %s", templatePath, blueprintContext.Name)
	}

	// Get blueprint definition file contents
	ymlContent, err := blueprintContext.fetchRemoteFile(blueprints[templatePath].DefinitionFile.Path)
	if err != nil {
		return nil, err
	}

	// Parse blueprint document contents
	blueprintDoc, err := parseTemplateMetadata(ymlContent, templatePath, blueprintContext, false)
	if err != nil {
		return nil, err
	}

	// Prepare full repository paths
	for i, config := range blueprintDoc.TemplateConfigs {
		config.FullPath = path.Join(templatePath, config.File)
		blueprintDoc.TemplateConfigs[i] = config
	}
	return blueprintDoc, err
}

func (blueprintContext *BlueprintContext) parseLocalDefinitionFile(templatePath string) (*BlueprintYaml, error) {
	// Parse blueprint document contents
	var ymlContent *[]byte
	var err error
	for _, ext := range repository.BlueprintMetadataFileExtensions {
		definitionFileName := repository.BlueprintMetadataFileName + ext
		filePath := filepath.Join(templatePath, definitionFileName)
		ymlContent, err = blueprintContext.fetchLocalFile(filePath)
		if err == nil {
			break
		}
	}
	if err != nil {
		return nil, err
	}
	blueprintDoc, err := parseTemplateMetadata(ymlContent, templatePath, blueprintContext, true)
	if err != nil {
		return nil, err
	}

	// Prepare full paths for the template files
	err = blueprintDoc.verifyTemplateDirAndGenFullPaths(templatePath)
	if err != nil {
		return nil, err
	}

	return blueprintDoc, err
}

/*
 * -----------------
 * Utility Functions
 * -----------------
 */
func getFilePathRelativeToTemplatePath(filePath string, templatePath string) string {
	util.Verbose("[repository] getting FilePath: %s relative to templatePath: %s \n", filePath, templatePath)
	chunks := strings.Split(filePath, util.AddSuffixIfNeeded(templatePath, string(os.PathSeparator)))
	if len(chunks) > 1 {
		return chunks[len(chunks)-1]
	}
	return filePath
}

func createTemplateConfigForSingleFile(blueprintTemplate string) ([]TemplateConfig, error) {
	if blueprintTemplate != "" {
		// could be a single remote or local file
		var templateConfigs []TemplateConfig
		_, fileName := filepath.Split(blueprintTemplate)
		templateConfigs = append(templateConfigs, TemplateConfig{
			File:     fileName,
			FullPath: blueprintTemplate,
		})
		return templateConfigs, nil
	}
	return nil, fmt.Errorf("unknown template specified for Blueprint : %s", blueprintTemplate)
}

package blueprint

import (
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strings"

	"github.com/spf13/pflag"
	"github.com/spf13/viper"

	"gopkg.in/AlecAivazis/survey.v1"

	"github.com/xebialabs/xl-cli/pkg/blueprint/repository"
	"github.com/xebialabs/xl-cli/pkg/blueprint/repository/github"
	"github.com/xebialabs/xl-cli/pkg/blueprint/repository/http"
	"github.com/xebialabs/xl-cli/pkg/blueprint/repository/mock"
	"github.com/xebialabs/xl-cli/pkg/models"
	"github.com/xebialabs/xl-cli/pkg/util"
)

const (
	// ContextPrefix - this is the key used in config
	ContextPrefix     = "blueprint-repository"
	templateExtension = ".tmpl"

	FlagBlueprintCurrentRepository     = ContextPrefix + "-current-repository"
	ViperKeyBlueprintCurrentRepository = ContextPrefix + ".current-repository"
)

// TemplateConfig holds the merged template file definitions with repository info
type TemplateConfig struct {
	File           string
	FullPath       string
	DependsOnTrue  VarField
	DependsOnFalse VarField
}

// BlueprintContext holds necessary remote/local repository information for connection
type BlueprintContext struct {
	ActiveRepo   *repository.BlueprintRepository
	DefinedRepos []*repository.BlueprintRepository
}

func SetRootFlags(rootFlags *pflag.FlagSet) {
	rootFlags.String(FlagBlueprintCurrentRepository, "", "Current active blueprint repository name")

	viper.BindPFlag(ViperKeyBlueprintCurrentRepository, rootFlags.Lookup(FlagBlueprintCurrentRepository))
}

func ConstructBlueprintContext(v *viper.Viper) (*BlueprintContext, error) {
	activeRepoName := strings.ToLower(v.GetString(fmt.Sprintf("%s.current-repository", ContextPrefix)))
	if activeRepoName == "" {
		return nil, fmt.Errorf("current active repository name cannot be empty")
	}

	var currentRepo *repository.BlueprintRepository
	var definedRepos []*repository.BlueprintRepository

	repoDefinitions := make([]map[string]string, 1, 1)
	err := v.UnmarshalKey(fmt.Sprintf("%s.repositories", ContextPrefix), &repoDefinitions)
	if err != nil {
		return nil, fmt.Errorf("bad format in blueprint context: blueprint repositories should be a non-empty YAML list")
	}

	for i, repoDefinition := range repoDefinitions {
		// Validate mandatory fields for all repository types
		if !util.MapContainsKeyWithVal(repoDefinition, "type") || !util.MapContainsKeyWithVal(repoDefinition, "name") {
			return nil, fmt.Errorf("repository with index %d doesn't have all mandatory fields set [type, name]", i)
		}

		// Get repository type
		var repo repository.BlueprintRepository
		repoProvider, err := models.GetRepoProvider(repoDefinition["type"])
		if err != nil {
			return nil, err
		}

		// Parse according to type string
		switch repoProvider {
		case models.ProviderMock: // only used for testing purposes
			repo, err = mock.NewMockBlueprintRepository(repoDefinition)
		case models.ProviderGitHub:
			repo, err = github.NewGitHubBlueprintRepository(repoDefinition)
		case models.ProviderHttp:
			repo, err = http.NewHttpBlueprintRepository(repoDefinition)
		default:
			return nil, fmt.Errorf("no blueprint provider implementation found for %s", repoProvider)
		}
		if err != nil {
			return nil, err
		}
		definedRepos = append(definedRepos, &repo)

		// Set current repo if name is matching
		if strings.ToLower(repo.GetName()) == activeRepoName {
			currentRepo = &repo
		}
	}

	// Check if active repo is set correctly
	if currentRepo == nil {
		return nil, fmt.Errorf("current repository name '%s' is not matching with any of the defined repositories", activeRepoName)
	}

	return &BlueprintContext{
		ActiveRepo:   currentRepo,
		DefinedRepos: definedRepos,
	}, nil
}

func (blueprintContext *BlueprintContext) initCurrentRepoClient() (map[string]*models.BlueprintRemote, error) {
	err := (*blueprintContext.ActiveRepo).Initialize()
	if err != nil {
		return nil, err
	}
	return blueprintContext.parseRepositoryTree()
}

func (blueprintContext *BlueprintContext) parseRepositoryTree() (map[string]*models.BlueprintRemote, error) {
	var blueprints map[string]*models.BlueprintRemote
	var blueprintDirs []string
	var err error

	// Parse file tree from provider
	blueprints, blueprintDirs, err = (*blueprintContext.ActiveRepo).ListBlueprintsFromRepo()
	if err != nil {
		return nil, err
	}

	// Clear non-blueprint items in result map
	for blueprintPath := range blueprints {
		if !util.IsStringInSlice(blueprintPath, blueprintDirs) {
			delete(blueprints, blueprintPath)
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
			return "", fmt.Errorf(
				"no blueprints found in repository [%s - %s]",
				(*blueprintContext.ActiveRepo).GetName(),
				(*blueprintContext.ActiveRepo).GetProvider(),
			)
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
	return (*blueprintContext.ActiveRepo).GetFileContents(filePath)
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
	}
	return blueprintContext.parseRemoteDefinitionFile(blueprints, templatePath)
}

func (blueprintContext *BlueprintContext) parseRemoteDefinitionFile(blueprints map[string]*models.BlueprintRemote, templatePath string) (*BlueprintYaml, error) {
	// Check if user provided/selected template path is in available blueprints map
	if _, ok := blueprints[templatePath]; !ok {
		return nil, fmt.Errorf("blueprint [%s] not found in repository %s", templatePath, (*blueprintContext.ActiveRepo).GetName())
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

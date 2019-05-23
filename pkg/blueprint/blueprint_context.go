package blueprint

import (
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strings"

	"github.com/xebialabs/yaml"

	"github.com/spf13/pflag"
	"github.com/spf13/viper"

	"gopkg.in/AlecAivazis/survey.v1"

	"github.com/xebialabs/xl-cli/pkg/blueprint/repository"
	"github.com/xebialabs/xl-cli/pkg/blueprint/repository/github"
	"github.com/xebialabs/xl-cli/pkg/blueprint/repository/http"
	"github.com/xebialabs/xl-cli/pkg/blueprint/repository/local"
	"github.com/xebialabs/xl-cli/pkg/blueprint/repository/mock"
	"github.com/xebialabs/xl-cli/pkg/models"
	"github.com/xebialabs/xl-cli/pkg/util"
)

const (
	// ContextPrefix - this is the key used in config
	ContextPrefix       = "blueprint"
	RepositoryConfigKey = ContextPrefix + ".repositories"
	templateExtension   = ".tmpl"

	FlagBlueprintCurrentRepository     = ContextPrefix + "-current-repository"
	ViperKeyBlueprintCurrentRepository = ContextPrefix + ".current-repository"
)

// BlueprintContext holds necessary remote/local repository information for connection
type BlueprintContext struct {
	ActiveRepo   *repository.BlueprintRepository
	DefinedRepos []*repository.BlueprintRepository
}

// using custom ConfMap to have list of configuration items
type ConfMap map[string]string
type ConfData struct {
	CurrentRepo  string    `yaml:"current-repository"`
	Repositories []ConfMap `yaml:"repositories"`
}

var defaultBlueprintRepo = ConfMap{
	"name": models.DefaultBlueprintRepositoryName,
	"type": models.DefaultBlueprintRepositoryProvider,
	"url":  models.DefaultBlueprintRepositoryUrl,
}

func GetDefaultBlueprintConfData() ConfData {
	return ConfData{models.DefaultBlueprintRepositoryName, []ConfMap{defaultBlueprintRepo}}
}

func GetDefaultBlueprintViperConfig(v *viper.Viper) *viper.Viper {
	v.Set(ViperKeyBlueprintCurrentRepository, models.DefaultBlueprintRepositoryName)

	repositories := make([]ConfMap, 0)
	err := v.UnmarshalKey(RepositoryConfigKey, &repositories)
	if err != nil || repositories == nil || len(repositories) == 0 {
		repositories = []ConfMap{defaultBlueprintRepo}
	} else {
		if !doesDefaultExist(repositories) {
			repositories = append(repositories, defaultBlueprintRepo)
		}
	}
	v.Set(RepositoryConfigKey, repositories)
	return v
}

func CreateOrUpdateBlueprintConfig(v *viper.Viper, configPath string) (*viper.Viper, error) {
	v = GetDefaultBlueprintViperConfig(v)

	c := util.SortMapStringInterface(v.AllSettings())
	yamlBytes, err := yaml.Marshal(c)
	if err != nil {
		return v, err
	}
	err = ioutil.WriteFile(configPath, yamlBytes, 0640)
	if err != nil {
		return v, err
	}
	return v, nil
}

func SetRootFlags(rootFlags *pflag.FlagSet) {
	rootFlags.String(FlagBlueprintCurrentRepository, "", "Current active blueprint repository name")

	viper.BindPFlag(ViperKeyBlueprintCurrentRepository, rootFlags.Lookup(FlagBlueprintCurrentRepository))
}

func ConstructBlueprintContext(v *viper.Viper, configPath string) (*BlueprintContext, error) {
	activeRepoName := v.GetString(ViperKeyBlueprintCurrentRepository)
	if activeRepoName == "" {
		util.Verbose("Updating CLI config %s with blueprint configuration\n", configPath)
		var err error
		v, err = CreateOrUpdateBlueprintConfig(v, configPath)
		if err != nil {
			util.Info("Failed to update default configuration for blueprint. Please update ~/.xebialabs/config.yaml manually.")
		}
		activeRepoName = models.DefaultBlueprintRepositoryName
	}

	var currentRepo *repository.BlueprintRepository
	var definedRepos []*repository.BlueprintRepository

	repoDefinitions := make([]ConfMap, 1, 1)
	err := v.UnmarshalKey(RepositoryConfigKey, &repoDefinitions)
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
		case models.ProviderLocal:
			repo, err = local.NewLocalBlueprintRepository(repoDefinition)
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
		if strings.ToLower(repo.GetName()) == strings.ToLower(activeRepoName) {
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

func (blueprintContext *BlueprintContext) parseDefinitionFile(blueprintLocalMode bool, blueprint *models.BlueprintRemote, templatePath string) (*BlueprintConfig, error) {
	// local/remote
	if blueprintLocalMode {
		return blueprintContext.parseLocalDefinitionFile(templatePath)
	}
	return blueprintContext.parseRemoteDefinitionFile(blueprint, templatePath)
}

func (blueprintContext *BlueprintContext) parseRemoteDefinitionFile(blueprint *models.BlueprintRemote, templatePath string) (*BlueprintConfig, error) {
	// Since we pass a reference from a map here, it could be nil
	if blueprint == nil {
		return nil, fmt.Errorf("blueprint [%s] not found in repository %s", templatePath, (*blueprintContext.ActiveRepo).GetName())
	}

	// Get blueprint definition file contents
	ymlContent, err := blueprintContext.fetchRemoteFile(blueprint.DefinitionFile.Path)
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
		config.FullPath = path.Join(templatePath, config.Path)
		blueprintDoc.TemplateConfigs[i] = config
	}
	return blueprintDoc, err
}

func (blueprintContext *BlueprintContext) parseLocalDefinitionFile(templatePath string) (*BlueprintConfig, error) {
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
	err = blueprintDoc.verifyTemplateDirAndPaths(templatePath)
	if err != nil {
		return nil, err
	}

	return blueprintDoc, err
}

func parseTemplateMetadata(blueprintVars *[]byte, templatePath string, blueprintRepository *BlueprintContext, isLocal bool) (*BlueprintConfig, error) {
	return parseTemplateMetadataV2(blueprintVars, templatePath, blueprintRepository, isLocal)
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
			Path:     fileName,
			FullPath: blueprintTemplate,
		})
		return templateConfigs, nil
	}
	return nil, fmt.Errorf("unknown template specified for Blueprint : %s", blueprintTemplate)
}

func doesDefaultExist(repositories []ConfMap) bool {
	for _, repo := range repositories {
		if repo["name"] == models.DefaultBlueprintRepositoryName {
			if repo["type"] == "" {
				repo["type"] = models.DefaultBlueprintRepositoryProvider
			}
			if repo["url"] == "" {
				repo["url"] = models.DefaultBlueprintRepositoryUrl
			}
			return true
		}
	}
	return false
}

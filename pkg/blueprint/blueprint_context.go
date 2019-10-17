package blueprint

import (
	"bytes"
	"fmt"
    "github.com/xebialabs/xl-cli/pkg/blueprint/repository/gitlab"
	"io/ioutil"
	"path"
	"sort"
	"strings"

	"github.com/xebialabs/yaml"

	"github.com/spf13/pflag"
	"github.com/spf13/viper"

	"gopkg.in/AlecAivazis/survey.v1"

	"github.com/xebialabs/xl-cli/pkg/blueprint/repository"
	"github.com/xebialabs/xl-cli/pkg/blueprint/repository/bitbucketserver"
	"github.com/xebialabs/xl-cli/pkg/blueprint/repository/github"
	"github.com/xebialabs/xl-cli/pkg/blueprint/repository/bitbucket"
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

func GetDefaultBlueprintViperConfig(v *viper.Viper, configPath string) (*viper.Viper, *viper.Viper, string, error) {
	activeRepoName := v.GetString(ViperKeyBlueprintCurrentRepository)
	if activeRepoName == "" {
		activeRepoName = models.DefaultBlueprintRepositoryName
		v.Set(ViperKeyBlueprintCurrentRepository, activeRepoName)
	}

	repositories := GetRepositoriesWithDefault(v)
	v.Set(RepositoryConfigKey, repositories)

	// existing config file on disk
	var vFromConfig *viper.Viper
	if WriteConfigFile && util.PathExists(configPath, false) {
		vFromConfig = viper.New()
		vFromConfig.SetConfigType("yaml")
		bytesread, err := ioutil.ReadFile(configPath)
		if err != nil {
			return v, nil, "", err
		}
		err = vFromConfig.ReadConfig(bytes.NewBuffer(bytesread))
		if err != nil {
			return v, nil, "", err
		}
		if vFromConfig != nil {
			if vFromConfig.GetString(ViperKeyBlueprintCurrentRepository) == "" {
				vFromConfig.Set(ViperKeyBlueprintCurrentRepository, models.DefaultBlueprintRepositoryName)
			}
			repositoriesFromConfig := GetRepositoriesWithDefault(vFromConfig)
			vFromConfig.Set(RepositoryConfigKey, repositoriesFromConfig)
		}
	}
	return v, vFromConfig, activeRepoName, nil
}

func GetRepositoriesWithDefault(v *viper.Viper) []ConfMap {
	repositories := make([]ConfMap, 0)
	err := v.UnmarshalKey(RepositoryConfigKey, &repositories)
	if err != nil || repositories == nil || len(repositories) == 0 {
		repositories = []ConfMap{defaultBlueprintRepo}
	} else {
		if !doesDefaultExist(repositories) {
			repositories = append(repositories, defaultBlueprintRepo)
		}
	}
	return repositories
}

// WriteConfigFile is used to suppress writing real config files during test
var WriteConfigFile = true

func CreateOrUpdateBlueprintConfig(v *viper.Viper, configPath string) (*viper.Viper, string, error) {
	v, vFromConfig, activeRepoName, err := GetDefaultBlueprintViperConfig(v, configPath)
	if err != nil {
		return v, activeRepoName, err
	}
	if WriteConfigFile && vFromConfig != nil {
		// write to existing config file
		c := util.SortMapStringInterface(vFromConfig.AllSettings())
		yamlBytes, err := yaml.Marshal(c)
		if err != nil {
			return v, activeRepoName, err
		}
		err = ioutil.WriteFile(configPath, yamlBytes, 0640)
		if err != nil {
			return v, activeRepoName, err
		}
	}
	return v, activeRepoName, nil
}

func SetRootFlags(rootFlags *pflag.FlagSet) {
	rootFlags.String(FlagBlueprintCurrentRepository, "", "Current active blueprint repository name")

	viper.BindPFlag(ViperKeyBlueprintCurrentRepository, rootFlags.Lookup(FlagBlueprintCurrentRepository))
}

func ConstructLocalBlueprintContext(localRepoPath string) (*BlueprintContext, error) {
	var definedRepos []*repository.BlueprintRepository
	var localRepo repository.BlueprintRepository
	var err error

	if !util.PathExists(localRepoPath, true) {
		return nil, fmt.Errorf("Error: Provided development local repository directory [%s] is not valid\n", localRepoPath)
	}

	localRepo, err = local.NewLocalBlueprintRepository(map[string]string{
		"type": models.ProviderLocal,
		"name": "cmd-arg",
		"path": localRepoPath,
	})
	if err != nil {
		return nil, err
	}
	definedRepos = append(definedRepos, &localRepo)

	return &BlueprintContext{
		ActiveRepo:   &localRepo,
		DefinedRepos: definedRepos,
	}, nil
}

func ConstructBlueprintContext(v *viper.Viper, configPath, CLIVersion string) (*BlueprintContext, error) {
	util.Verbose("Updating CLI config %s with blueprint configuration\n", configPath)
	v, activeRepoName, err := CreateOrUpdateBlueprintConfig(v, configPath)
	if err != nil {
		util.Info("Failed to update default configuration for blueprint. Please update ~/.xebialabs/config.yaml manually.")
	}

	var currentRepo *repository.BlueprintRepository
	var definedRepos []*repository.BlueprintRepository

	repoDefinitions := make([]ConfMap, 1, 1)
	err = v.UnmarshalKey(RepositoryConfigKey, &repoDefinitions)
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
		util.Verbose("Creating blueprint repo configuration %v\n", repoDefinition)
		// Parse according to type string
		switch repoProvider {
		case models.ProviderMock: // only used for testing purposes
			repo, err = mock.NewMockBlueprintRepository(repoDefinition)
		case models.ProviderLocal:
			repo, err = local.NewLocalBlueprintRepository(repoDefinition)
		case models.ProviderGitHub:
			repo, err = github.NewGitHubBlueprintRepository(repoDefinition)
		case models.ProviderBitbucket:
			repo, err = bitbucket.NewBitbucketBlueprintRepository(repoDefinition)
		case models.ProviderBitbucketServer:
			repo, err = bitbucketserver.NewBitbucketServerBlueprintRepository(repoDefinition)
		case models.ProviderHttp:
			repo, err = http.NewHttpBlueprintRepository(repoDefinition, CLIVersion)
        case models.ProviderGitLab:
            repo, err = gitlab.NewGitLabBlueprintRepository(repoDefinition)
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

func (blueprintContext *BlueprintContext) fetchFileContents(filePath string, addSuffix bool) (*[]byte, error) {
	if addSuffix {
		filePath = util.AddSuffixIfNeeded(filePath, templateExtension)
	}
	return (*blueprintContext.ActiveRepo).GetFileContents(filePath)
}

func (blueprintContext *BlueprintContext) parseDefinitionFile(blueprint *models.BlueprintRemote, templatePath string) (*BlueprintConfig, error) {
	// Since we pass a reference from a map here, it could be nil
	if blueprint == nil {
		return nil, fmt.Errorf("blueprint [%s] not found in repository %s", templatePath, (*blueprintContext.ActiveRepo).GetName())
	}

	// Get blueprint definition file contents
	ymlContent, err := blueprintContext.fetchFileContents(blueprint.DefinitionFile.Path, false)
	if err != nil {
		return nil, err
	}

	// Parse blueprint document contents
	blueprintDoc, err := parseTemplateMetadata(ymlContent, templatePath, blueprintContext)
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

func parseTemplateMetadata(ymlContent *[]byte, templatePath string, blueprintRepository *BlueprintContext) (*BlueprintConfig, error) {
	decoder := yaml.NewDecoder(bytes.NewReader(*ymlContent))
	yamlDoc := struct {
		ApiVersion string `yaml:"apiVersion"`
	}{}
	err := decoder.Decode(&yamlDoc)
	if err != nil {
		return nil, err
	}
	if yamlDoc.ApiVersion == models.BlueprintYamlFormatV1 {
		return parseTemplateMetadataV1(ymlContent, templatePath, blueprintRepository)
	}
	return parseTemplateMetadataV2(ymlContent, templatePath, blueprintRepository)
}

/*
 * -----------------
 * Utility Functions
 * -----------------
 */

func doesDefaultExist(repositories []ConfMap) bool {
	for _, repo := range repositories {
		if repo["name"] == models.DefaultBlueprintRepositoryName {
			if repo["type"] == "" {
				repo["type"] = models.DefaultBlueprintRepositoryProvider
			}
			if repo["url"] != models.DefaultBlueprintRepositoryUrl {
				repo["url"] = models.DefaultBlueprintRepositoryUrl
			}
			return true
		}
	}
	return false
}

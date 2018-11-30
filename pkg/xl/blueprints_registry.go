package xl

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"gopkg.in/AlecAivazis/survey.v1"
)

// TemplateConfig holds the merged template file definitions with repository info
type TemplateConfig struct {
	File           string
	FullPath       string
	DependsOnTrue  VarField
	DependsOnFalse VarField
	Repository     BlueprintRepository
}

var blueprintMetadataFileExtensions = []string{".yaml", ".yml"}

const blueprintMetadataFileName = "blueprint"
const repositoryIndexFile = "index.json"
const templateExtension = ".tmpl"

// MakeHTTPCallForBlueprintRepositoryFn is the type definition for makeHTTPCallForBlueprintRepository method
type MakeHTTPCallForBlueprintRepositoryFn func(url string, blueprintRepository BlueprintRepository) ([]byte, int, error)

// makeHTTPCallForBlueprintRepository does unauthenticated get requests for blueprint files
func makeHTTPCallForBlueprintRepository(url string, blueprintRepository BlueprintRepository) ([]byte, int, error) {
	// TODO: Try to use BlueprintRepository request methods instead
	request, err := http.NewRequest("GET", url, nil)
	if blueprintRepository.Server.Username != "" && blueprintRepository.Server.Password != "" {
		request.SetBasicAuth(blueprintRepository.Server.Username, blueprintRepository.Server.Password)
	}
	if err != nil {
		return nil, 0, err
	}
	httpClient := &http.Client{}
	response, err := httpClient.Do(request)
	if err != nil {
		return nil, 0, err
	}
	if response.StatusCode >= 400 {
		return nil, response.StatusCode, nil
	}
	bodyText, err := ioutil.ReadAll(response.Body)
	if err != nil {
		return nil, 0, err
	}
	return bodyText, response.StatusCode, nil
}

func parseRepositoryIndexFile(blueprintRepository BlueprintRepository, makeHTTPCall MakeHTTPCallForBlueprintRepositoryFn) ([]string, error) {
	indexURL := addSuffixIfNeeded(blueprintRepository.Server.Url.String(), "/") + repositoryIndexFile
	bodyText, statusCode, err := makeHTTPCall(indexURL, blueprintRepository)
	if err != nil {
		return nil, err
	}

	err = translateHTTPStatusCodeErrors(statusCode, blueprintRepository.Server.Url.String())
	if err != nil {
		return nil, err
	}

	var items []string
	err = json.Unmarshal(bodyText, &items)
	if err != nil {
		return nil, err
	}
	return items, nil
}

// fetchBlueprintFromPath will fetch the blueprint files form the following sources
// 1. A HTTP URL pointing to single blueprint file
// 2. A local blueprint file
func (config *TemplateConfig) fetchBlueprintFromPath(addSuffix bool, makeHTTPCall MakeHTTPCallForBlueprintRepositoryFn) ([]byte, error) {
	filePath := config.FullPath
	if addSuffix {
		filePath = addSuffixIfNeeded(config.FullPath, templateExtension)
	}
	// determine protocol
	if strings.HasPrefix(filePath, "http") {
		// fetch blueprints from http path
		bodyText, statusCode, err := makeHTTPCall(filePath, config.Repository)
		if err != nil {
			return nil, err
		}

		err = translateHTTPStatusCodeErrors(statusCode, filePath)
		if err != nil {
			return nil, err
		}
		Verbose("[repository] Read file %s\n", filePath)
		return bodyText, nil
	} else if PathExists(filePath, false) {
		// fetch templates from local path
		content, err := ioutil.ReadFile(filePath)
		if err != nil {
			return nil, err
		}
		return content, nil
	}
	return nil, fmt.Errorf("template not found in path %s", filePath)
}

func (config *TemplateConfig) generateFullURLPath(templatePath string, blueprintRepository BlueprintRepository) {
	repositoryURL := ""
	if !PathExists(templatePath, true) {
		repositoryURL = blueprintRepository.Server.Url.String()
		if repositoryURL != "" {
			repositoryURL = addSuffixIfNeeded(repositoryURL, "/")
		}
		config.Repository = blueprintRepository
	}
	config.FullPath = fmt.Sprintf("%s%s/%s", repositoryURL, templatePath, config.File)
}

// --utility functions
func getFilePathRelativeToTemplatePath(filePath string, templatePath string) string {
	Verbose("[repository] getting FilePath: %s relative to templatePath: %s \n", filePath, templatePath)
	chunks := strings.Split(filePath, addSuffixIfNeeded(templatePath, string(os.PathSeparator)))
	if len(chunks) > 1 {
		return chunks[len(chunks)-1]
	}
	return filePath
}

func getFromRelativeFolder(templatePath string) ([]TemplateConfig, error) {
	if PathExists(templatePath, true) {
		Verbose("[repository] Relative path found: %s \n", templatePath)
		var templates []TemplateConfig
		var filePaths []string

		// Walk the root directory
		err := filepath.Walk(templatePath, func(path string, fileInfo os.FileInfo, err error) error {
			if err != nil {
				return err
			}
			if !fileInfo.IsDir() {
				filePaths = append(filePaths, path)
			}
			return nil
		})
		if err != nil {
			return nil, err
		}

		if len(filePaths) == 0 {
			return nil, fmt.Errorf("path [%s] doesn't include any valid files", templatePath)
		}
		sort.Strings(filePaths)
		for _, filePath := range filePaths {
			file := getFilePathRelativeToTemplatePath(filePath, templatePath)
			if !strings.Contains(file, blueprintMetadataFileName+".yaml") && !strings.Contains(file, blueprintMetadataFileName+".yml") {
				templates = append(templates, TemplateConfig{
					File:     file,
					FullPath: filePath,
				})
			}
		}
		return templates, nil
	}
	return nil, fmt.Errorf("path [%s] doesn't exist", templatePath)
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

func getBlueprintVariableConfig(templatePath string, blueprintRepository BlueprintRepository, blueprintFileName string, makeHTTPCallFn MakeHTTPCallForBlueprintRepositoryFn) (*[]byte, error) {
	// read blueprint variables file
	filePath := fmt.Sprintf("%s/%s", templatePath, blueprintFileName)
	if !PathExists(filePath, false) && blueprintRepository.Server.Url.String() != "" {
		filePath = fmt.Sprintf("%s%s", addSuffixIfNeeded(blueprintRepository.Server.Url.String(), "/"), filePath)
	}
	variableConfigs, err := createTemplateConfigForSingleFile(filePath)
	variableConfig := variableConfigs[0]
	if !PathExists(filePath, false) {
		variableConfig.Repository = blueprintRepository
	}

	blueprintVars, err := variableConfig.fetchBlueprintFromPath(false, makeHTTPCallFn)
	if err != nil {
		return nil, err
	}
	return &blueprintVars, nil
}

func GetBlueprintConfig(templatePath string, blueprintRepository BlueprintRepository, makeHTTPCall ...MakeHTTPCallForBlueprintRepositoryFn) (*BlueprintYaml, error) {
	makeHTTPCallFn := makeHTTPCallForBlueprintRepository
	if makeHTTPCall != nil && makeHTTPCall[0] != nil {
		// this is in order to make this testable with mocks
		makeHTTPCallFn = makeHTTPCall[0]
	}
	// read blueprint metadata file - try both .yml and .yaml extensions
	var ymlContent *[]byte
	var blueprintDoc *BlueprintYaml
	var blueprintReadErr error
	for _, extension := range blueprintMetadataFileExtensions {
		ymlContent, blueprintReadErr = getBlueprintVariableConfig(templatePath, blueprintRepository, blueprintMetadataFileName+extension, makeHTTPCallFn)
		if blueprintReadErr == nil {
			break
		}
	}
	if blueprintReadErr != nil {
		return nil, blueprintReadErr
	}
	blueprintDoc, err := parseTemplateMetadata(ymlContent, templatePath, blueprintRepository)
	if err != nil {
		return nil, err
	}

	if blueprintDoc.TemplateConfigs == nil {
		Verbose("[repository] Remote config not found. Fetching from relative path: %s \n", templatePath)
		templateConfigs, err := getFromRelativeFolder(templatePath)
		if err != nil {
			return nil, err
		}
		blueprintDoc.TemplateConfigs = templateConfigs
	}

	return blueprintDoc, err
}

func GetBlueprintTemplateFromUser(blueprintTemplate string, blueprintRepository BlueprintRepository, surveyOpts ...survey.AskOpt) (string, error) {
	if blueprintTemplate == "" {
		blueprints, err := parseRepositoryIndexFile(blueprintRepository, makeHTTPCallForBlueprintRepository)
		if err != nil {
			return "", err
		}
		_ = survey.AskOne(
			&survey.Select{
				Message: "Choose a blueprint:",
				Options: blueprints,
				Default: blueprints[0],
			},
			&blueprintTemplate,
			survey.Required,
			surveyOpts...,
		)
	}
	return blueprintTemplate, nil
}

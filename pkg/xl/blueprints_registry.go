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

// TemplateConfig holds the merged template file definitions with registry info
type TemplateConfig struct {
	File     string
	FullPath string
	Registry TemplateRegistry
}

const blueprintMetadataFileName = "blueprint"

var blueprintMetadataFileExtensions = []string{".yaml", ".yml"}

const registryIndexFile = "index.json"
const templateExtn = ".tmpl"

func getTemplateTypes(mergedRegistryIndex map[string]TemplateRegistry) []string {
	var keys []string
	for k := range mergedRegistryIndex {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

// MakeHTTPCallForTemplateFn is the type definition for MakeHTTPCallForTemplate method
type MakeHTTPCallForTemplateFn func(indexURL string, registry TemplateRegistry) ([]byte, int, error)

// makeHTTPCallForTemplate does unauthenticated get requests for template files
func makeHTTPCallForTemplate(URL string, registry TemplateRegistry) ([]byte, int, error) {
	request, err := http.NewRequest("GET", URL, nil)
	if registry.Username != "" && registry.Password != "" {
		request.SetBasicAuth(registry.Username, registry.Password)
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

// fetchTemplateFromPath will fetch the template files form the following sources
// 1. A HTTP URL pointing to single template file
// 2. A local template file
func fetchTemplateFromPath(config TemplateConfig, addSuffix bool, makeHTTPCall MakeHTTPCallForTemplateFn) ([]byte, error) {
	filePath := config.FullPath
	if addSuffix {
		filePath = addSuffixIfNeeded(config.FullPath, templateExtn)
	}
	// determine protocol
	if strings.HasPrefix(filePath, "http") {
		// fetch templates from http path
		bodyText, statusCode, err := makeHTTPCall(filePath, config.Registry)
		if err != nil {
			return nil, err
		}

		err = translateHTTPStatusCodeErrors(statusCode, filePath)
		if err != nil {
			return nil, err
		}
		Verbose("[registry] Read file %s", filePath)
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

func getIndexJsonFromRegistry(urlVal string, registry TemplateRegistry, makeHTTPCall MakeHTTPCallForTemplateFn) ([]string, error) {
	indexURL := addSuffixIfNeeded(urlVal, "/") + registryIndexFile
	bodyText, statusCode, err := makeHTTPCall(indexURL, registry)
	if err != nil {
		return nil, err
	}

	err = translateHTTPStatusCodeErrors(statusCode, urlVal)
	if err != nil {
		return nil, err
	}

	var resp []string
	err = json.Unmarshal(bodyText, &resp)
	if err != nil {
		return nil, err
	}
	return resp, nil
}

// merge the index.json files from all configured registry
func mergeRegistryIndex(templateRegistries []TemplateRegistry, makeHTTPCall MakeHTTPCallForTemplateFn) (map[string]TemplateRegistry, error) {
	mergedIndex := make(map[string]TemplateRegistry)
	for _, registry := range templateRegistries {
		resp, err := getIndexJsonFromRegistry(registry.URL.String(), registry, makeHTTPCall)
		if err != nil {
			return nil, err
		}
		Verbose("[registry] Registry index for %s: %s\n", registry.URL.String(), resp)
		for _, key := range resp {
			mergedIndex[key] = registry
		}
	}

	return mergedIndex, nil
}

// makeFullURLPath prefixes each template file with full URL of registry and template path
func makeFullURLPath(index []string, templatePath string, registry TemplateRegistry) []TemplateConfig {
	var val []TemplateConfig
	for _, item := range index {
		config := TemplateConfig{
			File:     item,
			FullPath: fmt.Sprintf("%s%s/%s", addSuffixIfNeeded(registry.URL.String(), "/"), templatePath, item),
			Registry: registry,
		}
		val = append(val, config)
	}
	return val
}

func getTemplateConfigs(templatePath string, registry TemplateRegistry, makeHTTPCall MakeHTTPCallForTemplateFn) ([]TemplateConfig, error) {
	urlVal := addSuffixIfNeeded(registry.URL.String(), "/") + templatePath

	resp, err := getIndexJsonFromRegistry(urlVal, registry, makeHTTPCall)
	if err != nil {
		return nil, err
	}
	return makeFullURLPath(resp, templatePath, registry), nil
}

func getFilePathRelativeToTemplatePath(filePath, templatePath string) string {
	Verbose("[registry] getting FilePath: %s relative to templatePath: %s \n", filePath, templatePath)
	chunks := strings.Split(filePath, addSuffixIfNeeded(templatePath, string(os.PathSeparator)))
	if len(chunks) > 1 {
		return chunks[len(chunks)-1]
	}
	return filePath
}

func getFromRelativeFolder(templatePath string) ([]TemplateConfig, error) {
	if PathExists(templatePath, true) {
		Verbose("[registry] Relative path found: %s \n", templatePath)
		var templates []TemplateConfig
		var files []string

		// Walk the root directory
		err := filepath.Walk(templatePath, func(path string, fileInfo os.FileInfo, err error) error {
			if err != nil {
				return err
			}
			if !fileInfo.IsDir() {
				files = append(files, path)
			}
			return nil
		})
		if err != nil {
			return nil, err
		}

		if len(files) == 0 {
			return nil, nil
		}
		sort.Strings(files)
		for _, filePath := range files {
			templates = append(templates, TemplateConfig{
				File:     getFilePathRelativeToTemplatePath(filePath, templatePath),
				FullPath: filePath,
			})
		}
		return templates, nil
	}
	return nil, nil
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

// ask user which template to use if not provided
func getBlueprintTemplateFromUser(blueprintTemplate string, mergedRegistryIndex map[string]TemplateRegistry, surveyOpts ...survey.AskOpt) string {
	if blueprintTemplate == "" {
		options := getTemplateTypes(mergedRegistryIndex)
		survey.AskOne(
			&survey.Select{
				Message: "Choose a template:",
				Options: options,
				Default: options[0],
			},
			&blueprintTemplate,
			survey.Required,
			surveyOpts...,
		)
	}
	return blueprintTemplate
}

func getAvailableBlueprintTemplates(blueprintTemplate string, templateRegistries []TemplateRegistry, surveyOpts ...survey.AskOpt) ([]TemplateConfig, string, error) {
	mergedRegistryIndex, err := mergeRegistryIndex(templateRegistries, makeHTTPCallForTemplate)
	if err != nil {
		return nil, "", err
	}
	Verbose("[registry] Merged registry index contains %d entries\n", len(mergedRegistryIndex))
	// get template details
	templatePath := getBlueprintTemplateFromUser(blueprintTemplate, mergedRegistryIndex, surveyOpts...)
	Verbose("[registry] Template path: %s \n", templatePath)

	templateRegistry := mergedRegistryIndex[templatePath]
	templateConfigs, err := getTemplateConfigs(templatePath, templateRegistry, makeHTTPCallForTemplate)
	if templateConfigs == nil {
		Verbose("[registry] Remote config not found. Fetching from relative path: %s \n", templatePath)
		templateConfigs, err = getFromRelativeFolder(templatePath)
		if err != nil {
			return nil, "", err
		}
	}
	return templateConfigs, templatePath, err
}

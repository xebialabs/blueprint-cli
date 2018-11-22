package xl

import (
	"bytes"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"text/template"

	"github.com/magiconair/properties"

	"io/ioutil"

	"github.com/Masterminds/sprig"
	"github.com/xebialabs/yaml"
	"gopkg.in/AlecAivazis/survey.v1"
)

const (
	valuesFile         = "values.xlvals"
	valuesFileHeader   = "# This file includes all non-secret values, you can add variables here and then refer them with '!value' tag in YAML files"
	secretsFile        = "secrets.xlvals"
	secretsFileHeader  = "# This file includes all secret values, and will be excluded from GIT. You can add new values and/or edit them and then refer to them using '!value' YAML tag"
	gitignoreFile      = ".gitignore"
)

// parse blueprint definition doc
func parseTemplateMetadata(blueprintVars *[]byte) (*BlueprintYaml, error) {
	decoder := yaml.NewDecoder(bytes.NewReader(*blueprintVars))
	decoder.SetStrict(true)
	doc := BlueprintYaml{}
	decoder.Decode(&doc)

	// parse & validate
	err := doc.parseSpec()
	if err != nil {
		return nil, err
	}
	err = doc.validate()
	return &doc, err
}

// read file contents
func readFileContents(fileStream []byte) (string, error) {
	content, ioErr := ioutil.ReadAll(bytes.NewReader(fileStream))
	if ioErr != nil {
		return "", ioErr
	}
	return string(content), nil
}

func getBlueprintVariableConfig(templatePath string, registry TemplateRegistry, blueprintFileName string) (*[]byte, error) {
	// read blueprint variables file
	filePath := fmt.Sprintf("%s/%s", templatePath, blueprintFileName)
	if registry.URL.String() != "" {
		filePath = fmt.Sprintf("%s%s", addSuffixIfNeeded(registry.URL.String(), "/"), filePath)
	}
	variableConfigs, err := createTemplateConfigForSingleFile(filePath)
	variableConfig := variableConfigs[0]
	variableConfig.Registry = registry
	blueprintVars, err := fetchTemplateFromPath(variableConfig, false, makeHTTPCallForTemplate)
	if err != nil {
		return nil, err
	}
	return &blueprintVars, nil
}

func getFuncMaps() template.FuncMap {
	funcMaps := sprig.TxtFuncMap()
	funcMaps["kebabcase"] = toKebabCase
	return funcMaps
}

func adjustPathSeperatorIfNeeded(blueprintTemplate string) string {
	re := regexp.MustCompile(`[\/\\]`)
	return re.ReplaceAllString(blueprintTemplate, string(os.PathSeparator))
}

// CreateBlueprint is entry point for the cli command
func CreateBlueprint(blueprintTemplate string, templateRegistries []TemplateRegistry, outputDir string, surveyOpts ...survey.AskOpt) error {
	blueprintTemplate = adjustPathSeperatorIfNeeded(blueprintTemplate)
	// get available blueprint templates from merged registry
	templateConfigs, templatePath, err := getAvailableBlueprintTemplates(blueprintTemplate, templateRegistries, surveyOpts...)
	if err != nil {
		return err
	}

	if templateConfigs == nil {
		return fmt.Errorf("template configuration not found for path %s", templatePath)
	}
	Verbose("[cmd] Reading Blueprint from %s\n", templatePath)

	// read blueprint metadata file - try both .yml and .yaml extensions
	var ymlContent *[]byte
	var blueprintDoc *BlueprintYaml
	var blueprintReadErr error
	for _, extn := range blueprintMetadataFileExtensions {
		ymlContent, blueprintReadErr = getBlueprintVariableConfig(templatePath, templateConfigs[0].Registry, blueprintMetadataFileName+extn)
		if blueprintReadErr == nil {
			break
		}
	}
	if blueprintReadErr != nil {
		return blueprintReadErr
	}
	blueprintDoc, err = parseTemplateMetadata(ymlContent)
	if err != nil {
		return err
	}
	Verbose("[dataPrep] Got blueprint metadata: %#v\n", blueprintDoc.Metadata)

	// ask for user input
	preparedData, err := blueprintDoc.prepareTemplateData(surveyOpts...)
	if err != nil {
		return err
	}
	Verbose("[dataPrep] Prepared data: %#v\n", preparedData)

	// save prepared data to values & secrets files
	err = writeConfigToFile(valuesFileHeader, preparedData.Values, path.Join(outputDir, valuesFile))
	if err != nil {
		return err
	}
	err = writeConfigToFile(secretsFileHeader, preparedData.Secrets, path.Join(outputDir, secretsFile))
	if err != nil {
		return err
	}

	// generate .gitignore file
	gitignoreData := secretsFile
	writeDataToFile(path.Join(outputDir, gitignoreFile), &gitignoreData)

	// execute each template file found
	for _, config := range templateConfigs {
		// filter blueprint metadata file
		if strings.Contains(config.File, blueprintMetadataFileName) || config.File == "index.json" {
			continue
		}

		Verbose("[file] Fetching template file %s\n", config.FullPath)
		templateFile, err := fetchTemplateFromPath(config, strings.HasSuffix(config.File, templateExtn), makeHTTPCallForTemplate)
		if err != nil {
			return err
		}

		// read template contents
		templateString, err := readFileContents(templateFile)
		if err != nil {
			return err
		}

		// process the template file (filter based on extension)
		if strings.HasSuffix(config.File, templateExtn) {
			Verbose("[file] Processing template file %s\n", config.FullPath)

			// read & process the template
			tmpl := template.Must(template.New(config.File).Funcs(getFuncMaps()).Parse(templateString))
			processedTmpl := &strings.Builder{}
			err = tmpl.Execute(processedTmpl, preparedData.TemplateData)
			if err != nil {
				return err
			}

			// write the processed template to a file
			finalTmpl := strings.TrimSpace(processedTmpl.String())
			writeDataToFile(strings.Replace(config.File, templateExtn, "", 1), &finalTmpl)
		} else {
			// handle non-template files - copy as-it-is
			Verbose("[file] Copying file %s\n", config.FullPath)
			writeDataToFile(config.File, &templateString)
		}
	}
	return nil
}

func createDirectoryIfNeeded(fileName string) error {
	dir, _ := filepath.Split(fileName)
	if dir != "" && !PathExists(dir, true) {
		Verbose("[file] Creating sub-directory %s\n", dir)
		return os.MkdirAll(dir, os.ModePerm)
	}
	return nil
}

// --utility functions
func writeDataToFile(outputFileName string, data *string) error {
	err := createDirectoryIfNeeded(outputFileName)
	if err != nil {
		return err
	}

	Verbose("[file] Creating blueprint output file %s\n", outputFileName)
	file, err := os.Create(outputFileName)
	if err != nil {
		return err
	}
	out, err := file.WriteString(*data)
	if err != nil {
		return err
	}
	Verbose("\tWrote %d bytes \n", out)
	file.Sync()
	file.Close()
	Info("[file] Blueprint output file '%s' generated successfully\n", outputFileName)
	return nil
}

func writeConfigToFile(header string, config map[string]interface{}, filename string) error {
	err := createDirectoryIfNeeded(filename)
	if err != nil {
		return err
	}
	props := properties.NewProperties()

	// sort based on keys
	var keys []string
	for k := range config {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		err := props.SetValue(k, config[k])
		if err != nil {
			return err
		}
	}

	// write properties to file
	f, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer f.Close()
	bytesWrittenHeader, err := f.Write([]byte(header+"\n"))
	if err != nil {
		return err
	}
	bytesWrittenConfig, err := props.Write(f, properties.UTF8)
	if err != nil {
		return err
	}
	Verbose("\tWrote %d bytes \n", bytesWrittenHeader+bytesWrittenConfig)
	Info("[file] Blueprint output file '%s' generated successfully\n", filename)
	return nil
}

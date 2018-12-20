package xl

import (
	"os"
	"path"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"text/template"

	"github.com/magiconair/properties"

	"github.com/Masterminds/sprig"
	"gopkg.in/AlecAivazis/survey.v1"
)

// SkipFinalPrompt is used in tests to skip the confirmation prompt
var SkipFinalPrompt = false

const (
	valuesFile        = "values.xlvals"
	valuesFileHeader  = "# This file includes all non-secret values, you can add variables here and then refer them with '!value' tag in YAML files"
	secretsFile       = "secrets.xlvals"
	secretsFileHeader = "# This file includes all secret values, and will be excluded from GIT. You can add new values and/or edit them and then refer to them using '!value' YAML tag"
	gitignoreFile     = ".gitignore"
)

func getFuncMaps() template.FuncMap {
	funcMaps := sprig.TxtFuncMap()
	funcMaps["kebabcase"] = toKebabCase
	return funcMaps
}

func adjustPathSeperatorIfNeeded(blueprintTemplate string) string {
	re := regexp.MustCompile(`[\/\\]`)
	return re.ReplaceAllString(blueprintTemplate, string(os.PathSeparator))
}

func shouldSkipFile(templateConfig TemplateConfig, variables *[]Variable) (bool, error) {
	if !isStringEmpty(templateConfig.DependsOnTrue.Val) {
		dependsOnTrueVal, err := ParseDependsOnValue(templateConfig.DependsOnTrue, variables)
		if err != nil {
			return false, err
		}
		return !dependsOnTrueVal, nil
	}
	if !isStringEmpty(templateConfig.DependsOnFalse.Val) {
		dependsOnFalseVal, err := ParseDependsOnValue(templateConfig.DependsOnFalse, variables)
		if err != nil {
			return false, err
		}
		return dependsOnFalseVal, nil
	}
	return false, nil
}

// InstantiateBlueprint is entry point for the cli command
func InstantiateBlueprint(blueprintTemplate string, blueprintRepository BlueprintRepository, outputDir string, surveyOpts ...survey.AskOpt) error {
	blueprintTemplate = adjustPathSeperatorIfNeeded(blueprintTemplate)
	// get available blueprint templates from merged registry
	templatePath, err := GetBlueprintTemplateFromUser(blueprintTemplate, blueprintRepository, surveyOpts...)
	if err != nil {
		return err
	}

	Verbose("[cmd] Reading Blueprint from %s\n", templatePath)
	blueprintDoc, err := GetBlueprintConfig(templatePath, blueprintRepository)
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
	err = writeDataToFile(path.Join(outputDir, gitignoreFile), &gitignoreData)
	if err != nil {
		return err
	}

	// execute each template file found
	for _, config := range blueprintDoc.TemplateConfigs {
		skipFile, err := shouldSkipFile(config, &blueprintDoc.Variables)
		if err != nil {
			return err
		}
		if skipFile {
			Verbose("[file] skipping file [%s] since it has dependsOn value set\n", config.File)
			continue
		}

		// read template contents
		Verbose("[file] Fetching template file %s\n", config.FullPath)
		templateContent, err := config.fetchBlueprintFromPath(strings.HasSuffix(config.File, templateExtension), makeHTTPCallForBlueprintRepository)
		if err != nil {
			return err
		}
		templateString := string(templateContent)

		// process the template file (filter based on extension)
		if strings.HasSuffix(config.File, templateExtension) {
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
			err = writeDataToFile(strings.Replace(config.File, templateExtension, "", 1), &finalTmpl)
			if err != nil {
				return err
			}
		} else {
			// handle non-template files - copy as-it-is
			Verbose("[file] Copying file %s\n", config.FullPath)
			err = writeDataToFile(config.File, &templateString)
			if err != nil {
				return err
			}
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
	err = file.Sync()
	if err != nil {
		return err
	}
	err = file.Close()
	if err != nil {
		return err
	}
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
	bytesWrittenHeader, err := f.Write([]byte(header + "\n"))
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

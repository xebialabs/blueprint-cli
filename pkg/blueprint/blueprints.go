package blueprint

import (
	"os"
	"path"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"github.com/thoas/go-funk"

	"text/template"

	"github.com/magiconair/properties"
	"github.com/xebialabs/xl-cli/pkg/models"
	"github.com/xebialabs/xl-cli/pkg/util"

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

var ignoredPaths = []string{"__test__"}

func getFuncMaps() template.FuncMap {
	funcMaps := sprig.TxtFuncMap()
	funcMaps["kebabcase"] = util.ToKebabCase
	return funcMaps
}

func AdjustPathSeperatorIfNeeded(blueprintTemplate string) string {
	re := regexp.MustCompile(`[\/\\]`)
	return re.ReplaceAllString(blueprintTemplate, string(os.PathSeparator))
}

func shouldSkipFile(templateConfig TemplateConfig, variables *[]Variable, parameters map[string]interface{}) (bool, error) {
	if !util.IsStringEmpty(templateConfig.DependsOnTrue.Val) {
		dependsOnTrueVal, err := ParseDependsOnValue(templateConfig.DependsOnTrue, variables, parameters)
		if err != nil {
			return false, err
		}
		return !dependsOnTrueVal, nil
	}
	if !util.IsStringEmpty(templateConfig.DependsOnFalse.Val) {
		dependsOnFalseVal, err := ParseDependsOnValue(templateConfig.DependsOnFalse, variables, parameters)
		if err != nil {
			return false, err
		}
		return dependsOnFalseVal, nil
	}
	return false, nil
}

// InstantiateBlueprint is entry point for the cli command
func InstantiateBlueprint(
	blueprintLocalMode bool,
	templatePath string,
	blueprintContext *BlueprintContext,
	outputDir string,
	answersFile string,
	strictAnswers bool,
	surveyOpts ...survey.AskOpt,
) error {
	var err error
	var blueprints map[string]*models.BlueprintRemote

	// if remote mode, initialize repository client
	if !blueprintLocalMode {
		util.Verbose("[cmd] Reading blueprints from remote provider: %s\n", (*blueprintContext.ActiveRepo).GetProvider())
		blueprints, err = blueprintContext.initCurrentRepoClient()
		if err != nil {
			return err
		}

		// if template path is not defined in cmd, get user selection
		if templatePath == "" {
			templatePath, err = blueprintContext.askUserToChooseBlueprint(blueprints, templatePath, surveyOpts...)
			if err != nil {
				return err
			}
		}
	} else {
		templatePath = AdjustPathSeperatorIfNeeded(templatePath)
	}

	// get local/remote blueprint definition
	util.Verbose("[cmd] Parsing Blueprint from %s\n", templatePath)
	blueprintDoc, err := blueprintContext.parseDefinitionFile(blueprintLocalMode, blueprints, templatePath)
	if err != nil {
		return err
	}
	util.Verbose("[dataPrep] Got blueprint metadata: %#v\n", blueprintDoc.Metadata)

	// ask for user input
	preparedData, err := blueprintDoc.prepareTemplateData(answersFile, strictAnswers, surveyOpts...)
	if err != nil {
		return err
	}
	util.Verbose("[dataPrep] Prepared data: %#v\n", preparedData)

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
		skipFile, err := shouldSkipFile(config, &blueprintDoc.Variables, preparedData.TemplateData)
		if err != nil {
			return err
		}
		if skipFile {
			util.Verbose("[file] skipping file [%s] since it has dependsOn value set\n", config.File)
			continue
		}

		// read template contents
		util.Verbose("[file] Fetching template file %s from %s\n", config.File, config.FullPath)
		templateContent, err := blueprintContext.fetchFileContents(config.FullPath, blueprintLocalMode, strings.HasSuffix(config.File, templateExtension))
		if err != nil {
			return err
		}
		templateString := string(*templateContent)

		// process the template file (filter based on extension)
		if strings.HasSuffix(config.File, templateExtension) {
			util.Verbose("[file] Processing template file %s\n", config.FullPath)

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
			if funk.ContainsString(ignoredPaths, filepath.Base(filepath.Dir(config.FullPath))) {
				// skip files under ignored directories
				util.Verbose("[file] Skipping file %s because path is under ignored list\n", config.FullPath)
			} else {
				// handle non-template files - copy as-it-is
				util.Verbose("[file] Copying file %s\n", config.FullPath)
				err = writeDataToFile(config.File, &templateString)
				if err != nil {
					return err
				}
			}
		}
	}
	return nil
}

func createDirectoryIfNeeded(fileName string) error {
	dir, _ := filepath.Split(fileName)
	if dir != "" && !util.PathExists(dir, true) {
		util.Verbose("[file] Creating sub-directory %s\n", dir)
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

	util.Verbose("[file] Creating blueprint output file %s\n", outputFileName)
	file, err := os.Create(outputFileName)
	if err != nil {
		return err
	}
	out, err := file.WriteString(*data)
	if err != nil {
		return err
	}
	util.Verbose("\tWrote %d bytes \n", out)
	err = file.Sync()
	if err != nil {
		return err
	}
	err = file.Close()
	if err != nil {
		return err
	}
	util.Info("[file] Blueprint output file '%s' generated successfully\n", outputFileName)
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
	util.Verbose("\tWrote %d bytes \n", bytesWrittenHeader+bytesWrittenConfig)
	util.Info("[file] Blueprint output file '%s' generated successfully\n", filename)
	return nil
}

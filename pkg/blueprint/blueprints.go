package blueprint

import (
	"fmt"
	"path/filepath"
	"sort"
	"strings"

	"text/template"

	"github.com/fatih/color"
	funk "github.com/thoas/go-funk"

	"github.com/magiconair/properties"
	"github.com/xebialabs/blueprint-cli/pkg/models"
	"github.com/xebialabs/blueprint-cli/pkg/util"

	survey "github.com/AlecAivazis/survey/v2"
	"github.com/Masterminds/sprig"
)

// SkipFinalPrompt is used in tests to skip the confirmation prompt
var SkipFinalPrompt = false

// SkipUpFinalPrompt is used in tests to skip the confirmation prompt from xl-up
var SkipUpFinalPrompt = false

// SkipUserInput is used in tests to skip the user input
var SkipUserInput = false

const (
	valuesFile        = "values.xlvals"
	valuesFileHeader  = "# This file includes all non-secret values, you can add variables here and then refer them with '!value' tag in YAML files"
	secretsFile       = "secrets.xlvals"
	secretsFileHeader = "# This file includes all secret values, and will be excluded from GIT. You can add new values and/or edit them and then refer to them using '!value' YAML tag"
	gitignoreFile     = ".gitignore"
)

var ignoredPaths = []string{"__test__"}

type ComposedBlueprint struct {
	Name            string
	BlueprintConfig *BlueprintConfig
	DependsOn       []VarField
	Parent          string
}

func getFuncMaps() template.FuncMap {
	funcMaps := sprig.TxtFuncMap()
	funcMaps["kebabcase"] = util.ToKebabCase
	return funcMaps
}

func shouldSkipFile(templateConfig TemplateConfig, parameters map[string]interface{}) (bool, error) {
	if !util.IsStringEmpty(templateConfig.DependsOn.Value) {
		dependsOnVal, err := ParseDependsOnValue(templateConfig.DependsOn, parameters)
		if err != nil {
			return false, err
		}
		if templateConfig.DependsOn.InvertBool {
			return dependsOnVal, nil
		}
		return !dependsOnVal, nil
	}
	return false, nil
}

func (config *TemplateConfig) ProcessExpression(parameters map[string]interface{}, overrideFns ExpressionOverrideFn) error {
	fieldsToSkip := []string{""} // these fields have special processing
	return ProcessExpressionField(config, fieldsToSkip, parameters, config.Path, overrideFns)
}

type BlueprintParams struct {
	TemplatePath         string
	AnswersFile          string
	StrictAnswers        bool
	UseDefaultsAsValue   bool
	FromUpCommand        bool
	PrintSummaryTable    bool
	ExistingPreparedData *PreparedData
	OverrideDefaults     map[string]string
	AnswersMap           map[string]string
}

// InstantiateBlueprint is entry point for the cli command
func InstantiateBlueprint(
	params BlueprintParams,
	blueprintContext *BlueprintContext,
	generatedBlueprint *GeneratedBlueprint,
	overrideFns ExpressionOverrideFn,
	surveyOpts ...survey.AskOpt,
) (*PreparedData, *BlueprintConfig, error) {
	var err error
	var blueprints map[string]*models.BlueprintRemote

	// initialize repository client
	util.Verbose("[cmd] Reading blueprints from provider: %s\n", (*blueprintContext.ActiveRepo).GetProvider())
	blueprints, err = blueprintContext.initCurrentRepoClient()
	if err != nil {
		return nil, nil, err
	}

	// if template path is not defined in cmd, get user selection
	if params.TemplatePath == "" {
		params.TemplatePath, err = blueprintContext.askUserToChooseBlueprint(blueprints, params.TemplatePath, surveyOpts...)
		if err != nil {
			return nil, nil, err
		}
	}

	preparedData, blueprintDoc, err := prepareMergedTemplateData(blueprintContext, blueprints, params, overrideFns, surveyOpts...)
	if err != nil {
		return nil, nil, err
	}
	util.Verbose("[dataPrep] Prepared data: %#v\n", preparedData)

	// Final prompt from user to start generation process
	toContinue := true
	toSaveFiles := true

	// if this is from UP command, ask confirmation for xl-up
	if params.FromUpCommand && params.PrintSummaryTable && !SkipUpFinalPrompt {

		err := survey.AskOne(&survey.Confirm{Message: models.UpFinalPrompt, Default: true}, &toContinue, surveyOpts...)

		if err != nil {
			return nil, nil, err
		}

		if !toContinue && toSaveFiles {
			isQuiet := util.IsQuiet
			util.IsQuiet = false
			util.Info(util.Green("Generating all files and exiting...\n"))
			util.IsQuiet = isQuiet
		}
	}

	if toSaveFiles {
		createXebiaLabsFolder := !blueprintDoc.Metadata.SuppressXebiaLabsFolder

		// save prepared data to values & secrets files
		if createXebiaLabsFolder || len(preparedData.Values) != 0 {
			err = writeConfigToFile(valuesFileHeader, preparedData.Values, generatedBlueprint, filepath.Join(generatedBlueprint.OutputDir, valuesFile))
			if err != nil {
				return nil, nil, err
			}
		}

		if createXebiaLabsFolder || len(preparedData.Secrets) != 0 {
			err = writeConfigToFile(secretsFileHeader, preparedData.Secrets, generatedBlueprint, filepath.Join(generatedBlueprint.OutputDir, secretsFile))
			if err != nil {
				return nil, nil, err
			}
			// generate .gitignore file
			gitignoreData := secretsFile
			err = writeDataToFile(generatedBlueprint, filepath.Join(generatedBlueprint.OutputDir, gitignoreFile), &gitignoreData)
			if err != nil {
				return nil, nil, err
			}
		}

		// execute each template file found
		for _, config := range blueprintDoc.TemplateConfigs {
			config.ProcessExpression(preparedData.TemplateData, overrideFns)
			skipFile, err := shouldSkipFile(config, preparedData.TemplateData)
			if err != nil {
				return nil, nil, err
			}

			if skipFile {
				util.Verbose("[file] skipping file [%s] since it has writeIf value set or is skipped by composed blueprint\n", config.Path)
				continue
			}

			// read template contents
			util.Verbose("[file] Fetching template file %s from %s\n", config.Path, config.FullPath)
			templateContent, err := blueprintContext.fetchFileContents(config.FullPath, strings.HasSuffix(config.Path, templateExtension))
			if err != nil {
				return nil, nil, err
			}
			templateString := string(*templateContent)
			finalFileName := config.Path
			if config.RenameTo.Value != "" {
				finalFileName = config.RenameTo.Value
				util.Verbose("[file] Renaming template file %s to %s as it is overridden by composed blueprint\n", config.Path, finalFileName)
			}

			// process the template file (filter based on extension)
			if strings.HasSuffix(config.Path, templateExtension) {
				util.Verbose("[file] Processing template file %s\n", config.FullPath)

				// read & process the template
				tmpl := template.Must(template.New(config.Path).Funcs(getFuncMaps()).Parse(templateString))
				processedTmpl := &strings.Builder{}
				err = tmpl.Execute(processedTmpl, preparedData.TemplateData)
				if err != nil {
					return nil, nil, err
				}

				// write the processed template to a file
				finalTmpl := strings.TrimSpace(processedTmpl.String())

				err = writeDataToFile(generatedBlueprint, strings.Replace(finalFileName, templateExtension, "", 1), &finalTmpl)
				if err != nil {
					return nil, nil, err
				}
			} else {
				if funk.ContainsString(ignoredPaths, filepath.Base(filepath.Dir(config.FullPath))) {
					// skip files under ignored directories
					util.Verbose("[file] Skipping file %s because path is under ignored list\n", config.FullPath)
				} else {
					// handle non-template files - copy as-it-is
					util.Verbose("[file] Copying file %s\n", config.FullPath)
					err = writeDataToFile(generatedBlueprint, finalFileName, &templateString)
					if err != nil {
						return nil, nil, err
					}
				}
			}
		}
		util.Info("Please refer to file 'xebialabs/secrets.xlvals' for the default secrets\n")
		if blueprintDoc.Metadata.Instructions != "" {
			util.Info("\n\n%s\n\n", color.GreenString(blueprintDoc.Metadata.Instructions))
		}
	}

	if toContinue {
		return preparedData, blueprintDoc, nil
	} else {
		return nil, nil, fmt.Errorf("xl execution cancelled")
	}
}

func prepareMergedTemplateData(
	blueprintContext *BlueprintContext,
	blueprints map[string]*models.BlueprintRemote,
	params BlueprintParams,
	overrideFns ExpressionOverrideFn,
	surveyOpts ...survey.AskOpt,
) (*PreparedData, *BlueprintConfig, error) {
	// get blueprint definition
	blueprintDocs, masterBlueprintDoc, err := getBlueprintConfig(blueprintContext, blueprints, params.TemplatePath, []VarField{VarField{}}, "")
	if err != nil {
		return nil, nil, err
	}

	mergedData := NewPreparedData()
	if params.ExistingPreparedData != nil {
		// merge from existing data if any
		util.CopyIntoStringInterfaceMap(params.ExistingPreparedData.TemplateData, mergedData.TemplateData)
		util.CopyIntoStringInterfaceMap(params.ExistingPreparedData.SummaryData, mergedData.SummaryData)
		util.CopyIntoStringInterfaceMap(params.ExistingPreparedData.Values, mergedData.Values)
		util.CopyIntoStringInterfaceMap(params.ExistingPreparedData.Secrets, mergedData.Secrets)
	}
	mergedBlueprintDoc := &BlueprintConfig{
		ApiVersion: masterBlueprintDoc.ApiVersion,
		Kind:       masterBlueprintDoc.Kind,
		Metadata:   masterBlueprintDoc.Metadata,
		Include:    masterBlueprintDoc.Include,
	}
	// A map holding skipped blueprint names
	var skippedBlueprints []string
	for _, blueprintDoc := range blueprintDocs {
		var ok = true
		// skip child templates when parents are skipped
		for _, v := range skippedBlueprints {
			if blueprintDoc.Parent == v {
				ok = false
				break
			}
		}
		if ok {
			// Evaluate dependsOn
			ok, err = evaluateAndSkipIfDependsOnIsFalse(blueprintDoc.DependsOn, mergedData, overrideFns)
			if err != nil {
				return nil, nil, err
			}
		}
		if ok {
			// ask for user input
			preparedData, err := blueprintDoc.BlueprintConfig.prepareTemplateData(params, mergedData, overrideFns, surveyOpts...)
			if err != nil {
				return nil, nil, err
			}

			// merge
			util.CopyIntoStringInterfaceMap(preparedData.TemplateData, mergedData.TemplateData)
			util.CopyIntoStringInterfaceMap(preparedData.SummaryData, mergedData.SummaryData)
			util.CopyIntoStringInterfaceMap(preparedData.Values, mergedData.Values)
			util.CopyIntoStringInterfaceMap(preparedData.Secrets, mergedData.Secrets)
			// append params
			mergedBlueprintDoc.Variables = append(mergedBlueprintDoc.Variables, blueprintDoc.BlueprintConfig.Variables...)
			// append files
			mergedBlueprintDoc.TemplateConfigs = append(mergedBlueprintDoc.TemplateConfigs, blueprintDoc.BlueprintConfig.TemplateConfigs...)
		} else {
			skippedBlueprints = append(skippedBlueprints, blueprintDoc.Name)
		}
	}

	// Print summary table
	if params.PrintSummaryTable {
		// use util.Print so that this is not skipped in quiet mode
		if params.UseDefaultsAsValue && params.AnswersFile == "" && params.AnswersMap == nil {
			util.Print("Using default values:\n")
		}

		util.Print(util.DataMapTable(&mergedData.SummaryData, util.TableAlignLeft, 30, 50, "\t", 1, params.FromUpCommand))
	}

	if !SkipFinalPrompt {
		// Final prompt from user to start generation process
		toContinue := false
		err := survey.AskOne(&survey.Confirm{Message: models.BlueprintFinalPrompt, Default: true}, &toContinue, surveyOpts...)

		if err != nil {
			return nil, nil, err
		}
		if !toContinue {
			return nil, nil, fmt.Errorf("blueprint generation cancelled")
		}
	}

	return mergedData, mergedBlueprintDoc, nil
}

func evaluateAndSkipIfDependsOnIsFalse(dependsOn []VarField, mergedData *PreparedData, overrideFns ExpressionOverrideFn) (bool, error) {
	for _, dependOn := range dependsOn {
		procDependsOn, err := GetProcessedExpressionValue(dependOn, mergedData.TemplateData, overrideFns)
		if err != nil {
			return false, err
		}
		if util.IsStringEmpty(procDependsOn.Value) {
			continue
		}
		dependsOnVal, err := ParseDependsOnValue(procDependsOn, mergedData.TemplateData)
		if err != nil {
			return false, err
		}
		if !dependsOnVal {
			return false, nil
		}
	}
	return true, nil
}

func getBlueprintConfig(
	blueprintContext *BlueprintContext,
	blueprints map[string]*models.BlueprintRemote,
	templatePath string,
	dependsOn []VarField,
	parentBlueprint string,
) ([]*ComposedBlueprint, *BlueprintConfig, error) {
	util.Verbose("[cmd] Parsing Blueprint from %s\n", templatePath)
	blueprintDocs := make([]*ComposedBlueprint, 0)
	blueprint := blueprints[templatePath]
	masterBlueprintDoc, err := blueprintContext.parseDefinitionFile(blueprint, templatePath)
	if err != nil {
		return nil, nil, err
	}

	util.Verbose("[compose] Found %d included blueprints\n", len(masterBlueprintDoc.Include))
	blueprintDocs, err = composeBlueprints(templatePath, masterBlueprintDoc, blueprintContext, blueprints, dependsOn, parentBlueprint)
	if err != nil {
		return nil, nil, err
	}
	return blueprintDocs, masterBlueprintDoc, nil
}

func composeBlueprints(
	blueprintName string,
	blueprintDoc *BlueprintConfig,
	blueprintContext *BlueprintContext,
	blueprints map[string]*models.BlueprintRemote,
	dependsOn []VarField, parentBlueprint string,
) ([]*ComposedBlueprint, error) {
	includeBefore := make([]*ComposedBlueprint, 0)
	blueprintDocs := make([]*ComposedBlueprint, 0)
	// add the master blueprint
	blueprintDocs = append(blueprintDocs, &ComposedBlueprint{blueprintName, blueprintDoc, dependsOn, parentBlueprint})
	for _, included := range blueprintDoc.Include {
		util.Verbose("[compose] Fetch included blueprint %s\n", included.Blueprint)

		// combine parent and child DependsOn fields into a single array
		dependencies := make([]VarField, 0)
		if dependsOn != nil {
			dependencies = append(dependencies, dependsOn...)
		}

		// don't add duplicate DependsOn conditions
		foundMatch := false
		for _, m := range dependencies {
			if included.DependsOn == m {
				foundMatch = true
			}
		}
		if !foundMatch {
			dependencies = append(dependencies, included.DependsOn)
		}

		// fetch blueprint from current repo
		composedBlueprintDocs, currentBlueprintDoc, err := getBlueprintConfig(blueprintContext, blueprints, included.Blueprint, dependencies, blueprintName)
		if err != nil {
			return nil, err
		}
		if included.ParameterOverrides != nil {
			for _, override := range included.ParameterOverrides {
				targetIndex := findParameter(currentBlueprintDoc.Variables, override.Name)
				if targetIndex != -1 {
					util.MergeStructFields(&(currentBlueprintDoc.Variables[targetIndex]), &override, []string{"Name", "Type"})
				} else {
					util.Verbose("[compose] Could not find parameterOverride for %s\n", override.Name.Value)
				}
			}
		}
		if included.FileOverrides != nil {
			for _, override := range included.FileOverrides {
				targetIndex := findTemplateConfig(currentBlueprintDoc.TemplateConfigs, override.Path)
				if targetIndex != -1 {
					util.MergeStructFields(&(currentBlueprintDoc.TemplateConfigs[targetIndex]), &override, []string{"Path"})
				} else {
					util.Verbose("[compose] Could not find fileOverride for %s\n", override.Path)
				}
			}
		}
		if currentBlueprintDoc != nil {
			if included.Stage == "before" {
				includeBefore = append(includeBefore, composedBlueprintDocs...)
			} else {
				blueprintDocs = append(blueprintDocs, composedBlueprintDocs...)
			}
		}
	}

	if len(includeBefore) > 0 {
		blueprintDocs = append(includeBefore, blueprintDocs...)
	}

	return blueprintDocs, nil
}

func findParameter(params []Variable, name VarField) int {
	for i, param := range params {
		if param.Name.Value == name.Value {
			return i
		}
	}
	return -1
}

func findTemplateConfig(configs []TemplateConfig, path string) int {
	for i, config := range configs {
		if config.Path == path {
			return i
		}
	}
	return -1
}

// --utility functions
func writeDataToFile(generatedBlueprint *GeneratedBlueprint, outputFileName string, data *string) error {
	util.Verbose("[file] Creating blueprint output file %s\n", outputFileName)
	file, err := generatedBlueprint.GetOutputFile(outputFileName)
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

func writeConfigToFile(header string, config map[string]interface{}, generatedBlueprint *GeneratedBlueprint, filename string) error {
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
	f, err := generatedBlueprint.GetOutputFile(filename)
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

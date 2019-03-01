package blueprint

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"reflect"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"github.com/thoas/go-funk"
	"github.com/xebialabs/xl-cli/pkg/cloud/aws"
	"github.com/xebialabs/xl-cli/pkg/models"
	"github.com/xebialabs/xl-cli/pkg/osHelper"
	"github.com/xebialabs/xl-cli/pkg/repository"
	"github.com/xebialabs/xl-cli/pkg/util"
	"github.com/xebialabs/yaml"
	"gopkg.in/AlecAivazis/survey.v1"
)

// Constants
const (
	FnAWS = "aws"
	FnOs  = "os"

	tagFn         = "!fn"
	tagExpression = "!expression"
	fmtTagValue   = "!value %s"
)

// InputType constants
const (
	TypeInput   = "Input"
	TypeEditor  = "Editor"
	TypeFile    = "File"
	TypeSelect  = "Select"
	TypeConfirm = "Confirm"
)

var validTypes = []string{TypeInput, TypeEditor, TypeFile, TypeSelect, TypeConfirm}

// Blueprint YAML doc definition
type BlueprintYaml struct {
	ApiVersion      string      `yaml:"apiVersion,omitempty"`
	Kind            string      `yaml:"kind,omitempty"`
	Metadata        interface{} `yaml:"metadata,omitempty"`
	Parameters      interface{} `yaml:"parameters,omitempty"`
	Files           interface{} `yaml:"files,omitempty"`
	Spec            Spec
	TemplateConfigs []TemplateConfig
	Variables       []Variable
}
type Spec struct {
	Parameters interface{} `yaml:"parameters,omitempty"`
	Files      interface{} `yaml:"files,omitempty"`
}
type VarField struct {
	Val  string
	Bool bool
	Tag  string
}
type Variable struct {
	Name           VarField
	Type           VarField
	Secret         VarField
	Value          VarField
	Description    VarField
	Default        VarField
	DependsOnTrue  VarField
	DependsOnFalse VarField
	Options        []VarField
	Pattern        VarField
	SaveInXlVals   VarField
}
type PreparedData struct {
	TemplateData map[string]interface{}
	Values       map[string]interface{}
	Secrets      map[string]interface{}
}

func NewPreparedData() *PreparedData {
	templateData := make(map[string]interface{})
	values := make(map[string]interface{})
	secrets := make(map[string]interface{})
	return &PreparedData{TemplateData: templateData, Values: values, Secrets: secrets}
}

// regular Expressions
var regExFn = regexp.MustCompile(`([\w\d]+).([\w\d]+)\(([,\s\w\d]*)\)(?:\.([\w\d]*)|\[([\d]+)\])*`)

// reflect utilities for VarField
func getVariableField(variable *Variable, fieldName string) reflect.Value {
	return reflect.ValueOf(variable).Elem().FieldByName(fieldName)
}

func setVariableField(field *reflect.Value, value *VarField) {
	if field.IsValid() {
		field.Set(reflect.ValueOf(*value))
	}
}

func ParseDependsOnValue(varField VarField, variables *[]Variable, parameters map[string]interface{}) (bool, error) {
	tagVal := varField.Tag
	fieldVal := varField.Val
	switch tagVal {
	case tagFn:
		values, err := processCustomFunction(fieldVal)
		if err != nil {
			return false, err
		}
		if len(values) == 0 {
			return false, fmt.Errorf("function [%s] results is empty", fieldVal)
		}
		util.Verbose("[fn] Processed value of function [%s] is: %s\n", fieldVal, values[0])

		dependsOnVal, err := strconv.ParseBool(values[0])
		if err != nil {
			return false, err
		}
		return dependsOnVal, nil
	case tagExpression:
		value, err := ProcessCustomExpression(fieldVal, parameters)
		if err != nil {
			return false, err
		}
		dependsOnVal, ok := value.(bool)
		if ok {
			util.Verbose("[expression] Processed value of expression [%s] is: %v\n", fieldVal, dependsOnVal)
			return dependsOnVal, nil
		}
		return false, fmt.Errorf("Expression [%s] result is invalid for a boolean field", fieldVal)
	}
	dependsOnVar, err := findVariableByName(variables, fieldVal)
	if err != nil {
		return false, err
	}
	return dependsOnVar.Value.Bool, nil
}

// GetDefaultVal variable struct functions
func (variable *Variable) GetDefaultVal(variables map[string]interface{}) interface{} {
	defaultVal := variable.Default.Val
	switch variable.Default.Tag {
	case tagFn:
		values, err := processCustomFunction(defaultVal)
		if err != nil {
			util.Info("Error while processing default value !fn [%s] for [%s]. %s", defaultVal, variable.Name.Val, err.Error())
			defaultVal = ""
		} else {
			util.Verbose("[fn] Processed value of function [%s] is: %s\n", defaultVal, values[0])
			if variable.Type.Val == TypeConfirm {
				boolVal, err := strconv.ParseBool(values[0])
				if err != nil {
					util.Info("Error while processing default value !fn [%s] for [%s]. %s", defaultVal, variable.Name.Val, err.Error())
					return false
				}
				variable.Default.Bool = boolVal
				return boolVal
			}
			return values[0]
		}
	case tagExpression:
		value, err := ProcessCustomExpression(defaultVal, variables)
		if err != nil {
			util.Info("Error while processing default value !expression [%s] for [%s]. %s", defaultVal, variable.Name.Val, err.Error())
			defaultVal = ""
		} else {
			util.Verbose("[expression] Processed value of expression [%s] is: %s\n", defaultVal, value)
			boolVal, ok := value.(bool)
			if ok {
				if variable.Type.Val == TypeConfirm {
					variable.Default.Bool = boolVal
				}
			}
			return value
		}
	}

	// return false if this is a skipped confirm question
	if defaultVal == "" && variable.Type.Val == TypeConfirm {
		return false
	}
	return defaultVal
}

func (variable *Variable) GetValueFieldVal(parameters map[string]interface{}) string {
	switch variable.Value.Tag {
	case tagFn:
		values, err := processCustomFunction(variable.Value.Val)
		if err != nil {
			util.Info("Error while processing !fn [%s]. Please update the value for [%s] manually. %s", variable.Value.Val, variable.Name.Val, err.Error())
			return ""
		}
		util.Verbose("[fn] Processed value of function [%s] is: %s\n", variable.Value.Val, values[0])
		if variable.Type.Val == TypeConfirm {
			boolVal, err := strconv.ParseBool(values[0])
			if err != nil {
				util.Info("Error while processing !fn [%s]. Please update the value for [%s] manually. %s", variable.Value.Val, variable.Name.Val, err.Error())
				return ""
			}
			variable.Value.Bool = boolVal
			return values[0]
		}
		return values[0]
	case tagExpression:
		value, err := ProcessCustomExpression(variable.Value.Val, parameters)
		if err != nil {
			util.Info("Error while processing !expression [%s]. Please update the value for [%s] manually. %s", variable.Value.Val, variable.Name.Val, err.Error())
			return ""
		} else {
			util.Verbose("[expression] Processed value of expression [%s] is: %s\n", variable.Value.Val, value)
			boolVal, ok := value.(bool)
			if ok {
				if variable.Type.Val == TypeConfirm {
					variable.Value.Bool = boolVal
				}
				return fmt.Sprint(boolVal)
			}
			return value.(string)
		}
	}
	return variable.Value.Val
}

func (variable *Variable) GetOptions(parameters map[string]interface{}) []string {
	var options []string
	for _, option := range variable.Options {
		switch option.Tag {
		case tagFn:
			opts, err := processCustomFunction(option.Val)
			if err != nil {
				util.Info("Error while processing !fn [%s]. Please update the value for [%s] manually. %s", option.Val, variable.Name.Val, err.Error())
				return nil
			}
			util.Verbose("[fn] Processed value of function [%s] is: %s\n", option.Val, opts)
			options = append(options, opts...)
		case tagExpression:
			opts, err := ProcessCustomExpression(option.Val, parameters)
			if err != nil {
				util.Info("Error while processing !expression [%s]. Please update the value for [%s] manually. %s", option.Val, variable.Name.Val, err.Error())
				return nil
			}
			switch val := opts.(type) {
			case []string:
				util.Verbose("[expression] Processed value of expression [%s] is: %v\n", option.Val, val)
				options = append(options, val...)
			case []interface{}:
				util.Verbose("[expression] Processed value of expression [%s] is: %v\n", option.Val, val)
				for _, option := range val {
					options = append(options, fmt.Sprint(option))
				}
			default:
				util.Info("Error while processing !expression [%s]. Please update the value for [%s] manually. %s", option.Val, variable.Name.Val, "Return type should be a string array")
				return nil
			}
		default:
			options = append(options, option.Val)
		}
	}
	return options
}

func (variable *Variable) GetUserInput(defaultVal interface{}, parameters map[string]interface{}, surveyOpts ...survey.AskOpt) (interface{}, error) {
	var answer string
	var err error
	switch variable.Type.Val {
	case TypeInput:
		if variable.Secret.Bool == true {
			questionMsg := prepareQuestionText(variable.Description.Val, fmt.Sprintf("What is the value of %s?", variable.Name.Val))
			if defaultVal != "" {
				questionMsg += fmt.Sprintf(" (%s)", defaultVal)
			}
			err = survey.AskOne(
				&survey.Password{Message: questionMsg},
				&answer,
				validatePrompt(variable.Pattern.Val, true),
				surveyOpts...,
			)

			// if user bypassed question, replace with default value
			if answer == "" {
				util.Verbose("[input] Got empty response for secret field '%s', replacing with default value: %s\n", variable.Name.Val, defaultVal)
				answer = defaultVal.(string)
			}
		} else {
			err = survey.AskOne(
				&survey.Input{
					Message: prepareQuestionText(variable.Description.Val, fmt.Sprintf("What is the value of %s?", variable.Name.Val)),
					Default: defaultVal.(string),
				},
				&answer,
				validatePrompt(variable.Pattern.Val, false),
				surveyOpts...,
			)
		}
	case TypeEditor:
		err = survey.AskOne(
			&survey.Editor{
				Message:       prepareQuestionText(variable.Description.Val, fmt.Sprintf("What is the value of %s?", variable.Name.Val)),
				Default:       defaultVal.(string),
				HideDefault:   true,
				AppendDefault: true,
			},
			&answer,
			validatePrompt(variable.Pattern.Val, false),
			surveyOpts...,
		)
	case TypeFile:
		var filePath string
		err = survey.AskOne(
			&survey.Input{
				Message: prepareQuestionText(variable.Description.Val, fmt.Sprintf("What is the file path (relative/absolute) for %s?", variable.Name.Val)),
				Default: defaultVal.(string),
			},
			&filePath,
			validateFilePath(),
			surveyOpts...,
		)

		// read file contents & save as answer
		util.Verbose("[input] Reading file contents from path: %s\n", filePath)
		data, err := ioutil.ReadFile(filePath)
		if err != nil {
			return "", err
		}
		answer = string(data)
	case TypeSelect:
		options := variable.GetOptions(parameters)
		if err != nil {
			return "", err
		}
		err = survey.AskOne(
			&survey.Select{
				Message:  prepareQuestionText(variable.Description.Val, fmt.Sprintf("Select value for %s?", variable.Name.Val)),
				Options:  options,
				Default:  defaultVal.(string),
				PageSize: 10,
			},
			&answer,
			validatePrompt(variable.Pattern.Val, false),
			surveyOpts...,
		)
	case TypeConfirm:
		var confirm bool
		err = survey.AskOne(
			&survey.Confirm{
				Message: prepareQuestionText(variable.Description.Val, fmt.Sprintf("%s?", variable.Name.Val)),
				Default: variable.Default.Bool,
			},
			&confirm,
			validatePrompt(variable.Pattern.Val, false),
			surveyOpts...,
		)
		if err != nil {
			return "", err
		}
		variable.Value.Bool = confirm
		// TypeConfirm returns a boolean type
		return confirm, nil
	}
	// This always returns string
	return answer, err
}

// parse blueprint definition doc
func parseTemplateMetadata(blueprintVars *[]byte, templatePath string, blueprintRepository *BlueprintContext, isLocal bool) (*BlueprintYaml, error) {
	decoder := yaml.NewDecoder(bytes.NewReader(*blueprintVars))
	decoder.SetStrict(true)
	doc := BlueprintYaml{}
	err := decoder.Decode(&doc)
	if err != nil {
		return nil, err
	}

	// parse & validate
	err = doc.parseParameters()
	if err != nil {
		return nil, err
	}
	err = doc.parseFiles(templatePath, blueprintRepository, isLocal)
	if err != nil {
		return nil, err
	}
	err = doc.validate()
	return &doc, err
}

// verify blueprint directory & generate full paths for local files
func (blueprintDoc *BlueprintYaml) verifyTemplateDirAndGenFullPaths(templatePath string) error {
	if util.PathExists(templatePath, true) {
		util.Verbose("[repository] Verifying local path and files within: %s \n", templatePath)
		var filePaths []string

		// walk the root directory
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
			return err
		}
		if len(filePaths) == 0 {
			return fmt.Errorf("path [%s] doesn't include any valid files", templatePath)
		}

		// generate full local paths
		sort.Strings(filePaths)
		for _, filePath := range filePaths {
			relativePath := getFilePathRelativeToTemplatePath(filePath, templatePath)
			_, filename := filepath.Split(filePath)
			if filename != repository.BlueprintMetadataFileName+".yaml" && filename != repository.BlueprintMetadataFileName+".yml" {
				// Append full path to existing file, or create new one
				configIndex := findInTemplateConfigs(blueprintDoc.TemplateConfigs, relativePath)
				if configIndex == -1 {
					blueprintDoc.TemplateConfigs = append(blueprintDoc.TemplateConfigs, TemplateConfig{
						File:     relativePath,
						FullPath: filePath,
					})
				} else {
					blueprintDoc.TemplateConfigs[configIndex].FullPath = filePath
				}
			}
		}
		return nil
	}
	return fmt.Errorf("path [%s] doesn't exist", templatePath)
}

// parse doc parameters into list of variables
func (blueprintDoc *BlueprintYaml) parseParameters() error {
	var parameters []map[interface{}]interface{}
	if blueprintDoc.Spec != (Spec{}) {
		parameters = util.TransformToMap(blueprintDoc.Spec.Parameters)
	} else {
		parameters = util.TransformToMap(blueprintDoc.Parameters)
	}
	for _, m := range parameters {
		parsedVar, err := parseParameterMap(&m)
		if err != nil {
			return err
		}
		blueprintDoc.Variables = append(blueprintDoc.Variables, parsedVar)
	}
	return nil
}

// parse doc files into list of TemplateConfig
func (blueprintDoc *BlueprintYaml) parseFiles(templatePath string, blueprintRepository *BlueprintContext, isLocal bool) error {
	var files []map[interface{}]interface{}
	if blueprintDoc.Spec != (Spec{}) {
		files = util.TransformToMap(blueprintDoc.Spec.Files)
	} else {
		files = util.TransformToMap(blueprintDoc.Files)
	}
	for _, m := range files {
		templateConfig, err := parseFileMap(&m)
		if err != nil {
			return err
		}
		if isLocal {
			// If local mode, fix path separator in needed cases
			templateConfig.File = AdjustPathSeperatorIfNeeded(templateConfig.File)
		}
		blueprintDoc.TemplateConfigs = append(blueprintDoc.TemplateConfigs, templateConfig)
	}
	return nil
}

// validate blueprint yaml document based on required fields
func (blueprintDoc *BlueprintYaml) validate() error {
	if blueprintDoc.ApiVersion != models.YamlFormatVersion {
		return fmt.Errorf("api version needs to be %s", models.YamlFormatVersion)
	}
	if blueprintDoc.Kind != models.BlueprintSpecKind {
		return fmt.Errorf("yaml document kind needs to be %s", models.BlueprintSpecKind)
	}
	err := validateVariables(&blueprintDoc.Variables)
	if err != nil {
		return err
	}
	return validateFiles(&blueprintDoc.TemplateConfigs)
}

// prepare template data by getting user input and calling named functions
func (blueprintDoc *BlueprintYaml) prepareTemplateData(surveyOpts ...survey.AskOpt) (*PreparedData, error) {
	data := NewPreparedData()
	for i, variable := range blueprintDoc.Variables {
		// process default field value
		defaultVal := variable.GetDefaultVal(data.TemplateData)

		// skip question based on DependsOn fields
		if !util.IsStringEmpty(variable.DependsOnTrue.Val) {
			dependsOnTrueVal, err := ParseDependsOnValue(variable.DependsOnTrue, &blueprintDoc.Variables, data.TemplateData)
			if err != nil {
				return nil, err
			}
			if skipQuestionOnCondition(&variable, variable.DependsOnTrue.Val, dependsOnTrueVal, data, defaultVal, false) {
				continue
			}
		}
		if !util.IsStringEmpty(variable.DependsOnFalse.Val) {
			dependsOnFalseVal, err := ParseDependsOnValue(variable.DependsOnFalse, &blueprintDoc.Variables, data.TemplateData)
			if err != nil {
				return nil, err
			}
			if skipQuestionOnCondition(&variable, variable.DependsOnFalse.Val, dependsOnFalseVal, data, defaultVal, true) {
				continue
			}
		}

		// skip user input if value field is present
		if variable.Value.Val != "" {
			parsedVal := variable.GetValueFieldVal(data.TemplateData)

			// check if resulting value is non-empty
			if parsedVal != "" {
				if variable.Type.Val == TypeConfirm {
					saveItemToTemplateDataMap(&variable, data, variable.Value.Bool)
				} else {
					saveItemToTemplateDataMap(&variable, data, parsedVal)
				}
				util.Verbose("[dataPrep] Skipping question for parameter [%s] because value [%s] is present\n", variable.Name.Val, variable.Value.Val)
				continue
			} else {
				util.Verbose("[dataPrep] Parsed value for parameter [%s] is empty, therefore not being skipped\n", variable.Name.Val)
			}
		}

		// ask question based on type to get value - if value field is not present
		util.Verbose("[dataPrep] Processing template variable [Name: %s, Type: %s]\n", variable.Name.Val, variable.Type.Val)
		answer, err := variable.GetUserInput(defaultVal, data.TemplateData, surveyOpts...)
		if err != nil {
			return nil, err
		}
		if variable.Type.Val == TypeConfirm {
			blueprintDoc.Variables[i] = variable
		}
		saveItemToTemplateDataMap(&variable, data, answer)
	}

	if !SkipFinalPrompt {
		// Final prompt from user to start generation process
		toContinue := false
		err := survey.AskOne(&survey.Confirm{Message: models.BlueprintFinalPrompt, Default: true}, &toContinue, nil, surveyOpts...)
		if err != nil {
			return nil, err
		}
		if !toContinue {
			return nil, fmt.Errorf("blueprint generation cancelled")
		}
	}

	return data, nil
}

func validateVariables(variables *[]Variable) error {
	var variableNames []string
	for _, userVar := range *variables {
		// validate non-empty
		if util.IsStringEmpty(userVar.Name.Val) || util.IsStringEmpty(userVar.Type.Val) {
			return fmt.Errorf("parameter [%s] is missing required fields: [type]", userVar.Name.Val)
		}

		// validate type field
		if !util.IsStringInSlice(userVar.Type.Val, validTypes) {
			return fmt.Errorf("type [%s] is not valid for parameter [%s]", userVar.Type.Val, userVar.Name.Val)
		}

		// validate select case
		if userVar.Type.Val == TypeSelect && len(userVar.Options) == 0 {
			return fmt.Errorf("at least one option field is need to be set for parameter [%s]", userVar.Name.Val)
		}

		// validate file case
		if userVar.Type.Val == TypeFile && !util.IsStringEmpty(userVar.Value.Val) {
			return fmt.Errorf("'value' field is not allowed for file input type")
		}

		variableNames = append(variableNames, userVar.Name.Val)
	}

	// Check if there are duplicate variable names
	if len(funk.UniqString(variableNames)) != len(*variables) {
		return fmt.Errorf("variable names must be unique within blueprint 'parameters' definition")
	}
	return nil
}

func validateFiles(configs *[]TemplateConfig) error {
	for _, file := range *configs {
		// validate non-empty
		if util.IsStringEmpty(file.File) {
			return fmt.Errorf("path is missing for file specification in files")
		}
		if filepath.IsAbs(file.File) || strings.HasPrefix(file.File, "..") || strings.HasPrefix(file.File, "."+string(os.PathSeparator)) {
			return fmt.Errorf("path for file specification cannot start with /, .. or ./")
		}
	}
	return nil
}

func parseParameterMap(m *map[interface{}]interface{}) (Variable, error) {
	parsedVar := Variable{}
	for k, v := range *m {
		switch val := v.(type) {
		case string:
			// Set string field
			field := getVariableField(&parsedVar, strings.Title(k.(string)))
			setVariableField(&field, &VarField{Val: val})
		case int, uint, uint8, uint16, uint32, uint64:
			// Set integer field
			field := getVariableField(&parsedVar, strings.Title(k.(string)))
			setVariableField(&field, &VarField{Val: fmt.Sprint(v)})
		case float32, float64:
			// Set float field
			field := getVariableField(&parsedVar, strings.Title(k.(string)))
			setVariableField(&field, &VarField{Val: fmt.Sprintf("%f", v)})
		case bool:
			// Set boolean field
			field := getVariableField(&parsedVar, strings.Title(k.(string)))
			setVariableField(&field, &VarField{Val: strconv.FormatBool(val), Bool: val})
		case []interface{}:
			// Set []VarField
			field := getVariableField(&parsedVar, strings.Title(k.(string)))
			list := val
			if len(list) > 0 {
				switch t := list[0].(type) {
				case int, uint, uint8, uint16, uint32, uint64, float32, float64, string, yaml.CustomTag: //handle list of options
					field.Set(reflect.MakeSlice(reflect.TypeOf([]VarField{}), len(list), len(list)))
					for i, w := range list {
						switch wType := w.(type) {
						case int, uint, uint8, uint16, uint32, uint64:
							field.Index(i).Set(reflect.ValueOf(VarField{Val: fmt.Sprint(v)}))
						case float32, float64:
							field.Index(i).Set(reflect.ValueOf(VarField{Val: fmt.Sprintf("%f", v)}))
						case string:
							field.Index(i).Set(reflect.ValueOf(VarField{Val: w.(string)}))
						case yaml.CustomTag:
							customTag := w.(yaml.CustomTag)
							field.Index(i).Set(reflect.ValueOf(VarField{Val: customTag.Value, Tag: customTag.Tag}))
						default:
							return Variable{}, fmt.Errorf("unknown list item type %s", wType)
						}
					}
				default:
					return Variable{}, fmt.Errorf("unknown list type: %s", t)
				}
			}
		case yaml.CustomTag:
			// Set string field with YAML tag
			switch val.Tag {
			case tagFn, tagExpression:
				field := getVariableField(&parsedVar, strings.Title(k.(string)))
				setVariableField(&field, &VarField{Val: val.Value, Tag: val.Tag})
			default:
				return Variable{}, fmt.Errorf("unknown tag %s %s", val.Tag, val.Value)
			}
		case nil:
			util.Verbose("[dataPrep] Got empty metadata variable field with key [%s]\n", k)
		default:
			return Variable{}, fmt.Errorf("unknown variable type [%s]", val)
		}
	}
	return parsedVar, nil
}

func parseFileMap(m *map[interface{}]interface{}) (TemplateConfig, error) {
	config := TemplateConfig{}
	for k, v := range *m {
		keyName, ok := k.(string)
		if ok {
			switch val := v.(type) {
			case string:
				if keyName == "path" {
					config.File = val
				} else {
					field := reflect.ValueOf(&config).Elem().FieldByName(strings.Title(keyName))
					setVariableField(&field, &VarField{Val: val})
				}
			case yaml.CustomTag:
				// Set string field with YAML tag
				switch val.Tag {
				case tagFn, tagExpression:
					field := reflect.ValueOf(&config).Elem().FieldByName(strings.Title(keyName))
					setVariableField(&field, &VarField{Val: val.Value, Tag: val.Tag})
				default:
					return TemplateConfig{}, fmt.Errorf("unknown tag %s %s in files", val.Tag, val.Value)
				}
			default:
				return TemplateConfig{}, fmt.Errorf("unknown variable value type in files [%s]", val)
			}
		} else {
			return TemplateConfig{}, fmt.Errorf("unknown variable key type in files [%s]", k)
		}
	}
	return config, nil
}

// --utility functions
func validatePrompt(pattern string, allowEmpty bool) func(val interface{}) error {
	return func(val interface{}) error {
		// if empty value is not allowed, check for any value
		if !allowEmpty {
			err := survey.Required(val)
			if err != nil {
				return err
			}
		}

		if pattern != "" {
			// the reflect value of the result
			value := reflect.ValueOf(val)

			match, err := regexp.MatchString("^"+pattern+"$", value.String())
			if err != nil {
				return err
			}
			if !match {
				return fmt.Errorf("Value should match pattern %s", pattern)
			}
		}
		return nil
	}
}

func validateFilePath() func(val interface{}) error {
	return func(val interface{}) error {
		err := survey.Required(val)
		if err != nil {
			return err
		}
		filePath := val.(string)

		if filePath != "" {
			info, err := os.Stat(filePath)
			if err != nil {
				util.Verbose("[input] error in file stat: %s\n", err.Error())
				return fmt.Errorf("file not found on path %s", filePath)
			}
			if info.IsDir() {
				return fmt.Errorf("given path is a directory, file path is needed")
			}
		}
		return nil
	}
}

func skipQuestionOnCondition(currentVar *Variable, dependsOnVal string, dependsOn bool, dataMap *PreparedData, defaultVal interface{}, condition bool) bool {
	if dependsOn == condition {
		saveItemToTemplateDataMap(currentVar, dataMap, defaultVal)
		util.Verbose("[dataPrep] Skipping question for parameter [%s] because DependsOn [%s] value is %t\n", currentVar.Name.Val, dependsOnVal, condition)
		return true
	}
	return false
}

func prepareQuestionText(desc string, fallbackQuestion string) string {
	if desc != "" {
		return desc
	}
	return fallbackQuestion
}

func findVariableByName(variables *[]Variable, name string) (*Variable, error) {
	for _, variable := range *variables {
		if variable.Name.Val == name {
			return &variable, nil
		}
	}
	return nil, fmt.Errorf("no variable found in list by name [%s]", name)
}

func saveItemToTemplateDataMap(variable *Variable, preparedData *PreparedData, data interface{}) {
	if variable.Secret.Bool == true {
		preparedData.Secrets[variable.Name.Val] = data
		preparedData.TemplateData[variable.Name.Val] = fmt.Sprintf(fmtTagValue, variable.Name.Val)
	} else {
		// Save to values file if switch is ON
		if variable.SaveInXlVals.Bool == true {
			preparedData.Values[variable.Name.Val] = data
		}
		preparedData.TemplateData[variable.Name.Val] = data
	}
}

func processCustomFunction(fnStr string) ([]string, error) {
	// validate function call string (DOMAIN.MODULE(PARAMS...).ATTR|[INDEX])
	util.Verbose("[fn] Calling fn [%s] for getting template variable value\n", fnStr)
	if regExFn.MatchString(fnStr) {
		groups := regExFn.FindStringSubmatch(fnStr)
		if len(groups) != 6 {
			return nil, fmt.Errorf("invalid syntax in function reference: %s", fnStr)
		} else {
			// prepare function parts
			domain := groups[1]
			module := groups[2]
			params := strings.Split(groups[3], ",")
			for i, param := range params {
				params[i] = strings.TrimSpace(param)
			}
			attr := groups[4]
			indexStr := groups[5]
			var index int
			if indexStr == "" {
				index = -1
			} else {
				var atoiErr error
				index, atoiErr = strconv.Atoi(indexStr)
				if atoiErr != nil {
					return nil, atoiErr
				}
			}

			// call related function with params
			switch domain {
			case FnAWS:
				awsResult, err := aws.CallAWSFuncByName(module, params...)
				if err != nil {
					return nil, err
				}
				return awsResult.GetResult(module, attr, index)
			case FnOs:
				osResult, err := osHelper.CallOSFuncByName(module, params...)
				if err != nil {
					return nil, err
				}
				return osResult.GetResult(module, attr, index)
			default:
				return nil, fmt.Errorf("unknown function type: %s", domain)
			}
		}
	} else {
		return nil, fmt.Errorf("invalid syntax in function reference: %s", fnStr)
	}
}

func findInTemplateConfigs(templateConfigs []TemplateConfig, filename string) int {
	for i, config := range templateConfigs {
		if config.File == filename {
			return i
		}
	}
	return -1
}

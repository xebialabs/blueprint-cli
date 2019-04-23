package blueprint

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"reflect"
	"regexp"
	"strconv"
	"strings"

	"github.com/thoas/go-funk"
	"github.com/xebialabs/xl-cli/pkg/cloud/aws"
	"github.com/xebialabs/xl-cli/pkg/cloud/k8s"
	"github.com/xebialabs/xl-cli/pkg/models"
	"github.com/xebialabs/xl-cli/pkg/osHelper"
	"github.com/xebialabs/xl-cli/pkg/util"
	"github.com/xebialabs/yaml"
	"gopkg.in/AlecAivazis/survey.v1"
)

// Constants
const (
	FnAWS = "aws"
	FnK8S = "k8s"
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

type PreparedData struct {
	TemplateData map[string]interface{}
	DefaultData  map[string]interface{}
	Values       map[string]interface{}
	Secrets      map[string]interface{}
}

func NewPreparedData() *PreparedData {
	templateData := make(map[string]interface{})
	defaultData := make(map[string]interface{})
	values := make(map[string]interface{})
	secrets := make(map[string]interface{})
	return &PreparedData{TemplateData: templateData, DefaultData: defaultData, Values: values, Secrets: secrets}
}

// regular Expressions
var regExFn = regexp.MustCompile(`([\w\d]+).([\w\d]+)\(([,/\-:\s\w\d]*)\)(?:\.([\w\d]*)|\[([\d]+)\])*`)

func ParseDependsOnValue(varField VarField, variables *[]Variable, parameters map[string]interface{}) (bool, error) {
	tagVal := varField.Tag
	fieldVal := varField.Val
	switch tagVal {
	case tagFn:
		values, err := ProcessCustomFunction(fieldVal)
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
		if varField.InvertBool {
			return !dependsOnVal, nil
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
			if varField.InvertBool {
				return !dependsOnVal, nil
			}
			return dependsOnVal, nil
		}
		return false, fmt.Errorf("Expression [%s] result is invalid for a boolean field", fieldVal)
	}
	dependsOnVar, err := findVariableByName(variables, fieldVal)
	if err != nil {
		return false, err
	}
	if varField.InvertBool {
		return !dependsOnVar.Value.Bool, nil
	}
	return dependsOnVar.Value.Bool, nil
}

// GetDefaultVal variable struct functions
func (variable *Variable) GetDefaultVal(variables map[string]interface{}) interface{} {
	defaultVal := variable.Default.Val
	switch variable.Default.Tag {
	case tagFn:
		values, err := ProcessCustomFunction(defaultVal)
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

	return defaultVal
}

func (variable *Variable) GetValueFieldVal(parameters map[string]interface{}) interface{} {
	switch variable.Value.Tag {
	case tagFn:
		values, err := ProcessCustomFunction(variable.Value.Val)
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
			return value
		}
	}
	return variable.Value.Val
}

func (variable *Variable) GetOptions(parameters map[string]interface{}) []string {
	var options []string
	for _, option := range variable.Options {
		switch option.Tag {
		case tagFn:
			opts, err := ProcessCustomFunction(option.Val)
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

// Get variable validate expression
func (variable *Variable) GetValidateExpr() (string, error) {
	if variable.Validate.Val == "" {
		return "", nil
	}

	switch variable.Validate.Tag {
	case tagExpression:
		return variable.Validate.Val, nil
	}
	return "", fmt.Errorf("only '!expression' tag is supported for validate attribute")
}

func (variable *Variable) VerifyVariableValue(value interface{}, parameters map[string]interface{}) (interface{}, error) {
	// get validate expression
	validateExpr, err := variable.GetValidateExpr()
	if err != nil {
		return nil, fmt.Errorf("error getting validation expression: %s", err.Error())
	}

	// specific conversions by type if needed
	switch variable.Type.Val {
	case TypeConfirm:
		var answerBool bool
		var err error
		switch value.(type) {
		case string:
			answerBool, err = strconv.ParseBool(value.(string))
			if err != nil {
				return nil, err
			}
			break
		case bool:
			answerBool = value.(bool)
			break
		default:
			return nil, fmt.Errorf("type of value [%v] is not supported", value)
		}

		variable.Value.Bool = answerBool
		return answerBool, nil
	case TypeSelect:
		// check if answer is one of the options, error if not
		options := variable.GetOptions(parameters)
		answerStr := fmt.Sprintf("%v", value)
		if !funk.Contains(options, answerStr) {
			return "", fmt.Errorf("answer [%s] is not one of the available options %v for variable [%s]", answerStr, options, variable.Name.Val)
		}
		return answerStr, nil
	case TypeFile:
		// read file contents
		filePath := value.(string)
		util.Verbose("[input] Reading file contents from path: %s\n", filePath)
		data, err := ioutil.ReadFile(filePath)
		if err != nil {
			return "", fmt.Errorf("error reading input file [%s]: %s", filePath, err.Error())
		}
		return string(data), nil
	default:
		// do pattern validation if needed
		if variable.Pattern.Val != "" {
			allowEmpty := false
			if variable.Type.Val == TypeInput && variable.Secret.Bool {
				allowEmpty = true
			}
			validationErr := validatePrompt(variable.Name.Val, validateExpr, variable.Pattern.Val, allowEmpty, parameters)(value)
			if validationErr != nil {
				return nil, fmt.Errorf("validation error for answer value [%v] for variable [%s]: %s", value, variable.Name.Val, validationErr.Error())
			}
		}
		return value, nil
	}
}

func (variable *Variable) GetUserInput(defaultVal interface{}, parameters map[string]interface{}, surveyOpts ...survey.AskOpt) (interface{}, error) {
	var answer string
	var err error
	defaultValStr := fmt.Sprintf("%v", defaultVal)

	// get validate expression
	validateExpr, err := variable.GetValidateExpr()
	if err != nil {
		return nil, fmt.Errorf("error getting validation expression: %s", err.Error())
	}

	switch variable.Type.Val {
	case TypeInput:
		if variable.Secret.Bool {
			questionMsg := prepareQuestionText(variable.Description.Val, fmt.Sprintf("What is the value of %s?", variable.Name.Val))
			if defaultVal != "" {
				questionMsg += fmt.Sprintf(" (%s)", defaultVal)
			}
			err = survey.AskOne(
				&survey.Password{Message: questionMsg},
				&answer,
				validatePrompt(variable.Name.Val, validateExpr, variable.Pattern.Val, true, parameters),
				surveyOpts...,
			)

			// if user bypassed question, replace with default value
			if answer == "" {
				util.Verbose("[input] Got empty response for secret field '%s', replacing with default value: %s\n", variable.Name.Val, defaultVal)
				answer = defaultValStr
			}
		} else {
			err = survey.AskOne(
				&survey.Input{
					Message: prepareQuestionText(variable.Description.Val, fmt.Sprintf("What is the value of %s?", variable.Name.Val)),
					Default: defaultValStr,
				},
				&answer,
				validatePrompt(variable.Name.Val, validateExpr, variable.Pattern.Val, false, parameters),
				surveyOpts...,
			)
		}
	case TypeEditor:
		err = survey.AskOne(
			&survey.Editor{
				Message:       prepareQuestionText(variable.Description.Val, fmt.Sprintf("What is the value of %s?", variable.Name.Val)),
				Default:       defaultValStr,
				HideDefault:   true,
				AppendDefault: true,
			},
			&answer,
			validatePrompt(variable.Name.Val, validateExpr, variable.Pattern.Val, false, parameters),
			surveyOpts...,
		)
	case TypeFile:
		var filePath string
		err = survey.AskOne(
			&survey.Input{
				Message: prepareQuestionText(variable.Description.Val, fmt.Sprintf("What is the file path (relative/absolute) for %s?", variable.Name.Val)),
				Default: defaultValStr,
			},
			&filePath,
			validateFilePath(),
			surveyOpts...,
		)

		// read file contents & save as answer
		util.Verbose("[input] Reading file contents from path: %s\n", filePath)
		data, err := getFileContents(filePath)
		if err != nil {
			return "", err
		}
		answer = string(data)
	case TypeSelect:
		options := variable.GetOptions(parameters)
		err = survey.AskOne(
			&survey.Select{
				Message:  prepareQuestionText(variable.Description.Val, fmt.Sprintf("Select value for %s?", variable.Name.Val)),
				Options:  options,
				Default:  defaultValStr,
				PageSize: 10,
			},
			&answer,
			validatePrompt(variable.Name.Val, validateExpr, variable.Pattern.Val, false, parameters),
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
			validatePrompt(variable.Name.Val, validateExpr, variable.Pattern.Val, false, parameters),
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
func parseTemplateMetadata(blueprintVars *[]byte, templatePath string, blueprintRepository *BlueprintContext, isLocal bool) (*BlueprintConfig, error) {
	decoder := yaml.NewDecoder(bytes.NewReader(*blueprintVars))
	decoder.SetStrict(true)
	yamlDoc := BlueprintYaml{}
	err := decoder.Decode(&yamlDoc)
	if err != nil {
		return nil, err
	}

	// parse & validate
	variables, err := yamlDoc.parseParameters()
	if err != nil {
		return nil, err
	}
	templateConfigs, err := yamlDoc.parseFiles(templatePath, isLocal)
	if err != nil {
		return nil, err
	}
	included, err := yamlDoc.parseIncludes()
	if err != nil {
		return nil, err
	}
	blueprintConfig := BlueprintConfig{
		ApiVersion:      yamlDoc.ApiVersion,
		Kind:            yamlDoc.Kind,
		Metadata:        yamlDoc.Metadata,
		Include:         included,
		TemplateConfigs: templateConfigs,
		Variables:       variables,
	}
	err = blueprintConfig.validate()
	return &blueprintConfig, err
}

// parse doc parameters into list of variables
func (blueprintDoc *BlueprintYaml) parseParameters() ([]Variable, error) {
	parameters := []Parameter{}
	variables := []Variable{}
	if blueprintDoc.Spec.Parameters != nil {
		parameters = blueprintDoc.Spec.Parameters
	} else {
		// for backward compatibility with v8.5
		parameters = blueprintDoc.Parameters
	}
	for _, m := range parameters {
		parsedVar, err := parseParameter(&m)
		if err != nil {
			return variables, err
		}
		variables = append(variables, parsedVar)
	}
	return variables, nil
}

// parse doc files into list of TemplateConfig
func (blueprintDoc *BlueprintYaml) parseFiles(templatePath string, isLocal bool) ([]TemplateConfig, error) {
	files := []File{}
	templateConfigs := []TemplateConfig{}
	if blueprintDoc.Spec.Files != nil {
		files = blueprintDoc.Spec.Files
	} else {
		// for backward compatibility with v8.5
		files = blueprintDoc.Files
	}
	for _, m := range files {
		templateConfig, err := parseFile(&m)
		if err != nil {
			return nil, err
		}
		if isLocal {
			// If local mode, fix path separator in needed cases
			adjustedPath := AdjustPathSeperatorIfNeeded(templateConfig.Path)
			templateConfig.Path = adjustedPath
			templateConfig.FullPath = filepath.Join(templatePath, adjustedPath)
		}
		templateConfigs = append(templateConfigs, templateConfig)
	}
	return templateConfigs, nil
}

// parse doc files into list of TemplateConfig
func (blueprintDoc *BlueprintYaml) parseIncludes() ([]IncludedBlueprintProcessed, error) {
	processedIncludes := []IncludedBlueprintProcessed{}
	for _, m := range blueprintDoc.Spec.Include {
		include, err := parseInclude(&m)
		if err != nil {
			return nil, err
		}
		processedIncludes = append(processedIncludes, include)
	}
	return processedIncludes, nil
}

func parseParameter(m *Parameter) (Variable, error) {
	parsedVar := Variable{}
	err := parseFieldsFromStruct(m, func() reflect.Value {
		return reflect.ValueOf(&parsedVar).Elem()
	})
	return parsedVar, err
}

func parseFile(m *File) (TemplateConfig, error) {
	parsedConfig := TemplateConfig{}
	err := parseFieldsFromStruct(m, func() reflect.Value {
		return reflect.ValueOf(&parsedConfig).Elem()
	})
	return parsedConfig, err
}

func parseInclude(m *IncludedBlueprint) (IncludedBlueprintProcessed, error) {
	parsedInclude := IncludedBlueprintProcessed{}
	err := parseFieldsFromStruct(m, func() reflect.Value {
		return reflect.ValueOf(&parsedInclude).Elem()
	})
	return parsedInclude, err
}

func parseFieldsFromStruct(original interface{}, getFieldByReflect func() reflect.Value) error {
	parameterR := reflect.ValueOf(original).Elem()
	typeOfT := parameterR.Type()
	// iterate over the struct fields and map them
	for i := 0; i < parameterR.NumField(); i++ {
		fieldR := parameterR.Field(i)
		fieldName := typeOfT.Field(i).Name
		value := fieldR.Interface()
		// for backward compatibility
		invertBool := false
		if (strings.ToLower(fieldName) == "dependsontrue" || strings.ToLower(fieldName) == "dependsonfalse") && value != nil {
			invertBool = strings.ToLower(fieldName) == "dependsonfalse"
			fieldName = "DependsOn"
		}
		field := getFieldByReflect().FieldByName(strings.Title(fieldName))
		switch val := value.(type) {
		case string:
			// Set string field
			setVariableField(&field, val, VarField{Val: val, InvertBool: invertBool})
		case int, uint, uint8, uint16, uint32, uint64:
			// Set integer field
			setVariableField(&field, fmt.Sprint(val), VarField{Val: fmt.Sprint(val), InvertBool: invertBool})
		case float32, float64:
			// Set float field
			setVariableField(&field, fmt.Sprintf("%f", val), VarField{Val: fmt.Sprintf("%f", val), InvertBool: invertBool})
		case bool:
			// Set boolean field
			setVariableField(&field, strconv.FormatBool(val), VarField{Val: strconv.FormatBool(val), Bool: val, InvertBool: invertBool})
		case []interface{}:
			// Set options array field for Parameters
			if len(val) > 0 {
				field.Set(reflect.MakeSlice(reflect.TypeOf([]VarField{}), len(val), len(val)))
				for i, it := range val {
					switch wVal := it.(type) {
					case int, uint, uint8, uint16, uint32, uint64:
						field.Index(i).Set(reflect.ValueOf(VarField{Val: fmt.Sprint(wVal)}))
					case float32, float64:
						field.Index(i).Set(reflect.ValueOf(VarField{Val: fmt.Sprintf("%f", wVal)}))
					case string:
						field.Index(i).Set(reflect.ValueOf(VarField{Val: wVal}))
					case yaml.CustomTag:
						field.Index(i).Set(reflect.ValueOf(VarField{Val: wVal.Value, Tag: wVal.Tag}))
					default:
						return fmt.Errorf("unknown list item type %s", wVal)
					}
				}
			}
		case []ParameterOverride:
			// Set ParameterOverride array field for Include
			if len(val) > 0 {
				field.Set(reflect.MakeSlice(reflect.TypeOf([]ParameterOverridesProcessed{}), len(val), len(val)))
				for i, it := range val {
					parsed := ParameterOverridesProcessed{}
					err := parseFieldsFromStruct(&it, func() reflect.Value {
						return reflect.ValueOf(&parsed).Elem()
					})
					if err != nil {
						return err
					}
					field.Index(i).Set(reflect.ValueOf(parsed))
				}
			}
		case []File:
			// Set File array field for Include
			if len(val) > 0 {
				field.Set(reflect.MakeSlice(reflect.TypeOf([]TemplateConfig{}), len(val), len(val)))
				for i, it := range val {
					parsed := TemplateConfig{}
					err := parseFieldsFromStruct(&it, func() reflect.Value {
						return reflect.ValueOf(&parsed).Elem()
					})
					if err != nil {
						return err
					}
					field.Index(i).Set(reflect.ValueOf(parsed))
				}
			}
		case yaml.CustomTag:
			// Set string field with YAML tag
			switch val.Tag {
			case tagFn, tagExpression:
				setVariableField(&field, val.Value, VarField{Val: val.Value, Tag: val.Tag, InvertBool: invertBool})
			default:
				return fmt.Errorf("unknown tag %s %s", val.Tag, val.Value)
			}
		case nil:
			// do nothing when field is not set
		default:
			return fmt.Errorf("unknown variable type [%s]", val)
		}
	}
	return nil
}

func setVariableField(field *reflect.Value, val interface{}, varField VarField) {
	if field.IsValid() && field.CanInterface() {
		switch field.Interface().(type) {
		case string:
			field.Set(reflect.ValueOf(val))
		case VarField:
			field.Set(reflect.ValueOf(varField))
		}
	}
}

// verify blueprint directory & generate full paths for local files
func (blueprintDoc *BlueprintConfig) verifyTemplateDirAndPaths(templatePath string) error {
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

		// verify full local paths
		for _, config := range blueprintDoc.TemplateConfigs {
			configIndex := findInFilePaths(config, filePaths)
			if configIndex == -1 {
				return fmt.Errorf("path [%s] doesn't exist", config.FullPath)
			}

		}
		return nil
	}
	return fmt.Errorf("path [%s] doesn't exist", templatePath)
}

// validate blueprint yaml document based on required fields
func (blueprintDoc *BlueprintConfig) validate() error {
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
func (blueprintDoc *BlueprintConfig) prepareTemplateData(answersFilePath string, strictAnswers bool, useDefaultsAsValue bool, surveyOpts ...survey.AskOpt) (*PreparedData, error) {
	data := NewPreparedData()

	// if exists, get map of answers from file
	var answerMap map[string]interface{}
	var err error
	usingAnswersFile := false
	if answersFilePath != "" {
		// parse answers file
		util.Verbose("[dataPrep] Using answers file [%s] (strict: %t) instead of asking questions from console\n", answersFilePath, strictAnswers)
		answerMap, err = getValuesFromAnswersFile(answersFilePath)
		if err != nil {
			return nil, err
		}

		// skip final prompt if in strict answers mode
		if strictAnswers {
			SkipFinalPrompt = true
		}
		usingAnswersFile = true
	}

	// for every variable defined in blueprint.yaml file
	for i, variable := range blueprintDoc.Variables {
		// process default field value
		defaultVal := variable.GetDefaultVal(data.TemplateData)

		// skip question based on DependsOn fields
		if !util.IsStringEmpty(variable.DependsOn.Val) {
			dependsOnVal, err := ParseDependsOnValue(variable.DependsOn, &blueprintDoc.Variables, data.TemplateData)
			if err != nil {
				return nil, err
			}
			if skipQuestionOnCondition(&variable, variable.DependsOn.Val, dependsOnVal, data, defaultVal, variable.DependsOn.InvertBool) {
				continue
			}
		}
		// skip user input if value field is present
		if variable.Value.Val != "" {
			parsedVal := variable.GetValueFieldVal(data.TemplateData)

			// check if resulting value is non-empty
			if parsedVal != nil && parsedVal != "" {
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

		// skip user input if it is in default mode and default value is present
		if useDefaultsAsValue && defaultVal != nil && defaultVal != "" {
			finalVal, err := variable.VerifyVariableValue(defaultVal, data.TemplateData)
			if err != nil {
				return nil, err
			}

			util.Verbose(
				"[dataPrep] Use Defaults as Value mode: Skipping question for parameter [%s] because default value [%v] is present\n",
				variable.Name.Val,
				finalVal,
			)
			if variable.Type.Val == TypeConfirm {
				blueprintDoc.Variables[i] = variable
			}
			saveItemToTemplateDataMap(&variable, data, finalVal)
			if variable.Secret.Bool && !variable.ShowValueOnSummary.Bool {
				data.DefaultData[variable.Name.Val] = "*****"
			} else {
				data.DefaultData[variable.Name.Val] = finalVal
			}
			continue
		}

		// check answers file for variable value, if exists
		if usingAnswersFile {
			if util.MapContainsKeyWithValInterface(answerMap, variable.Name.Val) {
				answer, err := variable.VerifyVariableValue(answerMap[variable.Name.Val], data.TemplateData)
				if err != nil {
					return nil, err
				}

				// if we have a valid answer, skip user input
				if variable.Type.Val == TypeConfirm {
					blueprintDoc.Variables[i] = variable
				}
				saveItemToTemplateDataMap(&variable, data, answer)
				util.Info("[dataPrep] Using answer file value [%v] for variable [%s]\n", answer, variable.Name.Val)
				continue
			} else {
				if strictAnswers {
					return nil, fmt.Errorf("variable with name [%s] could not be found in answers file", variable.Name.Val)
				} // do not return error when in non-strict answers mode, instead ask user input for the variable value
			}
		}

		// ask question based on type to get value - on the following conditions in order
		// * if dependsOn fields exists, they have boolean result TRUE
		// * if value field is not present
		// * if not in default mode and default value is present
		// * if answers file is not present or isPartial is set to TRUE and answer not found on file for the variable
		util.Verbose("[dataPrep] Processing template variable [Name: %s, Type: %s]\n", variable.Name.Val, variable.Type.Val)
		var answer interface{}
		if !SkipUserInput {
			answer, err = variable.GetUserInput(defaultVal, data.TemplateData, surveyOpts...)
		}
		if err != nil {
			return nil, err
		}
		if variable.Type.Val == TypeConfirm {
			blueprintDoc.Variables[i] = variable
		}
		saveItemToTemplateDataMap(&variable, data, answer)
	}

	if useDefaultsAsValue {
		// Print summary default values table if in useDefaultsAsValues mode
		// use util.Print so that this is not skipped in quiet mode
		util.Print("Using default values:\n")
		util.Print(util.DataMapTable(&data.DefaultData, util.TableAlignLeft, 30, 50, "\t"))
	}

	return data, nil
}

// get values from answers file
func getValuesFromAnswersFile(answersFilePath string) (map[string]interface{}, error) {
	if util.PathExists(answersFilePath, false) {
		// read file contents
		content, err := ioutil.ReadFile(answersFilePath)
		if err != nil {
			return nil, err
		}

		// parse answers file
		answers := make(map[string]interface{})
		err = yaml.Unmarshal(content, answers)
		if err != nil {
			return nil, err
		}
		return answers, nil
	}
	return nil, fmt.Errorf("blueprint answers file not found in path %s", answersFilePath)
}

// utility functions
func getFileContents(filepath string) (string, error) {
	data, err := ioutil.ReadFile(filepath)
	if err != nil {
		return "", err
	}
	return string(data), nil
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
		if util.IsStringEmpty(file.Path) {
			return fmt.Errorf("path is missing for file specification in files")
		}
		if filepath.IsAbs(file.Path) || strings.HasPrefix(file.Path, "..") || strings.HasPrefix(file.Path, "."+string(os.PathSeparator)) {
			return fmt.Errorf("path for file specification cannot start with /, .. or ./")
		}
	}
	return nil
}

func validatePrompt(varName string, validateExpr string, pattern string, allowEmpty bool, parameters map[string]interface{}) func(val interface{}) error {
	return func(val interface{}) error {
		// if empty value is not allowed, check for any value
		if !allowEmpty {
			err := survey.Required(val)
			if err != nil {
				return err
			}
		}

		// do pattern validation - TODO: to be removed after v9.0
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

		// run validation function
		if validateExpr != "" {
			// add this value to the map of parameters for expression
			if varName != "" {
				parameters[varName] = val
			}
			isSuccess, err := ProcessCustomExpression(validateExpr, parameters)
			if err != nil {
				return err
			}
			if !isSuccess.(bool) {
				return fmt.Errorf("validation failed for field [%s] with value [%s] and expression [%s]", varName, val, validateExpr)
			}
			return nil
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
		// return false if this is a skipped confirm question
		if defaultVal == "" && currentVar.Type.Val == TypeConfirm {
			defaultVal = false
		}

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
	if variable.Secret.Bool {
		preparedData.Secrets[variable.Name.Val] = data
		// Use raw value of secret field if flag is set
		if variable.UseRawValue.Bool {
			preparedData.TemplateData[variable.Name.Val] = data
		} else {
			preparedData.TemplateData[variable.Name.Val] = fmt.Sprintf(fmtTagValue, variable.Name.Val)
		}
	} else {
		// Save to values file if switch is ON
		if variable.SaveInXlVals.Bool {
			preparedData.Values[variable.Name.Val] = data
		}
		preparedData.TemplateData[variable.Name.Val] = data
	}
}

func ProcessCustomFunction(fnStr string) ([]string, error) {
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
			case FnK8S:
				k8sResult, err := k8s.CallK8SFuncByName(module, params...)
				if err != nil {
					return nil, err
				}
				return k8sResult.GetResult(module, attr, index)
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

func findInFilePaths(templateConfig TemplateConfig, filePaths []string) int {
	for i, filepath := range filePaths {
		if templateConfig.FullPath == filepath {
			return i
		}
	}
	return -1
}

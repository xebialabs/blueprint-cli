package blueprint

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"reflect"
	"regexp"
	"strconv"
	"strings"

	"github.com/AlecAivazis/survey/v2"
	"github.com/AlecAivazis/survey/v2/core"
	"github.com/thoas/go-funk"

	"github.com/xebialabs/blueprint-cli/pkg/cloud/aws"
	"github.com/xebialabs/blueprint-cli/pkg/cloud/k8s"
	"github.com/xebialabs/blueprint-cli/pkg/models"
	"github.com/xebialabs/blueprint-cli/pkg/util"
	"github.com/xebialabs/yaml"
)

// Constants
const (
	FnAWS = "aws"
	FnK8S = "k8s"
	FnOs  = "os"

	tagFnV1          = "!fn"
	tagExpressionV1  = "!expression"
	tagExpressionV2  = "!expr"
	fmtTagValue      = "!value %s"
	optionTextFormat = "%s [%s]"
)

// InputType constants
const (
	TypeInput        = "Input"
	TypeEditor       = "Editor"
	TypeFile         = "File"
	TypeSelect       = "Select"
	TypeConfirm      = "Confirm"
	TypeSecret       = "SecretInput"
	TypeSecretEditor = "SecretEditor"
	TypeSecretFile   = "SecretFile"
)

var validTypes = []string{TypeInput, TypeEditor, TypeFile, TypeSelect, TypeConfirm, TypeSecret, TypeSecretEditor, TypeSecretFile}

type PreparedData struct {
	// Storing values for all fields
	TemplateData map[string]interface{}
	// Used to store data to be shown in summary table
	SummaryData map[string]interface{}
	// Used to store data to be saved in values.xlvals
	Values map[string]interface{}
	// Used to store data to be saved in secrets.xlvals
	Secrets map[string]interface{}
}

func NewPreparedData() *PreparedData {
	templateData := make(map[string]interface{})
	summaryData := make(map[string]interface{})
	values := make(map[string]interface{})
	secrets := make(map[string]interface{})
	return &PreparedData{TemplateData: templateData, SummaryData: summaryData, Values: values, Secrets: secrets}
}

// regular Expressions
var regExFn = regexp.MustCompile(`([\w\d]+).([\w\d]+)\(([,/\-:\s\w\d]*)\)(?:\.([\w\d]*)|\[([\d]+)\])*`)

func GetProcessedExpressionValue(val VarField, parameters map[string]interface{}, overrideFns ExpressionOverrideFn) (VarField, error) {
	switch val.Tag {
	case tagExpressionV1, tagExpressionV2:
		procVal, err := ProcessCustomExpression(val.Value, parameters, overrideFns)
		if err != nil {
			return val, err
		}
		util.Verbose("[expression] Processed value of expression [%s] is: %s\n", val.Value, procVal)
		switch finalVal := procVal.(type) {
		case string:
			val.Value = finalVal
			break
		case bool:
			val.Value = strconv.FormatBool(finalVal)
			val.Bool = finalVal
			break
		case nil:
			val.Value = ""
		case float32, float64:
			val.Value = fmt.Sprintf("%g", finalVal)
			if !strings.Contains(val.Value, ".") {
				val.Value = fmt.Sprintf("%s.0", val.Value)
			}
		}
		return val, nil
	}
	return val, nil
}

func (variable *Variable) ProcessExpression(parameters map[string]interface{}, overrideFns ExpressionOverrideFn) error {
	fieldsToSkip := []string{"Validate", "Options"} // these fields have special processing
	return ProcessExpressionField(variable, fieldsToSkip, parameters, variable.Name.Value, overrideFns)
}

func ProcessExpressionField(item interface{}, fieldsToSkip []string, parameters map[string]interface{}, id string, overrideFns ExpressionOverrideFn) error {
	itemR := reflect.ValueOf(item).Elem()
	typeOfT := itemR.Type()
	// iterate over the struct fields and map them
	for i := 0; i < itemR.NumField(); i++ {
		fieldR := itemR.Field(i)
		fieldName := typeOfT.Field(i).Name
		value := fieldR.Interface()
		field := reflect.ValueOf(item).Elem().FieldByName(strings.Title(fieldName))
		if !util.IsStringInSlice(fieldName, fieldsToSkip) && field.IsValid() {
			switch val := value.(type) {
			case VarField:
				procVal, err := GetProcessedExpressionValue(val, parameters, overrideFns)
				if err != nil {
					return fmt.Errorf("error while processing !expr [%s] for [%s] of [%s]. %s", val.Value, fieldName, id, err.Error())
				}
				field.Set(reflect.ValueOf(procVal))
			}
		}
	}
	return nil
}

// GetDefaultVal variable struct functions
func (variable *Variable) GetDefaultVal() interface{} {
	defaultVal := variable.Default.Value
	switch variable.Default.Tag {
	case tagFnV1:
		values, err := ProcessCustomFunction(defaultVal)
		if err != nil {
			util.Info("Error while processing default value !fn [%s] for [%s]. %s", defaultVal, variable.Name.Value, err.Error())
			defaultVal = ""
		} else {
			util.Verbose("[fn] Processed value of function [%s] is: %s\n", defaultVal, values[0])
			if variable.Type.Value == TypeConfirm {
				boolVal, err := strconv.ParseBool(values[0])
				if err != nil {
					util.Info("Error while processing default value !fn [%s] for [%s]. %s", defaultVal, variable.Name.Value, err.Error())
					return false
				}
				variable.Default.Bool = boolVal
				return boolVal
			}
			return values[0]
		}
	}

	return defaultVal
}

func (variable *Variable) GetValueFieldVal() interface{} {
	switch variable.Value.Tag {
	case tagFnV1:
		values, err := ProcessCustomFunction(variable.Value.Value)
		if err != nil {
			util.Info("Error while processing !fn [%s]. Please update the value for [%s] manually. %s", variable.Value.Value, variable.Name.Value, err.Error())
			return ""
		}
		util.Verbose("[fn] Processed value of function [%s] is: %s\n", variable.Value.Value, values[0])
		if variable.Type.Value == TypeConfirm {
			boolVal, err := strconv.ParseBool(values[0])
			if err != nil {
				util.Info("Error while processing !fn [%s]. Please update the value for [%s] manually. %s", variable.Value.Value, variable.Name.Value, err.Error())
				return ""
			}
			variable.Value.Bool = boolVal
			return values[0]
		}
		return values[0]
	}
	return variable.Value.Value
}

func (variable *Variable) GetOptions(parameters map[string]interface{}, withLabel bool, overrideFns ExpressionOverrideFn) []string {
	options := []string{}
	for _, option := range variable.Options {
		switch option.Tag {
		case tagFnV1:
			opts, err := ProcessCustomFunction(option.Value)
			if err != nil {
				util.Info("Error while processing !fn [%s]. Please update the value for [%s] manually. %s\n", option.Value, variable.Name.Value, err.Error())
				return options
			}
			util.Verbose("[fn] Processed value of function [%s] is: %s\n", option.Value, opts)
			options = append(options, opts...)
		case tagExpressionV1, tagExpressionV2:
			opts, err := ProcessCustomExpression(option.Value, parameters, overrideFns)
			if err != nil {
				util.Info("Error while processing !expr [%s]. Please update the value for [%s] manually. %s\n", option.Value, variable.Name.Value, err.Error())
				return options
			}
			switch val := opts.(type) {
			case []string:
				util.Verbose("[expression] Processed value of expression [%s] is: %v\n", option.Value, val)
				if len(val) == 0 {
					util.Info("Empty response while processing !expr [%s]. Please update the value for [%s] manually. %s\n", option.Value, variable.Name.Value, "Empty array returned.")
				} else {
					options = append(options, val...)
				}
			case []interface{}:
				util.Verbose("[expression] Processed value of expression [%s] is: %v\n", option.Value, val)
				if len(val) == 0 {
					util.Info("Empty response while processing !expr [%s]. Please update the value for [%s] manually. %s\n", option.Value, variable.Name.Value, "Empty array returned.")
				} else {
					for _, option := range val {
						options = append(options, fmt.Sprint(option))
					}
				}
			default:
				util.Info("Error while processing !expr [%s]. Please update the value for [%s] manually. %s\n", option.Value, variable.Name.Value, "Return type should be a string array.")
				return options
			}
		default:
			if withLabel {
				options = append(options, getOptionTextWithLabel(option))
			} else {
				options = append(options, option.Value)
			}
		}
	}
	return options
}

func getOptionTextWithLabel(option VarField) string {
	optionText := option.Value
	if option.Label != "" {
		optionText = fmt.Sprintf(optionTextFormat, optionText, option.Label)
	}
	return optionText
}

func getMatchingOptionTextWithLabelIfPresent(optionValue string, options []VarField) string {
	for _, option := range options {
		if option.Label != "" && option.Value == optionValue {
			return getOptionTextWithLabel(option)
		}
	}
	return optionValue
}

func getDefaultTextWithLabel(defVal string, optionVars []VarField, options []string) string {
	if len(optionVars) > 0 && len(options) > 0 {
		if defVal == "" { // when no default is set in blueprints, return first option text as default
			return options[0]
		} else { // when default value set in blueprint, return default value/optiontext if it matches any of options[]
			defValWithOptionalLabel := getMatchingOptionTextWithLabelIfPresent(defVal, optionVars)
			for _, optionText := range options {
				if optionText == defValWithOptionalLabel {
					return defValWithOptionalLabel
				}
			}
		}
	}
	// return default value itself, when options do not match or no options present
	return defVal
}

func findLabelValueFromOptions(val string, options []VarField) string {
	for _, o := range options {
		if getOptionTextWithLabel(o) == val {
			return o.Value
		}
	}
	return val
}

// Get variable validate expression
func (variable *Variable) GetValidateExpr() (string, error) {
	if variable.Validate.Value == "" {
		return "", nil
	}

	switch variable.Validate.Tag {
	case tagExpressionV1, tagExpressionV2:
		return variable.Validate.Value, nil
	}
	return "", fmt.Errorf("only '!expr' tag is supported for validate attribute")
}

func validateField(validateExpr string, variable *Variable, parameters map[string]interface{}, value interface{}, overrideFns ExpressionOverrideFn) error {
	if validateExpr != "" {
		allowEmpty := false
		if IsSecretType(variable.Type.Value) || variable.AllowEmpty.Bool {
			allowEmpty = true
		}
		validationErr := validatePrompt(variable.Name.Value, validateExpr, allowEmpty, parameters, overrideFns)(value)
		if validationErr != nil {
			return fmt.Errorf("validation error for answer value [%v] for variable [%s]: %s", value, variable.Name.Value, validationErr.Error())
		}
	}
	return nil
}

func (variable *Variable) VerifyVariableValue(value interface{}, parameters map[string]interface{}, overrideFns ExpressionOverrideFn) (interface{}, error) {
	// get validate expression
	validateExpr, err := variable.GetValidateExpr()
	if err != nil {
		return nil, fmt.Errorf("error getting validation expression: %s", err.Error())
	}

	// specific conversions by type if needed
	switch variable.Type.Value {
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
		options := variable.GetOptions(parameters, false, overrideFns)
		util.Verbose("[input] Select options verify for %s: \n%+v\n", variable.Name.Value, options)
		answerStr := fmt.Sprintf("%v", value)
		if !funk.Contains(options, answerStr) {
			return "", fmt.Errorf("answer [%s] is not one of the available options %v for variable [%s]", answerStr, options, variable.Name.Value)
		}
		return answerStr, nil
	case TypeFile, TypeSecretFile:
		// do validation if needed
		err := validateField(validateExpr, variable, parameters, value, overrideFns)
		if err != nil {
			return "", err
		}
		// read file contents
		filePath := value.(string)
		util.Verbose("[input] File path %s\n", filePath)
		data, err := ioutil.ReadFile(filePath)
		if err != nil {
			return "", fmt.Errorf("error reading input file [%s]: %s", filePath, err.Error())
		}
		return string(data), nil
	default:
		// do validation if needed
		err := validateField(validateExpr, variable, parameters, value, overrideFns)
		if err != nil {
			return "", err
		}
		return value, nil
	}
}

func (variable *Variable) GetHelpText() string {
	if variable.Description.Value != "" && variable.Description.Value != variable.Prompt.Value {
		return variable.Description.Value
	}
	return ""
}

func (variable *Variable) GetUserInput(defaultVal interface{}, parameters map[string]interface{}, overrideFns ExpressionOverrideFn, surveyOpts ...survey.AskOpt) (interface{}, error) {
	var answer string
	var err error
	defaultValStr := fmt.Sprintf("%v", defaultVal)

	// get validate expression
	validateExpr, err := variable.GetValidateExpr()
	if err != nil {
		return nil, fmt.Errorf("error getting validation expression: %s", err.Error())
	}

	switch variable.Type.Value {
	case TypeInput:
		surveyOpts = append(surveyOpts, survey.WithValidator(validatePrompt(variable.Name.Value, validateExpr, variable.AllowEmpty.Bool, parameters, overrideFns)))
		err = survey.AskOne(
			&survey.Input{
				Message: prepareQuestionText(variable.Prompt.Value, fmt.Sprintf("What is the value of %s?", variable.Name.Value)),
				Default: defaultValStr,
				Help:    variable.GetHelpText(),
			},
			&answer,
			surveyOpts...,
		)
	case TypeSecret:
		questionMsg := prepareQuestionText(variable.Prompt.Value, fmt.Sprintf("What is the value of %s?", variable.Name.Value))
		if defaultVal != "" {
			questionMsg += fmt.Sprintf(" (%s)", defaultVal)
		}
		surveyOpts = append(surveyOpts, survey.WithValidator(validatePrompt(variable.Name.Value, validateExpr, true, parameters, overrideFns)))
		err = survey.AskOne(
			&survey.Password{
				Message: questionMsg,
				Help:    variable.GetHelpText(),
			},
			&answer,
			surveyOpts...,
		)

		// if user bypassed question, replace with default value
		if answer == "" {
			util.Verbose("[input] Got empty response for secret field '%s', replacing with default value: %s\n", variable.Name.Value, defaultVal)
			answer = defaultValStr
		}
	case TypeEditor, TypeSecretEditor:
		questionMsg := prepareQuestionText(variable.Prompt.Value, fmt.Sprintf("What is the value of %s?", variable.Name.Value))
		surveyOpts = append(surveyOpts, survey.WithValidator(validatePrompt(variable.Name.Value, validateExpr, false, parameters, overrideFns)))
		err = survey.AskOne(
			&survey.Editor{
				Message:       questionMsg,
				Default:       defaultValStr,
				HideDefault:   true,
				AppendDefault: true,
				Help:          variable.GetHelpText(),
			},
			&answer,
			surveyOpts...,
		)
		// if user bypassed question, replace with default value
		if answer == "" {
			util.Verbose("[input] Got empty response for secret field '%s', replacing with default value: %s\n", variable.Name.Value, defaultVal)
			answer = defaultValStr
		}
	case TypeFile, TypeSecretFile:
		var filePath string
		surveyOpts = append(surveyOpts, survey.WithValidator(validateFilePath(variable.Name.Value, validateExpr, false, parameters, overrideFns)))
		err = survey.AskOne(
			&survey.Input{
				Message: prepareQuestionText(variable.Prompt.Value, fmt.Sprintf("What is the file path (relative/absolute) for %s?", variable.Name.Value)),
				Default: defaultValStr,
				Help:    variable.GetHelpText(),
			},
			&filePath,
			surveyOpts...,
		)
		filePath = strings.TrimSpace(filePath)
		// read file contents & save as answer
		util.Verbose("[input] Reading file contents from path: %s\n", filePath)
		data, err := getFileContents(filePath)
		if err != nil {
			return "", err
		}
		answer = string(data)
	case TypeSelect:
		options := variable.GetOptions(parameters, true, overrideFns)
		surveyOpts = append(surveyOpts, survey.WithValidator(validatePrompt(variable.Name.Value, validateExpr, false, parameters, overrideFns)))
		defaultValue := getDefaultTextWithLabel(defaultValStr, variable.Options, options)
		util.Verbose("[input] Select options prompt for %s with default value '%s' \n%+v\n", variable.Name.Value, defaultValue, options)
		err = survey.AskOne(
			&survey.Select{
				Message:  prepareQuestionText(variable.Prompt.Value, fmt.Sprintf("Select value for %s?", variable.Name.Value)),
				Options:  options,
				Default:  defaultValue,
				PageSize: 10,
				Help:     variable.GetHelpText(),
			},
			&answer,
			surveyOpts...,
		)
		if err != nil {
			return nil, fmt.Errorf("error rendering '%s', for the field %s: %s", variable.Prompt.Value, variable.Name.Value, err.Error())
		}
		answer = findLabelValueFromOptions(answer, variable.Options)
	case TypeConfirm:
		var confirm bool
		surveyOpts = append(surveyOpts, survey.WithValidator(validatePrompt(variable.Name.Value, validateExpr, false, parameters, overrideFns)))
		err = survey.AskOne(
			&survey.Confirm{
				Message: prepareQuestionText(variable.Prompt.Value, fmt.Sprintf("%s?", variable.Name.Value)),
				Default: variable.Default.Bool,
				Help:    variable.GetHelpText(),
			},
			&confirm,
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
	return strings.TrimSpace(answer), err
}

// validate blueprint yaml document based on required fields
func (blueprintDoc *BlueprintConfig) validate() error {
	if !util.IsStringInSlice(blueprintDoc.ApiVersion, models.BlueprintYamlFormatSupportedVersions) {
		return fmt.Errorf("api version needs to be %s or %s", models.BlueprintYamlFormatV2, models.BlueprintYamlFormatV1)
	}
	if blueprintDoc.ApiVersion != models.BlueprintYamlFormatV2 {
		util.Info("This blueprint uses a deprecated blueprint.yaml schema for apiVersion %s\n", models.BlueprintYamlFormatV1)
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
func (blueprintDoc *BlueprintConfig) prepareTemplateData(params BlueprintParams, data *PreparedData, overrideFns ExpressionOverrideFn, surveyOpts ...survey.AskOpt) (*PreparedData, error) {
	// if exists, get map of answers from file
	var answerMap map[string]string
	var err error
	usingAnswersFile := false
	if params.AnswersFile != "" || params.AnswersMap != nil {
		if params.AnswersMap != nil {
			util.Verbose("[dataPrep] Using answers map (strict: %t) instead of asking questions from console\n", params.StrictAnswers)
			answerMap = params.AnswersMap
		} else {
			// parse answers file
			util.Verbose("[dataPrep] Using answers file [%s] (strict: %t) instead of asking questions from console\n", params.AnswersFile, params.StrictAnswers)
			answerMap, err = GetValuesFromAnswersFile(params.AnswersFile)
			if err != nil {
				return nil, err
			}
		}

		// skip final prompt if in strict answers mode
		if params.StrictAnswers {
			SkipFinalPrompt = true
		}
		usingAnswersFile = true
	}

	// for every variable defined in blueprint.yaml file
	for i, variable := range blueprintDoc.Variables {
		variable.ProcessExpression(data.TemplateData, overrideFns)
		var defaultVal interface{}
		// override the default value if its passed and if the param is overridable.
		if variable.OverrideDefault.Bool && util.MapContainsKeyWithVal(params.OverrideDefaults, variable.Name.Value) {
			defaultVal = params.OverrideDefaults[variable.Name.Value]
			if variable.Type.Value == TypeConfirm {
				boolVal, err := strconv.ParseBool(params.OverrideDefaults[variable.Name.Value])
				if err != nil {
					util.Info("Error while processing default value !fn [%s] for [%s]. %s", defaultVal, variable.Name.Value, err.Error())
					return nil, err
				}
				variable.Default.Bool = boolVal
			}
			// remove the variable from answers if this is coming from UP command so that question will be asked in the upgrade flow
			if params.FromUpCommand {
				delete(answerMap, variable.Name.Value)
			}
		} else {
			// process default field value
			defaultVal = variable.GetDefaultVal()
		}

		// skip question based on DependsOn fields, the default value if present is set as value
		if !util.IsStringEmpty(variable.DependsOn.Value) {
			dependsOnVal, err := ParseDependsOnValue(variable.DependsOn, data.TemplateData)
			if err != nil {
				return nil, err
			}
			if skipQuestionOnCondition(&variable, variable.DependsOn.Value, dependsOnVal, data, defaultVal, variable.DependsOn.InvertBool) {
				continue
			}
		}
		// skip user input if value field is present
		if variable.Value.Value != "" {
			parsedVal := variable.GetValueFieldVal()

			// check if resulting value is non-empty
			if parsedVal != nil && parsedVal != "" {
				if variable.Type.Value == TypeConfirm {
					saveItemToTemplateDataMap(&variable, data, variable.Value.Bool)
				} else {
					saveItemToTemplateDataMap(&variable, data, parsedVal)
				}
				util.Verbose("[dataPrep] Skipping question for parameter [%s] because value [%s] is present\n", variable.Name.Value, variable.Value.Value)
				continue
			} else {
				util.Verbose("[dataPrep] Parsed value for parameter [%s] is empty, therefore not being skipped\n", variable.Name.Value)
			}
		}

		// check answers file for variable value, if exists
		if usingAnswersFile {
			if util.MapContainsKeyWithVal(answerMap, variable.Name.Value) {
				answer, err := variable.VerifyVariableValue(answerMap[variable.Name.Value], data.TemplateData, overrideFns)
				if err != nil {
					return nil, err
				}

				if variable.Type.Value == TypeConfirm {
					blueprintDoc.Variables[i] = variable
				}
				// if we have a valid answer, save it and skip user input
				saveItemToTemplateDataMap(&variable, data, answer)
				util.Info("[dataPrep] Using answer file value [%v] for variable [%s]\n", answer, variable.Name.Value)
				continue
			}
		}

		// skip user input if it is in default mode and default value is present
		if params.UseDefaultsAsValue && defaultVal != nil && defaultVal != "" {
			finalVal, err := variable.VerifyVariableValue(defaultVal, data.TemplateData, overrideFns)
			if err != nil {
				return nil, err
			}

			util.Verbose(
				"[dataPrep] Use Defaults as Value mode: Skipping question for parameter [%s] because default value [%v] is present\n",
				variable.Name.Value,
				finalVal,
			)
			if variable.Type.Value == TypeConfirm {
				blueprintDoc.Variables[i] = variable
			}
			saveItemToTemplateDataMap(&variable, data, finalVal)
			continue
		}
		// do not return error when in non-strict answers mode, instead ask user input for the variable value
		if usingAnswersFile && params.StrictAnswers && !variable.IgnoreIfSkipped.Bool {
			return nil, fmt.Errorf("variable with name [%s] could not be found in answers file", variable.Name.Value)
		}

		// ask question based on type to get value - on the following conditions in order
		// * if dependsOn fields exists, they have boolean result TRUE
		// * if value field is not present
		// * if not in default mode and default value is present
		// * if answers file is not present or isPartial is set to TRUE and answer not found on file for the variable
		util.Verbose("[dataPrep] Processing template variable [Name: %s, Type: %s]\n", variable.Name.Value, variable.Type.Value)
		var answer interface{}
		if shouldAskForInput(variable) {
			answer, err = variable.GetUserInput(defaultVal, data.TemplateData, overrideFns, surveyOpts...)
		}
		if err != nil {
			return nil, err
		}
		if variable.Type.Value == TypeConfirm {
			blueprintDoc.Variables[i] = variable
		}
		saveItemToTemplateDataMap(&variable, data, answer)
	}

	return data, nil
}

func shouldAskForInput(variable Variable) bool {
	if SkipUserInput {
		return false
	}
	if variable.IgnoreIfSkipped.Bool {
		return variable.Prompt != (VarField{}) && variable.Prompt.Value != ""
	}
	return true
}

// GetValuesFromAnswersFile get values from answers file
func GetValuesFromAnswersFile(answersFilePath string) (map[string]string, error) {
	if util.PathExists(answersFilePath, false) {
		// read file contents
		content, err := ioutil.ReadFile(answersFilePath)
		if err != nil {
			return nil, err
		}

		// parse answers file
		answers := make(map[string]string)
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
		// validate select case
		if userVar.Type.Value == TypeSelect && len(userVar.Options) == 0 {
			return fmt.Errorf("at least one option field is need to be set for parameter [%s]", userVar.Name.Value)
		}

		// validate file case
		if userVar.Type.Value == TypeFile && !util.IsStringEmpty(userVar.Value.Value) {
			return fmt.Errorf("'value' field is not allowed for file input type")
		}

		variableNames = append(variableNames, userVar.Name.Value)
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

func validatePrompt(varName string, validateExpr string, allowEmpty bool, parameters map[string]interface{}, overrideFns ExpressionOverrideFn) func(val interface{}) error {
	return func(val interface{}) error {
		var value interface{}
		switch valType := val.(type) {
		case string:
			value = strings.TrimSpace(valType)
		case core.OptionAnswer:
			value = val.(core.OptionAnswer).Value
			beforeLabel, _, found := strings.Cut(value.(string), "[")
			if found {
				value = strings.TrimSpace(beforeLabel)
			}
		default:
			value = val
		}
		// if empty value is not allowed, check for any value
		if !allowEmpty {
			err := survey.Required(value)
			if err != nil {
				return err
			}
		}

		// run validation function
		if validateExpr != "" {
			// add this value to the map of parameters for expression
			if varName != "" {
				parameters[varName] = value
			}
			isSuccess, err := ProcessCustomExpression(validateExpr, parameters, overrideFns)
			if err != nil {
				return err
			}
			if !isSuccess.(bool) {
				return fmt.Errorf("validation [%s] failed with value [%s]", validateExpr, value)
			}
			return nil
		}

		return nil
	}
}

func validateFilePath(varName string, validateExpr string, allowEmpty bool, parameters map[string]interface{}, overrideFns ExpressionOverrideFn) func(val interface{}) error {
	return func(val interface{}) error {
		err := survey.Required(val)
		if err != nil {
			return err
		}
		validationErr := validatePrompt(varName, validateExpr, allowEmpty, parameters, overrideFns)(val)

		if validationErr != nil {
			return fmt.Errorf("validation error for answer value [%v] for variable [%s]: %s", val, varName, validationErr.Error())
		}

		filePath := strings.TrimSpace(val.(string))

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
		if defaultVal == "" && currentVar.Type.Value == TypeConfirm {
			defaultVal = false
		}
		currentVar.Meta.PromptSkipped = true
		saveItemToTemplateDataMap(currentVar, dataMap, defaultVal)
		util.Verbose("[dataPrep] Skipping question for parameter [%s] because PromptIf [%s] value is %t\n", currentVar.Name.Value, dependsOnVal, condition)
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

func saveItemToTemplateDataMap(variable *Variable, preparedData *PreparedData, data interface{}) {
	skipParam := variable.IgnoreIfSkipped.Bool && (variable.Meta.PromptSkipped || data == nil || data == "")

	switch variable.Type.Value {
	case TypeConfirm:
		if data != nil && (data == "true" || data == true) {
			data = true
		} else {
			data = false
		}
	default:
		if data == nil {
			data = ""
		}
	}

	if IsSecretType(variable.Type.Value) {
		if !skipParam {
			util.Verbose("[dataPrep] Skipping secret parameter [%s] from summary-table/value-files because IgnoreIfSkipped is true and PromptIf is false\n", variable.Name.Value)

			if variable.RevealOnSummary.Bool {
				preparedData.SummaryData[variable.Label.Value] = data
			} else {
				preparedData.SummaryData[variable.Label.Value] = "*****"
			}

			preparedData.Secrets[variable.Name.Value] = data
		}
		// Use raw value of secret field if flag is set
		if variable.ReplaceAsIs.Bool {
			preparedData.TemplateData[variable.Name.Value] = data
		} else {
			preparedData.TemplateData[variable.Name.Value] = fmt.Sprintf(fmtTagValue, variable.Name.Value)
		}
	} else {
		if !skipParam {
			util.Verbose("[dataPrep] Skipping parameter [%s] from summary-table/value-files because IgnoreIfSkipped is true and PromptIf is false\n", variable.Name.Value)

			preparedData.SummaryData[variable.Label.Value] = data

			// Save to values file if switch is ON
			if variable.SaveInXlvals.Bool {
				preparedData.Values[variable.Name.Value] = data
			}
		}

		preparedData.TemplateData[variable.Name.Value] = data
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

				util.Info("Using AWS-SDK is deprecated and will be removed in the future versions. Consider not using this method in future.")

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
			default:
				return nil, fmt.Errorf("unknown function type: %s", domain)
			}
		}
	} else {
		return nil, fmt.Errorf("invalid syntax in function reference: %s", fnStr)
	}
}

package blueprint

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	"github.com/xebialabs/xl-cli/pkg/version"

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
	FnAWS     = "aws"
	FnK8S     = "k8s"
	FnOs      = "os"
	FnVersion = "version"

	tagFnV1         = "!fn"
	tagExpressionV1 = "!expression"
	tagExpressionV2 = "!expr"
	fmtTagValue     = "!value %s"
)

// InputType constants
const (
	TypeInput   = "Input"
	TypeSecret  = "SecretInput"
	TypeEditor  = "Editor"
	TypeFile    = "File"
	TypeSelect  = "Select"
	TypeConfirm = "Confirm"
)

var validTypes = []string{TypeInput, TypeEditor, TypeFile, TypeSelect, TypeConfirm, TypeSecret}

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

// GetDefaultVal variable struct functions
func (variable *Variable) GetDefaultVal(variables map[string]interface{}) interface{} {
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
	case tagExpressionV1, tagExpressionV2:
		value, err := ProcessCustomExpression(defaultVal, variables)
		if err != nil {
			util.Info("Error while processing default value !expr [%s] for [%s]. %s", defaultVal, variable.Name.Value, err.Error())
			defaultVal = ""
		} else {
			util.Verbose("[expression] Processed value of expression [%s] is: %s\n", defaultVal, value)
			boolVal, ok := value.(bool)
			if ok {
				if variable.Type.Value == TypeConfirm {
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
	case tagExpressionV1, tagExpressionV2:
		value, err := ProcessCustomExpression(variable.Value.Value, parameters)
		if err != nil {
			util.Info("Error while processing !expr [%s]. Please update the value for [%s] manually. %s", variable.Value.Value, variable.Name.Value, err.Error())
			return ""
		} else {
			util.Verbose("[expression] Processed value of expression [%s] is: %s\n", variable.Value.Value, value)
			boolVal, ok := value.(bool)
			if ok {
				if variable.Type.Value == TypeConfirm {
					variable.Value.Bool = boolVal
				}
				return fmt.Sprint(boolVal)
			}
			return value
		}
	}
	return variable.Value.Value
}

func (variable *Variable) GetOptions(parameters map[string]interface{}) []string {
	var options []string
	for _, option := range variable.Options {
		switch option.Tag {
		case tagFnV1:
			opts, err := ProcessCustomFunction(option.Value)
			if err != nil {
				util.Info("Error while processing !fn [%s]. Please update the value for [%s] manually. %s", option.Value, variable.Name.Value, err.Error())
				return nil
			}
			util.Verbose("[fn] Processed value of function [%s] is: %s\n", option.Value, opts)
			options = append(options, opts...)
		case tagExpressionV1, tagExpressionV2:
			opts, err := ProcessCustomExpression(option.Value, parameters)
			if err != nil {
				util.Info("Error while processing !expr [%s]. Please update the value for [%s] manually. %s", option.Value, variable.Name.Value, err.Error())
				return nil
			}
			switch val := opts.(type) {
			case []string:
				util.Verbose("[expression] Processed value of expression [%s] is: %v\n", option.Value, val)
				options = append(options, val...)
			case []interface{}:
				util.Verbose("[expression] Processed value of expression [%s] is: %v\n", option.Value, val)
				for _, option := range val {
					options = append(options, fmt.Sprint(option))
				}
			default:
				util.Info("Error while processing !expr [%s]. Please update the value for [%s] manually. %s", option.Value, variable.Name.Value, "Return type should be a string array")
				return nil
			}
		default:
			optionText := option.Value
			if option.Label != "" {
				optionText = fmt.Sprintf("%s (%s)", option.Label, optionText)
			}
			options = append(options, optionText)
		}
	}
	return options
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

func (variable *Variable) VerifyVariableValue(value interface{}, parameters map[string]interface{}) (interface{}, error) {
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
		options := variable.GetOptions(parameters)
		answerStr := fmt.Sprintf("%v", value)
		if !funk.Contains(options, answerStr) {
			return "", fmt.Errorf("answer [%s] is not one of the available options %v for variable [%s]", answerStr, options, variable.Name.Value)
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
		// do validation if needed
		if validateExpr != "" {
			allowEmpty := false
			if variable.Type.Value == TypeSecret {
				allowEmpty = true
			}
			validationErr := validatePrompt(variable.Name.Value, validateExpr, allowEmpty, parameters)(value)
			if validationErr != nil {
				return nil, fmt.Errorf("validation error for answer value [%v] for variable [%s]: %s", value, variable.Name.Value, validationErr.Error())
			}
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

func (variable *Variable) GetUserInput(defaultVal interface{}, parameters map[string]interface{}, surveyOpts ...survey.AskOpt) (interface{}, error) {
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
		err = survey.AskOne(
			&survey.Input{
				Message: prepareQuestionText(variable.Prompt.Value, fmt.Sprintf("What is the value of %s?", variable.Name.Value)),
				Default: defaultValStr,
				Help:    variable.GetHelpText(),
			},
			&answer,
			validatePrompt(variable.Name.Value, validateExpr, false, parameters),
			surveyOpts...,
		)
	case TypeSecret:
		questionMsg := prepareQuestionText(variable.Prompt.Value, fmt.Sprintf("What is the value of %s?", variable.Name.Value))
		if defaultVal != "" {
			questionMsg += fmt.Sprintf(" (%s)", defaultVal)
		}
		err = survey.AskOne(
			&survey.Password{
				Message: questionMsg,
				Help:    variable.GetHelpText(),
			},
			&answer,
			validatePrompt(variable.Name.Value, validateExpr, true, parameters),
			surveyOpts...,
		)

		// if user bypassed question, replace with default value
		if answer == "" {
			util.Verbose("[input] Got empty response for secret field '%s', replacing with default value: %s\n", variable.Name.Value, defaultVal)
			answer = defaultValStr
		}
	case TypeEditor:
		err = survey.AskOne(
			&survey.Editor{
				Message:       prepareQuestionText(variable.Prompt.Value, fmt.Sprintf("What is the value of %s?", variable.Name.Value)),
				Default:       defaultValStr,
				HideDefault:   true,
				AppendDefault: true,
				Help:          variable.GetHelpText(),
			},
			&answer,
			validatePrompt(variable.Name.Value, validateExpr, false, parameters),
			surveyOpts...,
		)
	case TypeFile:
		var filePath string
		err = survey.AskOne(
			&survey.Input{
				Message: prepareQuestionText(variable.Prompt.Value, fmt.Sprintf("What is the file path (relative/absolute) for %s?", variable.Name.Value)),
				Default: defaultValStr,
				Help:    variable.GetHelpText(),
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
				Message:  prepareQuestionText(variable.Prompt.Value, fmt.Sprintf("Select value for %s?", variable.Name.Value)),
				Options:  options,
				Default:  defaultValStr,
				PageSize: 10,
				Help:     variable.GetHelpText(),
			},
			&answer,
			validatePrompt(variable.Name.Value, validateExpr, false, parameters),
			surveyOpts...,
		)
	case TypeConfirm:
		var confirm bool
		err = survey.AskOne(
			&survey.Confirm{
				Message: prepareQuestionText(variable.Prompt.Value, fmt.Sprintf("%s?", variable.Name.Value)),
				Default: variable.Default.Bool,
				Help:    variable.GetHelpText(),
			},
			&confirm,
			validatePrompt(variable.Name.Value, validateExpr, false, parameters),
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
	if !util.IsStringInSlice(blueprintDoc.ApiVersion, models.BlueprintYamlFormatSupportedVersions) {
		return fmt.Errorf("api version needs to be %s or %s", models.BlueprintYamlFormatV2, models.BlueprintYamlFormatV1)
	}
	if blueprintDoc.ApiVersion != models.BlueprintYamlFormatV1 {
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
func (blueprintDoc *BlueprintConfig) prepareTemplateData(answersFilePath string, strictAnswers bool, useDefaultsAsValue bool, surveyOpts ...survey.AskOpt) (*PreparedData, error) {
	data := NewPreparedData()

	// if exists, get map of answers from file
	var answerMap map[string]string
	var err error
	usingAnswersFile := false
	if answersFilePath != "" {
		// parse answers file
		util.Verbose("[dataPrep] Using answers file [%s] (strict: %t) instead of asking questions from console\n", answersFilePath, strictAnswers)
		answerMap, err = GetValuesFromAnswersFile(answersFilePath)
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
		if !util.IsStringEmpty(variable.DependsOn.Value) {
			dependsOnVal, err := ParseDependsOnValue(variable.DependsOn, &blueprintDoc.Variables, data.TemplateData)
			if err != nil {
				return nil, err
			}
			if skipQuestionOnCondition(&variable, variable.DependsOn.Value, dependsOnVal, data, defaultVal, variable.DependsOn.InvertBool) {
				continue
			}
		}
		// skip user input if value field is present
		if variable.Value.Value != "" {
			parsedVal := variable.GetValueFieldVal(data.TemplateData)

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
				answer, err := variable.VerifyVariableValue(answerMap[variable.Name.Value], data.TemplateData)
				if err != nil {
					return nil, err
				}

				// if we have a valid answer, skip user input
				if variable.Type.Value == TypeConfirm {
					blueprintDoc.Variables[i] = variable
				}
				saveItemToTemplateDataMap(&variable, data, answer)
				util.Info("[dataPrep] Using answer file value [%v] for variable [%s]\n", answer, variable.Name.Value)
				continue
			} else {
				if strictAnswers {
					return nil, fmt.Errorf("variable with name [%s] could not be found in answers file", variable.Name.Value)
				} // do not return error when in non-strict answers mode, instead ask user input for the variable value
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
				variable.Name.Value,
				finalVal,
			)
			if variable.Type.Value == TypeConfirm {
				blueprintDoc.Variables[i] = variable
			}
			saveItemToTemplateDataMap(&variable, data, finalVal)
			if variable.Type.Value == TypeSecret && !variable.RevealOnSummary.Bool {
				data.DefaultData[variable.Label.Value] = "*****"
			} else {
				data.DefaultData[variable.Label.Value] = finalVal
			}
			continue
		}

		// ask question based on type to get value - on the following conditions in order
		// * if dependsOn fields exists, they have boolean result TRUE
		// * if value field is not present
		// * if not in default mode and default value is present
		// * if answers file is not present or isPartial is set to TRUE and answer not found on file for the variable
		util.Verbose("[dataPrep] Processing template variable [Name: %s, Type: %s]\n", variable.Name.Value, variable.Type.Value)
		var answer interface{}
		if !SkipUserInput {
			answer, err = variable.GetUserInput(defaultVal, data.TemplateData, surveyOpts...)
		}
		if err != nil {
			return nil, err
		}
		if variable.Type.Value == TypeConfirm {
			blueprintDoc.Variables[i] = variable
		}
		saveItemToTemplateDataMap(&variable, data, answer)
	}

	if useDefaultsAsValue && !usingAnswersFile {
		// Print summary default values table if in useDefaultsAsValues mode
		// use util.Print so that this is not skipped in quiet mode
		util.Print("Using default values:\n")
		util.Print(util.DataMapTable(&data.DefaultData, util.TableAlignLeft, 30, 50, "\t"))
	}

	return data, nil
}

// get values from answers file
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

func validatePrompt(varName string, validateExpr string, allowEmpty bool, parameters map[string]interface{}) func(val interface{}) error {
	return func(val interface{}) error {
		// if empty value is not allowed, check for any value
		if !allowEmpty {
			err := survey.Required(val)
			if err != nil {
				return err
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
				return fmt.Errorf("validation [%s] failed with value [%s]", validateExpr, val)
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
		if defaultVal == "" && currentVar.Type.Value == TypeConfirm {
			defaultVal = false
		}

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

func findVariableByName(variables *[]Variable, name string) (*Variable, error) {
	for _, variable := range *variables {
		if variable.Name.Value == name {
			return &variable, nil
		}
	}
	return nil, fmt.Errorf("no variable found in list by name [%s]", name)
}

func saveItemToTemplateDataMap(variable *Variable, preparedData *PreparedData, data interface{}) {
	if variable.Type.Value == TypeSecret {
		preparedData.Secrets[variable.Name.Value] = data
		// Use raw value of secret field if flag is set
		if variable.ReplaceAsIs.Bool {
			preparedData.TemplateData[variable.Name.Value] = data
		} else {
			preparedData.TemplateData[variable.Name.Value] = fmt.Sprintf(fmtTagValue, variable.Name.Value)
		}
	} else {
		// Save to values file if switch is ON
		if variable.SaveInXlvals.Bool {
			preparedData.Values[variable.Name.Value] = data
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
			case FnVersion:
				versionResult, err := version.CallVersionFuncByName(module, params...)
				if err != nil {
					return nil, err
				}
				return versionResult.GetResult(module, attr, index)
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

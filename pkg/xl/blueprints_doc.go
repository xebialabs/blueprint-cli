package xl

import (
	"fmt"
	"reflect"
	"regexp"
	"strconv"
	"strings"

	"github.com/xebialabs/xl-cli/pkg/cloud/aws"
	"github.com/xebialabs/xl-cli/pkg/models"
	"github.com/xebialabs/yaml"
	"gopkg.in/AlecAivazis/survey.v1"
)

// Constants
const (
	FnAWS = "aws"

	tagFn       = "!fn"
	fmtTagValue = "!value %s"
)

// InputType constants
const (
	TypeInput   = "Input"
	TypeSelect  = "Select"
	TypeConfirm = "Confirm"
)

var validTypes = []string{TypeInput, TypeSelect, TypeConfirm}

// Blueprint YAML doc definition
type BlueprintYaml struct {
	ApiVersion string      `yaml:"apiVersion,omitempty"`
	Kind       string      `yaml:"kind,omitempty"`
	Metadata   interface{} `yaml:"metadata,omitempty"`
	Spec       interface{} `yaml:"spec,omitempty"`

	Variables []Variable
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

// variable struct functions
func (variable *Variable) GetDefaultVal() string {
	defaultVal := variable.Default.Val
	if variable.Default.Tag == tagFn {
		values, err := processCustomFunction(defaultVal)
		if err != nil {
			Info("Error while processing default value !fn %s for %s. %s", defaultVal, variable.Name.Val, err.Error())
			defaultVal = ""
		} else {
			Verbose("[fn] Processed value of function [%s] is: %s\n", defaultVal, values[0])
			return values[0]
		}
	}

	// return false if this is a skipped confirm question
	if defaultVal == "" && variable.Type.Val == TypeConfirm {
		return strconv.FormatBool(false)
	}
	return defaultVal
}

func (variable *Variable) ParseDependsOnValue(tagVal string, fieldVal string, variables *[]Variable) (bool, error) {
	if tagVal == tagFn {
		values, err := processCustomFunction(fieldVal)
		if err != nil {
			return false, err
		}
		if len(values) == 0 {
			return false, fmt.Errorf("function [%s] results is empty", fieldVal)
		}
		Verbose("[fn] Processed value of function [%s] is: %s\n", fieldVal, values[0])

		dependsOnVal, err := strconv.ParseBool(values[0])
		if err != nil {
			return false, err
		}
		return dependsOnVal, nil
	} else {
		dependsOnVar, err := findVariableByName(variables, fieldVal)
		if err != nil {
			return false, err
		}
		return dependsOnVar.Value.Bool, nil
	}
}

func (variable *Variable) GetValueFieldVal() string {
	if variable.Value.Tag == tagFn {
		values, err := processCustomFunction(variable.Value.Val)
		if err != nil {
			Info("Error while processing !fn %s. Please update the value for %s manually. %s", variable.Value.Val, variable.Name.Val, err.Error())
			return ""
		}
		Verbose("[fn] Processed value of function [%s] is: %s\n", variable.Value.Val, values[0])
		return values[0]
	}
	return variable.Value.Val
}

func (variable *Variable) GetOptions() []string {
	var options []string
	for _, option := range variable.Options {
		if option.Tag == tagFn {
			opts, err := processCustomFunction(option.Val)
			if err != nil {
				Info("Error while processing !fn %s. Please update the value for %s manually. %s", option.Val, variable.Name.Val, err.Error())
				return nil
			}
			Verbose("[fn] Processed value of function [%s] is: %s\n", option.Val, opts)
			options = append(options, opts...)
		} else {
			options = append(options, option.Val)
		}
	}
	return options
}

func (variable *Variable) GetUserInput(defaultVal string, surveyOpts ...survey.AskOpt) (string, error) {
	var answer string
	var err error
	switch variable.Type.Val {
	case TypeInput:
		if variable.Secret.Bool == true {
			err = survey.AskOne(
				&survey.Password{Message: prepareQuestionText(variable.Description.Val, fmt.Sprintf("What is the value of %s?", variable.Name.Val))},
				&answer,
				validatePrompt(variable.Pattern.Val),
				surveyOpts...,
			)
		} else {
			err = survey.AskOne(
				&survey.Input{Message: prepareQuestionText(variable.Description.Val, fmt.Sprintf("What is the value of %s?", variable.Name.Val)), Default: defaultVal},
				&answer,
				validatePrompt(variable.Pattern.Val),
				surveyOpts...,
			)
		}
	case TypeSelect:
		options := variable.GetOptions()
		if err != nil {
			return "", err
		}
		err = survey.AskOne(
			&survey.Select{
				Message: prepareQuestionText(variable.Description.Val, fmt.Sprintf("Select value for %s?", variable.Name.Val)),
				Options: options,
				Default: defaultVal,
			},
			&answer,
			validatePrompt(variable.Pattern.Val),
			surveyOpts...,
		)
	case TypeConfirm:
		var confirm bool
		err = survey.AskOne(
			&survey.Confirm{Message: prepareQuestionText(variable.Description.Val, fmt.Sprintf("%s?", variable.Name.Val))},
			&confirm,
			validatePrompt(variable.Pattern.Val),
			surveyOpts...,
		)
		if err != nil {
			return "", err
		}
		answer = strconv.FormatBool(confirm)
		variable.Value.Bool = confirm
	}
	return answer, err
}

// parse doc spec into list of variables
func (blueprintDoc *BlueprintYaml) parseSpec() error {
	specs := TransformToMap(blueprintDoc.Spec)
	for _, m := range specs {
		parsedVar, err := blueprintDoc.parseSpecMap(&m)
		if err != nil {
			return err
		}
		blueprintDoc.Variables = append(blueprintDoc.Variables, parsedVar)
	}
	return nil
}
func (blueprintDoc *BlueprintYaml) parseSpecMap(m *map[interface{}]interface{}) (Variable, error) {
	parsedVar := Variable{}
	for k, v := range *m {
		switch vType := v.(type) {
		case string:
			// Set string field
			field := getVariableField(&parsedVar, strings.Title(k.(string)))
			setVariableField(&field, &VarField{Val: v.(string)})
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
			setVariableField(&field, &VarField{Bool: v.(bool)})
		case []interface{}:
			// Set []VarField
			field := getVariableField(&parsedVar, strings.Title(k.(string)))
			list := v.([]interface{})
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
			tag := v.(yaml.CustomTag)
			switch tag.Tag {
			case tagFn:
				field := getVariableField(&parsedVar, strings.Title(k.(string)))
				setVariableField(&field, &VarField{Val: tag.Value, Tag: tag.Tag})
			default:
				return Variable{}, fmt.Errorf("unknown tag %s %s", tag.Tag, tag.Value)
			}
		case nil:
			Verbose("[dataPrep] Got empty metadata variable field with key [%s]\n", k)
		default:
			return Variable{}, fmt.Errorf("unknown variable type [%s]", vType)
		}
	}
	return parsedVar, nil
}

// validate blueprint yaml document based on required fields
func (blueprintDoc *BlueprintYaml) validate() error {
	if blueprintDoc.ApiVersion != models.YamlFormatVersion {
		return fmt.Errorf("api version needs to be %s", models.YamlFormatVersion)
	}
	if blueprintDoc.Kind != models.BlueprintSpecKind {
		return fmt.Errorf("yaml document kind needs to be %s", models.BlueprintSpecKind)
	}
	return validateVariables(&blueprintDoc.Variables)
}
func validateVariables(variables *[]Variable) error {
	for _, userVar := range *variables {
		// validate non-empty
		if isStringEmpty(userVar.Name.Val) || isStringEmpty(userVar.Type.Val) {
			return fmt.Errorf("parameter [%s] is missing required fields: [type]", userVar.Name.Val)
		}

		// validate type field
		if !isStringInSlice(userVar.Type.Val, validTypes) {
			return fmt.Errorf("type [%s] is not valid for parameter [%s]", userVar.Type.Val, userVar.Name.Val)
		}

		// validate select case
		if userVar.Type.Val == TypeSelect && len(userVar.Options) == 0 {
			return fmt.Errorf("at least one option field is need to be set for parameter [%s]", userVar.Name.Val)
		}
	}
	return nil
}

// prepare template data by getting user input and calling named functions
func (blueprintDoc *BlueprintYaml) prepareTemplateData(surveyOpts ...survey.AskOpt) (*PreparedData, error) {
	data := NewPreparedData()
	for i, variable := range blueprintDoc.Variables {
		// process default field value
		defaultVal := variable.GetDefaultVal()

		// skip question based on DependsOn fields
		if !isStringEmpty(variable.DependsOnTrue.Val) {
			dependsOnTrueVal, err := variable.ParseDependsOnValue(variable.DependsOnTrue.Tag, variable.DependsOnTrue.Val, &blueprintDoc.Variables)
			if err != nil {
				return nil, err
			}
			if skipQuestionOnCondition(&variable, variable.DependsOnTrue.Val, dependsOnTrueVal, data, defaultVal, false) {
				continue
			}
		}
		if !isStringEmpty(variable.DependsOnFalse.Val) {
			dependsOnFalseVal, err := variable.ParseDependsOnValue(variable.DependsOnFalse.Tag, variable.DependsOnFalse.Val, &blueprintDoc.Variables)
			if err != nil {
				return nil, err
			}
			if skipQuestionOnCondition(&variable, variable.DependsOnFalse.Val, dependsOnFalseVal, data, defaultVal, true) {
				continue
			}
		}

		// skip user input if value field is present
		if variable.Value.Val != "" {
			parsedVal := variable.GetValueFieldVal()

			// check if resulting value is non-empty
			if parsedVal != "" {
				saveItemToTemplateDataMap(&variable, data, parsedVal)
				Verbose("[dataPrep] Skipping question for parameter [%s] because value [%s] is present\n", variable.Name.Val, variable.Value.Val)
				continue
			} else {
				Verbose("[dataPrep] Parsed value for parameter [%s] is empty, therefore not being skipped\n", variable.Name.Val)
			}
		}

		// ask question based on type to get value - if value field is not present
		Verbose("[dataPrep] Processing template variable [Name: %s, Type: %s]\n", variable.Name.Val, variable.Type.Val)
		answer, err := variable.GetUserInput(defaultVal, surveyOpts...)
		if err != nil {
			return nil, err
		}
		if variable.Type.Val == TypeConfirm {
			blueprintDoc.Variables[i] = variable
		}
		saveItemToTemplateDataMap(&variable, data, answer)
	}
	return data, nil
}

// --utility functions
func validatePrompt(pattern string) func(val interface{}) error {
	return func(val interface{}) error {
		err := survey.Required(val)
		if err != nil {
			return err
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

func skipQuestionOnCondition(currentVar *Variable, dependsOnVal string, dependsOn bool, dataMap *PreparedData, defaultVal string, condition bool) bool {
	if dependsOn == condition {
		saveItemToTemplateDataMap(currentVar, dataMap, defaultVal)
		Verbose("[dataPrep] Skipping question for parameter [%s] because DependsOn [%s] value is %t\n", currentVar.Name.Val, dependsOnVal, condition)
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

func saveItemToTemplateDataMap(variable *Variable, preparedData *PreparedData, data string) {
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
	Verbose("[fn] Calling fn [%s] for getting template variable value\n", fnStr)
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
			default:
				return nil, fmt.Errorf("unknown function type: %s", domain)
			}
		}
	} else {
		return nil, fmt.Errorf("invalid syntax in function reference: %s", fnStr)
	}
}

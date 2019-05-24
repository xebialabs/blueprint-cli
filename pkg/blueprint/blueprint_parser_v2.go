package blueprint

import (
	"bytes"
	"fmt"
	"path/filepath"
	"reflect"
	"strconv"
	"strings"

	"github.com/xebialabs/xl-cli/pkg/util"
	"github.com/xebialabs/yaml"
)

// parse blueprint definition doc
func parseTemplateMetadataV2(ymlContent *[]byte, templatePath string, blueprintRepository *BlueprintContext, isLocal bool) (*BlueprintConfig, error) {
	decoder := yaml.NewDecoder(bytes.NewReader(*ymlContent))
	decoder.SetStrict(true)
	yamlDoc := BlueprintYamlV2{}
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
		Metadata:        yamlDoc.parseToMetadata(),
		Include:         included,
		TemplateConfigs: templateConfigs,
		Variables:       variables,
	}
	err = blueprintConfig.validate()
	return &blueprintConfig, err
}

func (yamlDoc *BlueprintYamlV2) parseToMetadata() Metadata {
	return Metadata{
		Name:         yamlDoc.Metadata.Name,
		Description:  yamlDoc.Metadata.Description,
		Author:       yamlDoc.Metadata.Author,
		Version:      yamlDoc.Metadata.Version,
		Instructions: yamlDoc.Metadata.Instructions,
	}
}

// parse doc parameters into list of variables
func (yamlDoc *BlueprintYamlV2) parseParameters() ([]Variable, error) {
	parameters := []ParameterV2{}
	variables := []Variable{}
	parameters = yamlDoc.Spec.Parameters
	for _, m := range parameters {
		parsedVar, err := parseParameterV2(&m)
		if err != nil {
			return variables, err
		}
		variables = append(variables, parsedVar)
	}
	return variables, nil
}

// parse doc files into list of TemplateConfig
func (yamlDoc *BlueprintYamlV2) parseFiles(templatePath string, isLocal bool) ([]TemplateConfig, error) {
	files := []FileV2{}
	templateConfigs := []TemplateConfig{}
	files = yamlDoc.Spec.Files
	for _, m := range files {
		templateConfig, err := parseFileV2(&m)
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
func (yamlDoc *BlueprintYamlV2) parseIncludes() ([]IncludedBlueprintProcessed, error) {
	processedIncludes := []IncludedBlueprintProcessed{}
	for _, m := range yamlDoc.Spec.IncludeBefore {
		include, err := parseIncludeV2(&m)
		if err != nil {
			return nil, err
		}
		include.Stage = "before"
		processedIncludes = append(processedIncludes, include)
	}
	for _, m := range yamlDoc.Spec.IncludeAfter {
		include, err := parseIncludeV2(&m)
		if err != nil {
			return nil, err
		}
		include.Stage = "after"
		processedIncludes = append(processedIncludes, include)
	}
	return processedIncludes, nil
}

func parseParameterV2(m *ParameterV2) (Variable, error) {
	parsedVar := Variable{}
	err := parseFieldsFromStructV2(m, &parsedVar)
	if err != nil {
		return parsedVar, err
	}
	if parsedVar.Label == (VarField{}) {
		parsedVar.Label = parsedVar.Name
	}
	err = parsedVar.validate()
	return parsedVar, err
}

func parameterValidationErrorMsg(params ...interface{}) error {
	if len(params) == 1 {
		return fmt.Errorf("parameter must have a '%s' field", params...)
	}
	if len(params) == 2 {
		return fmt.Errorf("parameter %s must have a '%s' field", params...)
	}
	if len(params) == 3 {
		return fmt.Errorf("parameter %s must not have a '%s' field when field '%s' is set", params...)
	}
	return fmt.Errorf("validation error")
}

func (variable *Variable) validate() error {
	varName := variable.Name.Value
	if util.IsStringEmpty(varName) {
		return parameterValidationErrorMsg("name")
	}
	if variable.Value == (VarField{}) {
		// variable used as prompt
		if util.IsStringEmpty(variable.Prompt.Value) {
			return parameterValidationErrorMsg(varName, "prompt")
		}
		if util.IsStringEmpty(variable.Type.Value) {
			return parameterValidationErrorMsg(varName, "type")
		}
		if !util.IsStringInSlice(variable.Type.Value, validTypes) {
			return fmt.Errorf("type [%s] is not valid for parameter [%s]", variable.Type.Value, variable.Name.Value)
		}
	} else {
		// variable used as constant
		if !util.IsStringEmpty(variable.Prompt.Value) {
			return parameterValidationErrorMsg(varName, "prompt", "value")
		}
		if variable.Default != (VarField{}) {
			return parameterValidationErrorMsg(varName, "default", "value")
		}
		if variable.Options != nil && len(variable.Options) != 0 {
			return parameterValidationErrorMsg(varName, "options", "value")
		}
		if variable.DependsOn != (VarField{}) {
			return parameterValidationErrorMsg(varName, "promptIf", "value")
		}
	}
	if variable.Type.Value != TypeSecret {
		if variable.ReplaceAsIs != (VarField{}) {
			return parameterValidationErrorMsg(varName, "replaceAsIs", "type=SecretInput")
		}
		if variable.RevealOnSummary != (VarField{}) {
			return parameterValidationErrorMsg(varName, "revealOnSummary", "type=SecretInput")
		}
	}
	return nil
}

func parseFileV2(m *FileV2) (TemplateConfig, error) {
	parsedConfig := TemplateConfig{}
	err := parseFieldsFromStructV2(m, &parsedConfig)
	return parsedConfig, err
}

func parseIncludeV2(m *IncludedBlueprintV2) (IncludedBlueprintProcessed, error) {
	parsedInclude := IncludedBlueprintProcessed{}
	err := parseFieldsFromStructV2(m, &parsedInclude)
	return parsedInclude, err
}

func parseFieldsFromStructV2(original interface{}, target interface{}) error {
	parameterR := reflect.ValueOf(original).Elem()
	typeOfT := parameterR.Type()
	// iterate over the struct fields and map them
	for i := 0; i < parameterR.NumField(); i++ {
		fieldR := parameterR.Field(i)
		fieldName := typeOfT.Field(i).Name
		value := fieldR.Interface()
		fieldNameLower := strings.ToLower(fieldName)
		if (fieldNameLower == "promptif" || fieldNameLower == "writeif" || fieldNameLower == "includeif") && value != nil {
			fieldName = "DependsOn"
		}
		field := reflect.ValueOf(target).Elem().FieldByName(strings.Title(fieldName))
		switch val := value.(type) {
		case string:
			// Set string field
			setVariableField(&field, val, VarField{Value: val})
		case int, uint, uint8, uint16, uint32, uint64:
			// Set integer field
			setVariableField(&field, fmt.Sprint(val), VarField{Value: fmt.Sprint(val)})
		case float32, float64:
			// Set float field
			setVariableField(&field, fmt.Sprintf("%f", val), VarField{Value: fmt.Sprintf("%f", val)})
		case bool:
			// Set boolean field
			setVariableField(&field, strconv.FormatBool(val), VarField{Value: strconv.FormatBool(val), Bool: val})
		case []interface{}:
			// Set options array field for Parameters
			if len(val) > 0 {
				field.Set(reflect.MakeSlice(reflect.TypeOf([]VarField{}), len(val), len(val)))
				for i, it := range val {
					switch wVal := it.(type) {
					case int, uint, uint8, uint16, uint32, uint64:
						field.Index(i).Set(reflect.ValueOf(VarField{Value: fmt.Sprint(wVal)}))
					case float32, float64:
						field.Index(i).Set(reflect.ValueOf(VarField{Value: fmt.Sprintf("%f", wVal)}))
					case string:
						field.Index(i).Set(reflect.ValueOf(VarField{Value: wVal}))
					case yaml.CustomTag:
						field.Index(i).Set(reflect.ValueOf(VarField{Value: wVal.Value, Tag: wVal.Tag}))
					case map[interface{}]interface{}:
						var label, value string
						switch l := wVal["label"].(type) {
						case string:
							label = l
						default:
							return fmt.Errorf("unknown list item type %s", l)
						}

						switch v := wVal["value"].(type) {
						case string:
							value = v
						case int, uint, uint8, uint16, uint32, uint64:
							value = fmt.Sprint(v)
						case float32, float64:
							value = fmt.Sprintf("%f", v)
						default:
							return fmt.Errorf("unknown list item type %s", v)
						}

						field.Index(i).Set(reflect.ValueOf(VarField{Value: value, Label: label}))
					default:
						return fmt.Errorf("unknown list item type %s", wVal)
					}
				}
			}
		case []ParameterV2:
			// Set ParameterOverride array field for Include
			if len(val) > 0 {
				field.Set(reflect.MakeSlice(reflect.TypeOf([]Variable{}), len(val), len(val)))
				for i, it := range val {
					parsed := Variable{}
					err := parseFieldsFromStructV2(&it, &parsed)
					if err != nil {
						return err
					}
					field.Index(i).Set(reflect.ValueOf(parsed))
				}
			}
		case []FileV2:
			// Set FileOverride array field for Include
			if len(val) > 0 {
				field.Set(reflect.MakeSlice(reflect.TypeOf([]TemplateConfig{}), len(val), len(val)))
				for i, it := range val {
					parsed := TemplateConfig{}
					err := parseFieldsFromStructV2(&it, &parsed)
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
				setVariableField(&field, val.Value, VarField{Value: val.Value, Tag: val.Tag})
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

// ParseDependsOnValue parse the functions and expressions set on the dependsOn fields
func ParseDependsOnValue(varField VarField, variables *[]Variable, parameters map[string]interface{}) (bool, error) {
	tagVal := varField.Tag
	fieldVal := varField.Value
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
	return dependsOnVar.Value.Bool, nil
}

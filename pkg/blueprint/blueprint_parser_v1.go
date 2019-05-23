package blueprint

import (
	"bytes"
	"fmt"
	"path/filepath"
	"reflect"
	"strconv"
	"strings"

	"github.com/xebialabs/yaml"
)

// parse blueprint definition doc
func parseTemplateMetadataV1(ymlContent *[]byte, templatePath string, blueprintRepository *BlueprintContext, isLocal bool) (*BlueprintConfig, error) {
	decoder := yaml.NewDecoder(bytes.NewReader(*ymlContent))
	decoder.SetStrict(true)
	yamlDoc := BlueprintYamlV1{}
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
	blueprintConfig := BlueprintConfig{
		ApiVersion:      yamlDoc.ApiVersion,
		Kind:            yamlDoc.Kind,
		Metadata:        yamlDoc.parseToMetadata(),
		TemplateConfigs: templateConfigs,
		Variables:       variables,
	}
	err = blueprintConfig.validate()
	return &blueprintConfig, err
}

func (yamlDoc *BlueprintYamlV1) parseToMetadata() Metadata {
	return Metadata{
		Name:         yamlDoc.Metadata.Name,
		Description:  yamlDoc.Metadata.Description,
		Author:       yamlDoc.Metadata.Author,
		Version:      yamlDoc.Metadata.Version,
		Instructions: yamlDoc.Metadata.Instructions,
	}
}

// parse doc parameters into list of variables
func (yamlDoc *BlueprintYamlV1) parseParameters() ([]Variable, error) {
	parameters := []ParameterV1{}
	variables := []Variable{}
	if yamlDoc.Spec.Parameters != nil {
		parameters = yamlDoc.Spec.Parameters
	} else {
		// for backward compatibility with v8.5
		parameters = yamlDoc.Parameters
	}
	for _, m := range parameters {
		parsedVar, err := parseParameterV1(&m)
		if err != nil {
			return variables, err
		}
		variables = append(variables, parsedVar)
	}
	return variables, nil
}

// parse doc files into list of TemplateConfig
func (blueprintDoc *BlueprintYamlV1) parseFiles(templatePath string, isLocal bool) ([]TemplateConfig, error) {
	files := []FileV1{}
	templateConfigs := []TemplateConfig{}
	if blueprintDoc.Spec.Files != nil {
		files = blueprintDoc.Spec.Files
	} else {
		// for backward compatibility with v8.5
		files = blueprintDoc.Files
	}
	for _, m := range files {
		templateConfig, err := parseFileV1(&m)
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

func parseParameterV1(m *ParameterV1) (Variable, error) {
	parsedVar := Variable{}
	err := parseFieldsFromStructV1(m, &parsedVar)

	return transformToV2(parsedVar, m), err
}

func transformToV2(parsedVar Variable, m *ParameterV1) Variable {
	// compatibility hacks for v1 -> v2
	parsedVar.Prompt = parsedVar.Description
	parsedVar.Label = parsedVar.Name
	if m.Secret != nil {
		switch val := m.Secret.(type) {
		case string:
			if val == "true" {
				parsedVar.Type.Value = TypeSecret
			}
		case bool:
			if val {
				parsedVar.Type.Value = TypeSecret
			}
		}
	}
	return parsedVar
}

func parseFileV1(m *FileV1) (TemplateConfig, error) {
	parsedConfig := TemplateConfig{}
	err := parseFieldsFromStructV1(m, &parsedConfig)
	return parsedConfig, err
}

func parseFieldsFromStructV1(original interface{}, target interface{}) error {
	parameterR := reflect.ValueOf(original).Elem()
	typeOfT := parameterR.Type()
	// iterate over the struct fields and map them
	for i := 0; i < parameterR.NumField(); i++ {
		fieldR := parameterR.Field(i)
		fieldName := typeOfT.Field(i).Name
		value := fieldR.Interface()
		// for backward compatibility
		invertBool := false
		fieldNameLower := strings.ToLower(fieldName)
		if (fieldNameLower == "dependsontrue" || fieldNameLower == "dependsonfalse") && value != nil {
			invertBool = fieldNameLower == "dependsonfalse"
			fieldName = "DependsOn"
		}
		field := reflect.ValueOf(target).Elem().FieldByName(strings.Title(fieldName))
		switch val := value.(type) {
		case string:
			// Set string field
			setVariableField(&field, val, VarField{Value: val, InvertBool: invertBool})
		case int, uint, uint8, uint16, uint32, uint64:
			// Set integer field
			setVariableField(&field, fmt.Sprint(val), VarField{Value: fmt.Sprint(val), InvertBool: invertBool})
		case float32, float64:
			// Set float field
			setVariableField(&field, fmt.Sprintf("%f", val), VarField{Value: fmt.Sprintf("%f", val), InvertBool: invertBool})
		case bool:
			// Set boolean field
			setVariableField(&field, strconv.FormatBool(val), VarField{Value: strconv.FormatBool(val), Bool: val, InvertBool: invertBool})
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
					default:
						return fmt.Errorf("unknown list item type %s", wVal)
					}
				}
			}
		case yaml.CustomTag:
			// Set string field with YAML tag
			switch val.Tag {
			case tagFn, tagExpression:
				setVariableField(&field, val.Value, VarField{Value: val.Value, Tag: val.Tag, InvertBool: invertBool})
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

// TODO
// func ParseDependsOnValue(varField VarField, variables *[]Variable, parameters map[string]interface{}) (bool, error) {
// 	tagVal := varField.Tag
// 	fieldVal := varField.Val
// 	switch tagVal {
// 	case tagFn:
// 		values, err := ProcessCustomFunction(fieldVal)
// 		if err != nil {
// 			return false, err
// 		}
// 		if len(values) == 0 {
// 			return false, fmt.Errorf("function [%s] results is empty", fieldVal)
// 		}
// 		util.Verbose("[fn] Processed value of function [%s] is: %s\n", fieldVal, values[0])

// 		dependsOnVal, err := strconv.ParseBool(values[0])
// 		if err != nil {
// 			return false, err
// 		}
// 		if varField.InvertBool {
// 			return !dependsOnVal, nil
// 		}
// 		return dependsOnVal, nil
// 	case tagExpression:
// 		value, err := ProcessCustomExpression(fieldVal, parameters)
// 		if err != nil {
// 			return false, err
// 		}
// 		dependsOnVal, ok := value.(bool)
// 		if ok {
// 			util.Verbose("[expression] Processed value of expression [%s] is: %v\n", fieldVal, dependsOnVal)
// 			if varField.InvertBool {
// 				return !dependsOnVal, nil
// 			}
// 			return dependsOnVal, nil
// 		}
// 		return false, fmt.Errorf("Expression [%s] result is invalid for a boolean field", fieldVal)
// 	}
// 	dependsOnVar, err := findVariableByName(variables, fieldVal)
// 	if err != nil {
// 		return false, err
// 	}
// 	return dependsOnVar.Value.Bool, nil
// }

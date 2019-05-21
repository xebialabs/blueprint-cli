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
func parseTemplateMetadata(blueprintVars *[]byte, templatePath string, blueprintRepository *BlueprintContext, isLocal bool) (*BlueprintConfig, error) {
	decoder := yaml.NewDecoder(bytes.NewReader(*blueprintVars))
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
	included, err := yamlDoc.parseIncludes()
	if err != nil {
		return nil, err
	}
	blueprintConfig := BlueprintConfig{
		ApiVersion:      yamlDoc.ApiVersion,
		Kind:            yamlDoc.Kind,
		Metadata:        ParseToMetadata(yamlDoc.Metadata),
		Include:         included,
		TemplateConfigs: templateConfigs,
		Variables:       variables,
	}
	err = blueprintConfig.validate()
	return &blueprintConfig, err
}

func ParseToMetadata(metadata MetadataV1) Metadata {
	return Metadata{
		Name:         metadata.Name,
		Description:  metadata.Description,
		Author:       metadata.Author,
		Version:      metadata.Version,
		Instructions: metadata.Instructions,
	}
}

// parse doc parameters into list of variables
func (blueprintDoc *BlueprintYamlV1) parseParameters() ([]Variable, error) {
	parameters := []ParameterV1{}
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
func (blueprintDoc *BlueprintYamlV1) parseIncludes() ([]IncludedBlueprintProcessed, error) {
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

func parseParameter(m *ParameterV1) (Variable, error) {
	parsedVar := Variable{}
	err := parseFieldsFromStruct(m, func() reflect.Value {
		return reflect.ValueOf(&parsedVar).Elem()
	})
	return parsedVar, err
}

func parseFile(m *FileV1) (TemplateConfig, error) {
	parsedConfig := TemplateConfig{}
	err := parseFieldsFromStruct(m, func() reflect.Value {
		return reflect.ValueOf(&parsedConfig).Elem()
	})
	return parsedConfig, err
}

func parseInclude(m *IncludedBlueprintV1) (IncludedBlueprintProcessed, error) {
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
		case []ParameterOverrideV1:
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
		case []FileV1:
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
	return dependsOnVar.Value.Bool, nil
}

package blueprint

// Blueprint YAML schema definition
type BlueprintYamlV1 struct {
	ApiVersion string `yaml:"apiVersion"`
	Kind       string `yaml:"kind"`
	Metadata   MetadataV1
	Parameters []ParameterV1
	Files      []FileV1
	Spec       SpecV1
}

type MetadataV1 struct {
	Name         string `yaml:"projectName"`
	Description  string `yaml:"description"`
	Author       string `yaml:"author"`
	Version      string `yaml:"version"`
	Instructions string `yaml:"instructions"`
}

type SpecV1 struct {
	Parameters []ParameterV1
	Files      []FileV1
	Include    []IncludedBlueprintV1 // TODO remove
}

type ParameterV1 struct {
	Name            interface{}   `yaml:"name"`
	Type            interface{}   `yaml:"type"`
	Secret          interface{}   `yaml:"secret"`
	Value           interface{}   `yaml:"value"`
	Description     interface{}   `yaml:"description"`
	Default         interface{}   `yaml:"default"`
	DependsOn       interface{}   `yaml:"dependsOn"` // TODO remove
	DependsOnTrue   interface{}   `yaml:"dependsOnTrue"`
	DependsOnFalse  interface{}   `yaml:"dependsOnFalse"`
	Options         []interface{} `yaml:"options"`
	Pattern         interface{}   `yaml:"pattern"`
	SaveInXlvals    interface{}   `yaml:"saveInXlVals"`
	ReplaceAsIs     interface{}   `yaml:"useRawValue"`
	RevealOnSummary interface{}   `yaml:"showValueOnSummary"`
	Validate        interface{}   `yaml:"validate"` // TODO remove
}

type FileV1 struct {
	Path           interface{} `yaml:"path"`
	Operation      interface{} `yaml:"operation"`   // TODO remove
	RenamedPath    interface{} `yaml:"renamedPath"` // TODO remove
	DependsOn      interface{} `yaml:"dependsOn"`   // TODO remove
	DependsOnTrue  interface{} `yaml:"dependsOnTrue"`
	DependsOnFalse interface{} `yaml:"dependsOnFalse"`
}

// TODO remove
type IncludedBlueprintV1 struct {
	Blueprint          string                `yaml:"blueprint"`
	Stage              string                `yaml:"stage"`
	ParameterOverrides []ParameterOverrideV1 `yaml:"parameterOverrides"`
	FileOverrides      []FileV1              `yaml:"fileOverrides"`
	DependsOn          interface{}           `yaml:"dependsOn"`
	DependsOnTrue      interface{}           `yaml:"dependsOnTrue"`
	DependsOnFalse     interface{}           `yaml:"dependsOnFalse"`
}

// TODO remove
type ParameterOverrideV1 struct {
	Name           string      `yaml:"name"`
	Value          interface{} `yaml:"value"`
	DependsOn      interface{} `yaml:"dependsOn"`
	DependsOnTrue  interface{} `yaml:"dependsOnTrue"`
	DependsOnFalse interface{} `yaml:"dependsOnFalse"`
}

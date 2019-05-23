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
}

type ParameterV1 struct {
	Name            interface{}   `yaml:"name"`
	Type            interface{}   `yaml:"type"`
	Secret          interface{}   `yaml:"secret"`
	Value           interface{}   `yaml:"value"`
	Description     interface{}   `yaml:"description"`
	Default         interface{}   `yaml:"default"`
	DependsOnTrue   interface{}   `yaml:"dependsOnTrue"`
	DependsOnFalse  interface{}   `yaml:"dependsOnFalse"`
	Options         []interface{} `yaml:"options"`
	Pattern         interface{}   `yaml:"pattern"`
	SaveInXlvals    interface{}   `yaml:"saveInXlVals"`
	ReplaceAsIs     interface{}   `yaml:"useRawValue"`
	RevealOnSummary interface{}   `yaml:"showValueOnSummary"`
}

type FileV1 struct {
	Path           interface{} `yaml:"path"`
	DependsOnTrue  interface{} `yaml:"dependsOnTrue"`
	DependsOnFalse interface{} `yaml:"dependsOnFalse"`
}

package blueprint

// Blueprint YAML schema definition V2
type BlueprintYamlV2 struct {
	ApiVersion string `yaml:"apiVersion"`
	Kind       string `yaml:"kind"`
	Metadata   MetadataV2
	Spec       SpecV2
}

type MetadataV2 struct {
	Name         string `yaml:"name"`
	Description  string `yaml:"description"`
	Author       string `yaml:"author"`
	Version      string `yaml:"version"`
	Instructions string `yaml:"instructions"`
}

type SpecV2 struct {
	Parameters    []ParameterV2
	Files         []FileV2
	IncludeBefore []IncludedBlueprintV2
	IncludeAfter  []IncludedBlueprintV2
}

type ParameterV2 struct {
	Name            interface{}   `yaml:"name"`
	Prompt          interface{}   `yaml:"prompt"`
	Description     interface{}   `yaml:"description"`
	Label           interface{}   `yaml:"label"`
	Type            interface{}   `yaml:"type"`
	Default         interface{}   `yaml:"default"`
	Value           interface{}   `yaml:"value"`
	PromptIf        interface{}   `yaml:"promptIf"`
	Options         []interface{} `yaml:"options"`
	SaveInXlvals    interface{}   `yaml:"saveInXlvals"`
	ReplaceAsIs     interface{}   `yaml:"replaceAsIs"`
	RevealOnSummary interface{}   `yaml:"revealOnSummary"`
	Validate        interface{}   `yaml:"validate"`
}

type FileV2 struct {
	Path     interface{} `yaml:"path"`
	RenameTo interface{} `yaml:"renameTo"`
	WriteIf  interface{} `yaml:"writeIf"`
}

type IncludedBlueprintV2 struct {
	Blueprint          string        `yaml:"blueprint"`
	ParameterOverrides []ParameterV2 `yaml:"parameterOverrides"`
	FileOverrides      []FileV2      `yaml:"fileOverrides"`
	IncludeIf          interface{}   `yaml:"includeIf"`
}

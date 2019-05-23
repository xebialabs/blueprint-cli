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
	IncludeBefore []IncludedBlueprintV2 `yaml:"includeBefore"`
	IncludeAfter  []IncludedBlueprintV2 `yaml:"includeAfter"`
}

type ParameterV2 struct {
	Name            interface{}   `yaml:"name"`
	Type            interface{}   `yaml:"type"`
	Default         interface{}   `yaml:"default"`
	Value           interface{}   `yaml:"value"`
	PromptIf        interface{}   `yaml:"promptIf"`
	Options         []interface{} `yaml:"options"`
	SaveInXlvals    interface{}   `yaml:"saveInXlvals"`
	ReplaceAsIs     interface{}   `yaml:"replaceAsIs"`
	RevealOnSummary interface{}   `yaml:"revealOnSummary"`
	Validate        interface{}   `yaml:"validate"`
	Secret          interface{}   `yaml:"secret"`  // TODO remove
	Pattern         interface{}   `yaml:"pattern"` // TODO remove
	Prompt          interface{}   `yaml:"prompt"`
	Description     interface{}   `yaml:"description"`
	Label           interface{}   `yaml:"label"`
}

type FileV2 struct {
	Path     interface{} `yaml:"path"`
	WriteIf  interface{} `yaml:"writeIf"`
	RenameTo interface{} `yaml:"renameTo"`
}

type IncludedBlueprintV2 struct {
	Blueprint          string        `yaml:"blueprint"`
	IncludeIf          interface{}   `yaml:"includeIf"`
	ParameterOverrides []ParameterV2 `yaml:"parameterOverrides"`
	FileOverrides      []FileV2      `yaml:"fileOverrides"`
}

package blueprint

// Blueprint YAML schema definition
type BlueprintYaml struct {
	ApiVersion string `yaml:"apiVersion"`
	Kind       string `yaml:"kind"`
	Metadata   Metadata
	Parameters []Parameter
	Files      []File
	Spec       Spec
}

type Metadata struct {
	ProjectName  string `yaml:"projectName"`
	Description  string `yaml:"description"`
	Author       string `yaml:"author"`
	Version      string `yaml:"version"`
	Instructions string `yaml:"instructions"`
}

type Spec struct {
	Parameters []Parameter
	Files      []File
	Include    []IncludedBlueprint
}

type Parameter struct {
	Name        interface{} `yaml:"name"`
	Type        interface{} `yaml:"type"`
	Secret      interface{} `yaml:"secret"`
	Value       interface{} `yaml:"value"`
	Description interface{} `yaml:"description"`
	Default     interface{} `yaml:"default"`
	DependsOn   interface{} `yaml:"dependsOn"`
	// for backward compatibility
	DependsOnTrue      interface{}   `yaml:"dependsOnTrue"`
	DependsOnFalse     interface{}   `yaml:"dependsOnFalse"`
	Options            []interface{} `yaml:"options"`
	Pattern            interface{}   `yaml:"pattern"`
	SaveInXlVals       interface{}   `yaml:"saveInXlVals"`
	UseRawValue        interface{}   `yaml:"useRawValue"`
	ShowValueOnSummary interface{}   `yaml:"showValueOnSummary"`
}

type File struct {
	Path        interface{} `yaml:"path"`
	Operation   interface{} `yaml:"operation"`
	RenamedPath interface{} `yaml:"renamedPath"`
	DependsOn   interface{} `yaml:"dependsOn"`
	// for backward compatibility
	DependsOnTrue  interface{} `yaml:"dependsOnTrue"`
	DependsOnFalse interface{} `yaml:"dependsOnFalse"`
}

type IncludedBlueprint struct {
	Blueprint          string              `yaml:"blueprint"`
	Stage              string              `yaml:"stage"`
	ParameterOverrides []ParameterOverride `yaml:"parameterOverrides"`
	FileOverrides      []File              `yaml:"fileOverrides"`
	DependsOn          interface{}         `yaml:"dependsOn"`
	// for backward compatibility
	DependsOnTrue  interface{} `yaml:"dependsOnTrue"`
	DependsOnFalse interface{} `yaml:"dependsOnFalse"`
}

type ParameterOverride struct {
	Name      string      `yaml:"name"`
	Value     interface{} `yaml:"value"`
	DependsOn interface{} `yaml:"dependsOn"`
	// for backward compatibility
	DependsOnTrue  interface{} `yaml:"dependsOnTrue"`
	DependsOnFalse interface{} `yaml:"dependsOnFalse"`
}

// Blueprint YAML processed definition
type BlueprintConfig struct {
	ApiVersion      string
	Kind            string
	Metadata        Metadata
	Include         []IncludedBlueprintProcessed
	TemplateConfigs []TemplateConfig
	Variables       []Variable
}

type Variable struct {
	Name               VarField
	Type               VarField
	Secret             VarField
	Value              VarField
	Description        VarField
	Default            VarField
	DependsOn          VarField
	DependsOnFalse     VarField
	Options            []VarField
	Pattern            VarField
	SaveInXlVals       VarField
	UseRawValue        VarField
	ShowValueOnSummary VarField
}

// TemplateConfig holds the merged template file definitions with repository info
type TemplateConfig struct {
	Path           string
	FullPath       string
	Operation      string
	RenamedPath    VarField
	DependsOn      VarField
	DependsOnFalse VarField
}

type VarField struct {
	Val  string
	Bool bool
	Tag  string
}

type IncludedBlueprintProcessed struct {
	Blueprint          string
	Stage              string
	ParameterOverrides []ParameterOverridesProcessed
	FileOverrides      []TemplateConfig
	DependsOn          VarField
	DependsOnFalse     VarField
}

type ParameterOverridesProcessed struct {
	Name           string
	Value          VarField
	DependsOn      VarField
	DependsOnFalse VarField
}

type FileOverridesProcessed struct {
	Path           string
	DependsOn      VarField
	DependsOnFalse VarField
}

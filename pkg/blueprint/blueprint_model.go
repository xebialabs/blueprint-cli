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

type Spec struct {
	Parameters []Parameter
	Files      []File
	Include    []IncludedBlueprint
}

type Parameter struct {
	Name            interface{}   `yaml:"name"`
	Type            interface{}   `yaml:"type"`
	Secret          interface{}   `yaml:"secret"`
	Value           interface{}   `yaml:"value"`
	Description     interface{}   `yaml:"description"`
	Default         interface{}   `yaml:"default"`
	DependsOn       interface{}   `yaml:"dependsOn"`
	DependsOnTrue   interface{}   `yaml:"dependsOnTrue"`
	DependsOnFalse  interface{}   `yaml:"dependsOnFalse"`
	Options         []interface{} `yaml:"options"`
	Pattern         interface{}   `yaml:"pattern"`
	SaveInXlvals    interface{}   `yaml:"saveInXlVals"`
	ReplaceAsIs     interface{}   `yaml:"useRawValue"`
	RevealOnSummary interface{}   `yaml:"showValueOnSummary"`
	Validate        interface{}   `yaml:"validate"`
}

type File struct {
	Path           interface{} `yaml:"path"`
	Operation      interface{} `yaml:"operation"`
	RenamedPath    interface{} `yaml:"renamedPath"`
	DependsOn      interface{} `yaml:"dependsOn"`
	DependsOnTrue  interface{} `yaml:"dependsOnTrue"`
	DependsOnFalse interface{} `yaml:"dependsOnFalse"`
}

type IncludedBlueprint struct {
	Blueprint          string              `yaml:"blueprint"`
	Stage              string              `yaml:"stage"`
	ParameterOverrides []ParameterOverride `yaml:"parameterOverrides"`
	FileOverrides      []File              `yaml:"fileOverrides"`
	DependsOn          interface{}         `yaml:"dependsOn"`
	DependsOnTrue      interface{}         `yaml:"dependsOnTrue"`
	DependsOnFalse     interface{}         `yaml:"dependsOnFalse"`
}

type ParameterOverride struct {
	Name           string      `yaml:"name"`
	Value          interface{} `yaml:"value"`
	DependsOn      interface{} `yaml:"dependsOn"`
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

type Metadata struct {
	Name         string `yaml:"projectName"`
	Description  string `yaml:"description"`
	Author       string `yaml:"author"`
	Version      string `yaml:"version"`
	Instructions string `yaml:"instructions"`
}

type Variable struct {
	Name            VarField
	Type            VarField
	Secret          VarField
	Value           VarField
	Description     VarField
	Default         VarField
	DependsOn       VarField
	Options         []VarField
	Pattern         VarField
	SaveInXlvals    VarField
	ReplaceAsIs     VarField
	RevealOnSummary VarField
	Validate        VarField
}

// TemplateConfig holds the merged template file definitions with repository info
type TemplateConfig struct {
	Path        string
	FullPath    string
	Operation   string
	RenamedPath VarField
	DependsOn   VarField
}

type VarField struct {
	Val        string
	Bool       bool
	Tag        string
	InvertBool bool
}

type IncludedBlueprintProcessed struct {
	Blueprint          string
	Stage              string
	ParameterOverrides []ParameterOverridesProcessed
	FileOverrides      []TemplateConfig
	DependsOn          VarField
}

type ParameterOverridesProcessed struct {
	Name      string
	Value     VarField
	DependsOn VarField
}

type FileOverridesProcessed struct {
	Path      string
	DependsOn VarField
}

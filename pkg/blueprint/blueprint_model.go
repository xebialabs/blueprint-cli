package blueprint

// Blueprint YAML schema definition
type BlueprintYaml struct {
	ApiVersion string `yaml:"apiVersion"`
	Kind       string `yaml:"kind"`
	Metadata   Metadata
	// TODO
	Parameters []Parameter
	Files      []File
	// Parameters interface{} `yaml:"parameters,omitempty"`
	// Files      interface{} `yaml:"files,omitempty"`
	Spec Spec
}

type Metadata struct {
	ProjectName  string `yaml:"projectName"`
	Description  string `yaml:"description"`
	Author       string `yaml:"author"`
	Version      string `yaml:"version"`
	Instructions string `yaml:"instructions"`
}

type Spec struct {
	// TODO
	Parameters []Parameter
	Files      []File
	// Parameters interface{} `yaml:"parameters,omitempty"`
	// Files      interface{} `yaml:"files,omitempty"`
	Include []IncludedBlueprint
}

type Parameter struct {
	Name           interface{}   `yaml:"name"`
	Type           interface{}   `yaml:"type"`
	Secret         interface{}   `yaml:"secret"`
	Value          interface{}   `yaml:"value"`
	Description    interface{}   `yaml:"description"`
	Default        interface{}   `yaml:"default"`
	DependsOn      interface{}   `yaml:"dependsOn"`
	DependsOnTrue  interface{}   `yaml:"dependsOnTrue"`
	DependsOnFalse interface{}   `yaml:"dependsOnFalse"`
	Options        []interface{} `yaml:"options"`
	Pattern        interface{}   `yaml:"pattern"`
	SaveInXlVals   interface{}   `yaml:"saveInXlVals"`
	UseRawValue    interface{}   `yaml:"useRawValue"`
}

type File struct {
	Path           interface{} `yaml:"path"`
	DependsOn      interface{} `yaml:"dependsOn"`
	DependsOnTrue  interface{} `yaml:"dependsOnTrue"`
	DependsOnFalse interface{} `yaml:"dependsOnFalse"`
}

type IncludedBlueprint struct {
	Blueprint       string `yaml:"blueprint"`
	Stage           string `yaml:"stage"`
	ParameterValues []ParameterValue
	SkipFiles       []File
	DependsOn       interface{} `yaml:"dependsOn"`
	DependsOnTrue   interface{} `yaml:"dependsOnTrue"`
	DependsOnFalse  interface{} `yaml:"dependsOnFalse"`
}

type ParameterValue struct {
	Name           string      `yaml:"name"`
	Value          string      `yaml:"value"`
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

type Variable struct {
	Name        VarField
	Type        VarField
	Secret      VarField
	Value       VarField
	Description VarField
	Default     VarField
	DependsOn   VarField
	// DependsOnTrue  VarField // TODO remove
	DependsOnFalse VarField // TODO remove
	Options        []VarField
	Pattern        VarField
	SaveInXlVals   VarField
	UseRawValue    VarField
}

// TemplateConfig holds the merged template file definitions with repository info
type TemplateConfig struct {
	File      string
	FullPath  string
	DependsOn VarField
	// DependsOnTrue  VarField // TODO remove
	DependsOnFalse VarField // TODO remove
}

type VarField struct {
	Val  string
	Bool bool
	Tag  string
}

type IncludedBlueprintProcessed struct {
	Blueprint       string
	Stage           string
	ParameterValues []ParameterValuesProcessed
	SkipFiles       []SkipFilesProcessed
	DependsOn       VarField
}

type ParameterValuesProcessed struct {
	Name      string
	Value     string
	DependsOn VarField
}

type SkipFilesProcessed struct {
	Path      string
	DependsOn VarField
}

package blueprint

// Blueprint YAML schema definition
type BlueprintYaml struct {
	ApiVersion string `yaml:"apiVersion,omitempty"`
	Kind       string `yaml:"kind,omitempty"`
	Metadata   Metadata
	// TODO
	// Parameters []Parameters
	// Files      []Files
	Parameters interface{} `yaml:"parameters,omitempty"`
	Files      interface{} `yaml:"files,omitempty"`
	Spec       Spec
}

type Metadata struct {
	ProjectName  string `yaml:"projectName,omitempty"`
	Description  string `yaml:"description,omitempty"`
	Author       string `yaml:"author,omitempty"`
	Version      string `yaml:"version,omitempty"`
	Instructions string `yaml:"instructions,omitempty"`
}

type Parameters struct {
	Name           string
	Type           string
	Secret         string
	Value          string
	Description    string
	Default        string
	DependsOn      string
	DependsOnTrue  string
	DependsOnFalse string
	Options        []string
	Pattern        string
	SaveInXlVals   string
	UseRawValue    string
}

type Files struct {
	Path           string
	DependsOn      string
	DependsOnTrue  string
	DependsOnFalse string
}

type Spec struct {
	// TODO
	// Parameters []Parameters
	// Files      []Files
	Parameters interface{} `yaml:"parameters,omitempty"`
	Files      interface{} `yaml:"files,omitempty"`
	Include    []IncludedBlueprint
}

type IncludedBlueprint struct {
	Blueprint       string
	Stage           string
	ParameterValues []ParameterValues
	SkipFiles       []SkipFiles
	DependsOn       string
	DependsOnTrue   string
	DependsOnFalse  string
}

type ParameterValues struct {
	Name           string
	Value          string
	DependsOn      string
	DependsOnTrue  string
	DependsOnFalse string
}

type SkipFiles struct {
	Path           string
	DependsOn      string
	DependsOnTrue  string
	DependsOnFalse string
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
	Name           VarField
	Type           VarField
	Secret         VarField
	Value          VarField
	Description    VarField
	Default        VarField
	DependsOn      VarField
	DependsOnTrue  VarField // TODO remove
	DependsOnFalse VarField // TODO remove
	Options        []VarField
	Pattern        VarField
	SaveInXlVals   VarField
	UseRawValue    VarField
}

// TemplateConfig holds the merged template file definitions with repository info
type TemplateConfig struct {
	File           string
	FullPath       string
	DependsOn      VarField
	DependsOnTrue  VarField // TODO remove
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

package blueprint

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
	Name         string
	Description  string
	Author       string
	Version      string
	Instructions string
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

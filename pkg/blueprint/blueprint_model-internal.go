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
	Name                    string
	Description             string
	Author                  string
	Version                 string
	Instructions            string
	SuppressXebiaLabsFolder bool
}

type Variable struct {
	Name            VarField
	Type            VarField
	Value           VarField
	Prompt          VarField
	Default         VarField
	DependsOn       VarField
	Options         []VarField
	SaveInXlvals    VarField
	ReplaceAsIs     VarField
	RevealOnSummary VarField
	Validate        VarField
	Description     VarField
	Label           VarField
	IgnoreIfSkipped VarField
	Meta            VariableMeta
}

type VariableMeta struct {
	PromptSkipped bool
}

// TemplateConfig holds the merged template file definitions with repository info
type TemplateConfig struct {
	Path      string
	FullPath  string
	RenameTo  VarField
	DependsOn VarField
}

type VarField struct {
	Label      string // used only for options
	Value      string
	Bool       bool
	Tag        string
	InvertBool bool
}

type IncludedBlueprintProcessed struct {
	Blueprint          string
	Stage              string
	ParameterOverrides []Variable
	FileOverrides      []TemplateConfig
	DependsOn          VarField
}

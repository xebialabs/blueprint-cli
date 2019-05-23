package blueprint

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/xebialabs/xl-cli/pkg/models"
	"github.com/xebialabs/yaml"
)

func getValidTestBlueprintMetadata(templatePath string, blueprintRepository BlueprintContext) (*BlueprintConfig, error) {
	metadata := []byte(
		fmt.Sprintf(`
         apiVersion: %s
         kind: Blueprint
         metadata:
           name: Test Project
           description: Is just a test blueprint project used for manual testing of inputs
           author: XebiaLabs
           version: 1.0
           instructions: These are the instructions for executing this blueprint
         spec:
           parameters:
           - name: pass
             type: Input
             description: password?
             secret: true
           - name: test
             type: Input
             default: lala
             saveInXlvals: true
             description: help text
           - name: fn
             type: Input
             value: !fn aws.regions(ecs)[0]
           - name: select
             type: Select
             description: select region
             options:
             - !fn aws.regions(ecs)[0]
             - b
             - c
             default: b
           - name: isit
             description: is it?
             type: Confirm
             value: true
           - name: isitnot
             description: negative question?
             type: Confirm
           - name: dep
             description: depends on others
             type: Input
             promptIf: !expression "isit && true"
           files:
           - path: xebialabs/foo.yaml
           - path: readme.md
             writeIf: isit
           - path: bar.md
             writeIf: isitnot
           - path: foo.md
             writeIf: !expression "!isitnot"
           includeBefore:
           - blueprint: kubernetes/gke-cluster
             parameterOverrides:
             - name: Foo
               value: hello
               promptIf: !expression "ExpTest1 == 'us-west' && AppName != 'foo' && TestDepends"
             - name: bar
               value: true
             fileOverrides:
             - path: xld-infrastructure.yml.tmpl
               writeIf: false
           includeAfter:
           - blueprint: kubernetes/namespace
             includeIf: !expression "ExpTest1 == 'us-west' && AppName != 'foo' && TestDepends"
             parameterOverrides:
             - name: Foo
               value: hello
`, models.BlueprintYamlFormatV2))
	return parseTemplateMetadataV2(&metadata, templatePath, &blueprintRepository, true)
}

func TestParseTemplateMetadataV2(t *testing.T) {
	templatePath := "test/blueprints"
	blueprintRepository := BlueprintContext{}
	tmpDir := filepath.Join("test", "blueprints")
	os.MkdirAll(tmpDir, os.ModePerm)
	defer os.RemoveAll("test")
	d1 := []byte("hello\ngo\n")
	ioutil.WriteFile(filepath.Join(tmpDir, "test.yaml.tmpl"), d1, os.ModePerm)

	t.Run("should error on invalid xl yaml", func(t *testing.T) {
		metadata := []byte("test: blueprint")
		_, err := parseTemplateMetadataV2(&metadata, templatePath, &blueprintRepository, true)
		require.NotNil(t, err)
		assert.Equal(t, fmt.Sprintf("yaml: unmarshal errors:\n  line 1: field test not found in type blueprint.BlueprintYamlV2"), err.Error())
	})

	t.Run("should error on missing api version", func(t *testing.T) {
		metadata := []byte("kind: blueprint")
		_, err := parseTemplateMetadataV2(&metadata, templatePath, &blueprintRepository, true)
		require.NotNil(t, err)
		assert.Equal(t, fmt.Sprintf("api version needs to be %s or %s", models.BlueprintYamlFormatV2, models.BlueprintYamlFormatV1), err.Error())
	})

	t.Run("should error on missing doc kind", func(t *testing.T) {
		metadata := []byte("apiVersion: " + models.BlueprintYamlFormatV2)
		_, err := parseTemplateMetadataV2(&metadata, templatePath, &blueprintRepository, true)
		require.NotNil(t, err)
		assert.Equal(t, "yaml document kind needs to be Blueprint", err.Error())
	})

	t.Run("should error on unknown field type", func(t *testing.T) {
		metadata := []byte(
			fmt.Sprintf(`
                apiVersion: %s
                kind: Blueprint
                metadata:
                spec:
                  parameters:
                  - name: Test
                    type: Invalid
                    value: testing`,
				models.BlueprintYamlFormatV2))
		_, err := parseTemplateMetadataV2(&metadata, templatePath, &blueprintRepository, true)
		require.NotNil(t, err)
		assert.Equal(t, "type [Invalid] is not valid for parameter [Test]", err.Error())
	})

	t.Run("should error on missing variable field", func(t *testing.T) {
		metadata := []byte(
			fmt.Sprintf(`
               apiVersion: %s
               kind: Blueprint
               metadata:
               spec:
                 parameters:
                 - name: Test
                   value: testing`, models.BlueprintYamlFormatV2))
		_, err := parseTemplateMetadataV2(&metadata, templatePath, &blueprintRepository, true)
		require.NotNil(t, err)
		assert.Equal(t, "parameter [Test] is missing required fields: [type]", err.Error())
	})

	t.Run("should error on missing options for variable", func(t *testing.T) {
		metadata := []byte(
			fmt.Sprintf(`
                apiVersion: %s
                kind: Blueprint
                metadata:
                spec:
                  parameters:
                  - name: Test
                    type: Select
                    options:`, models.BlueprintYamlFormatV2))
		_, err := parseTemplateMetadataV2(&metadata, templatePath, &blueprintRepository, true)
		require.NotNil(t, err)
		assert.Equal(t, "at least one option field is need to be set for parameter [Test]", err.Error())
	})
	t.Run("should error on missing path for files", func(t *testing.T) {
		metadata := []byte(
			fmt.Sprintf(`
                apiVersion: %s
                kind: Blueprint
                metadata:
                spec:
                  parameters:
                  - name: Test
                    type: Confirm
                  files:
                  - writeIf: Test
                  - path: xbc.yaml`, models.BlueprintYamlFormatV2))
		_, err := parseTemplateMetadataV2(&metadata, "aws/test", &blueprintRepository, true)
		require.NotNil(t, err)
		assert.Equal(t, "path is missing for file specification in files", err.Error())
	})
	t.Run("should error on invalid path for files", func(t *testing.T) {
		metadata := []byte(
			fmt.Sprintf(`
               apiVersion: %s
               kind: Blueprint
               metadata:
               spec:
                 parameters:
                 - name: Test
                   type: Confirm
                 files:
                 - path: ../xbc.yaml`, models.BlueprintYamlFormatV2))
		_, err := parseTemplateMetadataV2(&metadata, "aws/test", &blueprintRepository, true)
		require.NotNil(t, err)
		assert.Equal(t, "path for file specification cannot start with /, .. or ./", err.Error())
	})
	t.Run("should error on duplicate variable names", func(t *testing.T) {
		metadata := []byte(
			fmt.Sprintf(`
               apiVersion: %s
               kind: Blueprint
               metadata:
               spec:
                 parameters:
                 - name: Test
                   type: Input
                   default: 1
                 - name: Test
                   type: Input
                   default: 2
                 files:`, models.BlueprintYamlFormatV2))
		_, err := parseTemplateMetadataV2(&metadata, "aws/test", &blueprintRepository, true)
		require.NotNil(t, err)
		assert.Equal(t, "variable names must be unique within blueprint 'parameters' definition", err.Error())
	})

	t.Run("should parse nested variables from valid metadata", func(t *testing.T) {
		doc, err := getValidTestBlueprintMetadata(templatePath, blueprintRepository)
		require.Nil(t, err)
		assert.Len(t, doc.Variables, 7)
		assert.Equal(t, Variable{
			Name:        VarField{Val: "pass"},
			Type:        VarField{Val: TypeInput},
			Description: VarField{Val: "password?"},
			Secret:      VarField{Bool: true, Val: "true"},
		}, doc.Variables[0])
		assert.Equal(t, Variable{
			Name:         VarField{Val: "test"},
			Type:         VarField{Val: TypeInput},
			Default:      VarField{Val: "lala"},
			Description:  VarField{Val: "help text"},
			SaveInXlvals: VarField{Bool: true, Val: "true"},
		}, doc.Variables[1])
		assert.Equal(t, Variable{
			Name:  VarField{Val: "fn"},
			Type:  VarField{Val: TypeInput},
			Value: VarField{Val: "aws.regions(ecs)[0]", Tag: tagFn},
		}, doc.Variables[2])
		assert.Equal(t, Variable{
			Name:        VarField{Val: "select"},
			Type:        VarField{Val: TypeSelect},
			Description: VarField{Val: "select region"},
			Options: []VarField{
				{Val: "aws.regions(ecs)[0]", Tag: tagFn},
				{Val: "b"},
				{Val: "c"},
			},
			Default: VarField{Val: "b"},
		}, doc.Variables[3])
		assert.Equal(t, Variable{
			Name:        VarField{Val: "isit"},
			Type:        VarField{Val: TypeConfirm},
			Description: VarField{Val: "is it?"},
			Value:       VarField{Bool: true, Val: "true"},
		}, doc.Variables[4])
		assert.Equal(t, Variable{
			Name:        VarField{Val: "isitnot"},
			Type:        VarField{Val: TypeConfirm},
			Description: VarField{Val: "negative question?"},
		}, doc.Variables[5])
		assert.Equal(t, Variable{
			Name:        VarField{Val: "dep"},
			Type:        VarField{Val: TypeInput},
			Description: VarField{Val: "depends on others"},
			DependsOn:   VarField{Val: "isit && true", Tag: "!expression"},
		}, doc.Variables[6])
	})
	t.Run("should parse files from valid metadata", func(t *testing.T) {
		doc, err := getValidTestBlueprintMetadata("templatePath/test", blueprintRepository)
		require.Nil(t, err)
		assert.Equal(t, 4, len(doc.TemplateConfigs))
		assert.Equal(t, TemplateConfig{
			Path:     "xebialabs/foo.yaml",
			FullPath: "templatePath/test/xebialabs/foo.yaml",
		}, doc.TemplateConfigs[0])
		assert.Equal(t, TemplateConfig{
			Path:      "readme.md",
			FullPath:  "templatePath/test/readme.md",
			DependsOn: VarField{Val: "isit"},
		}, doc.TemplateConfigs[1])
		assert.Equal(t, TemplateConfig{
			Path:      "bar.md",
			FullPath:  "templatePath/test/bar.md",
			DependsOn: VarField{Val: "isitnot"},
		}, doc.TemplateConfigs[2])
		assert.Equal(t, TemplateConfig{
			Path:      "foo.md",
			FullPath:  "templatePath/test/foo.md",
			DependsOn: VarField{Val: "!isitnot", Tag: tagExpression},
		}, doc.TemplateConfigs[3])
	})
	t.Run("should parse includes from valid metadata", func(t *testing.T) {
		doc, err := getValidTestBlueprintMetadata("templatePath/test", blueprintRepository)
		require.Nil(t, err)
		assert.Equal(t, 2, len(doc.Include))
		assert.Equal(t, IncludedBlueprintProcessed{
			Blueprint: "kubernetes/gke-cluster",
			Stage:     "before",
			ParameterOverrides: []Variable{
				{
					Name:      VarField{Val: "Foo"},
					Value:     VarField{Val: "hello"},
					DependsOn: VarField{Tag: "!expression", Val: "ExpTest1 == 'us-west' && AppName != 'foo' && TestDepends"},
				},
				{
					Name:  VarField{Val: "bar"},
					Value: VarField{Val: "true", Bool: true},
				},
			},
			FileOverrides: []TemplateConfig{
				{
					Path:      "xld-infrastructure.yml.tmpl",
					DependsOn: VarField{Val: "false", Bool: false},
				},
			},
		}, doc.Include[0])
		assert.Equal(t, IncludedBlueprintProcessed{
			Blueprint: "kubernetes/namespace",
			Stage:     "after",
			ParameterOverrides: []Variable{
				{
					Name:  VarField{Val: "Foo"},
					Value: VarField{Val: "hello"},
				},
			},
			DependsOn: VarField{Tag: "!expression", Val: "ExpTest1 == 'us-west' && AppName != 'foo' && TestDepends"},
		}, doc.Include[1])
	})
	t.Run("should parse metadata fields", func(t *testing.T) {
		doc, err := getValidTestBlueprintMetadata("templatePath/test", blueprintRepository)
		require.Nil(t, err)
		assert.Equal(t, "Test Project", doc.Metadata.Name)
		assert.Equal(t,
			"Is just a test blueprint project used for manual testing of inputs",
			doc.Metadata.Description)
		assert.Equal(t,
			"XebiaLabs",
			doc.Metadata.Author)
		assert.Equal(t,
			"1.0",
			doc.Metadata.Version)
		assert.Equal(t,
			"These are the instructions for executing this blueprint",
			doc.Metadata.Instructions)
	})
	t.Run("should parse multiline instructions", func(t *testing.T) {
		metadata := []byte(
			fmt.Sprintf(`
                apiVersion: %s
                kind: Blueprint
                metadata:
                  name: allala
                  instructions: |
                    This is a multiline instruction:

                    The instructions continue here:
                      1. First step
                      2. Second step
                spec:`, models.BlueprintYamlFormatV2))
		doc, err := parseTemplateMetadataV2(&metadata, "aws/test", &blueprintRepository, true)
		require.Nil(t, err)
		assert.Equal(t,
			"This is a multiline instruction:\n\nThe instructions continue here:\n  1. First step\n  2. Second step\n",
			doc.Metadata.Instructions)
	})
}

func TestBlueprintYaml_parseParameters(t *testing.T) {
	tests := []struct {
		name    string
		spec    SpecV2
		want    []Variable
		wantErr bool
	}{
		{
			"should error on invalid tag in promptIf ",
			SpecV2{
				Parameters: []ParameterV2{
					{
						Name:        "test",
						Type:        "Input",
						Secret:      true, //TODO
						Value:       "string",
						Description: "desc",
						Default:     "string2",
						PromptIf:    yaml.CustomTag{Tag: "!foo", Value: "1 > 2"},
						Options: []interface{}{
							"test", "foo", 10, 13.4,
						},
						Pattern:      "pat", //TODO
						SaveInXlvals: true,
						ReplaceAsIs:  false,
					},
				},
			},
			[]Variable{},
			true,
		},
		{
			"should error on invalid type in list ",
			SpecV2{
				Parameters: []ParameterV2{
					{
						Name:        "test",
						Type:        "Input",
						Secret:      true, //TODO
						Value:       "string",
						Description: "desc",
						Default:     "string2",
						Options: []interface{}{
							"test", "foo", true,
						},
						Pattern:      "pat", //TODO
						SaveInXlvals: true,
						ReplaceAsIs:  false,
					},
				},
			},
			[]Variable{},
			true,
		},
		{
			"should parse parameters under spec",
			SpecV2{
				Parameters: []ParameterV2{
					{
						Name:        "test",
						Type:        "Input",
						Secret:      true, //TODO
						Value:       "string",
						Description: "desc",
						Default:     "string2",
						PromptIf:    yaml.CustomTag{Tag: "!expression", Value: "1 > 2"},
						Options: []interface{}{
							"test", "foo", 10, 13.4,
						},
						Pattern:      "pat", //TODO
						SaveInXlvals: true,
						ReplaceAsIs:  false,
					},
					{
						Name:        "test",
						Type:        "Confirm",
						Secret:      false, //TODO
						Value:       true,
						Description: "desc",
						Default:     false,
						PromptIf:    yaml.CustomTag{Tag: "!expression", Value: "1 > 2"},
						Options: []interface{}{
							"test", yaml.CustomTag{Tag: "!expression", Value: "1 > 2"},
						},
						Pattern:      "pat", //TODO
						SaveInXlvals: true,
						ReplaceAsIs:  false,
					},
				},
			},
			[]Variable{
				{
					Name:        VarField{Val: "test"},
					Type:        VarField{Val: "Input"},
					Secret:      VarField{Bool: true, Val: "true"},
					Value:       VarField{Val: "string"},
					Description: VarField{Val: "desc"},
					Default:     VarField{Val: "string2"},
					DependsOn:   VarField{Tag: "!expression", Val: "1 > 2"},
					Options: []VarField{
						VarField{Val: "test"}, VarField{Val: "foo"}, VarField{Val: "10"}, VarField{Val: "13.400000"},
					},
					Pattern:      VarField{Val: "pat"},
					SaveInXlvals: VarField{Bool: true, Val: "true"},
					ReplaceAsIs:  VarField{Bool: false, Val: "false"},
				},
				{
					Name:        VarField{Val: "test"},
					Type:        VarField{Val: "Confirm"},
					Secret:      VarField{Bool: false, Val: "false"},
					Value:       VarField{Bool: true, Val: "true"},
					Description: VarField{Val: "desc"},
					Default:     VarField{Bool: false, Val: "false"},
					DependsOn:   VarField{Tag: "!expression", Val: "1 > 2"},
					Options: []VarField{
						VarField{Val: "test"}, VarField{Tag: "!expression", Val: "1 > 2"},
					},
					Pattern:      VarField{Val: "pat"},
					SaveInXlvals: VarField{Bool: true, Val: "true"},
					ReplaceAsIs:  VarField{Bool: false, Val: "false"},
				},
			},
			false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			blueprintDoc := &BlueprintYamlV2{
				ApiVersion: "",
				Kind:       "",
				Spec:       tt.spec,
			}
			got, err := blueprintDoc.parseParameters()
			if (err != nil) != tt.wantErr {
				t.Errorf("BlueprintYaml.parseParameters() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestBlueprintYaml_parseFiles(t *testing.T) {
	templatePath := "aws/monolith"
	tests := []struct {
		name    string
		fields  BlueprintYamlV2
		want    []TemplateConfig
		wantErr error
	}{
		{
			"parse a valid file declaration",
			BlueprintYamlV2{
				Spec: SpecV2{
					Files: []FileV2{
						{Path: "test.yaml"},
						{Path: "test2.yaml"},
					},
				},
			},
			[]TemplateConfig{
				{Path: "test.yaml", FullPath: filepath.Join(templatePath, "test.yaml")},
				{Path: "test2.yaml", FullPath: filepath.Join(templatePath, "test2.yaml")},
			},
			nil,
		},
		{
			"parse a valid file declaration with WriteIf that refers to existing variables",
			BlueprintYamlV2{
				Spec: SpecV2{
					Parameters: []ParameterV2{
						{Name: "foo", Type: "Confirm", Value: "true"},
						{Name: "bar", Type: "Confirm", Value: "false"},
					},
					Files: []FileV2{
						{Path: "test.yaml"},
						{Path: "test2.yaml", WriteIf: "foo"},
						{Path: "test3.yaml", WriteIf: yaml.CustomTag{Tag: "!expression", Value: "!bar"}},
						{Path: "test4.yaml", WriteIf: "bar"},
						{Path: "test5.yaml", WriteIf: yaml.CustomTag{Tag: "!expression", Value: "!foo"}},
					},
				},
			},
			[]TemplateConfig{
				{Path: "test.yaml", FullPath: filepath.Join(templatePath, "test.yaml")},
				{Path: "test2.yaml", FullPath: filepath.Join(templatePath, "test2.yaml"), DependsOn: VarField{Val: "foo", Tag: ""}},
				{Path: "test3.yaml", FullPath: filepath.Join(templatePath, "test3.yaml"), DependsOn: VarField{Val: "!bar", Tag: "!expression"}},
				{Path: "test4.yaml", FullPath: filepath.Join(templatePath, "test4.yaml"), DependsOn: VarField{Val: "bar", Tag: ""}},
				{Path: "test5.yaml", FullPath: filepath.Join(templatePath, "test5.yaml"), DependsOn: VarField{Val: "!foo", Tag: "!expression"}},
			},
			nil,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			blueprintDoc := &BlueprintYamlV2{
				ApiVersion: tt.fields.ApiVersion,
				Kind:       tt.fields.Kind,
				Metadata:   tt.fields.Metadata,
				Spec:       tt.fields.Spec,
			}
			tconfigs, err := blueprintDoc.parseFiles(templatePath, true)
			if tt.wantErr == nil || err == nil {
				assert.Equal(t, tt.wantErr, err)
			} else {
				assert.Equal(t, tt.wantErr.Error(), err.Error())
			}
			assert.Equal(t, tt.want, tconfigs)
		})
	}
}

func TestBlueprintYaml_parseIncludes(t *testing.T) {
	tests := []struct {
		name    string
		fields  BlueprintYamlV2
		want    []IncludedBlueprintProcessed
		wantErr error
	}{
		{
			"parse a valid include declaration with IncludeIf that is an expression",
			BlueprintYamlV2{
				Spec: SpecV2{
					IncludeAfter: []IncludedBlueprintV2{
						{
							Blueprint: "bar",
							IncludeIf: yaml.CustomTag{Tag: "!expression", Value: "1 > 2"},
						},
					},
					IncludeBefore: []IncludedBlueprintV2{
						{
							Blueprint: "foo",
							ParameterOverrides: []ParameterV2{
								{
									Name:     "foo",
									Value:    "bar",
									PromptIf: yaml.CustomTag{Tag: "!expression", Value: "1 > 2"},
								},
								{
									Name:     "bar",
									Value:    true,
									PromptIf: yaml.CustomTag{Tag: "!fn", Value: "foo"},
								},
								{
									Name:     "barr",
									Value:    10.5,
									PromptIf: yaml.CustomTag{Tag: "!fn", Value: "!foo"},
								},
							},
							FileOverrides: []FileV2{
								{
									Path:    "foo/bar.md",
									WriteIf: false,
								},
								{
									Path:    "foo/bar2.md",
									WriteIf: yaml.CustomTag{Tag: "!expression", Value: "1 > 2"},
								},
								{
									Path:     "foo/baar.md",
									RenameTo: "foo/baaar.md",
								},
								{
									Path:     "foo/baar2.md",
									RenameTo: yaml.CustomTag{Tag: "!expression", Value: "1 > 2 ? 'foo' : 'bar'"},
									WriteIf:  yaml.CustomTag{Tag: "!fn", Value: "foo"},
								},
							},
							IncludeIf: yaml.CustomTag{Tag: "!expression", Value: "1 > 2"},
						},
					},
				},
			},
			[]IncludedBlueprintProcessed{
				{
					Blueprint: "foo",
					Stage:     "before",
					ParameterOverrides: []Variable{
						{
							Name:      VarField{Val: "foo"},
							Value:     VarField{Val: "bar"},
							DependsOn: VarField{Val: "1 > 2", Tag: "!expression"},
						},
						{
							Name:      VarField{Val: "bar"},
							Value:     VarField{Val: "true", Bool: true},
							DependsOn: VarField{Tag: "!fn", Val: "foo"},
						},
						{
							Name:      VarField{Val: "barr"},
							Value:     VarField{Val: "10.500000"},
							DependsOn: VarField{Tag: "!fn", Val: "!foo"},
						},
					},
					FileOverrides: []TemplateConfig{
						{
							Path:      "foo/bar.md",
							DependsOn: VarField{Bool: false, Val: "false"},
						},
						{
							Path:      "foo/bar2.md",
							DependsOn: VarField{Tag: "!expression", Val: "1 > 2"},
						},
						{
							Path:     "foo/baar.md",
							RenameTo: VarField{Val: "foo/baaar.md"},
						},
						{
							Path:      "foo/baar2.md",
							RenameTo:  VarField{Tag: "!expression", Val: "1 > 2 ? 'foo' : 'bar'"},
							DependsOn: VarField{Tag: "!fn", Val: "foo"},
						},
					},
					DependsOn: VarField{Tag: "!expression", Val: "1 > 2"},
				},
				{
					Blueprint: "bar",
					Stage:     "after",
					DependsOn: VarField{Tag: "!expression", Val: "1 > 2"},
				},
			},
			nil,
		},
	}
	for _, tt := range tests {

		t.Run(tt.name, func(t *testing.T) {
			blueprintDoc := &BlueprintYamlV2{
				ApiVersion: tt.fields.ApiVersion,
				Kind:       tt.fields.Kind,
				Metadata:   tt.fields.Metadata,
				Spec:       tt.fields.Spec,
			}
			tconfigs, err := blueprintDoc.parseIncludes()
			if tt.wantErr == nil || err == nil {
				assert.Equal(t, tt.wantErr, err)
			} else {
				assert.Equal(t, tt.wantErr.Error(), err.Error())
			}
			assert.Equal(t, tt.want, tconfigs)
		})
	}
}

func TestParseFileV2(t *testing.T) {
	tests := []struct {
		name    string
		args    *FileV2
		want    TemplateConfig
		wantErr error
	}{
		{
			"return empty for empty map",
			&FileV2{},
			TemplateConfig{},
			nil,
		},
		{
			"parse a file declaration with only path",
			&FileV2{
				Path: "test.yaml",
			},
			TemplateConfig{Path: "test.yaml"},
			nil,
		},
		{
			"parse a file declaration with only path and nil for WriteIf",
			&FileV2{
				Path: "test.yaml", WriteIf: "",
			},
			TemplateConfig{Path: "test.yaml"},
			nil,
		},
		{
			"parse a file declaration with path and WriteIf",
			&FileV2{
				Path: "test.yaml", WriteIf: "foo",
			},
			TemplateConfig{Path: "test.yaml", DependsOn: VarField{Val: "foo"}},
			nil,
		},
		{
			"parse a file declaration with path and dependsOn",
			&FileV2{
				Path: "test.yaml", WriteIf: "foo",
			},
			TemplateConfig{Path: "test.yaml", DependsOn: VarField{Val: "foo"}},
			nil,
		},
		{
			"parse a file declaration with path and dependsOn as !fn tag",
			&FileV2{
				Path: "test.yaml", WriteIf: yaml.CustomTag{Tag: "!fn", Value: "aws.credentials().IsAvailable"},
			},
			TemplateConfig{Path: "test.yaml", DependsOn: VarField{Val: "aws.credentials().IsAvailable", Tag: "!fn"}},
			nil,
		},
		{
			"parse a file declaration with path and dependsOn as !expression tag",
			&FileV2{
				Path: "test.yaml", WriteIf: yaml.CustomTag{Tag: "!expression", Value: "1 > 2"},
			},
			TemplateConfig{Path: "test.yaml", DependsOn: VarField{Val: "1 > 2", Tag: "!expression"}},
			nil,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseFileV2(tt.args)
			if tt.wantErr == nil || err == nil {
				assert.Equal(t, tt.wantErr, err)
			} else {
				assert.Equal(t, tt.wantErr.Error(), err.Error())
			}
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestParseDependsOnValue(t *testing.T) {
	t.Run("should error when unknown function in DependsOn", func(t *testing.T) {
		v := Variable{
			Name:      VarField{Val: "test"},
			Type:      VarField{Val: TypeInput},
			DependsOn: VarField{Val: "aws.creds", Tag: "!fn"},
		}
		_, err := ParseDependsOnValue(v.DependsOn, &[]Variable{}, dummyData)
		require.NotNil(t, err)
	})
	t.Run("should return parsed bool value for DependsOn field from function", func(t *testing.T) {
		v := Variable{
			Name:      VarField{Val: "test"},
			Type:      VarField{Val: TypeInput},
			DependsOn: VarField{Val: "aws.credentials().IsAvailable", Tag: "!fn"},
		}
		out, err := ParseDependsOnValue(v.DependsOn, &[]Variable{}, dummyData)
		require.Nil(t, err)
		assert.Equal(t, true, out)
	})
	t.Run("should error when invalid expression in DependsOn", func(t *testing.T) {
		v := Variable{
			Name:      VarField{Val: "test"},
			Type:      VarField{Val: TypeInput},
			DependsOn: VarField{Val: "aws.creds", Tag: tagExpression},
		}
		_, err := ParseDependsOnValue(v.DependsOn, &[]Variable{}, dummyData)
		require.NotNil(t, err)
	})
	t.Run("should return parsed bool value for DependsOn field from expression", func(t *testing.T) {
		v := Variable{
			Name:      VarField{Val: "test"},
			Type:      VarField{Val: TypeInput},
			DependsOn: VarField{Val: "Foo > 10", Tag: tagExpression},
		}

		val, err := ParseDependsOnValue(v.DependsOn, &[]Variable{}, map[string]interface{}{
			"Foo": 100,
		})
		require.Nil(t, err)
		require.True(t, val)
	})
	t.Run("should return bool value from referenced var for dependsOn field", func(t *testing.T) {
		vars := make([]Variable, 2)
		vars[0] = Variable{
			Name:  VarField{Val: "confirm"},
			Type:  VarField{Val: TypeConfirm},
			Value: VarField{Bool: true, Val: "true"},
		}
		vars[1] = Variable{
			Name:      VarField{Val: "test"},
			Type:      VarField{Val: TypeInput},
			DependsOn: VarField{Val: "confirm"},
		}
		val, err := ParseDependsOnValue(vars[1].DependsOn, &vars, dummyData)
		require.Nil(t, err)
		assert.Equal(t, vars[0].Value.Bool, val)
	})
}

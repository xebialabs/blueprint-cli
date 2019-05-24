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
             label: testLabel
             type: SecretInput
             prompt: password?
           - name: test
             type: Input
             default: lala
             saveInXlvals: true
             prompt: help text
           - name: fn
             value: !fn aws.regions(ecs)[0]
           - name: select
             type: Select
             prompt: select region
             options:
             - !fn aws.regions(ecs)[0]
             - b
             - label: test
               value: testVal
             default: b
           - name: isit
             description: is it?
             value: true
           - name: isitnot
             prompt: negative question?
             type: Confirm
           - name: dep
             prompt: depends on others
             type: Input
             promptIf: !expr "isit && true"
           files:
           - path: xebialabs/foo.yaml
           - path: readme.md
             writeIf: isit
           - path: bar.md
             writeIf: isitnot
           - path: foo.md
             writeIf: !expr "!isitnot"
           includeBefore:
           - blueprint: kubernetes/gke-cluster
             parameterOverrides:
             - name: Foo
               value: hello
               promptIf: !expr "ExpTest1 == 'us-west' && AppName != 'foo' && TestDepends"
             - name: bar
               value: true
             fileOverrides:
             - path: xld-infrastructure.yml.tmpl
               writeIf: false
           includeAfter:
           - blueprint: kubernetes/namespace
             includeIf: !expr "ExpTest1 == 'us-west' && AppName != 'foo' && TestDepends"
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
                    prompt: what?
                    default: testing`,
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
                   default: testing`, models.BlueprintYamlFormatV2))
		_, err := parseTemplateMetadataV2(&metadata, templatePath, &blueprintRepository, true)
		require.NotNil(t, err)
		assert.Equal(t, "parameter Test must have a 'prompt' field", err.Error())
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
                    prompt: what?
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
                    prompt: what?
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
                   prompt: what?
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
                   prompt: what?
                   default: 1
                 - name: Test
                   type: Input
                   prompt: what?
                   default: 2
                 files:`, models.BlueprintYamlFormatV2))
		_, err := parseTemplateMetadataV2(&metadata, "aws/test", &blueprintRepository, true)
		require.NotNil(t, err)
		assert.Equal(t, "variable names must be unique within blueprint 'parameters' definition", err.Error())
	})

	t.Run("should parse different option types", func(t *testing.T) {
		metadata := []byte(
			fmt.Sprintf(`
               apiVersion: %s
               kind: Blueprint
               metadata:
               spec:
                 parameters:
                 - name: select
                   type: Select
                   prompt: select region
                   default: testVal
                   options:
                   - !fn aws.regions(ecs)[0]
                   - b
                   - label: test1
                     value: teeee
                   - label: test2
                     value: 10
                   - label: test3
                     value: 10.5
                   - label: test4
                     value: testVal`, models.BlueprintYamlFormatV2))
		doc, err := parseTemplateMetadataV2(&metadata, "aws/test", &blueprintRepository, true)
		require.Nil(t, err)
		assert.Equal(t, Variable{
			Name:   VarField{Value: "select"},
			Label:  VarField{Value: "select"},
			Type:   VarField{Value: TypeSelect},
			Prompt: VarField{Value: "select region"},
			Options: []VarField{
				{Value: "aws.regions(ecs)[0]", Tag: tagFn},
				{Value: "b"},
				{Value: "teeee", Label: "test1"},
				{Value: "10", Label: "test2"},
				{Value: "10.500000", Label: "test3"},
				{Value: "testVal", Label: "test4"},
			},
			Default: VarField{Value: "testVal"},
		}, doc.Variables[0])
	})

	t.Run("should error on invalid option types", func(t *testing.T) {
		metadata := []byte(
			fmt.Sprintf(`
               apiVersion: %s
               kind: Blueprint
               metadata:
               spec:
                 parameters:
                 - name: select
                   type: Select
                   prompt: select region
                   options:
                   - !fn aws.regions(ecs)[0]
                   - b
                   - key: test1
                     value: test
                   - label: test4
                     value: testVal`, models.BlueprintYamlFormatV2))
		_, err := parseTemplateMetadataV2(&metadata, "aws/test", &blueprintRepository, true)
		require.NotNil(t, err)
	})

	t.Run("should parse nested variables from valid metadata", func(t *testing.T) {
		doc, err := getValidTestBlueprintMetadata(templatePath, blueprintRepository)
		require.Nil(t, err)
		assert.Len(t, doc.Variables, 7)
		assert.Equal(t, Variable{
			Name:   VarField{Value: "pass"},
			Label:  VarField{Value: "testLabel"},
			Type:   VarField{Value: TypeSecret},
			Prompt: VarField{Value: "password?"},
		}, doc.Variables[0])
		assert.Equal(t, Variable{
			Name:         VarField{Value: "test"},
			Label:        VarField{Value: "test"},
			Type:         VarField{Value: TypeInput},
			Default:      VarField{Value: "lala"},
			Prompt:       VarField{Value: "help text"},
			SaveInXlvals: VarField{Bool: true, Value: "true"},
		}, doc.Variables[1])
		assert.Equal(t, Variable{
			Name:  VarField{Value: "fn"},
			Label: VarField{Value: "fn"},
			Value: VarField{Value: "aws.regions(ecs)[0]", Tag: tagFn},
		}, doc.Variables[2])
		assert.Equal(t, Variable{
			Name:   VarField{Value: "select"},
			Label:  VarField{Value: "select"},
			Type:   VarField{Value: TypeSelect},
			Prompt: VarField{Value: "select region"},
			Options: []VarField{
				{Value: "aws.regions(ecs)[0]", Tag: tagFn},
				{Value: "b"},
				{Value: "testVal", Label: "test"},
			},
			Default: VarField{Value: "b"},
		}, doc.Variables[3])
		assert.Equal(t, Variable{
			Name:        VarField{Value: "isit"},
			Label:       VarField{Value: "isit"},
			Description: VarField{Value: "is it?"},
			Value:       VarField{Bool: true, Value: "true"},
		}, doc.Variables[4])
		assert.Equal(t, Variable{
			Name:   VarField{Value: "isitnot"},
			Label:  VarField{Value: "isitnot"},
			Type:   VarField{Value: TypeConfirm},
			Prompt: VarField{Value: "negative question?"},
		}, doc.Variables[5])
		assert.Equal(t, Variable{
			Name:      VarField{Value: "dep"},
			Label:     VarField{Value: "dep"},
			Type:      VarField{Value: TypeInput},
			Prompt:    VarField{Value: "depends on others"},
			DependsOn: VarField{Value: "isit && true", Tag: tagExpressionV2},
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
			DependsOn: VarField{Value: "isit"},
		}, doc.TemplateConfigs[1])
		assert.Equal(t, TemplateConfig{
			Path:      "bar.md",
			FullPath:  "templatePath/test/bar.md",
			DependsOn: VarField{Value: "isitnot"},
		}, doc.TemplateConfigs[2])
		assert.Equal(t, TemplateConfig{
			Path:      "foo.md",
			FullPath:  "templatePath/test/foo.md",
			DependsOn: VarField{Value: "!isitnot", Tag: tagExpressionV2},
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
					Name:      VarField{Value: "Foo"},
					Value:     VarField{Value: "hello"},
					DependsOn: VarField{Tag: tagExpressionV2, Value: "ExpTest1 == 'us-west' && AppName != 'foo' && TestDepends"},
				},
				{
					Name:  VarField{Value: "bar"},
					Value: VarField{Value: "true", Bool: true},
				},
			},
			FileOverrides: []TemplateConfig{
				{
					Path:      "xld-infrastructure.yml.tmpl",
					DependsOn: VarField{Value: "false", Bool: false},
				},
			},
		}, doc.Include[0])
		assert.Equal(t, IncludedBlueprintProcessed{
			Blueprint: "kubernetes/namespace",
			Stage:     "after",
			ParameterOverrides: []Variable{
				{
					Name:  VarField{Value: "Foo"},
					Value: VarField{Value: "hello"},
				},
			},
			DependsOn: VarField{Tag: tagExpressionV2, Value: "ExpTest1 == 'us-west' && AppName != 'foo' && TestDepends"},
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
						Type:        TypeSecret,
						Description: "desc",
						Default:     "string2",
						PromptIf:    yaml.CustomTag{Tag: "!foo", Value: "1 > 2"},
						Options: []interface{}{
							"test", "foo", 10, 13.4,
						},
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
						Type:        TypeSecret,
						Description: "desc",
						Default:     "string2",
						Options: []interface{}{
							"test", "foo", true,
						},
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
						Label:       "testLabel",
						Type:        TypeSecret,
						Description: "desc",
						Prompt:      "desc?",
						Default:     "string2",
						PromptIf:    yaml.CustomTag{Tag: tagExpressionV2, Value: "1 > 2"},
						Options: []interface{}{
							"test", "foo", 10, 13.4,
						},
						SaveInXlvals: true,
						ReplaceAsIs:  true,
					},
					{
						Name:     "test",
						Type:     "Confirm",
						Prompt:   "desc",
						Default:  false,
						PromptIf: yaml.CustomTag{Tag: tagExpressionV2, Value: "1 > 2"},
						Options: []interface{}{
							"test", yaml.CustomTag{Tag: tagExpressionV2, Value: "1 > 2"},
						},
						SaveInXlvals: true,
					},
					{
						Name:  "test2",
						Value: true,
					},
				},
			},
			[]Variable{
				{
					Name:        VarField{Value: "test"},
					Label:       VarField{Value: "testLabel"},
					Type:        VarField{Value: TypeSecret},
					Prompt:      VarField{Value: "desc?"},
					Description: VarField{Value: "desc"},
					Default:     VarField{Value: "string2"},
					DependsOn:   VarField{Tag: tagExpressionV2, Value: "1 > 2"},
					Options: []VarField{
						VarField{Value: "test"}, VarField{Value: "foo"}, VarField{Value: "10"}, VarField{Value: "13.400000"},
					},
					SaveInXlvals: VarField{Bool: true, Value: "true"},
					ReplaceAsIs:  VarField{Bool: true, Value: "true"},
				},
				{
					Name:      VarField{Value: "test"},
					Label:     VarField{Value: "test"},
					Type:      VarField{Value: "Confirm"},
					Prompt:    VarField{Value: "desc"},
					Default:   VarField{Bool: false, Value: "false"},
					DependsOn: VarField{Tag: tagExpressionV2, Value: "1 > 2"},
					Options: []VarField{
						VarField{Value: "test"}, VarField{Tag: tagExpressionV2, Value: "1 > 2"},
					},
					SaveInXlvals: VarField{Bool: true, Value: "true"},
				},
				{
					Name:  VarField{Value: "test2"},
					Label: VarField{Value: "test2"},
					Value: VarField{Bool: true, Value: "true"},
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
						{Path: "test3.yaml", WriteIf: yaml.CustomTag{Tag: tagExpressionV2, Value: "!bar"}},
						{Path: "test4.yaml", WriteIf: "bar"},
						{Path: "test5.yaml", WriteIf: yaml.CustomTag{Tag: tagExpressionV2, Value: "!foo"}},
					},
				},
			},
			[]TemplateConfig{
				{Path: "test.yaml", FullPath: filepath.Join(templatePath, "test.yaml")},
				{Path: "test2.yaml", FullPath: filepath.Join(templatePath, "test2.yaml"), DependsOn: VarField{Value: "foo", Tag: ""}},
				{Path: "test3.yaml", FullPath: filepath.Join(templatePath, "test3.yaml"), DependsOn: VarField{Value: "!bar", Tag: tagExpressionV2}},
				{Path: "test4.yaml", FullPath: filepath.Join(templatePath, "test4.yaml"), DependsOn: VarField{Value: "bar", Tag: ""}},
				{Path: "test5.yaml", FullPath: filepath.Join(templatePath, "test5.yaml"), DependsOn: VarField{Value: "!foo", Tag: tagExpressionV2}},
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
							IncludeIf: yaml.CustomTag{Tag: tagExpressionV2, Value: "1 > 2"},
						},
					},
					IncludeBefore: []IncludedBlueprintV2{
						{
							Blueprint: "foo",
							ParameterOverrides: []ParameterV2{
								{
									Name:     "foo",
									Value:    "bar",
									PromptIf: yaml.CustomTag{Tag: tagExpressionV2, Value: "1 > 2"},
								},
								{
									Name:     "bar",
									Value:    true,
									PromptIf: yaml.CustomTag{Tag: tagFn, Value: "foo"},
								},
								{
									Name:     "barr",
									Value:    10.5,
									PromptIf: yaml.CustomTag{Tag: tagFn, Value: "!foo"},
								},
							},
							FileOverrides: []FileV2{
								{
									Path:    "foo/bar.md",
									WriteIf: false,
								},
								{
									Path:    "foo/bar2.md",
									WriteIf: yaml.CustomTag{Tag: tagExpressionV2, Value: "1 > 2"},
								},
								{
									Path:     "foo/baar.md",
									RenameTo: "foo/baaar.md",
								},
								{
									Path:     "foo/baar2.md",
									RenameTo: yaml.CustomTag{Tag: tagExpressionV2, Value: "1 > 2 ? 'foo' : 'bar'"},
									WriteIf:  yaml.CustomTag{Tag: tagFn, Value: "foo"},
								},
							},
							IncludeIf: yaml.CustomTag{Tag: tagExpressionV2, Value: "1 > 2"},
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
							Name:      VarField{Value: "foo"},
							Value:     VarField{Value: "bar"},
							DependsOn: VarField{Value: "1 > 2", Tag: tagExpressionV2},
						},
						{
							Name:      VarField{Value: "bar"},
							Value:     VarField{Value: "true", Bool: true},
							DependsOn: VarField{Tag: tagFn, Value: "foo"},
						},
						{
							Name:      VarField{Value: "barr"},
							Value:     VarField{Value: "10.500000"},
							DependsOn: VarField{Tag: tagFn, Value: "!foo"},
						},
					},
					FileOverrides: []TemplateConfig{
						{
							Path:      "foo/bar.md",
							DependsOn: VarField{Bool: false, Value: "false"},
						},
						{
							Path:      "foo/bar2.md",
							DependsOn: VarField{Tag: tagExpressionV2, Value: "1 > 2"},
						},
						{
							Path:     "foo/baar.md",
							RenameTo: VarField{Value: "foo/baaar.md"},
						},
						{
							Path:      "foo/baar2.md",
							RenameTo:  VarField{Tag: tagExpressionV2, Value: "1 > 2 ? 'foo' : 'bar'"},
							DependsOn: VarField{Tag: tagFn, Value: "foo"},
						},
					},
					DependsOn: VarField{Tag: tagExpressionV2, Value: "1 > 2"},
				},
				{
					Blueprint: "bar",
					Stage:     "after",
					DependsOn: VarField{Tag: tagExpressionV2, Value: "1 > 2"},
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
			TemplateConfig{Path: "test.yaml", DependsOn: VarField{Value: "foo"}},
			nil,
		},
		{
			"parse a file declaration with path and dependsOn",
			&FileV2{
				Path: "test.yaml", WriteIf: "foo",
			},
			TemplateConfig{Path: "test.yaml", DependsOn: VarField{Value: "foo"}},
			nil,
		},
		{
			"parse a file declaration with path and dependsOn as !fn tag",
			&FileV2{
				Path: "test.yaml", WriteIf: yaml.CustomTag{Tag: tagFn, Value: "aws.credentials().IsAvailable"},
			},
			TemplateConfig{Path: "test.yaml", DependsOn: VarField{Value: "aws.credentials().IsAvailable", Tag: tagFn}},
			nil,
		},
		{
			"parse a file declaration with path and dependsOn as !expr tag",
			&FileV2{
				Path: "test.yaml", WriteIf: yaml.CustomTag{Tag: tagExpressionV2, Value: "1 > 2"},
			},
			TemplateConfig{Path: "test.yaml", DependsOn: VarField{Value: "1 > 2", Tag: tagExpressionV2}},
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
			Name:      VarField{Value: "test"},
			Label:     VarField{Value: "test"},
			Type:      VarField{Value: TypeInput},
			DependsOn: VarField{Value: "aws.creds", Tag: tagFn},
		}
		_, err := ParseDependsOnValue(v.DependsOn, &[]Variable{}, dummyData)
		require.NotNil(t, err)
	})
	t.Run("should return parsed bool value for DependsOn field from function", func(t *testing.T) {
		v := Variable{
			Name:      VarField{Value: "test"},
			Label:     VarField{Value: "test"},
			Type:      VarField{Value: TypeInput},
			DependsOn: VarField{Value: "aws.credentials().IsAvailable", Tag: tagFn},
		}
		out, err := ParseDependsOnValue(v.DependsOn, &[]Variable{}, dummyData)
		require.Nil(t, err)
		assert.Equal(t, true, out)
	})
	t.Run("should error when invalid expression in DependsOn", func(t *testing.T) {
		v := Variable{
			Name:      VarField{Value: "test"},
			Label:     VarField{Value: "test"},
			Type:      VarField{Value: TypeInput},
			DependsOn: VarField{Value: "aws.creds", Tag: tagExpressionV2},
		}
		_, err := ParseDependsOnValue(v.DependsOn, &[]Variable{}, dummyData)
		require.NotNil(t, err)
	})
	t.Run("should return parsed bool value for DependsOn field from expression", func(t *testing.T) {
		v := Variable{
			Name:      VarField{Value: "test"},
			Label:     VarField{Value: "test"},
			Type:      VarField{Value: TypeInput},
			DependsOn: VarField{Value: "Foo > 10", Tag: tagExpressionV2},
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
			Name:  VarField{Value: "confirm"},
			Label: VarField{Value: "confirm"},
			Type:  VarField{Value: TypeConfirm},
			Value: VarField{Bool: true, Value: "true"},
		}
		vars[1] = Variable{
			Name:      VarField{Value: "test"},
			Label:     VarField{Value: "test"},
			Type:      VarField{Value: TypeInput},
			DependsOn: VarField{Value: "confirm"},
		}
		val, err := ParseDependsOnValue(vars[1].DependsOn, &vars, dummyData)
		require.Nil(t, err)
		assert.Equal(t, vars[0].Value.Bool, val)
	})
}

func TestVariable_validate(t *testing.T) {
	tests := []struct {
		name   string
		fields Variable
		errMsg string
	}{
		{
			"should error on validation failure for a parameter with missing name field",
			Variable{
				Type: VarField{Value: TypeInput},
			},
			"parameter must have a 'name' field",
		},
		{
			"should error on validation failure for a parameter with missing prompt field",
			Variable{
				Name: VarField{Value: "test"},
				Type: VarField{Value: TypeInput},
			},
			"parameter test must have a 'prompt' field",
		},
		{
			"should error on validation failure for a parameter with missing type field",
			Variable{
				Name:   VarField{Value: "test"},
				Prompt: VarField{Value: "??"},
			},
			"parameter test must have a 'type' field",
		},
		{
			"should validate without error for a valid parameter",
			Variable{
				Name:    VarField{Value: "test"},
				Type:    VarField{Value: TypeInput},
				Prompt:  VarField{Value: "??"},
				Default: VarField{Value: "hello"},
			},
			"",
		},
		{
			"should error on validation failure for a parameter with value which has invalid field 'prompt' set",
			Variable{
				Name:   VarField{Value: "test"},
				Value:  VarField{Value: "val"},
				Prompt: VarField{Value: "??"},
			},
			"parameter test must not have a 'prompt' field when field 'value' is set",
		},
		{
			"should error on validation failure for a parameter with value which has invalid field 'default' set",
			Variable{
				Name:    VarField{Value: "test"},
				Value:   VarField{Value: "val"},
				Default: VarField{Value: "hello"},
			},
			"parameter test must not have a 'default' field when field 'value' is set",
		},
		{
			"should error on validation failure for a parameter with value which has invalid field 'options' set",
			Variable{
				Name:    VarField{Value: "test"},
				Value:   VarField{Value: "val"},
				Options: []VarField{{Value: "test"}},
			},
			"parameter test must not have a 'options' field when field 'value' is set",
		},
		{
			"should error on validation failure for a parameter with value which has invalid field 'promptIf' set",
			Variable{
				Name:      VarField{Value: "test"},
				Value:     VarField{Value: "val"},
				DependsOn: VarField{Value: "test"},
			},
			"parameter test must not have a 'promptIf' field when field 'value' is set",
		},
		{
			"should validate without error for a valid parameter used as constant",
			Variable{
				Name:  VarField{Value: "test"},
				Value: VarField{Value: "hello"},
			},
			"",
		},
		{
			"should error on validation failure for a parameter of TypeInput which has invalid field 'replaceAsIs' set",
			Variable{
				Name:        VarField{Value: "test"},
				Type:        VarField{Value: TypeInput},
				Prompt:      VarField{Value: "test"},
				ReplaceAsIs: VarField{Value: "true", Bool: true},
			},
			"parameter test must not have a 'replaceAsIs' field when field 'type=SecretInput' is set",
		},
		{
			"should validate without error for a parameter of TypeSecret which has 'replaceAsIs' & 'revealOnSummary' set",
			Variable{
				Name:            VarField{Value: "test"},
				Type:            VarField{Value: TypeSecret},
				Prompt:          VarField{Value: "test"},
				ReplaceAsIs:     VarField{Value: "true", Bool: true},
				RevealOnSummary: VarField{Value: "true", Bool: true},
			},
			"",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			variable := &Variable{
				Name:            tt.fields.Name,
				Type:            tt.fields.Type,
				Value:           tt.fields.Value,
				Prompt:          tt.fields.Prompt,
				Default:         tt.fields.Default,
				DependsOn:       tt.fields.DependsOn,
				Options:         tt.fields.Options,
				SaveInXlvals:    tt.fields.SaveInXlvals,
				ReplaceAsIs:     tt.fields.ReplaceAsIs,
				RevealOnSummary: tt.fields.RevealOnSummary,
				Validate:        tt.fields.Validate,
				Description:     tt.fields.Description,
				Label:           tt.fields.Label,
			}
			err := variable.validate()
			if tt.errMsg != "" {
				require.NotNil(t, err)
				assert.Equal(t, tt.errMsg, err.Error())
			} else {
				require.Nil(t, err)
			}
		})
	}
}

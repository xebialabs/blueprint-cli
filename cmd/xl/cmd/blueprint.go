package cmd

import (
	"net/url"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/xebialabs/xl-cli/pkg/xl"
)

var blueprintCmd = &cobra.Command{
	Use:   "blueprint",
	Short: "Create a Blueprint",
	Long:  `Create a Blueprint for XL Platform Releases and Deployments`,
	Run: func(cmd *cobra.Command, args []string) {
		context, err := xl.BuildContext(viper.GetViper(), nil, nil)
		if err != nil {
			xl.Fatal("Error while reading configuration: %s\n", err)
		}
		if xl.IsVerbose {
			context.PrintConfiguration()
		}

		DoBlueprint(context)
	},
}

var blueprintTemplate string
var templateRegistry string

// DoBlueprint creates blueprint templates
func DoBlueprint(context *xl.Context) {
	templateRegistries := context.TemplateRegistries
	templateRegistryURL, err := url.ParseRequestURI(templateRegistry)
	if err != nil {
		xl.Fatal("Invalid template-registry URL: %s\n", err)
	}
	if templateRegistry != "" && !registryExists(*templateRegistryURL, templateRegistries) {
		xl.Verbose("appending registry from CLI flag %s\n", templateRegistry)
		templateRegistries = append(templateRegistries, xl.TemplateRegistry{
			Name: "adhoc", URL: *templateRegistryURL,
		})
	}

	err = xl.CreateBlueprint(blueprintTemplate, templateRegistries)
	if err != nil {
		xl.Fatal("Error while creating Blueprint: %s\n", err)
	}
}

func registryExists(templateRegistry url.URL, templateRegistries []xl.TemplateRegistry) bool {
	for _, v := range templateRegistries {
		if v.URL == templateRegistry {
			return true
		}
	}
	return false
}

func init() {
	rootCmd.AddCommand(blueprintCmd)

	blueprintFlags := blueprintCmd.Flags()
	blueprintFlags.StringVarP(&blueprintTemplate, "template", "t", "", "The template path/url to use")
	blueprintFlags.StringVarP(&templateRegistry, "template-registry", "r", "https://s3.amazonaws.com/xl-cli/blueprints", "Registry URL for Blueprint templates")
}

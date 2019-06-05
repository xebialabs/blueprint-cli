package cmd

import (
	"fmt"
	"github.com/xebialabs/xl-cli/pkg/models"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/xebialabs/xl-cli/pkg/util"
	"github.com/xebialabs/xl-cli/pkg/xl"
)

var schemaFilename string
var generateXld bool
var generateXlr bool
var override bool

var ideCmd = &cobra.Command{
	Use:   "ide",
	Short: "IDE commands",
	Long:  `IDE commands`,
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("Please use a subcommand like for example: `xl ide schema`")
	},
}

var schemaCmd = &cobra.Command{
	Use:   "schema",
	Short: "Generate a schema to be used with your IDE",
	Long:  `Generate a schema to be used with your IDE. This schema can be used together with VSCode and the Devops As Code by XebiaLabs extension.`,
	Run: func(cmd *cobra.Command, args []string) {
		if !(generateXld || generateXlr) {
			util.Fatal("Error missing product flags. You need to specify a product you want to generate a schema for. " +
				"Try adding --xl-deploy or --xl-release or both.\n")
		}
		context, err := xl.BuildContext(viper.GetViper(), nil, []string{}, nil, CliVersion)
		if err != nil {
			util.Fatal("Error while reading configuration: %s\n", err)
		}
		if util.IsVerbose {
			context.PrintConfiguration()
		}
		DoGenerateSchema(context)
	},
}

func DoGenerateSchema(context *xl.Context) {
	err := context.GenerateSchema(schemaFilename, generateXld, generateXlr, override)
	if err != nil {
		util.Fatal("Error while generating schema: %s\n", err)
	}
	util.Info("Done\n")
}

func init() {
	rootCmd.AddCommand(ideCmd)
	ideCmd.AddCommand(schemaCmd)
	schemaFlags := schemaCmd.Flags()
	schemaFlags.StringVarP(&schemaFilename, "file", "f", "schema.json", "Path of the file where the generated schema file will be stored")
	schemaFlags.BoolVarP(&generateXld, string(models.XLD), "d", false, "Set to true to generate schema for XL Deploy")
	schemaFlags.BoolVarP(&generateXlr, string(models.XLR), "r", false, "Set to true to generate schema for XL Release")
	schemaFlags.BoolVarP(&override, "override", "o", false, "Set to true to override the generated file")
}

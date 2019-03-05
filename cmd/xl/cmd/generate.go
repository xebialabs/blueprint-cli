package cmd

import (
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/xebialabs/xl-cli/pkg/models"
	"github.com/xebialabs/xl-cli/pkg/util"
	"github.com/xebialabs/xl-cli/pkg/xl"
)

var generateFilename string
var generatePath string
var generateServer string
var generateOverride bool
var globalPermissions bool
var users bool
var roles bool

var generateCmd = &cobra.Command{
	Use:   "generate",
	Short: "Generate configuration",
	Long:  `Generate configuration`,
	Run: func(cmd *cobra.Command, args []string) {
		context, err := xl.BuildContext(viper.GetViper(), nil, []string{})
		if err != nil {
			util.Fatal("Error while reading configuration: %s\n", err)
		}
		if util.IsVerbose {
			context.PrintConfiguration()
		}

		DoGenerate(context)
	},
}

func DoGenerate(context *xl.Context) {
	err := context.GenerateSingleDocument(generateServer, generateFilename, generatePath, generateOverride, globalPermissions, users, roles)
	if err != nil {
		util.Fatal("Error while generating document: %s\n", err)
	}
}

func init() {
	rootCmd.AddCommand(generateCmd)

	generateFlags := generateCmd.Flags()
	generateFlags.StringVarP(&generateFilename, "file", "f", "", "Path of the file where the generated yaml file will be stored (required)")
	generateCmd.MarkFlagRequired("file")
	generateFlags.StringVarP(&generatePath, "path", "p", "", "Server path which will be used for definitions generation")
	generateFlags.StringVarP(&generateServer, "server", "s", string(models.XLD), "Which server to generate from, either \"xl-deploy\" or \"xl-release\"")
	generateFlags.BoolVarP(&generateOverride, "override", "o", false, "Set to true to override the generated file")
	generateFlags.BoolVarP(&globalPermissions, "globalPermissions", "g", false, "Add to the generated file all the global permissions")
	generateFlags.BoolVarP(&users, "users", "u", false, "Add to the generated file all the users in system")
	generateFlags.BoolVarP(&roles, "roles", "r", false, "Add to the generated file all the roles in system")
}

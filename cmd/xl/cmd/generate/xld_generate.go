package generate

import (
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/xebialabs/xl-cli/pkg/util"
	"github.com/xebialabs/xl-cli/pkg/xl"
)

var xldGenerateFilename string
var xldGeneratePath string
var xldGenerateOverride bool
var xldGlobalPermissions bool
var xldUsers bool
var xldRoles bool
var xldIncludeSecrets bool
var xldIncludeDefaults bool

var GenerateCmdXLD = &cobra.Command{
	Use:   "xl-deploy",
	Short: "xl-deploy configuration generator",
	Long:  `Generate configuration for XL Deploy`,
	Run: func(cmd *cobra.Command, args []string) {
		context, err := xl.BuildContext(viper.GetViper(), nil, []string{}, nil, "")
		if err != nil {
			util.Fatal("Error while reading configuration: %s\n", err)
		}
		if util.IsVerbose {
			context.PrintConfiguration()
		}

		doGenerateXld(context)
	},
}

func doGenerateXld(context *xl.Context) {
	err := context.GenerateSingleXLDDocument(xldGenerateFilename, xldGeneratePath, xldGenerateOverride, xldGlobalPermissions, xldUsers, xldRoles, xldIncludeSecrets, xldIncludeDefaults)
	if err != nil {
		util.Fatal("Error while generating document: %s\n", err)
	}
}

func init() {
	generateFlags := GenerateCmdXLD.Flags()
	generateFlags.StringVarP(&xldGenerateFilename, "file", "f", "", "Path of the file where the generated yaml file will be stored (required)")
	err := GenerateCmdXLD.MarkFlagRequired("file")
	if err != nil {
		util.Fatal("Error while generating document: %s\n", err)
	}
	generateFlags.StringVarP(&xldGeneratePath, "path", "p", "", "Server path which will be used for definitions generation")
	generateFlags.BoolVarP(&xldGenerateOverride, "override", "o", false, "Set to true to override the generated file")
	generateFlags.BoolVarP(&xldGlobalPermissions, "globalPermissions", "g", false, "Add to the generated file all the global permissions")
	generateFlags.BoolVarP(&xldUsers, "users", "u", false, "Add to the generated file all the users in system")
	generateFlags.BoolVarP(&xldRoles, "roles", "r", false, "Add to the generated file all the roles in system")
	generateFlags.BoolVarP(&xldIncludeSecrets, "secrets", "s", false, "Generate a secrets.xlvals file that contains all secrets. (Requires ADMIN permissions)")
	generateFlags.BoolVarP(&xldIncludeDefaults, "defaults", "d", false, "Include properties that have default values. (Only works for XL Deploy)")
}

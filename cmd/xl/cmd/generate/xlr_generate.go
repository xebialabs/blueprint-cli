package generate

import (
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/xebialabs/xl-cli/pkg/util"
	"github.com/xebialabs/xl-cli/pkg/xl"
)

var xlrEnvironments bool
var xlrApplications bool
var xlrGenerateFilename string
var xlrGeneratePath string
var xlrGenerateOverride bool
var xlrUsers bool
var xlrRoles bool
var xlrIncludeSecrets bool
var xlrCiName string
var xlrTemplates bool
var xlrDeliveryPatterns bool
var xlrDashboards bool
var xlrConfigurations bool
var xlrPermissions bool
var xlrRiskProfiles bool

var GenerateCmdXLR = &cobra.Command{
	Use:   "xl-release",
	Short: "xl-release configuration generator",
	Long:  `Generate configuration for XL Release`,
	Run: func(cmd *cobra.Command, args []string) {
		context, err := xl.BuildContext(viper.GetViper(), nil, []string{}, nil, "")
		if err != nil {
			util.Fatal("Error while reading configuration: %s\n", err)
		}
		if util.IsVerbose {
			context.PrintConfiguration()
		}
		if cmd.Flags().Changed("path") && len(xlrGeneratePath) == 0 {
			util.Fatal("Path parameter cannot be empty\n")
		}
		if cmd.Flags().Changed("name") && len(xlrCiName) == 0 {
			util.Fatal("Name parameter cannot be empty\n")
		}

		doGenerateXlr(context)
	},
}

func doGenerateXlr(context *xl.Context) {
	err := context.GenerateSingleXLRDocument(
		xlrGenerateFilename,
		xlrGeneratePath,
		xlrGenerateOverride,
		xlrUsers,
		xlrRoles,
		xlrEnvironments,
		xlrApplications,
		xlrIncludeSecrets,
		xlrCiName,
		xlrTemplates,
		xlrDeliveryPatterns,
		xlrDashboards,
		xlrConfigurations,
		xlrPermissions,
		xlrRiskProfiles,
	)
	if err != nil {
		util.Fatal("Error while generating document: %s\n", err)
	}
}

func init() {
	generateFlags := GenerateCmdXLR.Flags()
	generateFlags.StringVarP(&xlrGenerateFilename, "file", "f", "", "Path of the file where the generated yaml file will be stored (required)")
	err := GenerateCmdXLR.MarkFlagRequired("file")
	if err != nil {
		util.Fatal("Error while generating document: %s\n", err)
	}
	generateFlags.StringVarP(&xlrGeneratePath, "path", "p", "", "Server folder path which will be used for definitions generation")
	generateFlags.StringVarP(&xlrCiName, "name", "n", "", "Server entity name which will be used to search for definitions generation")
	generateFlags.BoolVarP(&xlrGenerateOverride, "override", "o", false, "Set to true to override the generated file")
	generateFlags.BoolVarP(&xlrUsers, "users", "u", false, "Add to the generated file all the users in system")
	generateFlags.BoolVarP(&xlrRoles, "roles", "r", false, "Add to the generated file all the roles in system")
	generateFlags.BoolVarP(&xlrIncludeSecrets, "secrets", "s", false, "Generate a secrets.xlvals file that contains all secrets. (Requires ADMIN permissions)")
	generateFlags.BoolVarP(&xlrEnvironments, "environments", "e", false, "Add to the generated file all environments in system")
	generateFlags.BoolVarP(&xlrApplications, "applications", "a", false, "Add to the generated file all the applications in system")
	generateFlags.BoolVarP(&xlrTemplates, "templates", "t", false, "Add to the generated file all the templates in system")
	generateFlags.BoolVarP(&xlrDeliveryPatterns, "deliveryPatterns", "", false, "Add to the generated file all the delivery patterns in system")
	generateFlags.BoolVarP(&xlrDashboards, "dashboards", "d", false, "Add to the generated file all the dashboards in system")
	generateFlags.BoolVarP(&xlrConfigurations, "configurations", "c", false, "Add to the generated file all the configurations in system")
	generateFlags.BoolVarP(&xlrPermissions, "permissions", "m", false, "Add to the generated file all the permissions in system")
	generateFlags.BoolVarP(&xlrRiskProfiles, "riskProfiles", "k", false, "Add to the generated file all the risk profiles in system")
}

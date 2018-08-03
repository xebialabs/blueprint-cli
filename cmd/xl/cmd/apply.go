package cmd

import (
	"github.com/spf13/cobra"
	"github.com/xebialabs/xl-cli/internal/app/xl"
)

// applyCmd represents the apply command
var applyCmd = &cobra.Command{
	Use:   "apply",
	Short: "Apply configuration changes",
	Long: `Apply configuration changes to XebiaLabs products from YAML files.
Flag url takes precedence over xld and xlr. Setting url disregards xld and xlr.
Omitting xld and xlr defaults to using "default" as the server name.
The configuration of these preconfigured default servers is possible with the login command.`,
	Run: func(cmd *cobra.Command, args []string) {
		xl.Apply(filename, xld, xlr)
	},
}

func init() {
	rootCmd.AddCommand(applyCmd)

	setServerNameFlags(applyCmd.Flags())
	setFilenameFlags(applyCmd.Flags())
	applyCmd.MarkFlagRequired("filename")
}

package cmd

import (
	"github.com/spf13/cobra"
	"github.com/xebialabs/xl-cli/internal/app/xl"
)

// applyCmd represents the apply command
var applyCmd = &cobra.Command{
	Use:   "apply",
	Short: "Apply configuration changes",
	Long:  `Apply configuration changes to XebiaLabs products from YAML files.`,
	Run: func(cmd *cobra.Command, args []string) {
		xl.Apply(filename, url)
	},
}

func init() {
	rootCmd.AddCommand(applyCmd)

	applyCmd.Flags().StringArrayVarP(&filename, "filename", "f", []string{}, "Filename that contains the configuration change eg. xld.yaml")
}

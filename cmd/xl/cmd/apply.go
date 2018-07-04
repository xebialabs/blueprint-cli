package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

var filename []string

// applyCmd represents the apply command
var applyCmd = &cobra.Command{
	Use:   "apply",
	Short: "Apply a configuration change",
	Long:  `Apply a configuration change to XebiaLabs products from a YAML file.`,
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("apply called. args:", args, "filename:", filename)
	},
}

func init() {
	rootCmd.AddCommand(applyCmd)

	applyCmd.Flags().StringArrayVarP(&filename, "filename", "f", []string{}, "Filename that contains the configuration change eg. ./xld.yaml")
}

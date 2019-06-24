package cmd

import (
	"github.com/spf13/cobra"
	"github.com/xebialabs/xl-cli/cmd/xl/cmd/generate"
)

var generateCmd = &cobra.Command{
	Use:   "generate",
	Short: "Generate configuration",
	Long: `Generate configuration for XL Product.
The generate command has changed. There are two sub-commands for xl-deploy and xl-release.
For example, if you want to generate xl-release configurations and templates inside a folder, now you can use the following command:

xl generate xl-release --templates --configurations -p your/path/to/your/folder -f filename.yml`,
}

func init() {
	rootCmd.AddCommand(generateCmd)
	generateCmd.AddCommand(generate.GenerateCmdXLR)
	generateCmd.AddCommand(generate.GenerateCmdXLD)
}

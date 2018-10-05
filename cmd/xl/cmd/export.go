package cmd

import (
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/xebialabs/xl-cli/pkg/xl"
)

var exportFilename string
var exportPath string
var exportServer string
var exportOverride bool

var exportCmd = &cobra.Command{
	Use:   "export",
	Short: "Export configuration",
	Long:  `Export configuration`,
	Run: func(cmd *cobra.Command, args []string) {
		context, err := xl.BuildContext(viper.GetViper(), nil, nil)
		if err != nil {
			xl.Fatal("Error while reading configuration: %s\n", err)
		}
		if xl.IsVerbose {
			context.PrintConfiguration()
		}

		DoExport(context)
	},
}

func DoExport(context *xl.Context) {
	if exportPath == "" {
		xl.Fatal("Please provide a path to export\n")
	}

	err := context.ExportSingleDocument(exportServer, exportFilename, exportPath, exportOverride)
	if err != nil {
		xl.Fatal("Error while exporting document: %s\n", err)
	}
}

func init() {
	rootCmd.AddCommand(exportCmd)

	exportFlags := exportCmd.Flags()
	exportFlags.StringVarP(&exportFilename, "file", "f", "export.zip", "Path of the file where the export Zip archive will be stored")
	exportFlags.StringVarP(&exportPath, "path", "p", "", "Server path which will be exported")
	exportFlags.StringVarP(&exportServer, "server", "s", "xl-deploy", "Which server to export from, either \"xl-deploy\" or \"xl-release\"")
	exportFlags.BoolVarP(&exportOverride, "override", "o", false, "Set to true to override the export file")
}

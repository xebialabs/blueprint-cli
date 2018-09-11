package cmd

import (
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/xebialabs/xl-cli/pkg/xl"
	"io"
	"os"
	"path/filepath"
)

var applyFilenames []string

var applyCmd = &cobra.Command{
	Use:   "apply",
	Short: "Apply configuration changes",
	Long:  `Apply configuration changes`,
	Run: func(cmd *cobra.Command, args []string) {
		context, err := xl.BuildContext(viper.GetViper())
		if err != nil {
			xl.Fatal("Error while reading configuration: %s\n", err)
		}
		xl.Verbose("Using configuration:\n %v\n", viper.AllSettings())

		DoApply(context, applyFilenames)
	},
}

func DoApply(context *xl.Context, applyFilenames []string) {
	if len(applyFilenames) == 0 {
		xl.Fatal("Please provide a yaml file to apply\n")
	}

	for _, applyFilename := range applyFilenames {
		applyDir := filepath.Dir(applyFilename)
		xl.Verbose("using relative directory `%s` for file references\n", applyDir)
		reader, err := os.Open(applyFilename)
		if err != nil {
			xl.Fatal("Error while opening file %s: %s\n", applyFilename, err)
		}
		docReader := xl.NewDocumentReader(reader)
		for {
			doc, err := docReader.ReadNextYamlDocument()
			if err != nil {
				if err == io.EOF {
					xl.Info("Done with file %s\n", applyFilename)
					break
				} else {
					xl.Fatal("Error while reading yaml document %s: %s\n", applyFilename, err)
				}
			}
			xl.Info("Applying %s\n", doc.Kind)
			err = context.ProcessSingleDocument(doc, applyDir)
			if err != nil {
				xl.Fatal("Error while processing yaml document %s: %s\n", applyFilename, err)
			}
		}
		reader.Close()
	}
}

func init() {
	rootCmd.AddCommand(applyCmd)

	applyFlags := applyCmd.Flags()
	applyFlags.StringArrayVarP(&applyFilenames, "file", "f", []string{}, "Path(s) to the file(s) to apply")
}

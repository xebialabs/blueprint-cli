package cmd

import (
	"io"
	"os"
	"path/filepath"
	"regexp"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/pkg/errors"
	"github.com/xebialabs/xl-cli/pkg/xl"
)

var applyFilenames []string
var applyValues map[string]string
var applySecrets map[string]string

var applyCmd = &cobra.Command{
	Use:   "apply",
	Short: "Apply configuration changes",
	Long:  `Apply configuration changes`,
	Run: func(cmd *cobra.Command, args []string) {
		context, err := xl.BuildContext(viper.GetViper(), &applyValues, &applySecrets)
		if err != nil {
			xl.Fatal("Error while reading configuration: %s\n", err)
		}
		if xl.IsVerbose {
			context.PrintConfiguration()
		}

		DoApply(context, applyFilenames)
	},
}

func DoApply(context *xl.Context, applyFilenames []string) {
	if len(applyFilenames) == 0 {
		xl.Fatal("No XL YAML file to apply\n")
	}

	for _, applyFilename := range applyFilenames {
		xl.StartProgress(applyFilename)

		applyDir := filepath.Dir(applyFilename)
		reader, err := os.Open(applyFilename)
		if err != nil {
			xl.Fatal("Error while opening XL YAML file %s: %s\n", applyFilename, err)
		}

		docReader := xl.NewDocumentReader(reader)
		for {
			doc, err := docReader.ReadNextYamlDocument()
			if err != nil {
				if err == io.EOF {
					break
				} else {
					reportFatalDocumentError(applyFilename, doc, err)
				}
			}

			xl.UpdateProgressStartDocument(applyFilename, doc)
			err = context.ProcessSingleDocument(doc, applyDir)
			if err != nil {
				reportFatalDocumentError(applyFilename, doc, err)
			}
			xl.UpdateProgressEndDocument()
		}

		reader.Close()
		xl.EndProgress()
	}

}

var isFieldAlreadySetErrorRegexp = regexp.MustCompile(`field \w+ already set in type`)

func reportFatalDocumentError(applyFilename string, doc *xl.Document, err error) {
	if isFieldAlreadySetErrorRegexp.MatchString(err.Error()) {
		err = errors.Wrap(err, "Possible missing triple dash (---) to separate multiple YAML documents")
	}

	xl.Fatal("Error while processing YAML document at line %d of XL YAML file %s: %s\n", doc.Line, applyFilename, err)
}

func init() {
	rootCmd.AddCommand(applyCmd)

	applyFlags := applyCmd.Flags()
	applyFlags.StringArrayVarP(&applyFilenames, "file", "f", []string{}, "Path(s) to the file(s) to apply")
	applyFlags.StringToStringVar(&applyValues, "values", map[string]string{}, "Values")
	applyFlags.StringToStringVar(&applySecrets, "secrets", map[string]string{}, "Secret values")
}

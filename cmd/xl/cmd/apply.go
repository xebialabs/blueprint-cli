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

var applyCmd = &cobra.Command{
	Use:   "apply",
	Short: "Apply configuration changes",
	Long: `Apply configuration changes to XebiaLabs products from YAML files.
Flag url takes precedence over xld and xlr. Setting url disregards xld and xlr.
Omitting xld and xlr defaults to using "default" as the server name.
The configuration of these preconfigured default servers is possible with the login command.`,
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
	applyFlags.String("xl-deploy-url", "http://localhost:4516/", "URL to access the XL Deploy server")
	applyFlags.String("xl-deploy-username", "admin", "Username to access the XL Deploy server")
	applyFlags.String("xl-deploy-password", "admin", "Password to access the XL Deploy server")
	viper.BindPFlag("xl-deploy.url", applyFlags.Lookup("xl-deploy-url"))
	viper.BindPFlag("xl-deploy.username", applyFlags.Lookup("xl-deploy-username"))
	viper.BindPFlag("xl-deploy.password", applyFlags.Lookup("xl-deploy-password"))

	applyFlags.String("xl-release-url", "http://localhost:5516/", "URL to access the XL Release server")
	applyFlags.String("xl-release-username", "admin", "Username to access the XL Release server")
	applyFlags.String("xl-release-password", "admin", "Password to access the XL Release server")
	viper.BindPFlag("xl-release.url", applyFlags.Lookup("xl-release-url"))
	viper.BindPFlag("xl-release.username", applyFlags.Lookup("xl-release-username"))
	viper.BindPFlag("xl-release.password", applyFlags.Lookup("xl-release-password"))
}

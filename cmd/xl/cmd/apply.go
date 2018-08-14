package cmd

import (
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/xebialabs/xl-cli/internal/pkg/lib"
	"os"
	"io"
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
		context, err := lib.BuildContext(viper.GetViper())
		if err != nil {
			lib.Fatal("Error while reading configuration: %s\n", err)
		}
		lib.Verbose("Using configuration:\n %v\n", viper.AllSettings())

		DoApply(context, applyFilenames)
	},
}

func DoApply(context *lib.Context, applyFilenames []string) {
	if len(applyFilenames) == 0 {
		lib.Fatal("Please provide a yaml file to apply\n")
	}
	for _, applyFilename := range applyFilenames {
		reader, err := os.Open(applyFilename)
		if err != nil {
			lib.Fatal("Error while opening file %s: %s\n", applyFilename, err)
		}
		docReader := lib.NewDocumentReader(reader)
		for {
			doc, err := docReader.ReadNextYamlDocument()
			if err != nil {
				if err == io.EOF {
					lib.Info("Done with file %s\n", applyFilename)
					break
				} else {
					lib.Fatal("Error while reading yaml document %s: %s\n", applyFilename, err)
				}
			}
			lib.Info("Applying %s\n", doc.Kind)
			err = context.ProcessSingleDocument(doc)
			if err != nil {
				lib.Fatal("Error while processing yaml document %s: %s\n", applyFilename, err)
			}
		}
		reader.Close()
	}
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

package cmd

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/xebialabs/xl-cli/pkg/xl"
	"gopkg.in/cheggaaa/pb.v1"
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
		xl.Fatal("Please provide a yaml file to apply\n")
	}

	totalNrOfDocs, err := countTotalNrOfDocs(applyFilenames)
	if err != nil {
		panic(err)
	}

	bar := pb.StartNew(totalNrOfDocs)
	for _, applyFilename := range applyFilenames {
		bar.Prefix(fmt.Sprintf("%s:", filepath.Base(applyFilename)))

		applyDir := filepath.Dir(applyFilename)
		xl.Verbose("using relative directory %s for file references\n", applyDir)
		reader, err := os.Open(applyFilename)
		if err != nil {
			xl.Fatal("Error while opening file %s: %s\n", applyFilename, err)
		}
		docReader := xl.NewDocumentReader(reader)
		for {
			doc, err := docReader.ReadNextYamlDocument()
			if err != nil {
				if err == io.EOF {
					bar.Prefix("")
					break
				} else {
					xl.Fatal("Error while reading yaml document from %s: %s\n", applyFilename, err)
				}
			}

			bar.Increment()
			time.Sleep(1000 * time.Millisecond)

			err = context.ProcessSingleDocument(doc, applyDir)
			if err != nil {
				xl.Fatal("Error while processing yaml document at line %d of %s: %s\n", doc.Line, applyFilename, err)
			}
		}
		reader.Close()
	}
	bar.FinishPrint("Done")
}

func countTotalNrOfDocs(applyFilenames []string) (int, error){
	var totalNrOfDocuments = 0

	for _, applyFilename := range applyFilenames {
		nrOfDocuments, err := estimateNrOfDocs(applyFilename)
		if err != nil {
			return 0, err
		}
		totalNrOfDocuments += nrOfDocuments
	}

	return totalNrOfDocuments, nil
}

func estimateNrOfDocs(applyFilename string) (int, error) {
	f, err := os.Open(applyFilename)
	if err != nil {
		return 0, err
	}
	defer f.Close()
	nrOfDocs := 1
	scanner := bufio.NewScanner(bufio.NewReader(f))
	for scanner.Scan() {
		if scanner.Text() == "---" {
			nrOfDocs++
		}
	}
	return nrOfDocs, nil
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

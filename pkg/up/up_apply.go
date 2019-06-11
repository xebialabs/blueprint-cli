package up

import (
	"fmt"
	"io/ioutil"
	"path/filepath"

	mapset "github.com/deckarep/golang-set"
	"github.com/spf13/viper"
	"github.com/xebialabs/xl-cli/pkg/blueprint/repository"
	"github.com/xebialabs/xl-cli/pkg/util"
	"github.com/xebialabs/xl-cli/pkg/xl"
)

func applyFilesAndSave() {

	files := getYamlFiles()

	docs := xl.ParseDocuments(util.ToAbsolutePaths(files), mapset.NewSet(), nil, xl.ToProcess{false, true, true}, false, false, xl.SCMInfo{} )

	for _, fileWithDocs := range docs {
		var applyFile = util.PrintableFileName(fileWithDocs.FileName)
		var fileContents []byte

		if fileWithDocs.Parent != nil {
			var parentFile = util.PrintableFileName(*fileWithDocs.Parent)
			util.Verbose("Applying %s (imported by %s) \n", applyFile, parentFile)
		} else {
			util.Verbose("Applying %s \n", applyFile)
		}

		allValueFiles := getValueFiles(fileWithDocs.FileName)

		context, err := xl.BuildContext(viper.GetViper(), &applyValues, allValueFiles, nil, "")

		if err != nil {
			util.Fatal("Error while reading configuration: %s \n", err)
		}

		applyDir := filepath.Dir(fileWithDocs.FileName)

		for index, doc := range fileWithDocs.Documents {
			existingFileContents, _ := context.ProcessSingleDocumentAndGetContents(doc, applyDir, fileWithDocs.FileName)

			if index != len(fileWithDocs.Documents)-1 {
				fileSeparator := []byte("\n---\n")
				existingFileContents = append(existingFileContents, fileSeparator...)
			}

			fileContents = append(fileContents, existingFileContents...)
		}

		ioutil.WriteFile(fileWithDocs.FileName, fileContents, 0644)
	}
}

// searches for YAML / YML files inside xebialabs and kubernetes folder
func getYamlFiles() []string {
	var ymlFiles []string

	folders := []string{Xebialabs, Kubernetes}

	for _, pattern := range repository.BlueprintMetadataFileExtensions {
		for _, folder := range folders {
			glob := fmt.Sprintf("%s/*%s", folder, pattern)
			files, err := filepath.Glob(glob)

			if err != nil {
				util.Fatal("Error while finding YAML files: %s \n", err)
			}

			ymlFiles = append(ymlFiles, files...)
		}
	}
	return ymlFiles
}

func getValueFiles(fileName string) []string {
	homeValueFiles, err := xl.ListHomeXlValsFiles()

	if err != nil {
		util.Fatal("Error while reading value files from home: %s \n", err)
	}

	relativeValueFiles, err := xl.ListRelativeXlValsFiles(filepath.Dir(fileName))

	if err != nil {
		util.Fatal("Error while reading value files from xl: %s \n", err)
	}

	return append(homeValueFiles, relativeValueFiles...)
}

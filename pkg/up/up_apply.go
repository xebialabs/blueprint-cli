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

func applyFilesAndSave() error {

	files, err := getYamlFiles()
	if err != nil {
		return err
	}

	docs := xl.ParseDocuments(util.ToAbsolutePaths(files), mapset.NewSet(), nil, xl.ToProcess{false, true, true}, false, false, xl.SCMInfo{})

	for _, fileWithDocs := range docs {
		var applyFile = util.PrintableFileName(fileWithDocs.FileName)
		var fileContents []byte

		if fileWithDocs.Parent != nil {
			var parentFile = util.PrintableFileName(*fileWithDocs.Parent)
			util.Verbose("Applying %s (imported by %s) \n", applyFile, parentFile)
		} else {
			util.Verbose("Applying %s \n", applyFile)
		}

		allValueFiles, err := getValueFiles(fileWithDocs.FileName)
		if err != nil {
			return err
		}

		context, err := xl.BuildContext(viper.GetViper(), &applyValues, allValueFiles, nil, "")

		if err != nil {
			return fmt.Errorf("error while reading configuration: %s", err)
		}

		applyDir := filepath.Dir(fileWithDocs.FileName)

		for index, doc := range fileWithDocs.Documents {
			existingFileContents, err := context.ProcessSingleDocumentAndGetContents(doc, applyDir, fileWithDocs.FileName)
			if err != nil {
				return err
			}

			if index != len(fileWithDocs.Documents)-1 {
				fileSeparator := []byte("\n---\n")
				existingFileContents = append(existingFileContents, fileSeparator...)
			}

			fileContents = append(fileContents, existingFileContents...)
		}

		return ioutil.WriteFile(fileWithDocs.FileName, fileContents, 0644)
	}
	return nil
}

// searches for YAML / YML files inside xebialabs and kubernetes folder
func getYamlFiles() ([]string, error) {
	var ymlFiles []string

	folders := []string{Xebialabs, Kubernetes}

	for _, pattern := range repository.BlueprintMetadataFileExtensions {
		for _, folder := range folders {
			glob := fmt.Sprintf("%s/*%s", folder, pattern)
			files, err := filepath.Glob(glob)

			if err != nil {
				return nil, fmt.Errorf("error while finding YAML files: %s", err)
			}

			ymlFiles = append(ymlFiles, files...)
		}
	}
	return ymlFiles, nil
}

func getValueFiles(fileName string) ([]string, error) {
	homeValueFiles, err := xl.ListHomeXlValsFiles()

	if err != nil {
		return nil, fmt.Errorf("error while reading value files from home: %s", err)
	}

	relativeValueFiles, err := xl.ListRelativeXlValsFiles(filepath.Dir(fileName))

	if err != nil {
		return nil, fmt.Errorf("error while reading value files from xl: %s", err)
	}

	return append(homeValueFiles, relativeValueFiles...), nil
}

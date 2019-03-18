package xl

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	mapset "github.com/deckarep/golang-set"
	"github.com/spf13/viper"
	"github.com/xebialabs/xl-cli/pkg/blueprint"
	"github.com/xebialabs/xl-cli/pkg/blueprint/repository"
	"github.com/xebialabs/xl-cli/pkg/blueprint/repository/github"
	"github.com/xebialabs/xl-cli/pkg/models"
	"github.com/xebialabs/xl-cli/pkg/util"
)

const (
	Docker    = "docker"
	SeedImage = "xl-docker.xebialabs.com/xl-seed:demo"
)

var pullSeedImage = models.Command{
	Name: Docker,
	Args: []string{"pull", SeedImage},
}

var applyValues map[string]string

func runSeed() models.Command {
	dir, err := os.Getwd()

	if err != nil {
		util.Fatal("Error while getting current work directory")
	}

	return models.Command{
		Name: Docker,
		Args: []string{"run", "-v", dir + ":/data", SeedImage, "--init", "xebialabs/common.yaml", "xebialabs.yaml"},
	}
}

// InvokeBlueprintAndSeed will invoke blueprint and then call XL Seed
func InvokeBlueprintAndSeed(context *Context, upLocalMode bool, blueprintTemplate string, cfgOverridden bool) {
	// Skip Generate blueprint file
	blueprint.SkipFinalPrompt = true
	util.IsQuiet = true

	if !upLocalMode && !cfgOverridden {
        blueprintTemplate = "xl-up"
        var repo repository.BlueprintRepository
        repo, err := github.NewGitHubBlueprintRepository(map[string]string{
            "name":      "xl-up-blueprint",
            "repo-name": "xl-up-blueprint",
            "owner":     "xebialabs",
        })
        if err != nil {
            util.Fatal("Error while creating Blueprint: %s \n", err)
        }
        context.BlueprintContext.ActiveRepo = &repo
    }

	err := blueprint.InstantiateBlueprint(upLocalMode, blueprintTemplate, context.BlueprintContext, models.BlueprintOutputDir)
	if err != nil {
		util.Fatal("Error while creating Blueprint: %s \n", err)
	}

	util.IsQuiet = false
	applyFilesAndSave()
	// TODO: Ask for the version to deploy
	util.Info("Generated files for deployment successfully! \nSpinning up xl seed! \n")
	runAndCaptureResponse("pulling", pullSeedImage)
	runAndCaptureResponse("running", runSeed())
}

func runAndCaptureResponse(status string, cmd models.Command) {

	outStr, errorStr := util.ExecuteCommandAndShowLogs(cmd)

	if outStr != "" {
		createLogFile("xl-seed-log.txt", outStr)
		indx := strings.Index(outStr, "***************")
		if indx != -1 {
			util.Info(outStr[indx:])
		}
	}

	if errorStr != "" {
		createLogFile("xl-seed-error.txt", errorStr)
	}
}

func createLogFile(fileName string, contents string) {
	f, err := os.Create(fileName)
	if err != nil {
		util.Fatal(" Error creating a file %s \n", err)
	}
	f.WriteString(contents)
	f.Close()
}

func applyFilesAndSave() {

	files := getYamlFiles()

	docs := ParseDocuments(util.ToAbsolutePaths(files), mapset.NewSet(), nil, ToProcess{false, true, true})

	for _, fileWithDocs := range docs {
		var applyFile = util.PrintableFileName(fileWithDocs.FileName)

		if fileWithDocs.Parent != nil {
			var parentFile = util.PrintableFileName(*fileWithDocs.Parent)
			util.Verbose("Applying %s (imported by %s) \n", applyFile, parentFile)
		} else {
			util.Verbose("Applying %s \n", applyFile)
		}

		allValsFiles := getValFiles(fileWithDocs.FileName)

		context, err := BuildContext(viper.GetViper(), &applyValues, allValsFiles)
		if err != nil {
			util.Fatal("Error while reading configuration: %s \n", err)
		}

		applyDir := filepath.Dir(fileWithDocs.FileName)
		var fileContents []byte

		for i, doc := range fileWithDocs.Documents {
			existingFileContents, _ := context.processSingleDocumentAndGetContents(doc, applyDir, fileWithDocs.FileName)

			if i != len(fileWithDocs.Documents)-1 {
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

	folders := []string{"xebialabs", "kubernetes"}

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

func getValFiles(fileName string) []string {
	homeValsFiles, e := ListHomeXlValsFiles()

	if e != nil {
		util.Fatal("Error while reading value files from home: %s \n", e)
	}

	relativeValsFiles, e := ListRelativeXlValsFiles(filepath.Dir(fileName))

	if e != nil {
		util.Fatal("Error while reading value files from xl: %s \n", e)
	}

	return append(homeValsFiles, relativeValsFiles...)
}

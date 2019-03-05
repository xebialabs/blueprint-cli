package xl

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	"github.com/deckarep/golang-set"
	"github.com/spf13/viper"
	"github.com/xebialabs/xl-cli/pkg/blueprint"
	"github.com/xebialabs/xl-cli/pkg/models"
	"github.com/xebialabs/xl-cli/pkg/repository"
	"github.com/xebialabs/xl-cli/pkg/util"
	"gopkg.in/AlecAivazis/survey.v1"
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

func getBlueprintLocation(surveyOpts ...survey.AskOpt) (string, error) {

	blueprintTemplate := ""

	_ = survey.AskOne(
		&survey.Input{
			Message: "Enter the blueprint repository or file:",
			Help:    "http://github.com/xebialabs/repo-containing-blueprint or /path/to/blueprint",
			Default: "/Users/sendilkumar/xl/xl-platform-k8s/blueprint/xl-up",
		},
		&blueprintTemplate,
		survey.Required,
		surveyOpts...,
	)

	return blueprintTemplate, nil
}

func isLocal(surveyOpts ...survey.AskOpt) (bool, error) {

	isLocal := true

	_ = survey.AskOne(
		&survey.Confirm{
			Message: "Is your blueprint available in local?",
			Help:    "Y for local, N for remote",
			Default: true,
		},
		&isLocal,
		survey.Required,
		surveyOpts...,
	)

	return isLocal, nil
}

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
func InvokeBlueprintAndSeed(context *Context) {
	// Skip Generate blueprint file
	blueprint.SkipFinalPrompt = true

	// TODO: Check for Docker installation
	util.Verbose("Fetching the blueprint template location")
	isLocal, err := isLocal()
	blueprintTemplate, err := getBlueprintLocation()

	util.Verbose("Starting Blueprint questions to generate necessary files")
	err = blueprint.InstantiateBlueprint(isLocal, blueprintTemplate, context.BlueprintContext, models.BlueprintOutputDir)
	if err != nil {
		util.Fatal("Error while creating Blueprint: %s\n", err)
	}

	applyFilesAndSave()

	// TODO: Ask for the version to deploy
	util.Info("Blueprint created successfully! Spinning up xl seed!! \n")
	runAndCaptureResponse("pulling", pullSeedImage)
	runAndCaptureResponse("running", runSeed())
	// TODO: fetch URLs of XLD and XLR
	util.Info("Seed successfully started the services!\n")
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
		util.Fatal("Error while %s the xl seed image", status)
	}
}

func createLogFile(fileName string, contents string) {
	f,err :=os.Create(fileName)
	if err != nil {
		util.Fatal(" Error creating a file %s",err)
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
			util.Info("Applying %s (imported by %s)\n", applyFile, parentFile)
		} else {
			util.Info("Applying %s\n", applyFile)
		}

		allValsFiles := getValFiles(fileWithDocs.FileName)

		context, err := BuildContext(viper.GetViper(), &applyValues, allValsFiles)
		if err != nil {
			util.Fatal("Error while reading configuration: %s\n", err)
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

func getYamlFiles() []string {
	var ymlFiles []string

	for _, pattern := range repository.BlueprintMetadataFileExtensions {
		glob := fmt.Sprintf("**/*%s", pattern)
		files, err := filepath.Glob(glob)

		if err != nil {
			util.Fatal("Error while finding YAML files: %s\n", err)
		}

		ymlFiles = append(ymlFiles, files...)
	}

	return ymlFiles
}

func getValFiles(fileName string) []string {
	homeValsFiles, e := ListHomeXlValsFiles()

	if e != nil {
		util.Fatal("Error while reading value files from home: %s\n", e)
	}

	relativeValsFiles, e := ListRelativeXlValsFiles(filepath.Dir(fileName))

	if e != nil {
		util.Fatal("Error while reading value files from xl: %s\n", e)
	}

	return append(homeValsFiles, relativeValsFiles...)
}

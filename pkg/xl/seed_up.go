package xl

import (
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/deckarep/golang-set"
	"github.com/spf13/viper"
	"github.com/xebialabs/xl-cli/pkg/blueprint"
	"github.com/xebialabs/xl-cli/pkg/models"
	"github.com/xebialabs/xl-cli/pkg/util"
	"gopkg.in/AlecAivazis/survey.v1"
)

var docker = "docker"
var seedImage = "xl-docker.xebialabs.com/xl-seed:demo"

var dockerPullImage = models.Command{
	docker,
	[]string{"pull", seedImage},
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

func runImage() models.Command {
	dir, err := os.Getwd()

	if err != nil {
		util.Fatal("Error while getting current work directory")
	}

	dockerRunImage := models.Command{
		docker,
		[]string{"run", "-v", dir + ":/data", seedImage, "--init", "xebialabs/common.yaml", "xebialabs.yaml"},
	}

	return dockerRunImage
}

func RunXlSeed(context *Context) {
	// TODO: Check for Docker installation
	util.Verbose("Fetching the blueprint template location")
	isLocal, err := isLocal()
	blueprintTemplate, err := getBlueprintLocation()

	util.Verbose("Starting Blueprint questions to generate necessary files")
	err = blueprint.InstantiateBlueprint(isLocal, blueprintTemplate, context.BlueprintContext, models.BlueprintOutputDir)
	if err != nil {
		util.Fatal("Error while creating Blueprint: %s\n", err)
	}

	fakeApplyFiles()

	// TODO: Ask for the version to deploy ?
	util.Info("Blueprint created successfully! Spinning up xl seed!! \n")
	runAndCaptureResponse("pulling", dockerPullImage)
	runAndCaptureResponse("running", runImage())
	// TODO: fetch URLs of XLD and XLR
	util.Info("Seed successfully started the services!\n")
}

func runAndCaptureResponse(status string, cmd models.Command) {

	_, errorStr := util.ExecuteCommandAndShowLogs(cmd)

	if errorStr != "" {
		util.Fatal("Error while %s the xl seed image: %s\n", status, errorStr)
	}
}

func fakeApplyFiles() {

	files := getFiles()

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
			existingFileContents, _ := context.FakeProcessSingleDocument(doc, applyDir, fileWithDocs.FileName)

			if i != len(fileWithDocs.Documents)-1 {
				fileSeparator := []byte("\n---\n")
				existingFileContents = append(existingFileContents, fileSeparator...)
			}

			fileContents = append(fileContents, existingFileContents...)

		}

		ioutil.WriteFile(fileWithDocs.FileName, fileContents, 0644)
	}
}

func getFiles() []string {
	files, err := filepath.Glob("**/*.yaml")

	if err != nil {
		util.Fatal("Error while finding YAML files: %s\n", err)
	}

	return files
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

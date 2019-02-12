package xl

import (
	"github.com/deckarep/golang-set"
	"github.com/spf13/viper"
	"github.com/xebialabs/xl-cli/pkg/models"
	"gopkg.in/AlecAivazis/survey.v1"
	"io/ioutil"
	"os"
	"path/filepath"
)

var docker = "docker"
var seedImage = "xl-docker.xebialabs.com/xl-seed:0.0.1"

type command struct {
	name string
	args []string
}

var dockerPullImage = command {
	docker,
	[]string{ "pull", seedImage },
}

type FileWithDocuments struct {
	imports   []string
	parent    *string
	documents []*Document
	fileName  string
}

var applyFilenames []string
var applyValues map[string]string

func getBlueprintLocation(surveyOpts ...survey.AskOpt) (string, error) {

	blueprintTemplate := ""

	_ = survey.AskOne(
		&survey.Input{
			Message: "Enter the blueprint repository or file:",
			Help: "http://github.com/xebialabs/repo-containing-blueprint or /path/to/blueprint",
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
			Help: "Y for local, N for remote",
			Default: true,
		},
		&isLocal,
		survey.Required,
		surveyOpts...,
	)

	return isLocal, nil
}

func runImage() command{
	dir, err := os.Getwd()

	if err != nil {
		Fatal("Error while getting current work directory")
	}

	dockerRunImage := command {
		docker,
		[]string{ "run", "--network=host", "-v", dir + ":/data", seedImage,  "--init", "xebialabs/common.yaml", "xebialabs.yaml"},
	}

	return dockerRunImage
}

func RunXlSeed(context *Context) {
	// TODO: Check for Docker installation
	Verbose("Fetching the blueprint template location")
	isLocal, err := isLocal()
	blueprintTemplate, err := getBlueprintLocation()

	Verbose("Starting Blueprint questions to generate necessary files")
	err = InstantiateBlueprint(isLocal, blueprintTemplate, context.BlueprintContext, models.BlueprintOutputDir)
	if err != nil {
		Fatal("Error while creating Blueprint: %s\n", err)
	}

	fakeApplyFiles()

	// TODO: Ask for the version to deploy ?
	Info("Blueprint created successfully! Spinning up xl seed!! \n")
	_, errorStr := ExecuteCommandAndShowLogs(dockerPullImage)

	if errorStr != "" {
		Fatal("Error while pulling the xl seed image: %s\n", errorStr)
	}

	Info("Running xl-seed\n")
	_, errorStr = ExecuteCommandAndShowLogs(runImage())

	if errorStr != "" {
		Fatal("Error while running the xl seed image: %s\n", errorStr)
	}

	// TODO: fetch URLs of XLD and XLR
	Info("Seed successfully started the services!\n")
}

func fakeApplyFiles() {

	files := getFiles()

	docs := ParseDocuments(ToAbsolutePaths(files), mapset.NewSet(), nil)

	for _, fileWithDocs := range docs {
		var applyFile = PrintableFileName(fileWithDocs.fileName)

		if fileWithDocs.parent != nil {
			var parentFile = PrintableFileName(*fileWithDocs.parent)
			Info("Applying %s (imported by %s)\n", applyFile, parentFile)
		} else {
			Info("Applying %s\n", applyFile)
		}

		allValsFiles := getValFiles(fileWithDocs.fileName)

		context, err := BuildContext(viper.GetViper(), &applyValues, allValsFiles)
		if err != nil {
			Fatal("Error while reading configuration: %s\n", err)
		}

		applyDir := filepath.Dir(fileWithDocs.fileName)
		var fileContents []byte

		for i, doc := range fileWithDocs.documents {
			existingFileContents, _:= context.FakeProcessSingleDocument(doc, applyDir, fileWithDocs.fileName)

			if i != len(fileWithDocs.documents) - 1 {
				fileSeparator := []byte("\n---\n")
				existingFileContents = append(existingFileContents, fileSeparator...)
			}
			fileContents = append(fileContents, existingFileContents...)
		}

		ioutil.WriteFile(fileWithDocs.fileName, fileContents, 0644)
	}

}


func getFiles() []string {
	files, err := filepath.Glob("**/*.yaml")
	if err != nil {
		Fatal("Error while creating Blueprint: %s\n", err)
	}
	//files = append(files, "xebialabs/common.yaml")

	return files
}

func getValFiles(fileName string) []string {
	return append(getHomeValFiles(), getRelativeValFiles(fileName)...)
}

func getHomeValFiles() []string{
	homeValsFiles, e := ListHomeXlValsFiles()

	if e != nil {
		Fatal("Error while reading value files from home: %s\n", e)
	}

	return homeValsFiles
}

func getRelativeValFiles(fileName string) []string {

	projectValsFiles, err := ListRelativeXlValsFiles(filepath.Dir(fileName))
	if err != nil {
		Fatal("Error while reading value files for %s from project: %s\n", fileName, err)
	}

	return projectValsFiles
}




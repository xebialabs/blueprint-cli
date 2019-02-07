package xl

import (
	"github.com/xebialabs/xl-cli/pkg/models"
	"gopkg.in/AlecAivazis/survey.v1"
	"os"
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

func getBlueprintLocation(surveyOpts ...survey.AskOpt) (string, error) {

	blueprintTemplate := ""

	_ = survey.AskOne(
		&survey.Input{
			Message: "Enter the blueprint repository or file:",
			Help: "http://github.com/xebialabs/repo-containing-blueprint or /path/to/blueprint",
			Default: "http://github.com/xebialabs/repo-containing-blueprint",
		},
		&blueprintTemplate,
		survey.Required,
		surveyOpts...,
	)

	return blueprintTemplate, nil
}

func runImage() command{
	dir, err := os.Getwd()

	if err != nil {
		Fatal("Error while getting current work directory")
	}

	dockerRunImage := command {
		docker,
		[]string{ "run", "-v", dir + ":/data", seedImage,  "--init", "common.yaml", "xebialabs.yaml" },
	}

	return dockerRunImage
}

func RunXlSeed(context *Context) {
	// TODO: Check for Docker installation
	Verbose("Fetching the blueprint template location")
	blueprintTemplate, err := getBlueprintLocation()

	Verbose("Starting Blueprint questions to generate necessary files")
	err = InstantiateBlueprint(false, blueprintTemplate, context.BlueprintContext, models.BlueprintOutputDir)
	if err != nil {
		Fatal("Error while creating Blueprint: %s\n", err)
	}

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

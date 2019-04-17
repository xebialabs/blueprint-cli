package up

import (
	"github.com/xebialabs/xl-cli/pkg/blueprint/repository"
	"github.com/xebialabs/xl-cli/pkg/blueprint/repository/github"
	"github.com/xebialabs/xl-cli/pkg/models"
	"github.com/xebialabs/xl-cli/pkg/util"
	"os"
)

const (
	Docker                   = "docker"
	SeedImage                = "xl-docker.xebialabs.com/xl-seed:demo"
	Kubernetes               = "kubernetes"
	Xebialabs                = "xebialabs"
	XlUpBlueprint            = "xl-up-blueprint"
	DefaultBlueprintTemplate = "xl-up"
)

var pullSeedImage = models.Command{
	Name: Docker,
	Args: []string{"pull", SeedImage},
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

func getRepo(branchVersion string) repository.BlueprintRepository {

	repo, err := github.NewGitHubBlueprintRepository(map[string]string{
		"name":      XlUpBlueprint,
		"repo-name": XlUpBlueprint,
		"owner":     Xebialabs,
		"branch":    branchVersion,
	})

	if err != nil {
		util.Fatal("Error while creating Blueprint: %s \n", err)
	}

	return repo
}

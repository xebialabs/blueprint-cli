package mock

import (
	"fmt"

	"github.com/xebialabs/xl-blueprint/pkg/models"
	"github.com/xebialabs/xl-blueprint/pkg/util"
)

// Mock Blueprint Repository Provider implementation
// only suitable for test usage as mock provider
type MockBlueprintRepository struct {
	Name   string
	Owner  string
	Branch string
}

func NewMockBlueprintRepository(confMap map[string]string) (*MockBlueprintRepository, error) {
	repo := new(MockBlueprintRepository)
	repo.Name = confMap["name"]

	// parse branch name, or set it to default
	if util.MapContainsKeyWithVal(confMap, "branch") {
		repo.Branch = confMap["branch"]
	} else {
		repo.Branch = "master"
	}

	return repo, nil
}

func (repo *MockBlueprintRepository) Initialize() error {
	return nil
}

func (repo *MockBlueprintRepository) GetName() string {
	return repo.Name
}

func (repo *MockBlueprintRepository) GetProvider() string {
	return models.ProviderMock
}

func (repo *MockBlueprintRepository) GetInfo() string {
	return fmt.Sprintf(
		"Provider: %s\n  Repository name: %s\n  Owner: %s\n  Branch: %s",
		repo.GetProvider(),
		repo.Name,
		repo.Owner,
		repo.Branch,
	)
}

func (repo *MockBlueprintRepository) ListBlueprintsFromRepo() (map[string]*models.BlueprintRemote, []string, error) {
	blueprints := make(map[string]*models.BlueprintRemote)
	var blueprintDirs []string

	blueprintDirs = append(blueprintDirs, "xl/test")
	blueprints["xl/test"] = &models.BlueprintRemote{
		Name: "xl/test",
		Path: "xl/test",
		DefinitionFile: models.RemoteFile{
			Filename: "blueprint.yaml",
			Path:     "xl/test/blueprint.yaml",
		},
		Files: []models.RemoteFile{
			{Filename: "test.yaml.tmpl", Path: "xl/test/test.yaml.tmpl"},
			{Filename: "readme.md", Path: "xl/test/readme.md"},
		},
	}

	return blueprints, blueprintDirs, nil
}

func (repo *MockBlueprintRepository) GetFileContents(filePath string) (*[]byte, error) {
	var contents []byte
	switch filePath {
	case "xl/test/blueprint.yaml":
		contents = []byte(`apiVersion: xl/v2
kind: Blueprint
metadata:
  name: Test Project
  description: Is just a test blueprint project
  author: XebiaLabs
  version: 1.0
spec:
  parameters:
  - name: Test
    type: Input

  files:
  - path: test.yaml.tmpl
  - path: readme.md`)
	case "xl/test/test.yaml.tmpl":
		contents = []byte("template")
	case "xl/test/readme.md":
		contents = []byte("readme")
	default:
		return nil, fmt.Errorf("file %s not found in mock repo", filePath)
	}

	return &contents, nil
}

package mock

import (
	"fmt"
	"github.com/xebialabs/xl-cli/pkg/models"
)

// Mock Blueprint Repository Provider implementation
// only suitable for test usage as mock provider
type MockBlueprintRepository struct {
	Name string
	Owner string
	Branch string
}

func NewMockBlueprintRepository(name string, owner string, branch string) *MockBlueprintRepository {
	repo := new(MockBlueprintRepository)
	repo.Name = name
	repo.Owner = owner
	repo.Branch = branch

	return repo
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
			Path: "xl/test/blueprint.yaml",
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
		contents = []byte(`apiVersion: xl/v1
kind: Blueprint
metadata:
  projectName: Test Project
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

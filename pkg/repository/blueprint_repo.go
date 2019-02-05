package repository

import (
	"github.com/xebialabs/xl-cli/pkg/models"
)

const BlueprintMetadataFileName = "blueprint"
var BlueprintMetadataFileExtensions = []string{".yaml", ".yml"}

type BlueprintRepository interface {
	ListBlueprintsFromRepo() (map[string]*models.BlueprintRemote, []string, error)
	GetFileContents(filePath string) (*[]byte, error)
}

package repository

import (
	"github.com/thoas/go-funk"
	"github.com/xebialabs/blueprint-cli/pkg/models"
	"net/url"
	"path"
	"strings"
)

const BlueprintMetadataFileName = "blueprint"

var BlueprintMetadataFileExtensions = []string{".yaml", ".yml"}

type BlueprintRepository interface {
	Initialize() error
	GetName() string
	GetProvider() string
	GetInfo() string
	ListBlueprintsFromRepo() (map[string]*models.BlueprintRemote, []string, error)
	GetFileContents(filePath string) (*[]byte, error)
}

// utility functions
func GenerateBlueprintFileDefinition(blueprints map[string]*models.BlueprintRemote, blueprintPath string, filename string, path string, parsedUrl *url.URL) models.RemoteFile {
	// Initialize map item if needed
	if _, exists := blueprints[blueprintPath]; !exists {
		blueprints[blueprintPath] = models.NewBlueprintRemote(blueprintPath, blueprintPath)
	}
	return models.RemoteFile{
		Filename: filename,
		Path:     path,
		Url:      parsedUrl,
	}
}

func CheckIfBlueprintDefinitionFile(filename string) bool {
	return (strings.ToLower(strings.TrimSuffix(filename, path.Ext(filename))) == BlueprintMetadataFileName) && (funk.Contains(BlueprintMetadataFileExtensions, strings.ToLower(path.Ext(filename))))
}

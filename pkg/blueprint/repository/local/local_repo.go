package local

import (
	"fmt"
    "os"

    "github.com/xebialabs/xl-cli/pkg/models"
	"github.com/xebialabs/xl-cli/pkg/util"
)

// Local Blueprint Repository Provider implementation
type LocalBlueprintRepository struct {
	Name   string
	Path   string
}

func NewLocalBlueprintRepository(confMap map[string]string) (*LocalBlueprintRepository, error) {
	repo := new(LocalBlueprintRepository)
	repo.Name = confMap["name"]

    // parse & check local blueprint repo path
    if !util.MapContainsKeyWithVal(confMap, "path") {
        return nil, fmt.Errorf("'path' config field must be set for Local repository type")
    }
    dirInfo, err := os.Stat(confMap["path"])
    if os.IsNotExist(err) {
        return nil, fmt.Errorf("local repository dir [%s] not found: %s", confMap["path"], err.Error())
    }
    switch mode := dirInfo.Mode(); {
    case mode.IsRegular():
        return nil, fmt.Errorf("got file path [%s] instead of a local directory path", confMap["path"])
    }
    repo.Path = confMap["path"]

	return repo, nil
}

func (repo *LocalBlueprintRepository) Initialize() error {
	return nil
}

func (repo *LocalBlueprintRepository) GetName() string {
	return repo.Name
}

func (repo *LocalBlueprintRepository) GetProvider() string {
	return models.ProviderLocal
}

func (repo *LocalBlueprintRepository) GetInfo() string {
	return fmt.Sprintf(
		"Provider: %s\n  Repository name: %s\n  Local path: %s",
		repo.GetProvider(),
		repo.Name,
		repo.Path,
	)
}

func (repo *LocalBlueprintRepository) ListBlueprintsFromRepo() (map[string]*models.BlueprintRemote, []string, error) {
	blueprints := make(map[string]*models.BlueprintRemote)
	var blueprintDirs []string

	// todo: LOVE-642

	return blueprints, blueprintDirs, nil
}

func (repo *LocalBlueprintRepository) GetFileContents(filePath string) (*[]byte, error) {
	var contents []byte
    // todo: LOVE-642
	return &contents, nil
}

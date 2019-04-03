package local

import (
	"fmt"
    "github.com/thoas/go-funk"
    "io/ioutil"
    "os"
    "path/filepath"
    "strings"

    "github.com/xebialabs/xl-cli/pkg/blueprint/repository"
    "github.com/xebialabs/xl-cli/pkg/models"
	"github.com/xebialabs/xl-cli/pkg/util"
)

// list of ignored files
var ignoredFiles = []string {".DS_Store"}

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

	// walk root directory
    currentPath := "."
	err := filepath.Walk(repo.Path, func(path string, info os.FileInfo, err error) error {
	    if !info.IsDir() {
	        // found a file
            filename := filepath.Base(path)

            // todo: fix incorrect mappings!
            // check if file should be skipped
            if !funk.Contains(ignoredFiles, filename) {
                if strings.ToLower(strings.TrimSuffix(filename, filepath.Ext(filename))) == repository.BlueprintMetadataFileName {
                    // If this is blueprints definition file, it is considered as the root path for the blueprint
                    currentPath, err = filepath.Rel(repo.Path, filepath.Dir(path))
                    if err != nil {
                        return err
                    }

                    // skip root folder
                    if currentPath != "." {
                        // Add local definition file to blueprint
                        blueprintDirs = append(blueprintDirs, currentPath)
                        blueprints[currentPath].DefinitionFile = createLocalFileDefinition(blueprints, currentPath, filename, path)
                    }
                } else {
                    // skip root folder
                    if currentPath != "." && filepath.Dir(path) != repo.Path {
                        // Add local template file to blueprint
                        blueprints[currentPath].AddFile(createLocalFileDefinition(blueprints, currentPath, filename, path))
                    }
                }
            } else {
                util.Verbose("[local] Ignoring local file [%s] because it's in ignore list\n", path)
            }
        }
        return nil
    })
	if err != nil {
	    return nil, nil, err
    }

    // todo: remove debug logging!
    for k, v := range blueprints {
        util.Verbose("====> %s:\n", k)
        for _, file := range v.Files {
            util.Verbose("\t[%s] - %s\n", k, file.Path)
        }
    }
	return blueprints, blueprintDirs, nil
}

func (repo *LocalBlueprintRepository) GetFileContents(filePath string) (*[]byte, error) {
    content, err := ioutil.ReadFile(filePath)
    if err != nil {
        return nil, err
    }
	return &content, nil
}

// utility functions
func createLocalFileDefinition(blueprints map[string]*models.BlueprintRemote, currentPath string, filename string, fullPath string) models.RemoteFile {
    // Initialize map item if needed
    if _, exists := blueprints[currentPath]; !exists {
        blueprints[currentPath] = models.NewBlueprintRemote(currentPath, currentPath)
    }
    return models.RemoteFile{
        Filename: filename,
        Path:     fullPath,
    }
}

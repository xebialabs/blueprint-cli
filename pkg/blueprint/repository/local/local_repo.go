package local

import (
	"fmt"
	"io/ioutil"
	"os"
	"os/user"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/thoas/go-funk"

	"github.com/xebialabs/xl-cli/pkg/blueprint/repository"
	"github.com/xebialabs/xl-cli/pkg/models"
	"github.com/xebialabs/xl-cli/pkg/util"
)

// Local Blueprint Repository Provider implementation
type LocalBlueprintRepository struct {
	Name          string
	Path          string
	IgnoredDirs   []string
	IgnoredFiles  []string
	LocalFiles    []string
	BlueprintDirs []string
}

func NewLocalBlueprintRepository(confMap map[string]string) (*LocalBlueprintRepository, error) {
	repo := new(LocalBlueprintRepository)
	repo.Name = confMap["name"]
	repo.LocalFiles = []string{}
	repo.BlueprintDirs = []string{}

	// expand home dir if needed
	if !util.MapContainsKeyWithVal(confMap, "path") {
		return nil, fmt.Errorf("'path' config field must be set for Local repository type")
	}
	currentUser, err := user.Current()
	if err != nil {
		return nil, fmt.Errorf("cannot get current user: %s", err.Error())
	}
	repoDir := util.ExpandHomeDirIfNeeded(confMap["path"], currentUser)

	// parse & check local blueprint repo path
	dirInfo, err := os.Stat(repoDir)
	if os.IsNotExist(err) {
		return nil, fmt.Errorf("local repository dir [%s] not found: %s", repoDir, err.Error())
	}
	switch mode := dirInfo.Mode(); {
	case mode.IsRegular():
		return nil, fmt.Errorf("got file path [%s] instead of a local directory path", repoDir)
	}
	repo.Path = repoDir

	// parse ignored dirs & files
	if util.MapContainsKeyWithVal(confMap, "ignored-dirs") {
		repo.IgnoredDirs = strings.Split(confMap["ignored-dirs"], ",")
		for i := range repo.IgnoredDirs {
			repo.IgnoredDirs[i] = strings.TrimSpace(repo.IgnoredDirs[i])
		}
	} else {
		repo.IgnoredDirs = []string{}
	}
	if util.MapContainsKeyWithVal(confMap, "ignored-files") {
		repo.IgnoredFiles = strings.Split(confMap["ignored-files"], ",")
		for i := range repo.IgnoredFiles {
			repo.IgnoredFiles[i] = strings.TrimSpace(repo.IgnoredFiles[i])
		}
	} else {
		repo.IgnoredFiles = []string{}
	}

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
		"Provider: %s\n  Repository name: %s\n  Local path: %s\n  Ignored directories: %s\n  Ignored files: %s",
		repo.GetProvider(),
		repo.Name,
		repo.Path,
		repo.IgnoredDirs,
		repo.IgnoredFiles,
	)
}

func (repo *LocalBlueprintRepository) traversePath(path string, info os.FileInfo, err error) error {
	if info.IsDir() {
		// skip ignored directories
		dir := filepath.Base(path)
		if funk.Contains(repo.IgnoredDirs, dir) {
			util.Verbose("[local] Ignoring local directory [%s] because it's in ignore list\n", path)
			return filepath.SkipDir
		}
	} else {
		// handle file
		filename := filepath.Base(path)
		if !funk.Contains(repo.IgnoredFiles, filename) {
			fileDir, err := filepath.Rel(repo.Path, filepath.Dir(path))
			if err != nil {
				return err
			}
			if fileDir != "." {
				repo.LocalFiles = append(repo.LocalFiles, path)

				if repository.CheckIfBlueprintDefinitionFile(filename) {
					// mark directory if this is a blueprint root dir
					repo.BlueprintDirs = append(repo.BlueprintDirs, filepath.Dir(path))
				}
			}
		} else {
			util.Verbose("[local] Ignoring local file [%s] because it's in ignore list\n", path)
		}
	}
	return nil
}

func (repo *LocalBlueprintRepository) ListBlueprintsFromRepo() (map[string]*models.BlueprintRemote, []string, error) {
	blueprints := make(map[string]*models.BlueprintRemote)
	var blueprintDirs []string

	// walk root directory
	err := filepath.Walk(repo.Path, repo.traversePath)
	if err != nil {
		return nil, nil, err
	}

	// construct blueprint map
	for _, file := range repo.LocalFiles {
		if blueprintDir := findRelatedBlueprintDir(repo.BlueprintDirs, file); blueprintDir != "" {
			// if local file is within any valid blueprint directory
			filename := filepath.Base(file)
			currentPath, _ := filepath.Rel(repo.Path, blueprintDir)
			filePath := filepath.Join(currentPath, filename)
			if repository.CheckIfBlueprintDefinitionFile(filename) {
				blueprints[currentPath].DefinitionFile = repository.GenerateBlueprintFileDefinition(
					blueprints,
					currentPath,
					filename,
					filePath,
					nil,
				)
				blueprintDirs = append(blueprintDirs, currentPath)
			} else {
				fileDef := repository.GenerateBlueprintFileDefinition(blueprints, currentPath, filename, filePath, nil)
				blueprints[currentPath].AddFile(fileDef)
			}
		}
	}
	return blueprints, blueprintDirs, nil
}

func (repo *LocalBlueprintRepository) GetFileContents(filePath string) (*[]byte, error) {
	fullPath := filepath.Join(repo.Path, filePath)
	content, err := ioutil.ReadFile(fullPath)
	if err != nil {
		return nil, err
	}
	return &content, nil
}

// utility functions
func findRelatedBlueprintDir(blueprintDirs []string, fullPath string) string {
	for _, blueprintDir := range blueprintDirs {
		if match, _ := regexp.MatchString("[/\\\\]?"+blueprintDir+"[/\\\\]", fullPath); match {
			return blueprintDir
		}
	}
	return ""
}

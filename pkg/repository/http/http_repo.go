package http

import (
	"encoding/json"
	"fmt"
	"github.com/xebialabs/xl-cli/pkg/models"
	"github.com/xebialabs/xl-cli/pkg/repository"
	"io/ioutil"
	"net/http"
	"net/url"
	"path"
)

const (
	RepoIndexFileName = "index.json"
)

// HTTP Blueprint Repository Provider implementation
type HttpBlueprintRepository struct {
	Client *http.Client
	Name string
	RepoUrl *url.URL
	Username string
	Password string
}

func NewHttpBlueprintRepository(name string, repoUrl *url.URL, username string, password string) *HttpBlueprintRepository {
	repo := new(HttpBlueprintRepository)
	repo.Client = &http.Client{}
	repo.Name = name
	repo.RepoUrl = repoUrl
	// TODO: LOVE-628 Support for username & password
	repo.Username = username
	repo.Password = password

	return repo
}

func (repo *HttpBlueprintRepository) ListBlueprintsFromRepo() (map[string]*models.BlueprintRemote, []string, error) {
	blueprints := make(map[string]*models.BlueprintRemote)
	var blueprintDirs []string

	// Read repository index file
	contents, err := repo.GetFileContents(RepoIndexFileName)
	if err != nil {
		return nil, nil, err
	}
	err = json.Unmarshal(*contents, &blueprintDirs)
	if err != nil {
		return nil, nil, err
	}

	// Create list of blueprint remote definitions based on index file
	for _, blueprintDir := range blueprintDirs {
		blueprintDefFileName, err := repo.checkBlueprintDefinitionFile(blueprintDir)
		if err != nil {
			return nil, nil, err
		}
		blueprints[blueprintDir] = &models.BlueprintRemote{
			Name: blueprintDir,
			Path: blueprintDir,
			DefinitionFile: models.RemoteFile{
				Filename: blueprintDefFileName,
				Path: path.Join(blueprintDir, blueprintDefFileName),
			},
			Files: []models.RemoteFile{},
		}
	}

	return blueprints, blueprintDirs, nil
}

func (repo *HttpBlueprintRepository) GetFileContents(filePath string) (*[]byte, error) {
	response, err := repo.getResponseFromUrl(filePath)
	if err != nil {
		return nil, err
	}
	if response.StatusCode >= 400 {
		return nil, fmt.Errorf("%d unable to read remote http file [%s]", response.StatusCode, filePath)
	}

	body, err := ioutil.ReadAll(response.Body)
	if err != nil {
		return nil, err
	}
	return &body, nil
}

// Utility functions
func (repo *HttpBlueprintRepository) checkBlueprintDefinitionFile(blueprintDir string) (string, error) {
	var err error
	for _, validExtension := range repository.BlueprintMetadataFileExtensions {
		blueprintDefFileName := repository.BlueprintMetadataFileName + validExtension
		response, err := repo.getResponseFromUrl(path.Join(blueprintDir, blueprintDefFileName))
		if err == nil && response.StatusCode < 400 {
			return blueprintDefFileName, nil
		}
	}
	return "", err
}

func (repo *HttpBlueprintRepository) getResponseFromUrl(filePath string) (*http.Response, error) {
	reqUrl, _ := url.Parse(repo.RepoUrl.String())
	reqUrl.Path = path.Join(repo.RepoUrl.Path, filePath)
	print(fmt.Sprintf("===> %s\n", reqUrl.String()))
	response, err := repo.Client.Get(reqUrl.String())
	if err != nil {
		return nil, err
	}
	return response, nil
}

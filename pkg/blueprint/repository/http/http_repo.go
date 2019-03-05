package http

import (
	"encoding/json"
	"fmt"
	"github.com/xebialabs/xl-cli/pkg/blueprint/repository"
	"github.com/xebialabs/xl-cli/pkg/models"
	"github.com/xebialabs/xl-cli/pkg/util"
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
	Client      *http.Client

	Name        string
	RepoUrl     *url.URL
	Username    string
	Password    string
}

func NewHttpBlueprintRepository(confMap map[interface{}]interface{}) (*HttpBlueprintRepository, error) {
	// Parse context config
	repo := new(HttpBlueprintRepository)
	repo.Name = confMap["name"].(string)

	// parse repository URL
	if !util.MapContainsKey(confMap, "url") {
		return nil, fmt.Errorf("'url' config field must be set for HTTP repository type")
	}
	parsedUrl, err := url.ParseRequestURI(confMap["url"].(string))
	if err != nil {
		return nil, fmt.Errorf("HTTP repository URL cannot be parsed: %s", err.Error())
	}
	repo.RepoUrl = parsedUrl

	// parse basic auth credentials, if exists
	if util.MapContainsKey(confMap, "username") {
		repo.Username = confMap["username"].(string)
	}
	if util.MapContainsKey(confMap, "password") {
		repo.Password = confMap["password"].(string)
	}

	return repo, nil
}

func (repo *HttpBlueprintRepository) Initialize() error {
	repo.Client = &http.Client{}
	return nil
}

func (repo *HttpBlueprintRepository) GetName() string {
	return repo.Name
}

func (repo *HttpBlueprintRepository) GetProvider() string {
	return models.ProviderHttp
}

func (repo *HttpBlueprintRepository) GetInfo() string {
	return fmt.Sprintf(
		"Provider: %s\n  Name: %s\n  Repository URL: %s\n  Username: %s",
		repo.GetProvider(),
		repo.Name,
		repo.RepoUrl.String(),
		repo.Username,
	)
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
	requestUrl, _ := url.Parse(repo.RepoUrl.String())
	requestUrl.Path = path.Join(repo.RepoUrl.Path, filePath)

	request, err := http.NewRequest(http.MethodGet, requestUrl.String(), nil)
	if err != nil {
		return nil, err
	}
	if repo.Username != "" {
		util.Verbose("[http-repo] Setting basic auth headers for request '%s' with user '%s'\n", request.URL.String(), repo.Username)
		request.SetBasicAuth(repo.Username, repo.Password)
	}

	response, err := repo.Client.Do(request)
	if err != nil {
		return nil, err
	}
	return response, nil
}

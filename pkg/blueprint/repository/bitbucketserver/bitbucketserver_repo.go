package bitbucketserver

import (
	"fmt"
	"net/url"
	"path"
	"strconv"

	"github.com/xebialabs/xl-cli/pkg/blueprint/repository"
	"github.com/xebialabs/xl-cli/pkg/models"
	"github.com/xebialabs/xl-cli/pkg/util"
)

type BitbucketServerBlueprintRepository struct {
	Client     *BitbucketServerClient
	Name       string
	Url        string
	RepoName   string
	ProjectKey string
	Branch     string
	Username   string
	Token      string
	IsMock     bool
}

func NewBitbucketServerBlueprintRepository(confMap map[string]string) (*BitbucketServerBlueprintRepository, error) {
	// Parse context config
	repo := new(BitbucketServerBlueprintRepository)
	repo.Name = confMap["name"]

	// parse repo name
	if !util.MapContainsKeyWithVal(confMap, "repo-name") {
		return nil, fmt.Errorf("'repo-name' config field must be set for Bitbucket server repository type")
	}
	repo.RepoName = confMap["repo-name"]

	// parse repo url
	if !util.MapContainsKeyWithVal(confMap, "url") {
		return nil, fmt.Errorf("'url' config field must be set for Bitbucket server repository type")
	}
	repo.Url = confMap["url"]

	// parse repo project-key name
	if !util.MapContainsKeyWithVal(confMap, "project-key") {
		return nil, fmt.Errorf("'project-key' config field must be set for Bitbucket server repository type")
	}
	repo.ProjectKey = confMap["project-key"]

	// parse branch name, or set it to default
	if util.MapContainsKeyWithVal(confMap, "branch") {
		repo.Branch = confMap["branch"]
	} else {
		repo.Branch = "master"
	}

	// parse username if exists
	if util.MapContainsKeyWithVal(confMap, "user") {
		repo.Username = confMap["user"]
	}

	// parse token if exists
	if util.MapContainsKeyWithVal(confMap, "token") {
		repo.Token = confMap["token"]
	}

	if repo.Token != "" && repo.Username == "" {
		return nil, fmt.Errorf("'user' config field must be set if the 'token' field is set for Bitbucket server repository type")
	}

	if repo.Username != "" && repo.Token == "" {
		return nil, fmt.Errorf("'token' config field must be set if the 'user' field is set for Bitbucket server repository type")
	}

	// parse mock switch if available
	repo.IsMock = false
	if util.MapContainsKeyWithVal(confMap, "isMock") {
		repo.IsMock, _ = strconv.ParseBool(confMap["isMock"])
	}

	return repo, nil
}

func (repo *BitbucketServerBlueprintRepository) Initialize() error {
	repo.Client = NewBitbucketServerClient(repo.Url, repo.Username, repo.Token, repo.IsMock)
	return nil
}

func (repo *BitbucketServerBlueprintRepository) GetName() string {
	return repo.Name
}

func (repo *BitbucketServerBlueprintRepository) GetProvider() string {
	return models.ProviderBitbucketServer
}

func (repo *BitbucketServerBlueprintRepository) GetInfo() string {
	return fmt.Sprintf(
		"Provider: %s\n  Name: %s\n  Repository name: %s\n  ProjectKey: %s\n  Branch: %s",
		repo.GetProvider(),
		repo.Name,
		repo.RepoName,
		repo.ProjectKey,
		repo.Branch,
	)
}

func (repo *BitbucketServerBlueprintRepository) ListBlueprintsFromRepo() (map[string]*models.BlueprintRemote, []string, error) {
	blueprints := make(map[string]*models.BlueprintRemote)
	var blueprintDirs []string

	branch, err := repo.Client.Repository.GetCommit(repo.ProjectKey, repo.RepoName, repo.Branch)
	if err != nil {
		return nil, nil, err
	}
	sha := branch["id"].(string)

	repositoryFiles, err := repo.Client.Repository.ListFiles(repo.ProjectKey, repo.RepoName, sha)
	if err != nil {
		return nil, nil, err
	}

	// Parse the tree
	currentPath := "."
	for _, entry := range repositoryFiles.Values {
		filename := path.Base(entry)
		link := entry
		parsedUrl, err := url.Parse(link)
		if err != nil {
			return nil, nil, err
		}

		if repository.CheckIfBlueprintDefinitionFile(filename) {
			// If this is blueprints definition, this is considered as the root path for the blueprint
			currentPath = path.Dir(entry)
			blueprintDirs = append(blueprintDirs, currentPath)

			// Add remote definition file to blueprint
			blueprints[currentPath].DefinitionFile = repository.GenerateBlueprintFileDefinition(
				blueprints,
				currentPath,
				filename,
				entry,
				parsedUrl,
			)
		} else {
			if currentPath != "." && path.Dir(entry) != "." {
				// Add remote template file to blueprint
				blueprints[currentPath].AddFile(
					repository.GenerateBlueprintFileDefinition(blueprints, currentPath, filename, entry, parsedUrl),
				)
			}
		}
	}
	return blueprints, blueprintDirs, nil
}

func (repo *BitbucketServerBlueprintRepository) GetFileContents(filePath string) (*[]byte, error) {
	branch, err := repo.Client.Repository.GetCommit(repo.ProjectKey, repo.RepoName, repo.Branch)
	if err != nil {
		return nil, err
	}
	sha := branch["id"].(string)

	fileBlob, err := repo.Client.Repository.GetFileContents(repo.ProjectKey, repo.RepoName, filePath, sha)
	if err != nil {
		return nil, err
	}
	return fileBlob, nil
}

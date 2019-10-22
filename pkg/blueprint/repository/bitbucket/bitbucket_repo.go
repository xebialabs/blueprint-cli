package bitbucket

import (
	"fmt"
	"net/url"
	"path"
	"strconv"

	"github.com/ktrysmt/go-bitbucket"
	"github.com/xebialabs/xl-cli/pkg/blueprint/repository"
	"github.com/xebialabs/xl-cli/pkg/models"
	"github.com/xebialabs/xl-cli/pkg/util"
)

type BitbucketBlueprintRepository struct {
	Client   *BitbucketClient
	Name     string
	RepoName string
	Owner    string
	Branch   string
	Token    string
	IsMock   bool
}

func NewBitbucketBlueprintRepository(confMap map[string]string) (*BitbucketBlueprintRepository, error) {
	// Parse context config
	repo := new(BitbucketBlueprintRepository)
	repo.Name = confMap["name"]

	// parse repo name
	if !util.MapContainsKeyWithVal(confMap, "repo-name") {
		return nil, fmt.Errorf("'repo-name' config field must be set for GitHub repository type")
	}
	repo.RepoName = confMap["repo-name"]

	// parse repo owner name
	if !util.MapContainsKeyWithVal(confMap, "owner") {
		return nil, fmt.Errorf("'owner' config field must be set for GitHub repository type")
	}
	repo.Owner = confMap["owner"]

	// parse branch name, or set it to default
	if util.MapContainsKeyWithVal(confMap, "branch") {
		repo.Branch = confMap["branch"]
	} else {
		repo.Branch = "master"
	}

	// parse token if exists
	if util.MapContainsKeyWithVal(confMap, "token") {
		repo.Token = confMap["token"]
	}

	// parse mock switch if available
	repo.IsMock = false
	if util.MapContainsKeyWithVal(confMap, "isMock") {
		repo.IsMock, _ = strconv.ParseBool(confMap["isMock"])
	}

	return repo, nil
}

func (repo *BitbucketBlueprintRepository) Initialize() error {
	repo.Client = NewBitbucketClient(repo.Owner, repo.Token, repo.IsMock)
	return nil
}

func (repo *BitbucketBlueprintRepository) GetName() string {
	return repo.Name
}

func (repo *BitbucketBlueprintRepository) GetProvider() string {
	return models.ProviderBitbucket
}

func (repo *BitbucketBlueprintRepository) GetInfo() string {
	return fmt.Sprintf(
		"Provider: %s\n  Name: %s\n  Repository name: %s\n  Owner: %s\n  Branch: %s",
		repo.GetProvider(),
		repo.Name,
		repo.RepoName,
		repo.Owner,
		repo.Branch,
	)
}

func (repo *BitbucketBlueprintRepository) ListBlueprintsFromRepo() (map[string]*models.BlueprintRemote, []string, error) {
	blueprints := make(map[string]*models.BlueprintRemote)
	var blueprintDirs []string

	co := &bitbucket.CommitsOptions{
		Owner:    repo.Owner,
		RepoSlug: repo.RepoName,
		Revision: repo.Branch,
	}

	branch, err := repo.Client.Commits.GetCommit(co)
	if err != nil {
		return nil, nil, err
	}

	sha := branch.(map[string]interface{})["hash"].(string)

	ro := &bitbucket.RepositoryFilesOptions{
		Owner:    repo.Owner,
		RepoSlug: repo.RepoName,
		Ref:      sha,
	}
	files, err := repo.Client.Repository.ListFiles(ro)
	if err != nil {
		return nil, nil, err
	}

	// Parse the tree
	currentPath := "."
	for _, entry := range files {
		filename := path.Base(entry.Path)
		link := entry.Links["self"].(map[string]interface{})
		parsedUrl, err := url.Parse(link["href"].(string))
		if err != nil {
			return nil, nil, err
		}

		if repository.CheckIfBlueprintDefinitionFile(filename) {
			// If this is blueprints definition, this is considered as the root path for the blueprint
			currentPath = path.Dir(entry.Path)
			blueprintDirs = append(blueprintDirs, currentPath)

			// Add remote definition file to blueprint
			blueprints[currentPath].DefinitionFile = repository.GenerateBlueprintFileDefinition(
				blueprints,
				currentPath,
				filename,
				entry.Path,
				parsedUrl,
			)
		} else {
			if currentPath != "." && path.Dir(entry.Path) != "." {
				// Add remote template file to blueprint
				blueprints[currentPath].AddFile(
					repository.GenerateBlueprintFileDefinition(blueprints, currentPath, filename, entry.Path, parsedUrl),
				)
			}
		}
	}
	return blueprints, blueprintDirs, nil
}

func (repo *BitbucketBlueprintRepository) GetFileContents(filePath string) (*[]byte, error) {
	rbo := &bitbucket.RepositoryBlobOptions{
		Owner:    repo.Owner,
		RepoSlug: repo.RepoName,
		Ref:      repo.Branch,
		Path:     filePath,
	}
	fileBlob, err := repo.Client.Repository.GetFileBlob(rbo)
	if err != nil {
		return nil, err
	}
	return &fileBlob.Content, nil
}

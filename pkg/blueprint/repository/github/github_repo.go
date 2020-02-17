package github

import (
	"bytes"
	"fmt"
	"net/url"
	"path"
	"strconv"

	"github.com/google/go-github/github"
	"github.com/xebialabs/blueprint-cli/pkg/blueprint/repository"
	"github.com/xebialabs/blueprint-cli/pkg/models"
	"github.com/xebialabs/blueprint-cli/pkg/util"
)

type GitHubBlueprintRepository struct {
	Client   *GithubClient
	Name     string
	RepoName string
	Owner    string
	Branch   string
	Token    string
	IsMock   bool
}

func NewGitHubBlueprintRepository(confMap map[string]string) (*GitHubBlueprintRepository, error) {
	// Parse context config
	repo := new(GitHubBlueprintRepository)
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

func (repo *GitHubBlueprintRepository) Initialize() error {
	repo.Client = NewGithubClient(repo.Token, repo.IsMock)
	return nil
}

func (repo *GitHubBlueprintRepository) GetName() string {
	return repo.Name
}

func (repo *GitHubBlueprintRepository) GetProvider() string {
	return models.ProviderGitHub
}

func (repo *GitHubBlueprintRepository) GetInfo() string {
	return fmt.Sprintf(
		"Provider: %s\n  Name: %s\n  Repository name: %s\n  Owner: %s\n  Branch: %s",
		repo.GetProvider(),
		repo.Name,
		repo.RepoName,
		repo.Owner,
		repo.Branch,
	)
}

func (repo *GitHubBlueprintRepository) ListBlueprintsFromRepo() (map[string]*models.BlueprintRemote, []string, error) {
	blueprints := make(map[string]*models.BlueprintRemote)
	var blueprintDirs []string

	// Get latest SHA of the requested branch
	branch, _, err := repo.Client.Repositories.GetBranch(repo.Client.Context, repo.Owner, repo.RepoName, repo.Branch)
	if err != nil {
		return nil, nil, err
	}
	sha := branch.GetCommit().GetSHA()

	// Get GIT tree
	tree, _, err := repo.Client.Git.GetTree(repo.Client.Context, repo.Owner, repo.RepoName, sha, true)
	if err != nil {
		if _, ok := err.(*github.RateLimitError); ok {
			return nil, nil, fmt.Errorf("GitHub rate limit error: %s", err.Error())
		} else if _, ok := err.(*github.AcceptedError); ok {
			return nil, nil, fmt.Errorf("GitHub API error (scheduled on GitHub side): %s", err.Error())
		} else {
			return nil, nil, err
		}
	}

	// Parse GIT tree
	currentPath := "."
	for _, entry := range tree.Entries {
		filename := path.Base(entry.GetPath())
		parsedUrl, err := url.Parse(entry.GetURL())
		if err != nil {
			return nil, nil, err
		}

		if repository.CheckIfBlueprintDefinitionFile(filename) {
			// If this is blueprints definition, this is considered as the root path for the blueprint
			currentPath = path.Dir(entry.GetPath())
			blueprintDirs = append(blueprintDirs, currentPath)

			// Add remote definition file to blueprint
			blueprints[currentPath].DefinitionFile = repository.GenerateBlueprintFileDefinition(
				blueprints,
				currentPath,
				filename,
				entry.GetPath(),
				parsedUrl,
			)
		} else if entry.GetType() == "tree" {
			// pass
		} else {
			// Bypass root items
			if currentPath != "." && path.Dir(entry.GetPath()) != "." {
				// Add remote template file to blueprint
				blueprints[currentPath].AddFile(
					repository.GenerateBlueprintFileDefinition(blueprints, currentPath, filename, entry.GetPath(), parsedUrl),
				)
			}
		}
	}
	return blueprints, blueprintDirs, nil
}

func (repo *GitHubBlueprintRepository) GetFileContents(filePath string) (*[]byte, error) {
	fileContent, _, _, err := repo.Client.Repositories.GetContents(
		repo.Client.Context,
		repo.Owner,
		repo.RepoName,
		filePath,
		&github.RepositoryContentGetOptions{Ref: repo.Branch},
	)
	if err != nil {
		if isTooLargeBlobError(err) {
			util.Verbose("[github] File '%s' is larger than 1MB, retrying with blob API\n", filePath)
			contentBytes, _, err := repo.GetLargeFileContents(filePath)
			if err != nil {
				return nil, err
			}
			return &contentBytes, nil
		} else {
			return nil, err
		}
	}
	content, err := fileContent.GetContent()
	if err != nil {
		return nil, err
	}
	contentBytes := []byte(content)
	return &contentBytes, nil
}

func (repo *GitHubBlueprintRepository) GetLargeFileContents(filePath string) ([]byte, int64, error) {
	reader, err := repo.Client.Repositories.DownloadContents(
		repo.Client.Context,
		repo.Owner,
		repo.RepoName,
		filePath,
		&github.RepositoryContentGetOptions{Ref: repo.Branch},
	)
	if err != nil {
		return nil, 0, err
	}
	buffer := new(bytes.Buffer)
	size, err := buffer.ReadFrom(reader)
	if err != nil {
		reader.Close()
		return nil, 0, err
	}
	err = reader.Close()
	if err != nil {
		return nil, 0, err
	}
	util.Verbose("[github] Read '%d' bytes of file '%s'\n", size, filePath)
	return buffer.Bytes(), size, nil
}

// utility functions
func isTooLargeBlobError(err error) bool {
	if giterr, ok := err.(*github.ErrorResponse); ok {
		if giterr != nil && giterr.Errors != nil {
			for _, entry := range giterr.Errors {
				if entry.Code == "too_large" {
					return true
				}
			}
		}
	}
	return false
}

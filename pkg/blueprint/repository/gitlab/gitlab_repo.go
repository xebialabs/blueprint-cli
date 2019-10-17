package gitlab

import (
	"fmt"
	"net/url"
	"path"
	"strconv"

	"github.com/xanzy/go-gitlab"

	"github.com/xebialabs/xl-cli/pkg/blueprint/repository"
	"github.com/xebialabs/xl-cli/pkg/models"
	"github.com/xebialabs/xl-cli/pkg/util"
)

type GitLabBlueprintRepository struct {
	Client   *GitLabClient
	Name     string
	Url      string
	RepoName string
	Owner    string
	Branch   string
	Token    string
	IsMock   bool
}

func NewGitLabBlueprintRepository(confMap map[string]string) (*GitLabBlueprintRepository, error) {
	// Parse context config
	repo := new(GitLabBlueprintRepository)
	repo.Name = confMap["name"]

	// parse repo name
	if !util.MapContainsKeyWithVal(confMap, "repo-name") {
		return nil, fmt.Errorf("'repo-name' config field must be set for GitLab repository type")
	}
	repo.RepoName = confMap["repo-name"]

	// parse repo owner name
	if !util.MapContainsKeyWithVal(confMap, "owner") {
		return nil, fmt.Errorf("'owner' config field must be set for GitLab repository type")
	}
	repo.Owner = confMap["owner"]

	// parse repo url
	if !util.MapContainsKeyWithVal(confMap, "url") {
		return nil, fmt.Errorf("'url' config field must be set for GitLab repository type")
	}
	repo.Url = confMap["url"]

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

func (repo *GitLabBlueprintRepository) Initialize() error {
	repo.Client = NewGitLabClient(repo.Token, repo.IsMock)
	if repo.Client.GitLabClient != nil {
        err := repo.Client.GitLabClient.SetBaseURL(fmt.Sprintf("%s/api/v4", repo.Url))
        if err != nil {
            return err
        }
    }
	return nil
}

func (repo *GitLabBlueprintRepository) GetName() string {
	return repo.Name
}

func (repo *GitLabBlueprintRepository) GetProvider() string {
	return models.ProviderGitLab
}

func (repo *GitLabBlueprintRepository) GetInfo() string {
	return fmt.Sprintf(
		"Provider: %s\n  Name: %s\n  Repository name: %s\n  Owner: %s\n  Branch: %s",
		repo.GetProvider(),
		repo.Name,
		repo.RepoName,
		repo.Owner,
		repo.Branch,
	)
}

func (repo *GitLabBlueprintRepository) ListBlueprintsFromRepo() (map[string]*models.BlueprintRemote, []string, error) {
	blueprints := make(map[string]*models.BlueprintRemote)
	var blueprintDirs []string

	// Get latest SHA of the requested branch
	branch, _, err := repo.Client.Branches.GetBranch(fmt.Sprintf("%s/%s", repo.Owner, repo.RepoName), repo.Branch, nil)
	if err != nil {
		return nil, nil, err
	}
	sha := branch.Commit.ID

	lto := &gitlab.ListTreeOptions{
		ListOptions: gitlab.ListOptions{
			PerPage: 100,
			Page:    1,
		},
		Ref:       gitlab.String(sha),
		Recursive: gitlab.Bool(true),
	}

	// Get GIT tree
	var tree []*gitlab.TreeNode

	for {
		tmpTree, response, err := repo.Client.Repositories.ListTree(fmt.Sprintf("%s/%s", repo.Owner, repo.RepoName), lto)
		if err != nil {
			return nil, nil, err
		}

		tree = append(tree, tmpTree...)

		if response.CurrentPage >= response.TotalPages {
			break
		}

		lto.Page = response.NextPage
	}

	// Parse GIT tree
	currentPath := "."
	for _, entry := range tree {
		filename := path.Base(entry.Path)
		parsedUrl, err := url.Parse(entry.Path)
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
		} else if entry.Type == "tree" {
			// pass
		} else {
			// Bypass root items
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

func (repo *GitLabBlueprintRepository) GetFileContents(filePath string) (*[]byte, error) {
	// Get latest SHA of the requested branch
	branch, _, err := repo.Client.Branches.GetBranch(fmt.Sprintf("%s/%s", repo.Owner, repo.RepoName), repo.Branch, nil)
	if err != nil {
		return nil, err
	}
	sha := branch.Commit.ID

	rfo := &gitlab.GetRawFileOptions{
		Ref: gitlab.String(sha),
	}

	contentBytes, _, err := repo.Client.RepositoryFiles.GetRawFile(fmt.Sprintf("%s/%s", repo.Owner, repo.RepoName), filePath, rfo, nil)
	if err != nil {
		return nil, err
	}
	return &contentBytes, nil
}

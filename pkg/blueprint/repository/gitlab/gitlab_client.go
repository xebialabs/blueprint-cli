package gitlab

import (
	"fmt"
	"net/url"
	"os"
	"path/filepath"

	gitlab "gitlab.com/gitlab-org/api/client-go"
)

/*
 * GitLab Client Interfaces & Wrapper
 */

// GitLab Repository Service Interface
type gitlabRepositoriesService interface {
	ListTree(pid interface{}, opt *gitlab.ListTreeOptions, options ...gitlab.RequestOptionFunc) ([]*gitlab.TreeNode, *gitlab.Response, error)
}

// GitLab Branches Service Interface
type gitlabBranchesService interface {
	GetBranch(pid interface{}, branch string, options ...gitlab.RequestOptionFunc) (*gitlab.Branch, *gitlab.Response, error)
}

// GitLab RepositoryFiles Service Interface
type gitlabRepositoryFilesService interface {
	GetRawFile(pid interface{}, fileName string, opt *gitlab.GetRawFileOptions, options ...gitlab.RequestOptionFunc) ([]byte, *gitlab.Response, error)
}

// GitHub Client Wrapper
type GitLabClient struct {
	GitLabClient    *gitlab.Client
	Repositories    gitlabRepositoriesService
	Branches        gitlabBranchesService
	RepositoryFiles gitlabRepositoryFilesService
	BaseURL         *url.URL
}

func NewGitLabClient(token string, isMock bool, baseURL string) *GitLabClient {
	if isMock {
		// return mock gitlab client for testing purposes
		workDir, _ := os.Getwd()
		testFileFetcher := localFileFetcher{
			SourceDir: filepath.Join(workDir, "..", "..", "..", "..", "mock", "gitlab"),
			FileExt:   ".json",
		}
		return &GitLabClient{
			GitLabClient:    nil,
			Repositories:    &mockRepositoriesService{client: testFileFetcher},
			Branches:        &mockBranchesService{client: testFileFetcher},
			RepositoryFiles: &mockRepositoryFilesService{client: testFileFetcher},
		}
	} else {
		// return GitLab API client with/without authentication
		client, err := gitlab.NewClient(token, gitlab.WithBaseURL(fmt.Sprintf("%s/api/v4", baseURL)))
		if err != nil {
			panic(err)
		}
		return &GitLabClient{
			GitLabClient:    client,
			Repositories:    client.Repositories,
			Branches:        client.Branches,
			RepositoryFiles: client.RepositoryFiles,
			BaseURL:         client.BaseURL(),
		}
	}
}

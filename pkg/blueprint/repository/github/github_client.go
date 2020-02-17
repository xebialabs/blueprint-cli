package github

import (
	"github.com/google/go-github/github"
	"golang.org/x/net/context"
	"golang.org/x/oauth2"
	"io"
	"net/http"
	"os"
	"path/filepath"
)

/*
 * GitHub Client Interfaces & Wrapper
 * Original implementation is wrapped for proper mocking in tests
 * Reference: https://github.com/google/go-github/issues/113
 */

// Github Repository Service Interface
type githubRepoService interface {
	GetBranch(ctx context.Context, owner, repo, branch string) (*github.Branch, *github.Response, error)
	GetContents(ctx context.Context, owner, repo, path string, opt *github.RepositoryContentGetOptions) (fileContent *github.RepositoryContent, directoryContent []*github.RepositoryContent, resp *github.Response, err error)
	DownloadContents(ctx context.Context, owner, repo, filepath string, opt *github.RepositoryContentGetOptions) (io.ReadCloser, error)
}

// Github GIT Service Interface
type githubGitService interface {
	GetTree(ctx context.Context, owner string, repo string, sha string, recursive bool) (*github.Tree, *github.Response, error)
}

// GitHub Client Wrapper
type GithubClient struct {
	GithubClient *github.Client
	Context      context.Context
	Repositories githubRepoService
	Git          githubGitService
}

func NewGithubClient(token string, isMock bool) *GithubClient {
	if isMock {
		// return mock github client for testing purposes
		workDir, _ := os.Getwd()
		testFileFetcher := localFileFetcher{
			SourceDir: filepath.Join(workDir, "..", "..", "..", "..", "mock", "github"),
			FileExt:   ".json",
		}
		return &GithubClient{
			GithubClient: nil,
			Context:      context.Background(),
			Repositories: &mockRepoService{client: testFileFetcher},
			Git:          &mockGitService{client: testFileFetcher},
		}
	} else {
		// return Github API client with/without authentication
		githubContext := context.Background()
		var tc *http.Client
		var ts oauth2.TokenSource
		if token != "" {
			ts = oauth2.StaticTokenSource(&oauth2.Token{AccessToken: token})
		}
		tc = oauth2.NewClient(githubContext, ts)
		client := github.NewClient(tc)

		return &GithubClient{
			GithubClient: client,
			Context:      githubContext,
			Repositories: client.Repositories,
			Git:          client.Git,
		}
	}
}

package bitbucketserver

import (
	"os"
	"path/filepath"
)

type bitbucketServerRepoService interface {
	GetCommit(projectKey string, repo string, branch string) (map[string]interface{}, error)
	ListFiles(projectKey string, repo string, sha string) (*RepositoryFiles, error)
	GetFileContents(projectKey string, repo string, filePath string, sha string) (*[]byte, error)
}

type BitbucketServerClient struct {
	BitbucketServerClient *Client
	Repository            bitbucketServerRepoService
}

func NewBitbucketServerClient(url string, username string, token string, isMock bool) *BitbucketServerClient {
	if isMock {
		// return mock github client for testing purposes
		workDir, _ := os.Getwd()
		testFileFetcher := localFileFetcher{
			SourceDir: filepath.Join(workDir, "..", "..", "..", "..", "mock", "bitbucketserver"),
			FileExt:   ".json",
		}
		return &BitbucketServerClient{
			BitbucketServerClient: nil,
			Repository:            &mockRepoService{client: testFileFetcher},
		}
	} else {
		client := NewClient(url, username, token)

		return &BitbucketServerClient{
			BitbucketServerClient: client,
			Repository:            client.Repository,
		}
	}
}

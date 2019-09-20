package bitbucket

import (
	"os"
	"path/filepath"

	"github.com/xebialabs/go-bitbucket"
	"golang.org/x/net/context"
)

type bitbucketRepoService interface {
	GetFileBlob(ro *bitbucket.RepositoryBlobOptions) (*bitbucket.RepositoryBlob, error)
	ListFiles(ro *bitbucket.RepositoryFilesOptions) ([]bitbucket.RepositoryFile, error)
}

type bitbucketCommitService interface {
	GetCommit(co *bitbucket.CommitsOptions) (interface{}, error)
}

// GitHub Client Wrapper
type BitbucketClient struct {
	BitbucketClient *bitbucket.Client
	Context         context.Context
	Commits         bitbucketCommitService
	Repository      bitbucketRepoService
}

func NewBitbucketClient(owner string, token string, isMock bool) *BitbucketClient {
	if isMock {
		// return mock github client for testing purposes
		workDir, _ := os.Getwd()
		testFileFetcher := localFileFetcher{
			SourceDir: filepath.Join(workDir, "..", "..", "..", "..", "bitbucket-mock"),
			FileExt:   ".json",
		}
		return &BitbucketClient{
			BitbucketClient: nil,
			Context:         context.Background(),
			Commits:         &mockCommitsService{client: testFileFetcher},
			Repository:      &mockRepoService{client: testFileFetcher},
		}
	} else {
		// return Github API client with/without authentication
		bitbucketContext := context.Background()
		client := bitbucket.NewBasicAuth(owner, token)

		client.Pagelen = 100
		client.MaxDepth = 10

		return &BitbucketClient{
			BitbucketClient: client,
			Context:         bitbucketContext,
			Commits:         client.Repositories.Commits,
			Repository:      client.Repositories.Repository,
		}
	}
}

package github

import (
    "context"
    "encoding/json"
    "github.com/google/go-github/github"
    "io"
    "net/url"
    "os"
    "path/filepath"
)

// GitHub Mock Service Base & Client to fetch local JSON test files
type githubMockService struct {
	client localFileFetcher
}

type localFileFetcher struct {
	SourceDir string
	FileExt   string
}

func (c *localFileFetcher) GetFileReader(appendExt bool, params ...string) (*os.File, error) {
    params = append([]string {c.SourceDir}, params...)
	sourcePath := filepath.Join(params...)
	if appendExt {
		// if this is an API call, append default file extension
		sourcePath += c.FileExt
	}
	return os.Open(sourcePath)
}

func (c *localFileFetcher) DecodeToGithubEntity(fileReader io.Reader, v interface{}) error {
	var err error
	if w, ok := v.(io.Writer); ok {
		_, err = io.Copy(w, fileReader)
	} else {
		decErr := json.NewDecoder(fileReader).Decode(v)
		if decErr == io.EOF {
			decErr = nil // ignore EOF errors caused by empty response body
		}
		if decErr != nil {
			err = decErr
		}
	}
	return err
}

// Repository Service Mock implementation for tests
type mockRepoService githubMockService

func (s *mockRepoService) GetBranch(ctx context.Context, owner, repo, branch string) (*github.Branch, *github.Response, error) {
	// try to find local file
	fileReader, err := s.client.GetFileReader(true, "repos", owner, repo, "branches", branch)
	if err != nil {
		return nil, nil, err
	}

	// decode file contents to Github entity type
	b := new(github.Branch)
	err = s.client.DecodeToGithubEntity(fileReader, b)
	return b, nil, err
}

// Repository Service Mock implementation for tests
func (s *mockRepoService) GetContents(ctx context.Context, owner, repo, path string, opt *github.RepositoryContentGetOptions) (fileContent *github.RepositoryContent, directoryContent []*github.RepositoryContent, resp *github.Response, err error) {
	// try to find local file
	escapedPath := (&url.URL{Path: path}).String()
	fileReader, err := s.client.GetFileReader(true, "repos", owner, repo, "contents", escapedPath)
	if err != nil {
		return nil, nil, nil, err
	}

	// decode file contents to raw JSon type
    var rawJSON json.RawMessage
    err = s.client.DecodeToGithubEntity(fileReader, &rawJSON)
    if err != nil {
        return nil, nil, nil, err
    }

    // unmarshal raw JSON to Github entity
    fileUnmarshalError := json.Unmarshal(rawJSON, &fileContent)
    if fileUnmarshalError == nil {
        return fileContent, nil, resp, nil
    }
	return fileContent, nil, nil, nil
}

func (s *mockRepoService) DownloadContents(ctx context.Context, owner, repo, filepath string, opt *github.RepositoryContentGetOptions) (io.ReadCloser, error) {
	// try to find local file
	escapedPath := (&url.URL{Path: filepath}).String()
	fileReader, err := s.client.GetFileReader(false, "repos", owner, repo, "contents", escapedPath)
	if err != nil {
		return nil, err
	}
	return fileReader, nil
}

// GIT Service Mock implementation for tests
type mockGitService githubMockService

func (s *mockGitService) GetTree(ctx context.Context, owner string, repo string, sha string, recursive bool) (*github.Tree, *github.Response, error) {
	// try to find local file
	fileReader, err := s.client.GetFileReader(true, "repos", owner, repo, "git", "trees", sha)
	if err != nil {
		return nil, nil, err
	}

	// decode file contents to Github entity type
	t := new(github.Tree)
	err = s.client.DecodeToGithubEntity(fileReader, t)
	return t, nil, err
}

package bitbucketserver

import (
	"encoding/json"
	"io"
	"net/url"
	"os"
	"path/filepath"
)

// Bitbucket Mock Service Base & Client to fetch local JSON test files
type bitbucketServerMockService struct {
	client localFileFetcher
}

type localFileFetcher struct {
	SourceDir string
	FileExt   string
}

func (c *localFileFetcher) GetFileReader(appendExt bool, params ...string) (*os.File, error) {
	params = append([]string{c.SourceDir}, params...)
	sourcePath := filepath.Join(params...)
	if appendExt {
		// if this is an API call, append default file extension
		sourcePath += c.FileExt
	}
	return os.Open(sourcePath)
}

func (c *localFileFetcher) DecodeToBitbucketEntity(fileReader io.Reader, v interface{}) error {
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
type mockRepoService bitbucketServerMockService

func (s *mockRepoService) GetCommit(projectKey string, repo string, branch string) (map[string]interface{}, error) {
	fileReader, err := s.client.GetFileReader(true, "repos", projectKey, repo, "branches", branch)
	if err != nil {
		return nil, err
	}

	var b map[string]interface{}
	if err = s.client.DecodeToBitbucketEntity(fileReader, &b); err != nil {
		return nil, err
	}

	// decode file contents to Bitbucket entity type
	return b, nil
}

func (s *mockRepoService) ListFiles(projectKey string, repo string, sha string) (*RepositoryFiles, error) {
	fileReader, err := s.client.GetFileReader(true, "repos", projectKey, repo, "git", "trees", sha)
	if err != nil {
		return nil, err
	}

	t := new(RepositoryFiles)
	err = s.client.DecodeToBitbucketEntity(fileReader, t)

	if err != nil {
		return nil, err
	}

	return t, nil
}

func (s *mockRepoService) GetFileContents(projectKey string, repo string, filePath string, sha string) (*[]byte, error) {
	//try to find local file
	escapedPath := (&url.URL{Path: filePath}).String()
	fileReader, err := s.client.GetFileReader(false, "repos", projectKey, repo, "contents", escapedPath)
	if err != nil {
		return nil, err
	}

	t := make([]byte, 0)
	tmp := make([]byte, 4096)

	for {
		bytesRead, err := fileReader.Read(tmp)
		if err != nil {
			return nil, err
		}
		t = append(t, tmp[0:bytesRead]...)

		if bytesRead < len(tmp) {
			break
		}
	}

	return &t, nil
}

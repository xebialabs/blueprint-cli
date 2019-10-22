package bitbucket

import (
	"encoding/json"
	"io"
	"net/url"
	"os"
	"path/filepath"

	"github.com/ktrysmt/go-bitbucket"
)

// Bitbucket Mock Service Base & Client to fetch local JSON test files
type bitbucketMockService struct {
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

type RepositoryFiles struct {
	Values []RepositoryFile `json:"values,omitempty"`
}

type RepositoryFile struct {
	Mimetype   string                 `json:"mimetype,omitempty"`
	Links      map[string]interface{} `json:"links,omitempty"`
	Path       string                 `json:"path,omitempty"`
	Commit     map[string]interface{} `json:"commit,omitempty"`
	Attributes []string               `json:"attributes,omitempty"`
	Type       string                 `json:"type,omitempty"`
	Size       int                    `json:"size,omitempty"`
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
type mockCommitsService bitbucketMockService

func (s *mockCommitsService) GetCommit(co *bitbucket.CommitsOptions) (interface{}, error) {
	// try to find local file
	fileReader, err := s.client.GetFileReader(true, "repos", co.Owner, co.RepoSlug, "branches", co.Revision)
	if err != nil {
		return nil, err
	}

	var b map[string]interface{}
	err = s.client.DecodeToBitbucketEntity(fileReader, &b)
	if err != nil {
		return nil, err
	}

	// decode file contents to Github entity type
	return b, nil
}

// Repository Service Mock implementation for tests
type mockRepoService bitbucketMockService

func (s *mockRepoService) GetFileBlob(ro *bitbucket.RepositoryBlobOptions) (fileContent *bitbucket.RepositoryBlob, err error) {
	//try to find local file
	escapedPath := (&url.URL{Path: ro.Path}).String()
	fileReader, err := s.client.GetFileReader(true, "repos", ro.Owner, ro.RepoSlug, "contents", escapedPath)
	if err != nil {
		return nil, err
	}

	var rawJSON json.RawMessage
	err = s.client.DecodeToBitbucketEntity(fileReader, &rawJSON)
	if err != nil {
		return nil, err
	}

	// unmarshal raw JSON to Github entity
	fileUnmarshalError := json.Unmarshal(rawJSON, &fileContent)
	if fileUnmarshalError == nil {
		return fileContent, nil
	}
	return fileContent, nil
}

func (s *mockRepoService) ListFiles(ro *bitbucket.RepositoryFilesOptions) ([]bitbucket.RepositoryFile, error) {
	// try to find local file
	fileReader, err := s.client.GetFileReader(true, "repos", ro.Owner, ro.RepoSlug, "git", "trees", ro.Ref)
	if err != nil {
		return nil, err
	}

	t := new(RepositoryFiles)
	err = s.client.DecodeToBitbucketEntity(fileReader, &t)

	if err != nil {
		return nil, err
	}

	repositoryFiles := make([]bitbucket.RepositoryFile, len(t.Values))
	for i, rf := range t.Values {
		repositoryFiles[i] = bitbucket.RepositoryFile{
			Mimetype:   rf.Mimetype,
			Links:      rf.Links,
			Path:       rf.Path,
			Commit:     rf.Commit,
			Attributes: rf.Attributes,
			Type:       rf.Type,
			Size:       rf.Size,
		}
	}

	return repositoryFiles, nil
}

package gitlab

import (
	"encoding/json"
	"io"
	"net/url"
	"os"
	"path/filepath"
	"strings"

	"github.com/xanzy/go-gitlab"
)

// GitLab Mock Service Base & Client to fetch local JSON test files
type gitlabMockService struct {
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

func (c *localFileFetcher) DecodeToGitLabEntity(fileReader io.Reader, v interface{}) error {
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
type mockRepositoriesService gitlabMockService

func (s *mockRepositoriesService) ListTree(pid interface{}, opt *gitlab.ListTreeOptions, options ...gitlab.OptionFunc) ([]*gitlab.TreeNode, *gitlab.Response, error) {
	ownerRepo := strings.Split(pid.(string), "/")

	// try to find local file
	fileReader, err := s.client.GetFileReader(true, "repos", ownerRepo[0], ownerRepo[1], "git", "trees", *opt.Ref)
	if err != nil {
		return nil, nil, err
	}

	// decode file contents to Github entity type
	var t []*gitlab.TreeNode

	err = s.client.DecodeToGitLabEntity(fileReader, &t)
	if err != nil {
		return nil, nil, err
	}

	response := gitlab.Response{
		TotalPages:  1,
		CurrentPage: 1,
	}

	return t, &response, nil
}

// Branches Service Mock implementation for tests
type mockBranchesService gitlabMockService

func (s *mockBranchesService) GetBranch(pid interface{}, branch string, options ...gitlab.OptionFunc) (*gitlab.Branch, *gitlab.Response, error) {
	ownerRepo := strings.Split(pid.(string), "/")

	// try to find local file
	fileReader, err := s.client.GetFileReader(true, "repos", ownerRepo[0], ownerRepo[1], "branches", branch)
	if err != nil {
		return nil, nil, err
	}

	// decode file contents to GitLab entity type
	b := new(gitlab.Branch)
	err = s.client.DecodeToGitLabEntity(fileReader, b)
	return b, nil, err
}

// RepositoryFiles Services Mock implementation for tests
type mockRepositoryFilesService gitlabMockService

func (s *mockRepositoryFilesService) GetRawFile(pid interface{}, fileName string, opt *gitlab.GetRawFileOptions, options ...gitlab.OptionFunc) ([]byte, *gitlab.Response, error) {
	ownerRepo := strings.Split(pid.(string), "/")
	escapedPath := (&url.URL{Path: fileName}).String()
	fileReader, err := s.client.GetFileReader(false, "repos", ownerRepo[0], ownerRepo[1], "contents", escapedPath)
	if err != nil {
		return nil, nil, err
	}

	t := make([]byte, 0)
	tmp := make([]byte, 4096)

	for {
		bytesRead, err := fileReader.Read(tmp)
		if err != nil {
			return nil, nil, err
		}
		t = append(t, tmp[0:bytesRead]...)

		if bytesRead < len(tmp) {
			break
		}
	}

	return t, nil, nil
}

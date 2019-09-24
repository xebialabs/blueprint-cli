package bitbucketserver

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
)

type RepositoryFiles struct {
	Values        []string `json:"values,omitempty"`
	Size          int      `json:"size,omitempty"`
	IsLastPage    bool     `json:"isLastPage,omitempty"`
	Start         int      `json:"start,omitempty"`
	Limit         int      `json:"limit,omitempty"`
	NextPageStart int      `json:"nextPageStart,omitempty"`
}

type Client struct {
	Url        string
	Username   string
	Token      string
	Repository *BitbucketServerRepository
}

func NewClient(url string, username string, token string) *Client {
	client := &Client{
		Url:      url,
		Username: username,
		Token:    token,
	}

	client.Repository = &BitbucketServerRepository{
		c: client,
	}

	return client
}

type BitbucketServerRepository struct {
	c *Client
}

func (b *BitbucketServerRepository) createRequest(url string) (*[]byte, error) {
	request, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}

	if b.c.Username != "" && b.c.Token != "" {
		request.SetBasicAuth(b.c.Username, b.c.Token)
	}

	client := &http.Client{}
	response, err := client.Do(request)
	if err != nil {
		return nil, err
	}

	if response.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("%d:%s", response.StatusCode, http.StatusText(response.StatusCode))
	}

	bytes, err := ioutil.ReadAll(response.Body)
	if err != nil {
		return nil, err
	}

	if len(bytes) == 0 {
		return nil, fmt.Errorf("received an empty response from the server")
	}

	return &bytes, nil
}

func (b *BitbucketServerRepository) GetCommit(projectKey string, repo string, branch string) (map[string]interface{}, error) {
	url := fmt.Sprintf("%s/rest/api/1.0/projects/%s/repos/%s/commits/%s", b.c.Url, projectKey, repo, branch)
	bytes, err := b.createRequest(url)
	if err != nil {
		return nil, err
	}

	var body map[string]interface{}
	if err = json.Unmarshal(*bytes, &body); err != nil {
		return nil, err
	}

	return body, nil
}

func (b *BitbucketServerRepository) ListFiles(projectKey string, repo string, sha string) (*RepositoryFiles, error) {
	url := fmt.Sprintf("%s/rest/api/1.0/projects/%s/repos/%s/files?limit=1000&at=%s", b.c.Url, projectKey, repo, sha)
	bytes, err := b.createRequest(url)
	if err != nil {
		return nil, err
	}

	var files *RepositoryFiles
	if err = json.Unmarshal(*bytes, &files); err != nil {
		return nil, err
	}

	return files, nil
}

func (b *BitbucketServerRepository) GetFileContents(projectKey string, repo string, filePath string, sha string) (*[]byte, error) {
	url := fmt.Sprintf("%s/rest/api/1.0/projects/%s/repos/%s/raw/%s?at=%s", b.c.Url, projectKey, repo, filePath, sha)
	bytes, err := b.createRequest(url)
	if err != nil {
		return nil, err
	}

	return bytes, nil
}

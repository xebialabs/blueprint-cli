package github

import (
	"fmt"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func getGitHubTokenFromEnvVar(t *testing.T) string {
	token := os.Getenv("XL_CLI_GITHUB_TOKEN")
	if token != "" {
		t.Log(fmt.Sprintf("Found GitHub token in env vars!"))
	}
	return token
}

func getDefaultConfMap(t *testing.T) map[string]string {
	return map[string]string{
		"name":      "test",
		"type":      "github",
		"repo-name": "blueprints",
		"owner":     "xebialabs",
		"branch":    "master",
		"token":     getGitHubTokenFromEnvVar(t),
	}
}

func TestGitHubBlueprintRepository_GetFileContents(t *testing.T) {
	repo, err := NewGitHubBlueprintRepository(getDefaultConfMap(t))
	require.Nil(t, err)
	err = repo.Initialize()
	require.Nil(t, err)

	tests := []struct {
		name     string
		filePath string
		want     bool
		wantErr  bool
	}{
		{
			"should error when file not found",
			"test",
			false,
			true,
		},
		{
			"should fetch file of size less than 1MB",
			"aws/datalake/cloudformation/data-lake-api.yaml",
			true,
			false,
		},
		{
			"should fetch blob of size more than 1MB",
			"aws/datalake/cloudformation/data-lake-artifacts.zip",
			true,
			false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := repo.GetFileContents(tt.filePath)
			if (err != nil) != tt.wantErr {
				t.Errorf("GitHubBlueprintRepository.GetFileContents() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.want {
				if got == nil {
					t.Errorf("GitHubBlueprintRepository.GetFileContents() is nil, expected not nil")
				}
			}
		})
	}
}

func TestGitHubBlueprintRepository_GetLargeFileContents(t *testing.T) {
	repo, err := NewGitHubBlueprintRepository(getDefaultConfMap(t))
	require.Nil(t, err)
	err = repo.Initialize()
	require.Nil(t, err)

	tests := []struct {
		name     string
		filePath string
		want     int64
		wantErr  bool
	}{
		{
			"should error on invalid filepath",
			"aws/datalake/cloudformation/foo.zip",
			0,
			true,
		},
		{
			"should fetch blob of size more than 1MB",
			"aws/datalake/cloudformation/data-lake-artifacts.zip",
			15111927,
			false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, size, err := repo.GetLargeFileContents(tt.filePath)
			if (err != nil) != tt.wantErr {
				t.Errorf("GitHubBlueprintRepository.GetLargeFileContents() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if size != tt.want {
				t.Errorf("GitHubBlueprintRepository.GetLargeFileContents() got = %v, want %v", size, tt.want)
			}
		})
	}
}

func TestNewGitHubBlueprintRepository(t *testing.T) {
	t.Run("should error when repo-name is not set", func(t *testing.T) {
		repo, err := NewGitHubBlueprintRepository(map[string]string{
			"name": "test",
			"type": "github",
		})
		require.NotNil(t, err)
		require.Nil(t, repo)
	})
	t.Run("should error when owner is not set", func(t *testing.T) {
		repo, err := NewGitHubBlueprintRepository(map[string]string{
			"name":      "test",
			"type":      "github",
			"repo-name": "blueprints",
		})
		require.NotNil(t, err)
		require.Nil(t, repo)
	})
	t.Run("should set master as branch when not set", func(t *testing.T) {
		repo, err := NewGitHubBlueprintRepository(map[string]string{
			"name":      "test",
			"type":      "github",
			"repo-name": "blueprints",
			"owner":     "xebialabs",
			"token":     getGitHubTokenFromEnvVar(t),
		})
		require.Nil(t, err)
		require.NotNil(t, repo)
		assert.Equal(t, "test", repo.GetName())
		assert.Equal(t, "master", repo.Branch)
		err = repo.Initialize()
		require.Nil(t, err)
	})
	t.Run("should create a new GitHub repo context", func(t *testing.T) {
		repo, err := NewGitHubBlueprintRepository(map[string]string{
			"name":      "test",
			"type":      "github",
			"repo-name": "blueprints",
			"owner":     "xebialabs",
			"branch":    "development",
			"token":     getGitHubTokenFromEnvVar(t),
		})
		require.Nil(t, err)
		require.NotNil(t, repo)
		assert.Equal(t, "test", repo.GetName())
		assert.Equal(t, "github", repo.GetProvider())
		assert.Equal(t, "blueprints", repo.RepoName)
		assert.Equal(t, "development", repo.Branch)
		err = repo.Initialize()
		require.Nil(t, err)
	})
}

func TestGitHubBlueprintRepository_Initialize(t *testing.T) {
	type fields struct {
		Name     string
		RepoName string
		Owner    string
		Branch   string
		Token    string
	}
	repo1, _ := NewGitHubBlueprintRepository(map[string]string{
		"name":      "test",
		"type":      "github",
		"repo-name": "blueprints",
		"owner":     "xebialabs",
		"branch":    "dev",
	})
	repo2, _ := NewGitHubBlueprintRepository(getDefaultConfMap(t))
	tests := []struct {
		name   string
		fields *GitHubBlueprintRepository
	}{
		{
			"should initialize a gitHub repo without token",
			repo1,
		},
		{
			"should initialize a gitHub repo with token",
			repo2,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := &GitHubBlueprintRepository{
				GithubContext: tt.fields.GithubContext,
				Client:        tt.fields.Client,
				Name:          tt.fields.Name,
				RepoName:      tt.fields.RepoName,
				Owner:         tt.fields.Owner,
				Branch:        tt.fields.Branch,
				Token:         tt.fields.Token,
			}
			repo.Initialize()
			if repo.Client == nil {
				t.Errorf("GitHubBlueprintRepository.Initialize() repo.Client = %v", repo.Client)
			}
		})
	}
}

func TestGitHubBlueprintRepository_ListBlueprintsFromRepo(t *testing.T) {
	repo, err := NewGitHubBlueprintRepository(getDefaultConfMap(t))
	require.Nil(t, err)
	err = repo.Initialize()
	require.Nil(t, err)

	t.Run("should get list of blueprints from default xl repo", func(t *testing.T) {
		blueprints, dirs, err := repo.ListBlueprintsFromRepo()
		require.Nil(t, err)
		require.NotNil(t, dirs)
		assert.NotEmptyf(t, dirs, "blueprint directory list is empty")
		require.NotNil(t, blueprints)
		assert.NotEmptyf(t, blueprints, "blueprints map is empty")

		t.Run("should get valid content for a remote blueprint file", func(t *testing.T) {
			contents, err := repo.GetFileContents(blueprints[dirs[0]].DefinitionFile.Path)
			require.Nil(t, err)
			require.NotNil(t, contents)
			assert.NotEmptyf(t, string(*contents), "blueprint definition file contents is empty")
		})
	})

	t.Run("should error on non existing repository name", func(t *testing.T) {
		repo.RepoName = "nonexistingname"
		_, _, err := repo.ListBlueprintsFromRepo()
		require.NotNil(t, err)
		assert.Contains(t, err.Error(), "404 Not Found")
	})

	t.Run("should get empty blueprints map from a non-blueprint containing repo", func(t *testing.T) {
		repo.RepoName = "devops-as-code-vscode"
		blueprints, _, err := repo.ListBlueprintsFromRepo()
		require.Nil(t, err)
		require.NotNil(t, blueprints)
		assert.Empty(t, blueprints)
	})
}

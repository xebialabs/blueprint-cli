package gitlab

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func getDefaultConfMap(t *testing.T) map[string]string {
	return map[string]string{
		"name":      "test",
		"type":      "gitlab",
		"url":       "http://localhost",
		"repo-name": "blueprints",
		"owner":     "xebialabs",
		"branch":    "master",
		"token":     "",
		"isMock":    "true",
	}
}

func TestGitLabBlueprintRepository_GetFileContents(t *testing.T) {
	repo, err := NewGitLabBlueprintRepository(getDefaultConfMap(t))
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
				t.Errorf("GitLabBlueprintRepository.GetFileContents() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.want {
				if got == nil {
					t.Errorf("GitLabBlueprintRepository.GetFileContents() is nil, expected not nil")
				}
			}
		})
	}
}

func TestNewGitLabBlueprintRepository(t *testing.T) {
	t.Run("should error when repo-name is not set", func(t *testing.T) {
		repo, err := NewGitLabBlueprintRepository(map[string]string{
			"url":  "http://localhost",
			"name": "test",
			"type": "gitlab",
		})
		require.NotNil(t, err)
		require.Nil(t, repo)
	})
	t.Run("should error when owner is not set", func(t *testing.T) {
		repo, err := NewGitLabBlueprintRepository(map[string]string{
			"name":      "test",
			"type":      "gitlab",
			"url":       "http://localhost",
			"repo-name": "blueprints",
		})
		require.NotNil(t, err)
		require.Nil(t, repo)
	})
	t.Run("should error when url is not set", func(t *testing.T) {
		repo, err := NewGitLabBlueprintRepository(map[string]string{
			"name":      "test",
			"type":      "gitlab",
			"owner":     "xebialabs",
			"repo-name": "blueprints",
		})
		require.NotNil(t, err)
		require.Nil(t, repo)
	})
	t.Run("should set master as branch when not set", func(t *testing.T) {
		repo, err := NewGitLabBlueprintRepository(map[string]string{
			"name":      "test",
			"type":      "gitlab",
			"url":       "http://localhost",
			"repo-name": "blueprints",
			"owner":     "xebialabs",
			"isMock":    "true",
		})
		require.Nil(t, err)
		require.NotNil(t, repo)
		assert.Equal(t, "test", repo.GetName())
		assert.Equal(t, "master", repo.Branch)
		err = repo.Initialize()
		require.Nil(t, err)
	})
	t.Run("should create a new GitLab repo context", func(t *testing.T) {
		repo, err := NewGitLabBlueprintRepository(map[string]string{
			"name":      "test",
			"type":      "gitlab",
			"url":       "http://localhost",
			"repo-name": "blueprints",
			"owner":     "xebialabs",
			"branch":    "development",
			"isMock":    "true",
		})
		require.Nil(t, err)
		require.NotNil(t, repo)
		assert.Equal(t, "test", repo.GetName())
		assert.Equal(t, "gitlab", repo.GetProvider())
		assert.Equal(t, "blueprints", repo.RepoName)
		assert.Equal(t, "development", repo.Branch)
		err = repo.Initialize()
		require.Nil(t, err)
	})
}

func TestGitLabBlueprintRepository_Initialize(t *testing.T) {
	type fields struct {
		Name     string
		RepoName string
		Owner    string
		Branch   string
		Token    string
	}
	repo1, _ := NewGitLabBlueprintRepository(map[string]string{
		"name":      "test",
		"type":      "gitlab",
		"url":       "http://localhost",
		"repo-name": "blueprints",
		"owner":     "xebialabs",
		"branch":    "dev",
	})
	repo2, _ := NewGitLabBlueprintRepository(getDefaultConfMap(t))
	tests := []struct {
		name   string
		fields *GitLabBlueprintRepository
	}{
		{
			"should initialize a gitlab repo without token",
			repo1,
		},
		{
			"should initialize a gitlab repo with token",
			repo2,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := &GitLabBlueprintRepository{
				Client:   tt.fields.Client,
				Name:     tt.fields.Name,
				RepoName: tt.fields.RepoName,
				Owner:    tt.fields.Owner,
				Branch:   tt.fields.Branch,
				Token:    tt.fields.Token,
			}
			repo.Initialize()
			if repo.Client == nil {
				t.Errorf("GitLabBlueprintRepository.Initialize() repo.Client = %v", repo.Client)
			}
		})
	}
}

func TestGitLabBlueprintRepository_ListBlueprintsFromRepo(t *testing.T) {
	repo, err := NewGitLabBlueprintRepository(getDefaultConfMap(t))
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
	})

	t.Run("should get empty blueprints map from a non-blueprint containing repo", func(t *testing.T) {
		repo.RepoName = "devops-as-code-vscode"
		blueprints, _, err := repo.ListBlueprintsFromRepo()
		require.Nil(t, err)
		require.NotNil(t, blueprints)
		assert.Empty(t, blueprints)
	})
}

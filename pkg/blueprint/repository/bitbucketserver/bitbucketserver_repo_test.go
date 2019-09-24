package bitbucketserver

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func getDefaultConfMap(t *testing.T) map[string]string {
	return map[string]string{
		"name":        "test",
		"type":        "bitbucketserver",
		"url":         "http://localhost:7990",
		"repo-name":   "blueprints",
		"project-key": "XEB",
		"branch":      "master",
		"user":        "admin",
		"token":       "some-bitbucket-token",
		"isMock":      "true",
	}
}

func TestBitbucketServerBlueprintRepository_GetFileContents(t *testing.T) {
	repo, err := NewBitbucketServerBlueprintRepository(getDefaultConfMap(t))
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
				t.Errorf("BitbucketServerBlueprintRepository.GetFileContents() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.want {
				if got == nil {
					t.Errorf("BitbucketServerBlueprintRepository.GetFileContents() is nil, expected not nil")
				}
			}
		})
	}
}

func TestNewBitbucketBlueprintRepository(t *testing.T) {
	t.Run("should error when repo-name is not set", func(t *testing.T) {
		repo, err := NewBitbucketServerBlueprintRepository(map[string]string{
			"name": "test",
			"type": "bitbucketserver",
			"url":  "http://localhost:7990",
		})
		require.NotNil(t, err)
		require.Nil(t, repo)
	})
	t.Run("should error when project-key is not set", func(t *testing.T) {
		repo, err := NewBitbucketServerBlueprintRepository(map[string]string{
			"name":      "test",
			"type":      "bitbucketserver",
			"url":       "http://localhost:7990",
			"repo-name": "blueprints",
		})
		require.NotNil(t, err)
		require.Nil(t, repo)
	})
	t.Run("should set master as branch when not set", func(t *testing.T) {
		repo, err := NewBitbucketServerBlueprintRepository(map[string]string{
			"name":        "test",
			"type":        "bitbucketserver",
			"url":         "http://localhost:7990",
			"repo-name":   "blueprints",
			"project-key": "XEB",
			"isMock":      "true",
		})
		require.Nil(t, err)
		require.NotNil(t, repo)
		assert.Equal(t, "test", repo.GetName())
		assert.Equal(t, "master", repo.Branch)
		err = repo.Initialize()
		require.Nil(t, err)
	})
	t.Run("should error when user is set, but token is not set", func(t *testing.T) {
		repo, err := NewBitbucketServerBlueprintRepository(map[string]string{
			"name":        "test",
			"type":        "bitbucketserver",
			"url":         "http://localhost:7990",
			"repo-name":   "blueprints",
			"project-key": "XEB",
			"user":        "admin",
			"isMock":      "true",
		})
		require.NotNil(t, err)
		require.Nil(t, repo)
	})
	t.Run("should error when token is set, but user is not set", func(t *testing.T) {
		repo, err := NewBitbucketServerBlueprintRepository(map[string]string{
			"name":        "test",
			"type":        "bitbucketserver",
			"url":         "http://localhost:7990",
			"repo-name":   "blueprints",
			"project-key": "XEB",
			"token":       "some-bitbucket-token",
			"isMock":      "true",
		})
		require.NotNil(t, err)
		require.Nil(t, repo)
	})
	t.Run("should create a new Bitbucket repo context", func(t *testing.T) {
		repo, err := NewBitbucketServerBlueprintRepository(map[string]string{
			"name":        "test",
			"type":        "bitbucketserver",
			"url":         "http://localhost:7990",
			"repo-name":   "blueprints",
			"project-key": "XEB",
			"branch":      "development",
			"isMock":      "true",
		})
		require.Nil(t, err)
		require.NotNil(t, repo)
		assert.Equal(t, "test", repo.GetName())
		assert.Equal(t, "bitbucketserver", repo.GetProvider())
		assert.Equal(t, "blueprints", repo.RepoName)
		assert.Equal(t, "development", repo.Branch)
		err = repo.Initialize()
		require.Nil(t, err)
	})
}

func TestBitbucketServerBlueprintRepository_Initialize(t *testing.T) {
	type fields struct {
		Name     string
		RepoName string
		Owner    string
		Branch   string
		Token    string
	}
	repo1, _ := NewBitbucketServerBlueprintRepository(map[string]string{
		"name":        "test",
		"type":        "bitbucket",
		"url":         "http://localhost:7990",
		"repo-name":   "blueprints",
		"project-key": "XEB",
		"branch":      "dev",
	})
	repo2, _ := NewBitbucketServerBlueprintRepository(getDefaultConfMap(t))
	tests := []struct {
		name   string
		fields *BitbucketServerBlueprintRepository
	}{
		{
			"should initialize a Bitbucket repo without token",
			repo1,
		},
		{
			"should initialize a Bitbucket repo with token",
			repo2,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := &BitbucketServerBlueprintRepository{
				Client:     tt.fields.Client,
				Name:       tt.fields.Name,
				RepoName:   tt.fields.RepoName,
				ProjectKey: tt.fields.ProjectKey,
				Branch:     tt.fields.Branch,
				Token:      tt.fields.Token,
			}
			repo.Initialize()
			if repo.Client == nil {
				t.Errorf("BitbucketBlueprintRepository.Initialize() repo.Client = %v", repo.Client)
			}
		})
	}
}

func TestBitbucketBlueprintRepository_ListBlueprintsFromRepo(t *testing.T) {
	repo, err := NewBitbucketServerBlueprintRepository(getDefaultConfMap(t))
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

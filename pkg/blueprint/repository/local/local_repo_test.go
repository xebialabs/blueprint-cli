package local

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	repoType = "local"
)

func GetLocalBlueprintTestRepoPath() string {
	currentDir, _ := os.Getwd()
	return filepath.Join(currentDir, "../../../../templates/test")
}

func TestNewGitHubBlueprintRepository(t *testing.T) {
	blueprintDir := GetLocalBlueprintTestRepoPath()

	t.Run("should error when path is not set", func(t *testing.T) {
		repo, err := NewLocalBlueprintRepository(map[string]string{
			"name": "test",
			"type": repoType,
		})
		require.NotNil(t, err)
		require.Nil(t, repo)
	})
	t.Run("should set ignore lists empty when not set", func(t *testing.T) {
		repo, err := NewLocalBlueprintRepository(map[string]string{
			"name": "test",
			"type": repoType,
			"path": blueprintDir,
		})
		require.Nil(t, err)
		require.NotNil(t, repo)
		assert.Equal(t, "test", repo.GetName())
		assert.Equal(t, blueprintDir, repo.Path)
		assert.Empty(t, repo.IgnoredFiles)
		assert.Empty(t, repo.IgnoredDirs)
		err = repo.Initialize()
		require.Nil(t, err)
	})
	t.Run("should get filled ignored lists when set", func(t *testing.T) {
		repo, err := NewLocalBlueprintRepository(map[string]string{
			"name":          "test",
			"type":          repoType,
			"path":          blueprintDir,
			"ignored-dirs":  ".git, .github,.vscode",
			"ignored-files": ".DS_Store, .test",
		})
		require.Nil(t, err)
		require.NotNil(t, repo)
		assert.Equal(t, "test", repo.GetName())
		assert.Equal(t, blueprintDir, repo.Path)
		assert.ElementsMatch(t, repo.IgnoredDirs, []string{".git", ".github", ".vscode"})
		assert.ElementsMatch(t, repo.IgnoredFiles, []string{".DS_Store", ".test"})
		err = repo.Initialize()
		require.Nil(t, err)
	})
}

func TestListBlueprintsFromRepo(t *testing.T) {
	blueprintDir := GetLocalBlueprintTestRepoPath()

	t.Run("should list blueprints from local test repo", func(t *testing.T) {
		repo, err := NewLocalBlueprintRepository(map[string]string{
			"name": "test",
			"type": repoType,
			"path": blueprintDir,
		})
		require.Nil(t, err)
		require.NotNil(t, repo)
		blueprints, blueprintDirs, err := repo.ListBlueprintsFromRepo()
		assert.NotEmpty(t, repo.LocalFiles)
		require.Nil(t, err)
		require.NotNil(t, blueprints)
		assert.NotEmpty(t, blueprints)
		assert.Len(t, blueprints, 6)
		require.NotNil(t, blueprintDirs)
		assert.NotEmpty(t, blueprintDirs)
		assert.Len(t, blueprintDirs, 6)
	})

    t.Run("should list empty blueprints list from local dir", func(t *testing.T) {
        repo, err := NewLocalBlueprintRepository(map[string]string{
            "name": "test",
            "type": repoType,
            "path": filepath.Join(blueprintDir, "invalid"),
        })
        require.Nil(t, err)
        require.NotNil(t, repo)
        blueprints, blueprintDirs, err := repo.ListBlueprintsFromRepo()
        assert.Empty(t, repo.LocalFiles)
        require.NotNil(t, blueprints)
        assert.Empty(t, blueprints)
        require.Nil(t, blueprintDirs)
    })
}

func TestGetFileContents(t *testing.T) {
	blueprintDir := GetLocalBlueprintTestRepoPath()

	t.Run("should get valid local repo file contents", func(t *testing.T) {
		repo, err := NewLocalBlueprintRepository(map[string]string{
			"name": "test",
			"type": repoType,
			"path": blueprintDir,
		})
		require.Nil(t, err)
		require.NotNil(t, repo)
		contents, err := repo.GetFileContents("answer-input/xlr-pipeline.yml")
		require.Nil(t, err)
		require.NotNil(t, contents)
	})

	t.Run("should error on invalid local repo path for get contents", func(t *testing.T) {
		repo, err := NewLocalBlueprintRepository(map[string]string{
			"name": "test",
			"type": repoType,
			"path": blueprintDir,
		})
		require.Nil(t, err)
		require.NotNil(t, repo)
		_, err = repo.GetFileContents("invalid-path/blueprint.yaml")
		require.NotNil(t, err)
	})
}

func TestFindRelatedBlueprintDir(t *testing.T) {
	tests := []struct {
		name          string
		blueprintDirs []string
		searchPath    string
		expected      string
	}{
		{
			"should find related blueprint dir given full path",
			[]string{"/path/to/blueprint/test", "/path/to/blueprint/another", "/path/to/blueprint/yet-another"},
			"/path/to/blueprint/test/file.yaml",
			"/path/to/blueprint/test",
		},
		{
			"should return empty given full path of a non-blueprint file",
			[]string{"/path/to/blueprint/test", "/path/to/blueprint/another", "/path/to/blueprint/yet-another"},
			"/path/to/blueprint/non-blueprint/file.yaml",
			"",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := findRelatedBlueprintDir(tt.blueprintDirs, tt.searchPath)
			assert.Equal(t, tt.expected, got)
		})
	}
}

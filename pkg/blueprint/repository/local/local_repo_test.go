package local

import (
    "os"
    "os/user"
    "path"
    "path/filepath"
    "runtime"
    "strings"
    "testing"

    "github.com/stretchr/testify/assert"
    "github.com/stretchr/testify/require"
)

const (
	repoType = "local"
)

func GetLocalBlueprintTestRepoPath() string {
    pwd, _ := os.Getwd()
    return strings.Replace(pwd, path.Join("pkg", "blueprint", "repository", "local"), path.Join("templates", "test"), -1)
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
    t.Run("should error when given path is a file", func(t *testing.T) {
        repo, err := NewLocalBlueprintRepository(map[string]string{
            "name": "test",
            "type": repoType,
            "path": filepath.Join(blueprintDir, "answer-input.yaml"),
        })
        require.NotNil(t, err)
        require.Nil(t, repo)
    })
	if runtime.GOOS != "windows" {
        t.Run("should expand home dir", func(t *testing.T) {
            currentUser, _ := user.Current()
            repo, err := NewLocalBlueprintRepository(map[string]string{
                "name": "test",
                "type": repoType,
                "path": "~/",
            })
            require.Nil(t, err)
            require.NotNil(t, repo)
            assert.Equal(t, "test", repo.GetName())
            assert.Equal(t, currentUser.HomeDir, repo.Path)
            err = repo.Initialize()
            require.Nil(t, err)
        })
    }
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
            "ignored-dirs":  ".git, __test__",
            "ignored-files": ".DS_Store",
		})
		require.Nil(t, err)
		require.NotNil(t, repo)
		blueprints, blueprintDirs, err := repo.ListBlueprintsFromRepo()
		assert.NotEmpty(t, repo.LocalFiles)
		require.Nil(t, err)
		require.NotNil(t, blueprints)
		assert.NotEmpty(t, blueprints)
		assert.Len(t, blueprints, 7)
		require.NotNil(t, blueprintDirs)
		assert.NotEmpty(t, blueprintDirs)
		assert.Len(t, blueprintDirs, 7)

		answerInputBlueprint := blueprints["answer-input"]
		assert.Equal(t, "answer-input", answerInputBlueprint.Path)
		assert.Len(t, answerInputBlueprint.Files, 3)

        validNoPromptBlueprint := blueprints["valid-no-prompt"]
        assert.Equal(t, "valid-no-prompt", validNoPromptBlueprint.Path)
        assert.Len(t, validNoPromptBlueprint.Files, 5)
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

func TestExpandHomeDirIfNeeded(t *testing.T) {
    // not to be tested on windows
    if runtime.GOOS != "windows" {
        currentUser, _ := user.Current()
        tests := []struct {
            name     string
            repoPath string
            expected string
        }{
            {
                "should expand home path when given ~",
                "~",
                currentUser.HomeDir,
            },
            {
                "should expand home path when given ~/",
                "~",
                currentUser.HomeDir,
            },
            {
                "should expand home path when given relative path to ~",
                "~/some/dir",
                filepath.Join(currentUser.HomeDir, "some/dir"),
            },
            {
                "should not expand home path when given a path including ~ in between",
                "/tmp/~/some/dir",
                "/tmp/~/some/dir",
            },
            {
                "should return original path when a full path is given",
                "/tmp/path/to/some/local/dir/",
                "/tmp/path/to/some/local/dir/",
            },
            {
                "should return original path when a root path is given",
                "/",
                "/",
            },
        }
        for _, tt := range tests {
            t.Run(tt.name, func(t *testing.T) {
                got := expandHomeDirIfNeeded(tt.repoPath, currentUser)
                assert.Equal(t, tt.expected, got)
            })
        }
    }
}

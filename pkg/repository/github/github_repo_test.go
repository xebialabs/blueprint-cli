package github

import (
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestGithubRepositoryClient(t *testing.T) {
	repo := NewGitHubBlueprintRepository("blueprints", "xebialabs", "master", "")

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
		repo.Name = "nonexistingname"
		_, _, err := repo.ListBlueprintsFromRepo()
		require.NotNil(t, err)
		assert.Contains(t, err.Error(), "404 Not Found")
	})

	t.Run("should get empty blueprints map from a non-blueprint containing repo", func(t *testing.T) {
		repo.Name = "devops-as-code-vscode"
		blueprints, _, err := repo.ListBlueprintsFromRepo()
		require.Nil(t, err)
		require.NotNil(t, blueprints)
		assert.Empty(t, blueprints)
	})
}
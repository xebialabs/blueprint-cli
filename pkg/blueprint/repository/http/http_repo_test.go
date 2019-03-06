package http

import (
    "github.com/jarcoal/httpmock"
    "github.com/stretchr/testify/assert"
    "github.com/stretchr/testify/require"
    "testing"
)

const mockEndpoint = "http://mock.repo.server.com/"

func getDefaultConfMap() map[interface{}]interface{} {
    return map[interface{}]interface{} {
        "name": "test",
        "url": "http://mock.repo.server.com/",
    }
}

func TestHttpRepositoryClientFail(t *testing.T) {
	repo, err := NewHttpBlueprintRepository(getDefaultConfMap())
	require.Nil(t, err)
	err = repo.Initialize()
	require.Nil(t, err)

	httpmock.Activate()
	defer httpmock.DeactivateAndReset()
	httpmock.RegisterResponder(
		"GET",
		mockEndpoint + RepoIndexFileName,
		httpmock.NewStringResponder(404, ""),
	)
	httpmock.RegisterResponder(
		"GET",
		mockEndpoint + "400.yaml",
		httpmock.NewStringResponder(400, ""),
	)
	httpmock.RegisterResponder(
		"GET",
		mockEndpoint + "500.yaml",
		httpmock.NewStringResponder(500, ""),
	)

	t.Run("should 404 on non-existing repo URL", func(t *testing.T) {
		_, _, err := repo.ListBlueprintsFromRepo()
		require.NotNil(t, err)
		assert.Contains(t, err.Error(), "404 unable to read remote http file [index.json]")
	})

	t.Run("should error on response code higher than 400", func(t *testing.T) {
		_, err1 := repo.GetFileContents("400.yaml")
		require.NotNil(t, err1)
		assert.Contains(t, err1.Error(), "400 unable to read remote http file [400.yaml]")

		_, err2 := repo.GetFileContents("500.yaml")
		require.NotNil(t, err2)
		assert.Contains(t, err2.Error(), "500 unable to read remote http file [500.yaml]")
	})
}

func TestHttpRepositoryClientSuccess(t *testing.T) {
    repo, err := NewHttpBlueprintRepository(getDefaultConfMap())
    require.Nil(t, err)
    err = repo.Initialize()
    require.Nil(t, err)

	httpmock.Activate()
	defer httpmock.DeactivateAndReset()
	httpmock.RegisterResponder(
		"GET",
		mockEndpoint + RepoIndexFileName,
		httpmock.NewStringResponder(200, `[
"aws/monolith",
"aws/microservice-ecommerce",
"aws/datalake",
"docker/simple-demo-app"
]`),
	)
	httpmock.RegisterResponder(
		"GET",
		mockEndpoint + "aws/monolith/test.txt",
		httpmock.NewStringResponder(200, `sample test text
with a new line`),
	)

	t.Run("should list available blueprints from http repo", func(t *testing.T) {
		blueprints, _, err := repo.ListBlueprintsFromRepo()
		require.Nil(t, err)
		require.NotNil(t, blueprints)
		assert.Len(t, blueprints, 4)
	})

	t.Run("should fetch remote file contents from http repo", func(t *testing.T) {
		content, err := repo.GetFileContents("aws/monolith/test.txt")
		require.Nil(t, err)
		require.NotNil(t, content)
		assert.Equal(t, "sample test text\nwith a new line", string(*content))
	})
}

func TestCheckBlueprintDefinitionFile(t *testing.T) {
    repo, err := NewHttpBlueprintRepository(getDefaultConfMap())
    require.Nil(t, err)
    err = repo.Initialize()
    require.Nil(t, err)

	httpmock.Activate()
	defer httpmock.DeactivateAndReset()
	httpmock.RegisterResponder(
		"GET",
		mockEndpoint + RepoIndexFileName,
		httpmock.NewStringResponder(200, `[
"aws/monolith",
"aws/microservice-ecommerce"
]`),
	)
	httpmock.RegisterResponder(
		"GET",
		mockEndpoint + "aws/monolith/blueprint.yml",
		httpmock.NewStringResponder(200, `sample test text
with a new line`),
	)
	httpmock.RegisterResponder(
		"GET",
		mockEndpoint + "aws/microservice-ecommerce/blueprint.yaml",
		httpmock.NewStringResponder(200, `sample test text
with a new line`),
	)

	t.Run("should list available blueprints with various YAML extensions", func(t *testing.T) {
		blueprints, _, err := repo.ListBlueprintsFromRepo()
		require.Nil(t, err)
		require.NotNil(t, blueprints)
		assert.Len(t, blueprints, 2)
		assert.Equal(t, "aws/monolith/blueprint.yml", blueprints["aws/monolith"].DefinitionFile.Path)
		assert.Equal(t, "aws/microservice-ecommerce/blueprint.yaml", blueprints["aws/microservice-ecommerce"].DefinitionFile.Path)
	})
}

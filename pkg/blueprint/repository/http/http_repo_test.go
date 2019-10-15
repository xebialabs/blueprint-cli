package http

import (
	"reflect"
	"testing"

	"github.com/jarcoal/httpmock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/xebialabs/xl-cli/pkg/models"
)

const mockEndpoint = "http://mock.repo.server.com/"
const DummyCLIVersion = "9.0.0-SNAPSHOT"

func getDefaultConfMap() map[string]string {
	return map[string]string{
		"name": "test",
		"url":  "http://mock.repo.server.com/",
	}
}

func TestNewHttpBlueprintRepository(t *testing.T) {
	tests := []struct {
		name    string
		confMap map[string]string
		want    string
		wantErr bool
	}{
		{
			"should error when url is not set",
			map[string]string{
				"name": "test",
			},
			"",
			true,
		},
		{
			"should error when invalid url is not set",
			map[string]string{
				"name": "test",
				"url":  "hoola",
			},
			"",
			true,
		},
		{
			"should create a http context",
			map[string]string{
				"name":     "test",
				"url":      "http://test.com",
				"username": "test",
			},
			"test",
			false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := NewHttpBlueprintRepository(tt.confMap, DummyCLIVersion)
			if (err != nil) != tt.wantErr {
				t.Errorf("NewHttpBlueprintRepository() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != nil && !reflect.DeepEqual(got.GetName(), tt.want) {
				t.Errorf("NewHttpBlueprintRepository() = %v, want %v", got, tt.want)
			}
		})
	}
}

func mockHttpFail() {
	httpmock.Activate()
	httpmock.RegisterResponder(
		"GET",
		mockEndpoint+RepoIndexFileName,
		httpmock.NewStringResponder(404, ""),
	)
	httpmock.RegisterResponder(
		"GET",
		mockEndpoint+"400.yaml",
		httpmock.NewStringResponder(400, ""),
	)
	httpmock.RegisterResponder(
		"GET",
		mockEndpoint+"500.yaml",
		httpmock.NewStringResponder(500, ""),
	)
}

func mockHttpSuccess() {
	httpmock.Activate()
	httpmock.RegisterResponder(
		"GET",
		mockEndpoint+RepoIndexFileName,
		httpmock.NewStringResponder(200, `[
            "aws/monolith",
            "aws/microservice-ecommerce",
            "aws/datalake",
            "docker/simple-demo-app"
        ]`),
	)
	httpmock.RegisterResponder(
		"GET",
		mockEndpoint+"aws/monolith/blueprint.yaml",
		httpmock.NewStringResponder(200, `sample test text`),
	)
	httpmock.RegisterResponder(
		"GET",
		mockEndpoint+"aws/microservice-ecommerce/blueprint.yaml",
		httpmock.NewStringResponder(200, `sample test text`),
	)
	httpmock.RegisterResponder(
		"GET",
		mockEndpoint+"aws/datalake/blueprint.yml",
		httpmock.NewStringResponder(200, `sample test text`),
	)
	httpmock.RegisterResponder(
		"GET",
		mockEndpoint+"docker/simple-demo-app/blueprint.yaml",
		httpmock.NewStringResponder(200, `sample test text`),
	)
	httpmock.RegisterResponder(
		"GET",
		mockEndpoint+"aws/monolith/test.txt",
		httpmock.NewStringResponder(200, `sample test text
with a new line`),
	)
}

func TestHttpBlueprintRepository_ListBlueprintsFromRepo(t *testing.T) {
	repo, err := NewHttpBlueprintRepository(getDefaultConfMap(), DummyCLIVersion)
	require.Nil(t, err)
	err = repo.Initialize()
	require.Nil(t, err)

	t.Run("should 404 on non-existing repo URL", func(t *testing.T) {
		defer httpmock.DeactivateAndReset()
		mockHttpFail()

		_, _, err := repo.ListBlueprintsFromRepo()
		require.NotNil(t, err)
		assert.Contains(t, err.Error(), "404 unable to read remote http file [index.json]")
	})

	t.Run("should list available blueprints from http repo", func(t *testing.T) {
		defer httpmock.DeactivateAndReset()
		mockHttpSuccess()

		blueprints, blueprintDirs, err := repo.ListBlueprintsFromRepo()
		require.Nil(t, err)
		require.NotNil(t, blueprints)
		assert.Len(t, blueprints, 4)
		assert.Len(t, blueprintDirs, 4)
	})
}

func TestHttpBlueprintRepository_GetFileContents(t *testing.T) {
	repo, err := NewHttpBlueprintRepository(getDefaultConfMap(), DummyCLIVersion)
	require.Nil(t, err)
	err = repo.Initialize()
	require.Nil(t, err)

	t.Run("should error on response code higher than 400", func(t *testing.T) {
		defer httpmock.DeactivateAndReset()
		mockHttpFail()

		_, err1 := repo.GetFileContents("400.yaml")
		require.NotNil(t, err1)
		assert.Contains(t, err1.Error(), "400 unable to read remote http file [400.yaml]")

		_, err2 := repo.GetFileContents("500.yaml")
		require.NotNil(t, err2)
		assert.Contains(t, err2.Error(), "500 unable to read remote http file [500.yaml]")
	})

	t.Run("should fetch remote file contents from http repo", func(t *testing.T) {
		defer httpmock.DeactivateAndReset()
		mockHttpSuccess()

		content, err := repo.GetFileContents("aws/monolith/test.txt")
		require.Nil(t, err)
		require.NotNil(t, content)
		assert.Equal(t, "sample test text\nwith a new line", string(*content))
	})
}

func TestHttpBlueprintRepository_checkBlueprintDefinitionFile(t *testing.T) {
	repo, err := NewHttpBlueprintRepository(getDefaultConfMap(), DummyCLIVersion)
	require.Nil(t, err)
	err = repo.Initialize()
	require.Nil(t, err)

	httpmock.Activate()
	defer httpmock.DeactivateAndReset()
	httpmock.RegisterResponder(
		"GET",
		mockEndpoint+RepoIndexFileName,
		httpmock.NewStringResponder(200, `[
"aws/monolith",
"aws/microservice-ecommerce"
"gcp/microservice-ecommerce"
]`),
	)
	httpmock.RegisterResponder(
		"GET",
		mockEndpoint+"aws/monolith/blueprint.yml",
		httpmock.NewStringResponder(200, `sample test text
with a new line`),
	)
	httpmock.RegisterResponder(
		"GET",
		mockEndpoint+"aws/microservice-ecommerce/blueprint.yaml",
		httpmock.NewStringResponder(200, `sample test text
with a new line`),
	)
	httpmock.RegisterResponder(
		"GET",
		mockEndpoint+"gcp/microservice-ecommerce/blueprint.json",
		httpmock.NewStringResponder(200, `sample test text
with a new line`),
	)

	t.Run("should fail for invalid extensions", func(t *testing.T) {
		blueprint, err := repo.checkBlueprintDefinitionFile("gcp/microservice-ecommerce")
		require.NotNil(t, err)
		require.Equal(t, "", blueprint)
	})
	t.Run("should pass for valid extensions", func(t *testing.T) {
		blueprint, err := repo.checkBlueprintDefinitionFile("aws/microservice-ecommerce")
		require.Nil(t, err)
		assert.Equal(t, "blueprint.yaml", blueprint)
		blueprint, err = repo.checkBlueprintDefinitionFile("aws/monolith")
		require.Nil(t, err)
		assert.Equal(t, "blueprint.yml", blueprint)
	})
}

func Test_getCLIVersionURL(t *testing.T) {
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()
	httpmock.RegisterResponder(
		"GET",
		"https://dist.xebialabs.com/public/blueprints/9.0.0/",
		httpmock.NewStringResponder(200, `[
            "aws/monolith"
        ]`),
	)
	httpmock.RegisterResponder(
		"GET",
		"https://dist.xebialabs.com/public/blueprints/9.1/",
		httpmock.NewStringResponder(200, `[
            "aws/monolith"
        ]`),
	)

	tests := []struct {
		name       string
		url        string
		CLIVersion string
		want       string
	}{
		{
			"should return the given url when it doesn't have a placeholder",
			"https://dist.xebialabs.com/public/blueprints/",
			"",
			"https://dist.xebialabs.com/public/blueprints/",
		},
		{
			"should return the correct url when it has a placeholder",
			models.DefaultBlueprintRepositoryUrl,
			"9.0.0-SNAPSHOT",
			"https://dist.xebialabs.com/public/blueprints/9.0.0/",
		},
		{
			"should return the given url when placeholder cannot be replaced",
			models.DefaultBlueprintRepositoryUrl,
			"FOO9.0.0-SNAPSHOT",
			models.DefaultBlueprintRepositoryUrl,
		},
		{
			"should return the given url when placeholder is invalid",
			"https://dist.xebialabs.com/public/blueprints/${foo}",
			"FOO9.0.0-SNAPSHOT",
			"https://dist.xebialabs.com/public/blueprints/${foo}",
		},
		{
			"should keep trying until it finds an existing version from patch",
			models.DefaultBlueprintRepositoryUrl,
			"9.0.1",
			"https://dist.xebialabs.com/public/blueprints/9.0.0/",
		},
		{
			"should keep trying until it finds an existing version from minor",
			models.DefaultBlueprintRepositoryUrl,
			"9.1.0",
			"https://dist.xebialabs.com/public/blueprints/9.1/",
		},
		{
			"should keep trying until it finds an existing version without patch",
			models.DefaultBlueprintRepositoryUrl,
			"9.2.0",
			"https://dist.xebialabs.com/public/blueprints/9.1/",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := getCLIVersionURL(tt.url, tt.CLIVersion); got != tt.want {
				t.Errorf("getCLIVersionURL() = %v, want %v", got, tt.want)
			}
		})
	}
}

package models

import (
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestGetRepoProviderWithName(t *testing.T) {
	t.Run("should error on not-supported provider name", func(t *testing.T) {
		_, err := GetRepoProvider("not-supported")
		require.NotNil(t, err)
		assert.Equal(t, "not-supported is not supported as repository provider", err.Error())
	})

	t.Run("should get mock repo provider with name", func(t *testing.T) {
		provider, err := GetRepoProvider("mock")
		require.Nil(t, err)
		require.NotNil(t, provider)
		assert.Equal(t, ProviderMock, provider)
	})

	t.Run("should get github repo provider with name", func(t *testing.T) {
		provider, err := GetRepoProvider("github")
		require.Nil(t, err)
		require.NotNil(t, provider)
		assert.Equal(t, ProviderGitHub, provider)
	})

	t.Run("should get github repo provider with name mixed with uppercase and lowercase", func(t *testing.T) {
		provider, err := GetRepoProvider("GitHub")
		require.Nil(t, err)
		require.NotNil(t, provider)
		assert.Equal(t, ProviderGitHub, provider)
	})
}

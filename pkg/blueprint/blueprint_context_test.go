package blueprint

import (
	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
	"github.com/xebialabs/xl-cli/pkg/models"
	"testing"
)

func TestBlueprintContextBuilder(t *testing.T) {
	t.Run("build simple context for Blueprint repository", func(t *testing.T) {
		v := viper.New()
		v.Set(ViperKeyBlueprintRepositoryProvider, models.ProviderGitHub)
		v.Set(ViperKeyBlueprintRepositoryName, "blueprints")
		v.Set(ViperKeyBlueprintRepositoryOwner, "xebialabs")

		c, err := ConstructBlueprintContext(v)

		assert.Nil(t, err)
		assert.NotNil(t, c)
		assert.Equal(t, models.ProviderGitHub, c.Provider)
		assert.Equal(t, "blueprints", c.Name)
		assert.Equal(t, "xebialabs", c.Owner)
	})
}

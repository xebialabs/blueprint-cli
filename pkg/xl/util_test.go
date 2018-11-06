package xl

import (
	"fmt"
	"github.com/stretchr/testify/assert"
	"os"
	"path"
	"testing"
)

func TestUtil(t *testing.T) {
	t.Run("should check for relative paths", func(t *testing.T) {
		assert.True(t, isRelativePath(path.Join("..", "provision.yaml")))
		assert.False(t, isRelativePath(path.Join("folder", "provision.yaml")))
	})

	t.Run("ValidateFilePath should prevent specifying absolute paths", func(t *testing.T) {
		absolute := path.Join(string(os.PathSeparator), "folder", "provision.yaml")
		assert.EqualError(t, ValidateFilePath(absolute, "test"), fmt.Sprintf("absolute path is not allowed in test: %s\n", absolute))
	})

	t.Run("ValidateFilePath should prevent specifying relative paths (starts from ..)", func(t *testing.T) {
		relativePath := path.Join("..", "folder", "provision.yaml")
		assert.EqualError(t, ValidateFilePath(relativePath, "test"), fmt.Sprintf("relative path with .. is not allowed in test: %s\n", relativePath))
	})

	t.Run("ValidateFilePath happy flow", func(t *testing.T) {
		assert.Nil(t, ValidateFilePath("file.yaml", "test"))
		assert.Nil(t, ValidateFilePath(path.Join("folder", "provision.yaml"), "test"))
	})
}

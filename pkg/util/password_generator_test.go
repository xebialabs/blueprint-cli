package util

import (
	"regexp"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestIsPasswordGeneration(t *testing.T) {
	t.Run("should generate the password is generated for the given length", func(t *testing.T) {
		assert.Equal(t, 8, len(GeneratePassword(8)))
		assert.Equal(t, 16, len(GeneratePassword(16)))
		assert.Equal(t, 32, len(GeneratePassword(32)))
	})

	t.Run("should generate empty password if the length is below 1", func(t *testing.T) {
		assert.Equal(t, "", GeneratePassword(0))
		assert.Equal(t, "", GeneratePassword(-1))
		assert.Equal(t, "", GeneratePassword(-100))
	})

	t.Run("should contain one numeric, one uppercase and one lowercase character at the end", func(t *testing.T) {
		pwd := GeneratePassword(8)
		lastThreeChars := pwd[len(pwd)-3:]

		assert.True(t, true, regexp.MustCompile(`[0-9][A-Z][a-z]`).MatchString(lastThreeChars))
		assert.True(t, true, regexp.MustCompile(`\w*[0-9][A-Z][a-z]$`).MatchString(pwd))
	})

}

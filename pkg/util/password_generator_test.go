package util

import (
	"github.com/stretchr/testify/assert"
	"testing"
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

	t.Run("should check for numerics if length is above 1", func(t *testing.T) {
		assert.Equal(t, hasNumeric(GeneratePassword(2)), true)
		assert.Equal(t, hasNumeric(GeneratePassword(8)), true)
	})

	t.Run("should recognize numerics in a string", func(t *testing.T) {
		assert.Equal(t, hasNumeric("Aa"), false)
		assert.Equal(t, hasNumeric("1A"), true)
		assert.Equal(t, hasNumeric("A1"), true)
	})

}

package xl

import (
	"testing"
	"github.com/stretchr/testify/assert"
)

func TestObfuscryption(t *testing.T) {
	t.Run("obfuscrypt", func(t *testing.T) {
		secretValue := "@adm!npassw0rd!"
		obfuscryptedValue, err := Obfuscrypt(secretValue)
		assert.Nil(t, err)

		deobfuscryptedValue, err := Deobfuscrypt(obfuscryptedValue)
		assert.Nil(t, err)

		assert.Equal(t, deobfuscryptedValue, secretValue)
	} )

	t.Run("dontdeobfuscrypt", func(t *testing.T) {
		_, err := Deobfuscrypt("plaintextValue")
		assert.NotNil(t, err)
	} )

}
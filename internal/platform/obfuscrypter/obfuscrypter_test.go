package obfuscrypter_test;

import (
	"testing"
	"github.com/stretchr/testify/assert"
	"github.com/xebialabs/xl-cli/internal/platform/obfuscrypter"
)

func TestObfuscryption(t *testing.T) {
	t.Run("obfuscrypt", func(t *testing.T) {
		secretValue := "@adm!npassw0rd!"
		obfuscryptedValue, err := obfuscrypter.Obfuscrypt(secretValue)
		assert.Nil(t, err)

		deobfuscryptedValue, err := obfuscrypter.Deobfuscrypt(obfuscryptedValue)
		assert.Nil(t, err)

		assert.Equal(t, deobfuscryptedValue, secretValue)
	} )

	t.Run("dontdeobfuscrypt", func(t *testing.T) {
		_, err := obfuscrypter.Deobfuscrypt("plaintextValue")
		assert.NotNil(t, err)
	} )

}
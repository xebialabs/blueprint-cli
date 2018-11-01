package xl

import (
	"testing"
	"github.com/stretchr/testify/assert"
)

func TestSubstitution(t *testing.T) {

	values := map[string]string{"a": "X"}

	// good
	t.Run("test substitution good 1", func(t *testing.T) {
		substituteAndExpect(t, values,"a %a% a",       "a X a")
	})

	t.Run("test substitution good 2", func(t *testing.T) {
		substituteAndExpect(t, values,"a %a% %% a",    "a X % a")
	})

	t.Run("test substitution good 3", func(t *testing.T) {
		substituteAndExpect(t, values,"a %%%a%%% a",   "a %X% a")
	})

	t.Run("test substitution good 4", func(t *testing.T) {
		substituteAndExpect(t, values,"a %%%%a%%%% a", "a %%a%% a")
	})

	t.Run("test substitution good 5", func(t *testing.T) {
		substituteAndExpect(t, values,"a %a%a%a% a",   "a XaX a")
	})

	// wrong
	t.Run("test substitution unknown variable name", func(t *testing.T) {
		substituteAndError(t, values,"%b%","unknown value: `b`")
	})

	t.Run("test substitution wrong 1", func(t *testing.T) {
		substituteAndError(t, values,"%","invalid format string: `%`")
	})

	t.Run("test substitution wrong 2", func(t *testing.T) {
		substituteAndError(t, values,"% % %","unknown value: ` `")
	})

	t.Run("test substitution wrong 3", func(t *testing.T) {
		substituteAndError(t, values,"a %","invalid format string: `a %`")
	})

	t.Run("test substitution wrong 4", func(t *testing.T) {
		substituteAndError(t, values,"a %%a% a","invalid format string: `a %%a% a`")
	})

	t.Run("test substitution wrong 5", func(t *testing.T) {
		substituteAndError(t, values,"a %a%% a","invalid format string: `a %a%% a`")
	})

	t.Run("test substitution wrong 6", func(t *testing.T) {
		substituteAndError(t, values,"a %a%a%a a","invalid format string: `a %a%a%a a`")
	})

	t.Run("test substitution wrong 7", func(t *testing.T) {
		substituteAndError(t, values,"a %a%%%a% a","invalid format string: `a %a%%%a% a`")
	})

}

func substituteAndExpect(t *testing.T, values map[string]string, in string, out string) {
	result, err := Substitute(in, values)
	assert.Nil(t, err)
	assert.Equal(t, out, result)
}

func substituteAndError(t *testing.T, values map[string]string, in string, msg string) {
	_, err := Substitute(in, values)
	assert.NotNil(t, err)
	assert.Equal(t, err.Error(), msg)
}

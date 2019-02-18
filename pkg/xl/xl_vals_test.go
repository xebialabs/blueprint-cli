package xl

import (
	"github.com/stretchr/testify/assert"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"
)

func TestXlVals(t *testing.T) {
	t.Run("should list xlvals files from relative directory in alphabetical order", func(t *testing.T) {
		dir, err := ioutil.TempDir("", "")
		defer os.Remove(dir)
		check(err)

		_, err1 := os.Create(filepath.Join(dir, "b.xlvals"))
		check(err1)

		_, err2 := os.Create(filepath.Join(dir, "c.xlvals"))
		check(err2)

		_, err3 := os.Create(filepath.Join(dir, "a.xlvals"))
		check(err3)

		subdir, err := ioutil.TempDir(dir, "")
		check(err)

		_, err4 := os.Create(filepath.Join(subdir, "d.xlvals"))
		check(err4)

		files, err := ListRelativeXlValsFiles(dir)
		check(err)

		assert.Len(t, files, 3)
		assert.Equal(t, filepath.Base(files[0]), "a.xlvals")
		assert.Equal(t, filepath.Base(files[1]), "b.xlvals")
		assert.Equal(t, filepath.Base(files[2]), "c.xlvals")
		assert.NotContains(t, files, "d.xlvals")
	})
}

func check(e error) {
	if e != nil {
		panic(e)
	}
}

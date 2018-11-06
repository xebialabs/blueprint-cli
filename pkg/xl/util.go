package xl

import (
	"fmt"
	"github.com/pkg/errors"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strings"
)

func FindByExtInDirSorted(parentPath string, ext string) ([]string, error) {
	var files []string
	err := filepath.Walk(parentPath, func(currentPath string, f os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !f.IsDir() && filepath.Ext(currentPath) == ext {
			files = append(files, currentPath)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}

	// sort by filename
	sort.Slice(files, func(i, j int) bool {
		_, f1 := path.Split(files[i])
		_, f2 := path.Split(files[j])
		return strings.Compare(f1, f2) == 0
	})

	return files, nil
}

func isRelativePath(filename string) bool {
	for _, p := range strings.Split(filename, string(os.PathSeparator)) {
		if p == ".." {
			return true
		}
	}
	return false
}

func ValidateFilePath(path string, in string) error {
	if filepath.IsAbs(path) {
		return errors.New(fmt.Sprintf("absolute path is not allowed in %s: %s\n", in, path))
	}
	if isRelativePath(path) {
		return errors.New(fmt.Sprintf("relative path with .. is not allowed in %s: %s\n", in, path))
	}
	return nil
}

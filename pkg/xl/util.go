package xl

import (
	"fmt"
	"github.com/pkg/errors"
	"github.com/thoas/go-funk"
	"io/ioutil"
	"os"
	"path"
	"path/filepath"
	"strings"
)

func FindByExtInDirSorted(parentPath string, ext string) ([]string, error) {
	var res []string
	files, err := ioutil.ReadDir(parentPath) // sorted by filename
	if err != nil {
		return nil, err
	}
	for _, f := range files {
		if !f.IsDir() && filepath.Ext(f.Name()) == ext {
			res = append(res, filepath.Join(parentPath, f.Name()))
		}
	}
	return res, nil
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

func ToAbsolutePaths(paths []string) []string {
	return funk.Map(paths, func(f string) string {
		abs, _ := filepath.Abs(filepath.FromSlash(f))
		return abs
	}).([]string)
}

func AbsoluteFileDir(fileName string) string {
	return filepath.FromSlash(path.Dir(filepath.ToSlash(fileName)))
}

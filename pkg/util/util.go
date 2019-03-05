package util

import (
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"path/filepath"

	funk "github.com/thoas/go-funk"
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

func ValidateFilePath(path string, in string) error {
	if filepath.IsAbs(path) {
		return fmt.Errorf("absolute path is not allowed in %s: %s\n", in, path)
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

// PathExists checks if the given relative path exists and has persmission to access it
func PathExists(filename string, mustBeDir bool) bool {
	info, err := os.Stat(filename)
	if os.IsNotExist(err) || os.IsPermission(err) {
		return false
	}

	if mustBeDir {
		return err == nil && info.IsDir()
	}
	return err == nil
}

func PrintableFileName(path string) string {
	if IsVerbose {
		return path
	} else {
		return filepath.Base(path)
	}
}

func TransformToMap(spec interface{}) []map[interface{}]interface{} {
	var convertedMap []map[interface{}]interface{}

	switch typeVal := spec.(type) {
	case []interface{}:
		list := make([]map[interface{}]interface{}, 0)
		for _, v := range typeVal {
			list = append(list, v.(map[interface{}]interface{}))
		}
		temporaryList := make([]map[interface{}]interface{}, len(list))

		for i, v := range list {
			temporaryList[i] = v
		}

		convertedMap = temporaryList
	case []map[interface{}]interface{}:
		convertedMap = typeVal
	case map[interface{}]interface{}:
		list := [...]map[interface{}]interface{}{typeVal}
		temporaryList := make([]map[interface{}]interface{}, 1)

		for i, v := range list {
			temporaryList[i] = v
		}

		convertedMap = temporaryList
	}

	return convertedMap
}

// Checks if the map contains value for the given key - empty values are not allowed
func MapContainsKey(dict map[interface{}]interface{}, key string) bool {
	val, ok := dict[key]
	if !ok {
		return false
	}
	return val != ""
}

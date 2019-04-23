package util

import (
	"fmt"
	"io/ioutil"
	"os"
    "os/user"
    "path"
	"path/filepath"
	"sort"
    "strings"

    "github.com/mitchellh/go-homedir"
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

// PathExists checks if the given relative path exists and has permission to access it
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

func ExpandHomeDirIfNeeded(path string, currentUser *user.User) string {
    if path == "~" || path == "~/" {
        Verbose("[path] path is user home directory [~]\n")
        return currentUser.HomeDir
    } else if strings.HasPrefix(path, "~/") {
        Verbose("[path] expanding local relative path [%s] for user [%s]\n", path, currentUser.Username)
        return filepath.Join(currentUser.HomeDir, path[2:])
    }
    return path
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
func MapContainsKeyWithVal(dict map[string]string, key string) bool {
	val, ok := dict[key]
	if !ok {
		return false
	}
	return val != ""
}
func MapContainsKeyWithValInterface(dict map[string]interface{}, key string) bool {
	val, ok := dict[key]
	if !ok {
		return false
	}
	return val != nil && val != ""
}

func DefaultConfigfilePath() (string, error) {
	home, err := homedir.Dir()
	if err != nil {
		return "", err
	}
	xebialabsFolder := filepath.Join(home, ".xebialabs", "config.yaml")
	return xebialabsFolder, nil
}

func SortMapStringInterface(m map[string]interface{}) map[string]interface{} {
	var keys []string
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	sorted := make(map[string]interface{})
	for _, k := range keys {
		switch v := m[k].(type) {
		case map[string]interface{}:
			sorted[k] = SortMapStringInterface(v)
		default:
			sorted[k] = v
		}
	}
	return sorted
}

// Returns difference between two slices: slice #1 - slice #2 as a set operation
func DiffBetweenStringSlices(slice1, slice2 []string) (diff []string) {
	diff = []string{}
	for _, item := range slice1 {
		if !funk.Contains(slice2, item) {
			diff = append(diff, item)
		}
	}
	return
}

func ExtractStringKeysFromMap(m map[string]interface{}) (keys []string) {
	keys = make([]string, len(m))
	i := 0
	for key := range m {
		keys[i] = key
		i += 1
	}
	return
}

func CopyIntoStringInterfaceMap(in map[string]interface{}, out map[string]interface{}) {
	for k, v := range in {
		out[k] = v
	}
}

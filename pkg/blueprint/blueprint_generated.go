package blueprint

import (
	"sort"
	"strings"

	"github.com/xebialabs/xl-cli/pkg/models"

	"github.com/xebialabs/xl-cli/pkg/util"

	"errors"
	"io"
	"os"
	"path/filepath"
)

// GeneratedBlueprint keeps track of all files and directories that were generated as part of the blueprint process.
type GeneratedBlueprint struct {
	OutputDir      string
	GeneratedFiles []string
}

// createDirectoryIfNeeded will create a Directory if it does not exist and add it to the GeneratedBlueprint context object.
func (generatedBlueprint *GeneratedBlueprint) createDirectoryIfNeeded(dirName string) error {
	util.Verbose("[file] Checking whether path %s exists\n", dirName)
	if exists(dirName) {
		if b, _ := isDirectory(dirName); !b {
			return errors.New(dirName + " exists but is not a directory.")
		}
		return nil
	}

	parentDir := filepath.Dir(dirName)
	generatedBlueprint.createDirectoryIfNeeded(parentDir)
	if err := generatedBlueprint.createDirectory(dirName); err != nil {
		return err
	}

	return nil
}

func (generatedBlueprint *GeneratedBlueprint) createDirectory(dirname string) error {
	util.Verbose("[file] Creating directory %s\n", dirname)
	err := os.Mkdir(dirname, os.ModePerm)
	if err != nil {
		return err
	}
	generatedBlueprint.GeneratedFiles = append(generatedBlueprint.GeneratedFiles, dirname)
	return nil
}

// GetOutputFile will return a newly created (or truncated) file.
func (generatedBlueprint *GeneratedBlueprint) GetOutputFile(fileName string) (*os.File, error) {
	if err := generatedBlueprint.createDirectoryIfNeeded(filepath.Dir(fileName)); err != nil {
		return nil, err
	}
	util.Verbose("[file] Creating file %s\n", fileName)
	file, err := os.Create(fileName)
	if err != nil {
		return nil, err
	}
	generatedBlueprint.GeneratedFiles = append(generatedBlueprint.GeneratedFiles, fileName)
	return file, nil
}

// Cleanup will cleanup all generated blueprint files
func (generatedBlueprint *GeneratedBlueprint) Cleanup() error {
	var directories []string

	// Clean all files first
	for _, file := range generatedBlueprint.GeneratedFiles {
		if isDir, _ := isDirectory(file); isDir {
			directories = append(directories, file)
		} else if util.PathExists(file, false) {
			if !(strings.Index(file, "cm_answer_file_auto") != -1 || strings.Index(file, "merged_answer_file") != -1) {
				if err := os.Remove(file); err != nil {
					return err
				}
			}
		}
	}
	// Reverse the directories
	sort.Sort(sort.Reverse(sort.StringSlice(directories)))

	xebialabsDir := ""

	for _, dir := range directories {
		util.Verbose("[file] Removing directory %s\n", dir)
		if util.PathExists(dir, true) {
			if empty, _ := isDirectoryEmpty(dir); empty {
				if err := os.Remove(dir); err != nil {
					return err
				}
			} else {
				if strings.HasSuffix(dir, models.BlueprintOutputDir) {
					xebialabsDir = dir
				}
			}
		}
	}

	// Manually remove the xebialabs directory
	if xebialabsDir != "" {
		if empty, _ := isDirectoryEmpty(xebialabsDir); empty {
			if err := os.Remove(xebialabsDir); err != nil {
				return err
			}
		}
	}

	return nil
}

func isDirectory(path string) (bool, error) {
	fileInfo, err := os.Stat(path)
	if err != nil {
		return false, err
	}
	return fileInfo.IsDir(), err
}

func isDirectoryEmpty(name string) (bool, error) {
	f, err := os.Open(name)
	if err != nil {
		return false, err
	}
	defer f.Close()

	_, err = f.Readdirnames(1) // Or f.Readdir(1)
	if err == io.EOF {
		return true, nil
	}
	return false, err // Either not empty or error, suits both cases
}

func exists(path string) bool {
	_, err := os.Stat(path)
	return err == nil || os.IsExist(err)
}

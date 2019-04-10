package xl

import (
	"os"
	"path/filepath"

	"github.com/mitchellh/go-homedir"
	"github.com/xebialabs/xl-cli/pkg/util"
)

func ListHomeXlValsFiles() ([]string, error) {
	home, err := homedir.Dir()
	if err != nil {
		return nil, err
	}
	xebialabsFolder := filepath.Join(home, ".xebialabs")
	if _, err := os.Stat(xebialabsFolder); os.IsNotExist(err) {
		return []string{}, nil
	}
	valfiles, err := util.FindByExtInDirSorted(xebialabsFolder, ".xlvals")
	if err != nil {
		return nil, err
	}
	return valfiles, nil
}

func ListRelativeXlValsFiles(dir string) ([]string, error) {
	valfiles, err := util.FindByExtInDirSorted(dir, ".xlvals")
	if err != nil {
		return nil, err
	}
	return valfiles, nil
}

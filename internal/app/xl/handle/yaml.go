package handle

import (
	"fmt"
	"github.com/xebialabs/xl-cli/internal/platform/yaml"
	"os"
)

func AddYamlFromFiles(pys *[]yaml.Yaml, fls []*os.File) error {
	for _, f := range fls {
		if ys, err := yaml.ParseFile(f); err != nil {
			return fmt.Errorf("error parsing YAML from file (%s): %v", f.Name(), err)
		} else {
			*pys = append(*pys, ys...)
		}
	}

	return nil
}

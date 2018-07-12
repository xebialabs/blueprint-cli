package xl

import (
	"github.com/xebialabs/xl-cli/internal/platform/handle"
	"github.com/xebialabs/xl-cli/internal/platform/yaml"
	"os"
)

func addYamlFromFiles(pys *[]yaml.Yaml, fls []*os.File) {
	for _, f := range fls {
		ys, err := yaml.ParseFile(f)

		handle.BasicError("error parsing YAML from file", err)

		*pys = append(*pys, ys...)
	}
}

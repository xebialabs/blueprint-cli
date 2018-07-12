package xl

import (
	"github.com/xebialabs/xl-cli/internal/platform/files"
	"github.com/xebialabs/xl-cli/internal/platform/handle"
	"github.com/xebialabs/xl-cli/internal/platform/yaml"
	"github.com/xebialabs/xl-cli/internal/servers"
)

func Apply(fs []string, url string) {
	defer handle.BasicPanicAsLog()

	fls, err := files.Open(fs...)

	defer closeFiles(fls)

	handle.BasicError("error opening files", err)

	ys := make([]yaml.Yaml, 0)
	addYamlFromFiles(&ys, fls)
	yg := yaml.Group(ys, "apiVersion")

	for key, val := range yg {
		s, err := yaml.ToString(val)

		handle.BasicError("error converting YAML to string", err)

		srv, err := servers.FromApiVersion(key.(string))

		handle.BasicError("error retrieving server", err)

		u := url

		if u == "" {
			u = srv.Url
		}

		postStringUrlAuth(u, s, "text/vnd.yaml")
	}
}

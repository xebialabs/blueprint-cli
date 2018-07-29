package xl

import (
	"github.com/xebialabs/xl-cli/internal/platform/files"
	"github.com/xebialabs/xl-cli/internal/platform/handle"
	"github.com/xebialabs/xl-cli/internal/platform/yaml"
	"github.com/xebialabs/xl-cli/internal/servers"
)

func Apply(fs []string, url string, xld string, xlr string) {
	defer handle.BasicPanicLog()

	fls, err := files.Open(fs...)

	defer closeFiles(fls)

	handle.BasicError("error opening files", err)

	ys := make([]yaml.Yaml, 0)
	addYamlFromFiles(&ys, fls)
	yg := yaml.Group(ys, "apiVersion")

	for key, val := range yg {
		s, err := yaml.String(val)

		handle.BasicError("error converting YAML to string", err)

		if url == "" {
			k := servers.ParseApiVersion(key.(string))
			srvN := map[string]string{
				servers.XldId: xld,
				servers.XlrId: xlr,
			}

			srv, err := servers.FromApiVersionAndName(k, srvN[k])

			handle.BasicError("error retrieving server", err)

			newBasicServerRequest(srv, methodPost, s, contentTypeYaml)
		} else {
			newBasicUrlRequest(url, methodPost, s, contentTypeYaml)
		}
	}
}

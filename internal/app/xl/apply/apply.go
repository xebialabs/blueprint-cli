package apply

import (
	"fmt"
	"github.com/xebialabs/xl-cli/internal/app/xl/handle"
	"github.com/xebialabs/xl-cli/internal/platform/files"
	"github.com/xebialabs/xl-cli/internal/platform/yaml"
	"github.com/xebialabs/xl-cli/internal/servers"
)

func Execute(fs []string, url string, xld string, xlr string) error {
	fls, err := files.Open(fs...)

	defer handle.CloseFiles(fls)

	if err != nil {
		return fmt.Errorf("error opening files: %v", err)
	}

	ys := make([]yaml.Yaml, 0)

	if err := handle.AddYamlFromFiles(&ys, fls); err != nil {
		return err
	}

	yg := yaml.Group(ys, "apiVersion")

	for key, val := range yg {
		if url == "" {
			k := servers.ParseApiVersion(key.(string))
			srvN := map[string]string{
				servers.XldId: xld,
				servers.XlrId: xlr,
			}

			srv, err := servers.FromApiVersionAndName(k, srvN[k])

			if err != nil {
				return fmt.Errorf("error retrieving server: %v", err)
			}

			if k == servers.XldId || k == servers.XlrId {
				for _, y := range val {
					if m, ok := y.Values["metadata"]; ok {
						meta := m.(map[interface{}]interface{})
						processServerMetadata(meta, srv)
					} else {
						meta := make(map[interface{}]interface{})
						processServerMetadata(meta, srv)
						y.Values["metadata"] = meta
					}
				}
			}

			if s, err := yamlToString(val); err != nil {
				return err
			} else {
				if _, _, respErr := handle.NewBasicServerRequest(srv, handle.MethodPost, s, handle.ContentTypeYaml); respErr != nil {
					return respErr
				}
			}
		} else {
			if s, err := yamlToString(val); err != nil {
				return err
			} else {
				if _, _, respErr := handle.NewBasicUrlRequest(url, handle.MethodPost, s, handle.ContentTypeYaml); respErr != nil {
					return respErr
				}
			}
		}
	}

	return nil
}

func processServerMetadata(meta map[interface{}]interface{}, srv *servers.Server) {
	for mKey, mVal := range srv.Metadata {
		if v, exist := meta[mKey]; !exist || v == "" {
			meta[mKey] = mVal
		}
	}
}

func yamlToString(ys []yaml.Yaml) (string, error) {
	if s, err := yaml.String(ys); err != nil {
		return "", fmt.Errorf("error converting YAML to string: %v", err)
	} else {
		return s, nil
	}
}

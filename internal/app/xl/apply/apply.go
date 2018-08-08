package apply

import (
	"fmt"
	"github.com/xebialabs/xl-cli/internal/app/xl/handle"
	"github.com/xebialabs/xl-cli/internal/platform/files"
	"github.com/xebialabs/xl-cli/internal/platform/yaml"
	"github.com/xebialabs/xl-cli/internal/servers"
	"github.com/xebialabs/xl-cli/internal/app/xl/login"
)

func Execute(fs []string, xld string, xlr string, xldUrl string, xldUsername string, xldPassword string, xldApplicationsHome string, xldConfigurationHome string,
	xldEnvironmentHome string, xldInfrastructureHome string, xlrUrl string, xlrUsername string, xlrPassword string, xlrHome string) error {

	xld, xlr, xldServer, xlrServer, err := processFlags(xld, xlr, xldUrl, xldUsername, xldPassword, xldApplicationsHome,
		xldConfigurationHome, xldEnvironmentHome, xldInfrastructureHome, xlrUrl, xlrUsername, xlrPassword, xlrHome)

	if err != nil {
		return err
	}

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
		k := servers.ParseApiVersion(key.(string))
		srvN := map[string]string{
			servers.XldId: xld,
			servers.XlrId: xlr,
		}

		var srv *servers.Server
		if k == servers.XldId && xldServer != nil {
			srv = xldServer
		} else if k == servers.XlrId && xlrServer != nil {
			srv = xlrServer
		} else {
			srv, err = servers.FromApiVersionAndName(k, srvN[k])
			if err != nil {
				return fmt.Errorf("error retrieving server: %v", err)
			}
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
			if _, _, respErr := handle.NewServerRequest(srv, handle.MethodPost, s, handle.ContentTypeYaml); respErr != nil {
				return respErr
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

func processFlags(xld string, xlr string, xldUrl string, xldUsername string, xldPassword string,
	xldApplicationsHome string, xldConfigurationHome string, xldEnvironmentHome string, xldInfrastructureHome string,
	xlrUrl string, xlrUsername string, xlrPassword string, xlrHome string) (string, string, *servers.Server, *servers.Server, error) {

	xldParams := xldUrl != "" || xldUsername != "" || xldPassword != "" || xldApplicationsHome != "" || xldConfigurationHome != "" || xldEnvironmentHome != "" || xldInfrastructureHome != ""
	if xld == "" && !xldParams {
		xld = "default"
	} else if xld != "" && xldParams {
		return xld, xlr, &servers.Server{}, &servers.Server{}, fmt.Errorf("xld flag can't be combined with xld-* flags")
	}

	if xldParams && (xldUrl == "" || xldUsername == "" || xldPassword == ""){
		return xld, xlr, &servers.Server{}, &servers.Server{}, fmt.Errorf("when using xld-* flags: xld-url, xld-username and xld-password are required")
	}

	xlrParams := xlrUrl != "" || xlrUsername != "" || xlrPassword != "" || xlrHome != ""
	if xlr == "" && !xlrParams {
		xlr = "default"
	} else if xlr != "" && xlrParams {
		return xld, xlr, &servers.Server{}, &servers.Server{}, fmt.Errorf("xlr flag can't be combined with xlr-* flags")
	}

	if xlrParams && (xlrUrl == "" || xlrUsername == "" || xlrPassword == ""){
		return xld, xlr, &servers.Server{}, &servers.Server{}, fmt.Errorf("when using xlr-* flags: xlr-url, xlr-username and xlr-password are required")
	}

	var xldServer *servers.Server
	var xlrServer *servers.Server

	if xldParams {
		xldMetadata := make(map[string]string)
		login.PopulateMetadata(servers.XldId, xldMetadata, xldApplicationsHome, xldConfigurationHome, xldEnvironmentHome, xldInfrastructureHome, "")
		xldServer = &servers.Server{
			Url:      xldUrl,
			Type:     servers.XldId,
			Username: xldUsername,
			Password: xldPassword,
			Metadata: xldMetadata,
		}
	}

	if xlrParams {
		xlrMetadata := make(map[string]string)
		login.PopulateMetadata(servers.XlrId, xlrMetadata, "", "", "", "", xlrHome)
		xlrServer = &servers.Server{
			Url:      xlrUrl,
			Type:     servers.XlrId,
			Username: xlrUsername,
			Password: xlrPassword,
			Metadata: xlrMetadata,
		}
	}

	return xld, xlr, xldServer, xlrServer, nil
}

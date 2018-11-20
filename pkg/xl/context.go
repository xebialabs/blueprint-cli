package xl

import (
	"fmt"
	"net/url"
)

type ChangedCis struct {
	Created *[]string
	Updated *[]string
}

type CiValidationError struct {
	CiId         string
	PropertyName string
	Message      string
}

type PermissionError struct {
	CiId       string
	Permission string
}

type DocumentFieldError struct {
	Field   string
	Problem string
}

type Errors struct {
	Validation *[]CiValidationError
	Permission *[]PermissionError
	Document   *DocumentFieldError
	Generic    *string
}

type TaskInfo struct {
	Id          string
	Description string
}

type Changes struct {
	Cis  *ChangedCis
	Task *TaskInfo
}

type AsCodeResponse struct {
	Changes *Changes
	Errors  *Errors
}

type TemplateRegistry struct {
	Name     string
	URL      url.URL
	Username string
	Password string
}

type Context struct {
	XLDeploy           XLServer
	XLRelease          XLServer
	values             map[string]string
	TemplateRegistries []TemplateRegistry
}

func (c *Context) PrintConfiguration() {
	Info("XL Deploy:\n  URL: %s\n  Username: %s\n  Applications home: %s\n  Environments home: %s\n  Infrastructure home: %s\n  Configuration home: %s\n",
		c.XLDeploy.(*XLDeployServer).Server.(*SimpleHTTPServer).Url.String(),
		c.XLDeploy.(*XLDeployServer).Server.(*SimpleHTTPServer).Username,
		c.XLDeploy.(*XLDeployServer).ApplicationsHome,
		c.XLDeploy.(*XLDeployServer).EnvironmentsHome,
		c.XLDeploy.(*XLDeployServer).InfrastructureHome,
		c.XLDeploy.(*XLDeployServer).ConfigurationHome)

	Info("XL Release:\n  URL: %s\n  Username: %s\n  Home: %s\n",
		c.XLRelease.(*XLReleaseServer).Server.(*SimpleHTTPServer).Url.String(),
		c.XLRelease.(*XLReleaseServer).Server.(*SimpleHTTPServer).Username,
		c.XLRelease.(*XLReleaseServer).Home)

	if len(c.values) > 0 {
		Info("Values:\n")
		for k, v := range c.values {
			Info("  %s: %s\n", k, v)
		}
	}

	if len(c.TemplateRegistries) > 0 {
		Info("Template Registries:\n")
		for _, v := range c.TemplateRegistries {
			Info("%s: %s \n", v.Name, v.URL.String())
		}
	}
}

func (c *Context) ProcessSingleDocument(doc *Document, artifactsDir string) (*Changes, error) {
	err := doc.Preprocess(c, artifactsDir)
	if err != nil {
		return nil, err
	}

	defer doc.Cleanup()

	if doc.ApiVersion == "" {
		return nil, fmt.Errorf("apiVersion missing")
	}

	if c.XLDeploy != nil && c.XLDeploy.AcceptsDoc(doc) {
		return c.XLDeploy.SendDoc(doc)
	}

	if c.XLRelease != nil && c.XLRelease.AcceptsDoc(doc) {
		return c.XLRelease.SendDoc(doc)
	}

	return nil, fmt.Errorf("unknown apiVersion: %s", doc.ApiVersion)
}

func (c *Context) ExportSingleDocument(exportServer string, exportFilename string, exportPath string, exportOverride bool) error {
	finalPath := url.QueryEscape(exportPath)

	if exportServer == "xl-deploy" {
		Info("Exporting %s from XL Deploy to %s\n", finalPath, exportFilename)
		return c.XLDeploy.ExportDoc(exportFilename, finalPath, exportOverride)
	}

	if exportServer == "xl-release" {
		Info("Exporting %s from XL Release to %s\n", finalPath, exportFilename)
		return c.XLRelease.ExportDoc(exportFilename, finalPath, exportOverride)
	}

	return fmt.Errorf("unknown server type: %s", exportServer)
}

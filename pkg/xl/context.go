package xl

import (
	"fmt"
)

type Context struct {
	XLDeploy  XLServer
	XLRelease XLServer
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
}

func (c *Context) ProcessSingleDocument(doc *Document, artifactsDir string) error {
	err := doc.Preprocess(c, artifactsDir)
	if err != nil {
		return err
	}

	defer doc.Cleanup()

	if doc.ApiVersion == "" {
		return fmt.Errorf("apiVersion missing")
	}

	if c.XLDeploy != nil && c.XLDeploy.AcceptsDoc(doc) {
		return c.XLDeploy.SendDoc(doc)
	}

	if c.XLRelease != nil && c.XLRelease.AcceptsDoc(doc) {
		return c.XLRelease.SendDoc(doc)
	}

	return fmt.Errorf("unknown apiVersion: %s", doc.ApiVersion)
}

func (c *Context) ExportSingleDocument(exportServer string, exportFilename string, exportPath string, exportOverride bool) error {

	if exportServer == "xl-deploy" {
		Info("Exporting %s from XL Deploy to %s\n", exportPath, exportFilename)
		return c.XLDeploy.ExportDoc(exportFilename, exportPath, exportOverride)
	}

	if exportServer == "xl-release" {
		Info("Exporting %s from XL Release to %s\n", exportPath, exportFilename)
		return c.XLRelease.ExportDoc(exportFilename, exportPath, exportOverride)
	}

	return fmt.Errorf("unknown server type: %s", exportServer)
}
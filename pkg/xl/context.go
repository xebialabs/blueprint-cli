package xl

import (
	"fmt"
)

type Context struct {
	XLDeploy  XLServer
	XLRelease XLServer
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
		return c.XLDeploy.ExportDoc(exportFilename, exportPath, exportOverride)
	}

	if exportServer == "xl-release" {
		return c.XLRelease.ExportDoc(exportFilename, exportPath, exportOverride)
	}

	return fmt.Errorf("unknown server type: %s", exportServer)
}
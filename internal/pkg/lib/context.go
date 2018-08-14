package lib

import "fmt"

type Context struct {
	XLDeploy  XLServer
	XLRelease XLServer
}

func (c *Context) ProcessSingleDocument(doc *Document) error {
	if doc.ApiVersion == "xl-deploy/v1alpha1" {
		return c.XLDeploy.SendDoc(doc)
	} else if doc.ApiVersion == "xl-release/v1" {
		return c.XLRelease.SendDoc(doc)
	} else if doc.ApiVersion == "" {
		return fmt.Errorf("apiVersion missing")
	} else {
		return fmt.Errorf("unknown apiVersion: %s", doc.ApiVersion)
	}
}

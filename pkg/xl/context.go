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
	RawBody string
}

type BlueprintRepository struct {
	Server SimpleHTTPServer
}

type Context struct {
	XLDeploy            XLServer
	XLRelease           XLServer
	BlueprintRepository BlueprintRepository
	values              map[string]string
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

	Info("Blueprint Repository:\n  URL: %s\n  Username: %s\n",
		c.BlueprintRepository.Server.Url.String(),
		c.BlueprintRepository.Server.Username)
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

func (c *Context) GenerateSingleDocument(generateServer string, generateFilename string, generatePath string, generateOverride bool) error {
	finalPath := url.QueryEscape(generatePath)

	if generateServer == "xl-deploy" {
		Info("Generating definitions for path %s from XL Deploy to %s\n", finalPath, generateFilename)
		return c.XLDeploy.GenerateDoc(generateFilename, finalPath, generateOverride)
	}

	if generateServer == "xl-release" {
		Info("Generating definitions for path %s from XL Release to %s\n", finalPath, generateFilename)
		return c.XLRelease.GenerateDoc(generateFilename, finalPath, generateOverride)
	}

	return fmt.Errorf("unknown server type: %s", generateServer)
}

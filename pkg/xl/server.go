package xl

import "fmt"

const XldApiVersion = "xl-deploy/v1beta1"
const XlrApiVersion = "xl-release/v1beta1"

type XLServer interface {
	AcceptsDoc(doc *Document) bool
	PreprocessDoc(doc *Document)
	SendDoc(doc *Document) error
	ExportDoc(filename string, path string, override bool) error
}

type XLDeployServer struct {
	Server             HTTPServer
	ApplicationsHome   string
	ConfigurationHome  string
	EnvironmentsHome   string
	InfrastructureHome string
}

type XLReleaseServer struct {
	Server HTTPServer
	Home   string
}

func (server *XLDeployServer) AcceptsDoc(doc *Document) bool {
	return doc.ApiVersion == XldApiVersion
}

func (server *XLReleaseServer) AcceptsDoc(doc *Document) bool {
	return doc.ApiVersion == XlrApiVersion
}

func (server *XLDeployServer) PreprocessDoc(doc *Document) {
	addHomeIfMissing(doc, server.ApplicationsHome, "Applications-home")
	addHomeIfMissing(doc, server.EnvironmentsHome, "Environments-home")
	addHomeIfMissing(doc, server.InfrastructureHome, "Infrastructure-home")
	addHomeIfMissing(doc, server.ConfigurationHome, "Configuration-home")
}

func (server *XLReleaseServer) PreprocessDoc(doc *Document) {
	addHomeIfMissing(doc, server.Home, "home")
}

func addHomeIfMissing(doc *Document, home string, key string) {
	if _, found := doc.Metadata[key]; home != "" && !found {
		doc.Metadata[key] = home
	}
}

func (server *XLDeployServer) ExportDoc(filename string, path string, override bool) error {
	return server.Server.ExportYamlDoc(filename, "deployit/devops-as-code/export/" + path, override)
}

func (server *XLReleaseServer) ExportDoc(filename string, path string, override bool) error {
	return server.Server.ExportYamlDoc(filename, "devops-as-code/export/" + path, override)
}

func (server *XLDeployServer) SendDoc(doc *Document) error {
	return sendDoc(server.Server, "deployit/devops-as-code/apply", doc)
}

func (server *XLReleaseServer) SendDoc(doc *Document) error {
	if doc.ApplyZip != "" {
		return fmt.Errorf("file tags found but XL Release does not support file references")
	}
	return sendDoc(server.Server, "devops-as-code/apply", doc)
}

func sendDoc(server HTTPServer, path string, doc *Document) error {
	if doc.ApplyZip != "" {
		Verbose("file references found, posting zip to server\n")
		return server.PostYamlZip(path, doc.ApplyZip)
	} else {
		Verbose("no file references found, posting yaml to server\n")
		documentBytes, err := doc.RenderYamlDocument()
		if err != nil {
			return err
		}
		return server.PostYamlDoc(path, documentBytes)
	}
}

package lib

type XLServer interface {
	AcceptsDoc(doc *Document) bool
	PreprocessDoc(doc *Document)
	SendDoc(doc *Document) error
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
	return doc.ApiVersion == "xl-deploy/v1alpha1"
}

func (server *XLReleaseServer) AcceptsDoc(doc *Document) bool {
	return doc.ApiVersion == "xl-release/v1"
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

func (server *XLDeployServer) SendDoc(doc *Document) error {
	return sendDoc(server.Server, "deployit/ascode", doc)
}

func (server *XLReleaseServer) SendDoc(doc *Document) error {
	return sendDoc(server.Server, "ascode", doc)
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
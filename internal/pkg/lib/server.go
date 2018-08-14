package lib

type XLServer interface {
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

func addHomeIfMissing(doc *Document, home string, key string) {
	if _, found := doc.Metadata[key]; home != "" && !found {
		doc.Metadata[key] = home
	}
}

func (server *XLDeployServer) SendDoc(doc *Document) error {
	if doc.Metadata == nil {
		doc.Metadata = make(map[interface{}]interface{})
	}

	addHomeIfMissing(doc, server.ApplicationsHome, "Applications-home")
	addHomeIfMissing(doc, server.EnvironmentsHome, "Environments-home")
	addHomeIfMissing(doc, server.InfrastructureHome, "Infrastructure-home")
	addHomeIfMissing(doc, server.ConfigurationHome, "Configuration-home")

	documentBytes, err := doc.RenderYamlDocument()
	if err != nil {
		return err
	}

	return server.Server.PostYaml("deployit/ascode", documentBytes)
}

func (server *XLReleaseServer) SendDoc(doc *Document) error {
	if doc.Metadata == nil {
		doc.Metadata = make(map[interface{}]interface{})
	}

	addHomeIfMissing(doc, server.Home, "home")

	documentBytes, err := doc.RenderYamlDocument()
	if err != nil {
		return err
	}

	return server.Server.PostYaml("ascode", documentBytes)
}

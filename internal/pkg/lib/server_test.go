package lib

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

type DummyHTTPServer struct {
	capturedBytes []byte
}

func (d *DummyHTTPServer) PostYaml(path string, body []byte) error {
	d.capturedBytes = body
	return nil
}

func TestServer(t *testing.T) {
	t.Run("should add xl-deploy homes if missing in yaml document", func(t *testing.T) {

		dummyServer := DummyHTTPServer{}
		server := XLDeployServer{Server: &dummyServer,
			ApplicationsHome:   "Applications/MyHome",
			EnvironmentsHome:   "Environments/MyHome",
			ConfigurationHome:  "Configuration/MyHome",
			InfrastructureHome: "Infrastructure/MyHome"}

		yaml := `
apiVersion: xl-deploy/v1
kind: Applications
`

		doc, err := ParseYamlDocument(yaml)
		assert.Nil(t, err)
		assert.NotNil(t, doc)

		err2 := server.SendDoc(doc)
		assert.Nil(t, err2)

		doc, err = ParseYamlDocument(string(dummyServer.capturedBytes))
		assert.Nil(t, err)

		// should not throw away existing fields
		assert.Equal(t, "xl-deploy/v1", doc.ApiVersion)
		assert.Equal(t, "Applications", doc.Kind)

		assert.Equal(t, "Applications/MyHome", doc.Metadata["Applications-home"])
		assert.Equal(t, "Environments/MyHome", doc.Metadata["Environments-home"])
		assert.Equal(t, "Configuration/MyHome", doc.Metadata["Configuration-home"])
		assert.Equal(t, "Infrastructure/MyHome", doc.Metadata["Infrastructure-home"])
	})

	t.Run("should add xl-release home if missing in yaml document", func(t *testing.T) {

		dummyServer := DummyHTTPServer{}
		server := XLReleaseServer{Server: &dummyServer, Home: "MyHome"}

		yaml := `
apiVersion: xl-release/v1
kind: Templates
`

		doc, err := ParseYamlDocument(yaml)
		assert.Nil(t, err)
		assert.NotNil(t, doc)

		err2 := server.SendDoc(doc)
		assert.Nil(t, err2)

		doc, err = ParseYamlDocument(string(dummyServer.capturedBytes))
		assert.Nil(t, err)

		assert.Equal(t, "MyHome", doc.Metadata["home"])
	})

	t.Run("should not replace homes if included in yaml document", func(t *testing.T) {
		dummyServer := DummyHTTPServer{}
		server := XLDeployServer{Server: &dummyServer,
			ApplicationsHome:   "Applications/MyHome",
			EnvironmentsHome:   "Environments/MyHome",
			ConfigurationHome:  "Configuration/MyHome",
			InfrastructureHome: "Infrastructure/MyHome"}

		yaml := `
apiVersion: xl-deploy/v1
kind: Applications
metadata:
  Applications-home: Applications/DoNotTouch
  Environments-home: Environments/DoNotTouch
  Configuration-home: Configuration/DoNotTouch
  Infrastructure-home: Infrastructure/DoNotTouch
`

		doc, err := ParseYamlDocument(yaml)
		assert.Nil(t, err)
		assert.NotNil(t, doc)

		err2 := server.SendDoc(doc)
		assert.Nil(t, err2)

		doc, err = ParseYamlDocument(string(dummyServer.capturedBytes))
		assert.Nil(t, err)

		assert.Equal(t, "Applications/DoNotTouch", doc.Metadata["Applications-home"])
		assert.Equal(t, "Environments/DoNotTouch", doc.Metadata["Environments-home"])
		assert.Equal(t, "Configuration/DoNotTouch", doc.Metadata["Configuration-home"])
		assert.Equal(t, "Infrastructure/DoNotTouch", doc.Metadata["Infrastructure-home"])
	})

	t.Run("should not add homes if empty", func(t *testing.T) {
		dummyServer := DummyHTTPServer{}
		server := XLDeployServer{Server: &dummyServer,
			ApplicationsHome:   "",
			EnvironmentsHome:   "",
			ConfigurationHome:  "",
			InfrastructureHome: ""}

		yaml := `
apiVersion: xl-deploy/v1
kind: Applications
`

		doc, err := ParseYamlDocument(yaml)
		assert.Nil(t, err)
		assert.NotNil(t, doc)

		err2 := server.SendDoc(doc)
		assert.Nil(t, err2)

		doc, err = ParseYamlDocument(string(dummyServer.capturedBytes))
		assert.Nil(t, err)

		assert.Nil(t, doc.Metadata["Applications-home"])
		assert.Nil(t, doc.Metadata["Environments-home"])
		assert.Nil(t, doc.Metadata["Configuration-home"])
		assert.Nil(t, doc.Metadata["Infrastructure-home"])
	})
}

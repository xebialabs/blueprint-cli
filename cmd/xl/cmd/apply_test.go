package cmd

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
	"github.com/xebialabs/xl-cli/internal/pkg/lib"
	"io/ioutil"
	"os"
	"github.com/magiconair/properties/assert"
	"fmt"
)

func TestApply(t *testing.T) {
	t.Run("should apply multiple yaml files in right order to both xlr and xld", func(t *testing.T) {

		file1, err := ioutil.TempFile("", "file1")
		if err != nil { panic(err) }
		defer os.Remove(file1.Name())
		file1.WriteString(fmt.Sprintf(`
apiVersion: %s
kind: Template
spec:
- name: Template1

---

apiVersion: %s
kind: Applications
spec:
- name: App1
`, lib.XlrApiVersion, lib.XldApiVersion))
		file1.Close()

		file2, err := ioutil.TempFile("", "file2")
		if err != nil { panic(err) }
		defer os.Remove(file2.Name())
		file2.WriteString(fmt.Sprintf(`
apiVersion: %s
kind: Template
spec:
- name: Template2

---

apiVersion: %s
kind: Applications
spec:
- name: App2
`, lib.XlrApiVersion, lib.XldApiVersion))
		file2.Close()

		var documents []lib.Document

		xldHandler := func(responseWriter http.ResponseWriter, request *http.Request) {
			body, err := ioutil.ReadAll(request.Body)
			if err != nil {panic(err)}
			doc, err := lib.ParseYamlDocument(string(body))
			if err != nil {panic(err)}
			documents = append(documents, *doc)
		}

		xlrHandler := func(responseWriter http.ResponseWriter, request *http.Request) {
			body, err := ioutil.ReadAll(request.Body)
			if err != nil {panic(err)}
			doc, err := lib.ParseYamlDocument(string(body))
			if err != nil {panic(err)}
			documents = append(documents, *doc)
		}

		applyFilenames := []string{file1.Name(), file2.Name()}

		xldTestServer := httptest.NewServer(http.HandlerFunc(xldHandler))
		defer xldTestServer.Close()

		xlrTestServer := httptest.NewServer(http.HandlerFunc(xlrHandler))
		defer xlrTestServer.Close()

		xldUrl, _ := url.Parse(xldTestServer.URL + "/")
		xlrUrl, _ := url.Parse(xlrTestServer.URL + "/")

		context := lib.Context{
			XLDeploy:  &lib.XLDeployServer{Server: &lib.SimpleHTTPServer{Url: *xldUrl, Username:"", Password:""},
										   ApplicationsHome: "Applications/XL", InfrastructureHome:"",
										   ConfigurationHome:"",EnvironmentsHome:""},
			XLRelease: &lib.XLReleaseServer{Server: &lib.SimpleHTTPServer{Url: *xlrUrl, Username:"", Password:""},
										   Home: "XL"},
		}

		DoApply(&context, applyFilenames)

		assert.Equal(t, documents[0].Spec[0]["name"], "Template1")
		assert.Equal(t, documents[0].Metadata["home"], "XL")
		assert.Equal(t, documents[1].Spec[0]["name"], "App1")
		assert.Equal(t, documents[1].Metadata["Applications-home"], "Applications/XL")
		assert.Equal(t, documents[2].Spec[0]["name"], "Template2")
		assert.Equal(t, documents[2].Metadata["home"], "XL")
		assert.Equal(t, documents[3].Spec[0]["name"], "App2")
		assert.Equal(t, documents[3].Metadata["Applications-home"], "Applications/XL")
	})
}

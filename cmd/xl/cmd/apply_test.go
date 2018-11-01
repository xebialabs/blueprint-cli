package cmd

import (
	"testing"
	"github.com/xebialabs/xl-cli/pkg/xl"
	"io/ioutil"
	"os"
	"github.com/magiconair/properties/assert"
	"fmt"
	"net/http"
	"github.com/spf13/viper"
	"net/http/httptest"
	"path/filepath"
)


func TestApply(t *testing.T) {

	xl.IsVerbose = true

	t.Run("should apply multiple yaml files in right order with value replacement to both xlr and xld", func(t *testing.T) {

		tempDir1, err := ioutil.TempDir("", "firstDir")
		check(err)
		prop1, err := os.Create(filepath.Join(tempDir1, "prop1.xlvals"))
		check(err)
		defer prop1.Close()
		_, err2 := prop1.WriteString("replaceme=success1")
		prop1.Sync()
		check(err2)
		yaml, err := ioutil.TempFile(tempDir1, "yaml1")
		check(err)
		defer os.RemoveAll(tempDir1)
		yaml.WriteString(fmt.Sprintf(`
apiVersion: %s
kind: Template
spec:
- name: Template1
- replaceTest: !value replaceme

---

apiVersion: %s
kind: Applications
spec:
- name: App1
`, xl.XlrApiVersion, xl.XldApiVersion))
		yaml.Close()

		tempDir2, err := ioutil.TempDir("", "secondDir")
		check(err)
		prop2, err := os.Create(filepath.Join(tempDir2, "prop2.xlvals"))
		check(err)
		defer prop2.Close()
		_, err3 := prop2.WriteString("replaceme=success2\noverrideme=notoverriden")
		prop2.Sync()
		check(err3)

		prop3, err := os.Create(filepath.Join(tempDir2, "prop3.xlvals"))
		check(err)
		defer prop3.Close()
		_, err4 := prop3.WriteString("overrideme=OVERRIDDEN")
		prop3.Sync()
		check(err4)

		yaml2, err := ioutil.TempFile(tempDir2, "yaml2")
		check(err)
		defer os.RemoveAll(tempDir2)
		yaml2.WriteString(fmt.Sprintf(`
apiVersion: %s
kind: Template
spec:
- name: Template2
- replaceTest: !value replaceme
- overrideTest: !value overrideme
---

apiVersion: %s
kind: Applications
spec:
- name: App2
`, xl.XlrApiVersion, xl.XldApiVersion))
		yaml2.Close()

		var documents []xl.Document

		xldHandler := func(responseWriter http.ResponseWriter, request *http.Request) {
			body, err := ioutil.ReadAll(request.Body)
			check(err)
			doc, err := xl.ParseYamlDocument(string(body))
			check(err)
			documents = append(documents, *doc)
		}

		xlrHandler := func(responseWriter http.ResponseWriter, request *http.Request) {
			body, err := ioutil.ReadAll(request.Body)
			check(err)
			doc, err := xl.ParseYamlDocument(string(body))
			check(err)
			documents = append(documents, *doc)
		}

		applyFilenames := []string{yaml.Name(), yaml2.Name()}

		xldTestServer := httptest.NewServer(http.HandlerFunc(xldHandler))
		defer xldTestServer.Close()

		xlrTestServer := httptest.NewServer(http.HandlerFunc(xlrHandler))
		defer xlrTestServer.Close()

		v := viper.GetViper()
		v.Set("xl-deploy.url", xldTestServer.URL)
		v.Set("xl-release.url", xlrTestServer.URL)
		v.Set("xl-deploy.applications-home", "Applications/XL")
		v.Set("xl-release.home", "XL")

		DoApply(applyFilenames)

		assert.Equal(t, xl.TransformToMap(documents[0].Spec)[0]["name"], "Template1")
		assert.Equal(t, xl.TransformToMap(documents[0].Spec)[1]["replaceTest"], "success1")
		assert.Equal(t, documents[0].Metadata["home"], "XL")
		assert.Equal(t, xl.TransformToMap(documents[1].Spec)[0]["name"], "App1")
		assert.Equal(t, documents[1].Metadata["Applications-home"], "Applications/XL")
		assert.Equal(t, xl.TransformToMap(documents[2].Spec)[0]["name"], "Template2")
		assert.Equal(t, xl.TransformToMap(documents[2].Spec)[1]["replaceTest"], "success2")
		assert.Equal(t, xl.TransformToMap(documents[2].Spec)[2]["overrideTest"], "OVERRIDDEN")
		assert.Equal(t, documents[2].Metadata["home"], "XL")
		assert.Equal(t, xl.TransformToMap(documents[3].Spec)[0]["name"], "App2")
		assert.Equal(t, documents[3].Metadata["Applications-home"], "Applications/XL")
	})
}


func check(e error) {
	if e != nil {
		panic(e)
	}
}
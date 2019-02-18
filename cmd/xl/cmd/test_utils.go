package cmd

import (
	"github.com/spf13/viper"
	"github.com/xebialabs/xl-cli/pkg/util"
	"github.com/xebialabs/xl-cli/pkg/xl"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"os"
)

type TestInfra struct {
	documents []xl.Document
	xldServer *httptest.Server
	xlrServer *httptest.Server
}

func (infra *TestInfra) appendDoc(document xl.Document) {
	infra.documents = append(infra.documents, document)
}

func (infra *TestInfra) shutdown() {
	infra.xldServer.Close()
	infra.xlrServer.Close()
}

func (infra *TestInfra) doc(index int) xl.Document {
	return infra.documents[index]
}

func (infra *TestInfra) spec(index int) []map[interface{}]interface{} {
	return util.TransformToMap(infra.doc(index).Spec)
}

func (infra *TestInfra) metadata(index int) map[interface{}]interface{} {
	return infra.doc(index).Metadata
}

func CreateTestInfra(viper *viper.Viper) *TestInfra {
	infra := &TestInfra{documents: make([]xl.Document, 0)}

	xldHandler := func(responseWriter http.ResponseWriter, request *http.Request) {
		body, err := ioutil.ReadAll(request.Body)
		check(err)
		doc, err := xl.ParseYamlDocument(string(body))
		check(err)
		infra.appendDoc(*doc)
		_, _ = responseWriter.Write([]byte("{}"))
	}

	xlrHandler := func(responseWriter http.ResponseWriter, request *http.Request) {
		body, err := ioutil.ReadAll(request.Body)
		check(err)
		doc, err := xl.ParseYamlDocument(string(body))
		check(err)
		infra.appendDoc(*doc)
		_, _ = responseWriter.Write([]byte("{}"))
	}

	infra.xldServer = httptest.NewServer(http.HandlerFunc(xldHandler))
	infra.xlrServer = httptest.NewServer(http.HandlerFunc(xlrHandler))

	viper.Set("xl-deploy.url", infra.xldServer.URL)
	viper.Set("xl-release.url", infra.xlrServer.URL)
	return infra
}

func createTempDir(name string) string {
	dir, err := ioutil.TempDir("", name)
	check(err)
	return dir
}

func writeToTempFile(tempDir string, fileName string, value string) *os.File {
	file, err := ioutil.TempFile(tempDir, fileName)
	check(err)
	return writeTo(file, value)
}

func writeToFile(filePath string, value string) *os.File {
	file, err := os.Create(filePath)
	check(err)
	return writeTo(file, value)
}

func writeTo(file *os.File, value string) *os.File {
	defer file.Close()
	_, err2 := file.WriteString(value)
	_ = file.Sync()
	check(err2)
	return file
}

func check(e error) {
	if e != nil {
		panic(e)
	}
}

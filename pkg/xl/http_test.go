package xl

import (
	"archive/zip"
	"encoding/base64"
	"fmt"
	"github.com/stretchr/testify/assert"
	"io"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"testing"
)

func TestHttp(t *testing.T) {
	t.Run("should post yaml with credentials and correct content type", func(t *testing.T) {

		handler := func(responseWriter http.ResponseWriter, request *http.Request) {
			// Useful for troubleshooting
			//requestDump, err := httputil.DumpRequest(request, true)
			//if err != nil {
			//	fmt.Println(err)
			//}
			//fmt.Println("REQUEST: \n" + string(requestDump))

			assert.Equal(t, "POST", request.Method)
			assert.Equal(t, "/devops-as-code/apply", request.URL.Path)
			assert.Equal(t, "text/vnd.yaml", request.Header.Get("Content-Type"))
			assert.Equal(t, "Basic "+base64.StdEncoding.EncodeToString([]byte("admin:admin")), request.Header.Get("Authorization"))
			body, err := ioutil.ReadAll(request.Body)
			assert.Nil(t, err)
			assert.Equal(t, "document body", string(body))
			_, _ = responseWriter.Write([]byte("{}"))
		}

		testServer := httptest.NewServer(http.HandlerFunc(handler))
		defer testServer.Close()

		res, _ := url.Parse(testServer.URL)
		server := SimpleHTTPServer{Url: *res, Username: "admin", Password: "admin"}

		_, e := server.PostYamlDoc("devops-as-code/apply", []byte("document body"))
		assert.Nil(t, e)
	})

	t.Run("should post ZIP", func(t *testing.T) {
		filesToUploaded := map[string]string{"file1": "this is the first file", "file2": "and this is the second file"}
		zipToUpload := createZip(t, filesToUploaded)
		defer os.Remove(zipToUpload.Name())

		handler := func(responseWriter http.ResponseWriter, request *http.Request) {
			assert.Equal(t, "POST", request.Method)
			assert.Equal(t, "/apply", request.URL.Path)
			assert.Equal(t, "application/zip", request.Header.Get("Content-Type"))
			assert.Equal(t, "Basic "+base64.StdEncoding.EncodeToString([]byte("root:s3cr3t")), request.Header.Get("Authorization"))

			uploadedZip, err := ioutil.TempFile("", "uploadedZip")
			if err != nil {
				assert.Fail(t, "cannot create temporary file to store request body", "cannot create temporary file to store request body: %s", err)
				return
			}
			defer os.Remove(uploadedZip.Name())

			contentLength, err := io.Copy(uploadedZip, request.Body)
			if err != nil {
				assert.Fail(t, "cannot write request body to temporary file", "cannot write request body to temporary file: %s", err)
				return
			}
			e := uploadedZip.Close()
			assert.Nil(t, e)

			r, err := os.Open(uploadedZip.Name())
			if err != nil {
				assert.Fail(t, "cannot open temporary file to read request body", "cannot open temporary file [%s] to read request body: %s", uploadedZip.Name(), err)
				return
			}
			defer r.Close()

			zipReader, err := zip.NewReader(r, contentLength)
			if err != nil {
				assert.FailNow(t, "cannot open uploaded ZIP file", "cannot open uploaded ZIP file: %s", err)
			}

			uploadedFiles := make(map[string]string)
			for _, f := range zipReader.File {
				fr, err := f.Open()
				if err != nil {
					assert.FailNow(t, "cannot open entry in uploaded ZIP file", "cannot open entry [%s] in uploaded ZIP file: %s", f.Name, err)
				}
				contents, err := ioutil.ReadAll(fr)
				if err != nil {
					assert.FailNow(t, "cannot read entry in uploaded ZIP file", "cannot read entry [%s] in uploaded ZIP file: %s", f.Name, err)
				}
				uploadedFiles[f.Name] = string(contents)
			}

			assert.Equal(t, filesToUploaded, uploadedFiles)
			_, _ = responseWriter.Write([]byte("{}"))
		}

		testServer := httptest.NewServer(http.HandlerFunc(handler))
		defer testServer.Close()

		res, _ := url.Parse(testServer.URL)
		server := SimpleHTTPServer{Url: *res, Username: "root", Password: "s3cr3t"}

		_, e := server.PostYamlZip("apply", zipToUpload.Name())
		assert.Nil(t, e)
	})

	t.Run("should generate yaml and depending files", func(t *testing.T) {

		handler := func(responseWriter http.ResponseWriter, request *http.Request) {
			assert.Equal(t, "GET", request.Method)
			assert.Equal(t, "/generate/Applications", request.URL.Path)
			assert.Equal(t, "Basic "+base64.StdEncoding.EncodeToString([]byte("root:s3cr3t")), request.Header.Get("Authorization"))

			archiveFiles := map[string]string{"index.yaml": "yaml: content", "otherfile": "otherfile content"}
			zipfile := createZip(t, archiveFiles)
			defer os.Remove(zipfile.Name())

			b, err := ioutil.ReadFile(zipfile.Name())
			if err != nil {
				panic(err)
			}
			_, e := responseWriter.Write(b)
			assert.Nil(t, e)
		}

		file, err := ioutil.TempFile("", "generated.yaml")
		if err != nil {
			panic(err)
		}
		defer os.Remove(file.Name())

		testServer := httptest.NewServer(http.HandlerFunc(handler))
		defer testServer.Close()

		res, _ := url.Parse(testServer.URL)
		server := SimpleHTTPServer{Url: *res, Username: "root", Password: "s3cr3t"}

		e := server.GenerateYamlDoc(file.Name(), "generate/Applications", true)
		assert.Nil(t, e)

		b, err := ioutil.ReadFile(file.Name())
		if err != nil {
			fmt.Print(err)
		}
		assert.Equal(t, "yaml: content", string(b))

		b2, err := ioutil.ReadFile(filepath.Join(filepath.Dir(file.Name()), "otherfile"))
		if err != nil {
			fmt.Print(err)
		}
		assert.Equal(t, "otherfile content", string(b2))
	})

	t.Run("should refuse to generate when file exists", func(t *testing.T) {
		file, err := ioutil.TempFile("", "generated.yaml")
		if err != nil {
			panic(err)
		}
		defer os.Remove(file.Name())

		res, _ := url.Parse("http://test")
		server := SimpleHTTPServer{Url: *res, Username: "", Password: ""}

		e := server.GenerateYamlDoc(file.Name(), "generate/Applications", false)
		assert.Contains(t, e.Error(), "already exists")
	})

	t.Run("should generate schema", func(t *testing.T) {
		handler := func(responseWriter http.ResponseWriter, request *http.Request) {
			assert.Equal(t, "GET", request.Method)
			assert.Equal(t, "/deployit/devops-as-code/schema", request.URL.Path)
			assert.Equal(t, "Basic "+base64.StdEncoding.EncodeToString([]byte("root:s3cr3t")), request.Header.Get("Authorization"))

			_, e := responseWriter.Write([]byte("schemabody"))
			assert.Nil(t, e)
		}

		testServer := httptest.NewServer(http.HandlerFunc(handler))
		defer testServer.Close()

		res, _ := url.Parse(testServer.URL)
		server := SimpleHTTPServer{Url: *res, Username: "root", Password: "s3cr3t"}

		bytes, e := server.DownloadSchema("deployit/devops-as-code/schema")
		assert.Nil(t, e)

		assert.Equal(t, "schemabody", string(bytes))
	})

	t.Run("should request task info and transform it to map-like structure", func(t *testing.T) {
		handler := func(responseWriter http.ResponseWriter, request *http.Request) {
			assert.Equal(t, "GET", request.Method)
			assert.Equal(t, "/deployit/tasks/v2/12345", request.URL.Path)
			assert.Equal(t, "application/json", request.Header.Get("Content-Type"))
			assert.Equal(t, "application/json", request.Header.Get("Accept"))
			_, _ = responseWriter.Write([]byte(`{"id":"12345","state":"EXECUTING","blocks":[{"id":"0_1","state":"EXECUTING"}]}`))
		}

		testServer := httptest.NewServer(http.HandlerFunc(handler))
		defer testServer.Close()

		res, _ := url.Parse(testServer.URL)
		server := SimpleHTTPServer{Url: *res, Username: "admin", Password: "admin"}

		response, err := server.TaskInfo("deployit/tasks/v2/12345")

		assert.Equal(t, "12345", response["id"])
		assert.Equal(t, "EXECUTING", response["state"])
		blocks := response["blocks"].([]interface{})
		assert.Len(t, blocks, 1)
		firstBlock := blocks[0].(map[string]interface{})
		assert.Equal(t, "0_1", firstBlock["id"])
		assert.Equal(t, "EXECUTING", firstBlock["state"])
		assert.Nil(t, err)
	})
}

func createZip(t *testing.T, filesToUploaded map[string]string) *os.File {
	zipToUpload, err := ioutil.TempFile("", "zipToUpload")
	if err != nil {
		assert.FailNow(t, "cannot create temporary file to create zip", "cannot create temporary file to create zip: %s", err)
	}
	zipWriter := zip.NewWriter(zipToUpload)
	for filename, fileContents := range filesToUploaded {
		w, err := zipWriter.Create(filename)
		if err != nil {
			assert.FailNow(t, "cannot add file to zip", "cannot add file [%s] to zip: %s", filename, err)
		}
		_, err = w.Write([]byte(fileContents))
		if err != nil {
			assert.FailNow(t, "cannot write file to zip", "cannot write file [%s] to zip: %s", filename, err)
		}

	}
	err = zipWriter.Close()
	if err != nil {
		assert.FailNow(t, "cannot close zip", "cannot close zip: %s", err)
	}
	err = zipToUpload.Close()
	if err != nil {
		assert.FailNow(t, "cannot close zip file", "cannot close zip file: %s", err)
	}
	return zipToUpload
}

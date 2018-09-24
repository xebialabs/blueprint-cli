package xl

import (
	"bytes"
	"fmt"
	"github.com/mholt/archiver"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
)

type HTTPServer interface {
	PostYamlDoc(path string, yamlDocBytes []byte) error
	PostYamlZip(path string, yamlZipFilename string) error
	ExportYamlDoc(exportFilename string, path string, override bool) error
}

type SimpleHTTPServer struct {
	Url      url.URL
	Username string
	Password string
}

var client = &http.Client{}

func (server *SimpleHTTPServer) ExportYamlDoc(exportFilename string, ciPath string, override bool) error {
	if override == false {
		if _, err := os.Stat(exportFilename); !os.IsNotExist(err) {
			return fmt.Errorf("file `%s` already exists. Use -o flag to overwrite it.", exportFilename)
		}
	}

	response, err := server.doRequest("GET", ciPath, "", nil)
	if err != nil {
		return err
	}
	defer response.Body.Close()

	tempArchive, err := ioutil.TempFile(os.TempDir(), "tempArchive")
	if err != nil {
		return err
	}
	defer os.Remove(tempArchive.Name())

	_, err = io.Copy(tempArchive, response.Body)
	if err != nil {
		return err
	}

	tempArchivePath, err := filepath.Abs(tempArchive.Name())
	if err != nil {
		return err
	}

	indexFilePath, err := filepath.Abs(exportFilename)
	if err != nil {
		return err
	}

	destinationDir := filepath.Dir(indexFilePath)
	err = archiver.Zip.Open(tempArchivePath, destinationDir)
	err = os.Rename(filepath.Join(destinationDir, "index.yaml"), filepath.Join(destinationDir, filepath.Base(exportFilename)))
	if err != nil {
		return err
	}

	return nil
}

func (server *SimpleHTTPServer) PostYamlDoc(resource string, yamlDocBytes []byte) error {
	_, err := server.doRequest("POST", resource, "text/vnd.yaml", bytes.NewReader(yamlDocBytes))
	return err
}

func (server *SimpleHTTPServer) PostYamlZip(resource string, yamlZipFilename string) error {
	f, err := os.Open(yamlZipFilename)
	if err != nil {
		return err
	}

	defer f.Close()

	_, err2 := server.doRequest("POST", resource, "application/zip", f)
	return err2
}

func (server *SimpleHTTPServer) doRequest(method string, path string, contentType string, body io.Reader) (*http.Response, error) {

	maybeSlash := ""
	if !strings.HasSuffix(server.Url.String(), "/") {
		maybeSlash = "/"
	}
	theUrl := server.Url.String() + maybeSlash + path

	request, err := http.NewRequest(method, theUrl, body)
	if err != nil {
		return nil, err
	}

	if contentType != "" {
		request.Header.Set("Content-Type", contentType)
	}
	request.SetBasicAuth(server.Username, server.Password)
	response, err := client.Do(request)

	if err != nil {
		return nil, err
	}

	if response.StatusCode == 401 {
		return nil, fmt.Errorf("401 Request unauthorized. Please check your credentials.")
	} else if response.StatusCode == 403 {
		return nil, fmt.Errorf("403 Request forbidden. Please check your permissions on the server.")
	} else if response.StatusCode == 404 {
		return nil, fmt.Errorf("404 Not found. Please specify the correct url.")
	} else if response.StatusCode >= 400 {
		bodyText, _ := ioutil.ReadAll(response.Body)
		return nil, fmt.Errorf("%d unexpected response: %s", response.StatusCode, string(bodyText))
	}

	return response, nil
}

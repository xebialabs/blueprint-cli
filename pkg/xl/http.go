package xl

import (
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
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

func (server *SimpleHTTPServer) ExportYamlDoc(exportFilename string, path string, override bool) error {
	if override == false {
		if _, err := os.Stat(exportFilename); !os.IsNotExist(err) {
			return fmt.Errorf("file `%s` already exists", exportFilename)
		}
	}

	response, err := server.doRequest("GET", path, "", nil)
	if err != nil {
		return err
	}
	defer response.Body.Close()

	// writing file
	outFile, err := os.Create(exportFilename)
	if err != nil {
		return err
	}
	defer outFile.Close()

	_, err = io.Copy(outFile, response.Body)

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
		return nil, fmt.Errorf("401 request unauthorized. did you configure the correct credentials?")
	} else if response.StatusCode == 403 {
		return nil, fmt.Errorf("403 request forbidden. do you have the correct permissions?")
	} else if response.StatusCode == 404 {
		return nil, fmt.Errorf("404 not found, did you specify the correct url/path?")
	} else if response.StatusCode >= 400 {
		bodyText, _ := ioutil.ReadAll(response.Body)
		return nil, fmt.Errorf("%d unexpected response: %s", response.StatusCode, string(bodyText))
	} else {
		Verbose("Response status %s\n", response.Status)
	}

	return response, nil
}

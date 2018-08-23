package lib

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
}

type SimpleHTTPServer struct {
	Url      url.URL
	Username string
	Password string
}

var client = &http.Client{}

func (server *SimpleHTTPServer) PostYamlDoc(resource string, yamlDocBytes []byte) error {
	return server.post(resource, "text/vnd.yaml", bytes.NewReader(yamlDocBytes))
}

func (server *SimpleHTTPServer) PostYamlZip(resource string, yamlZipFilename string) error {
	f, err := os.Open(yamlZipFilename)
	if err != nil {
		return err
	}

	defer f.Close()

	return server.post(resource, "application/zip", f)
}

func (server *SimpleHTTPServer) post(resource string, contentType string, body io.Reader) error {
	maybeSlash := ""
	if !strings.HasSuffix(server.Url.String(), "/") {
		maybeSlash = "/"
	}
	theUrl := server.Url.String() + maybeSlash + resource
	request, err := http.NewRequest("POST", theUrl, body)
	if err != nil {
		return err
	}

	request.Header.Set("Content-Type", contentType)
	request.SetBasicAuth(server.Username, server.Password)
	response, err := client.Do(request)

	if err != nil {
		return err
	}

	if response.StatusCode == 401 {
		return fmt.Errorf("401 request unauthorized. did you configure the correct credentials?")
	} else if response.StatusCode == 403 {
		return fmt.Errorf("403 request forbidden. do you have the correct permissions?")
	} else if response.StatusCode >= 400 {
		bodyText, _ := ioutil.ReadAll(response.Body)
		return fmt.Errorf("%d unexpected response: %s", response.StatusCode, string(bodyText))
	} else {
		Verbose("Response status %s\n", response.Status)
	}

	return nil
}

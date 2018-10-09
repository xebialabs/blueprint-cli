package xl

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/mholt/archiver"
	"github.com/olekukonko/tablewriter"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
)

type HTTPServer interface {
	PostYamlDoc(path string, yamlDocBytes []byte) (*ChangedCis, error)
	PostYamlZip(path string, yamlZipFilename string) (*ChangedCis, error)
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

func processAsCodeResponse(response http.Response) *AsCodeResponse {
	var resp AsCodeResponse
	bodyText, _ := ioutil.ReadAll(response.Body)
	json.Unmarshal(bodyText, &resp)
	return &resp
}

func (server *SimpleHTTPServer) PostYamlDoc(resource string, yamlDocBytes []byte) (*ChangedCis, error) {
	response, err := server.doRequest("POST", resource, "text/vnd.yaml", bytes.NewReader(yamlDocBytes))
	if err != nil {
		return nil, err
	}
	var asCodeResponse = processAsCodeResponse(*response)
	return asCodeResponse.Cis, nil
}

func (server *SimpleHTTPServer) PostYamlZip(resource string, yamlZipFilename string) (*ChangedCis, error) {
	f, err := os.Open(yamlZipFilename)
	if err != nil {
		return nil, err
	}

	defer f.Close()

	response, err2 := server.doRequest("POST", resource, "application/zip", f)
	if err2 != nil {
		return nil, err2
	}
	var asCodeResponse = processAsCodeResponse(*response)
	return asCodeResponse.Cis, nil
}

func printTable(initialString string, header []string, data [][]string) error {
	var buf = bytes.NewBufferString(fmt.Sprintf("\n%s\n\n", initialString))
	var table = tablewriter.NewWriter(buf)
	table.SetBorder(false)
	table.SetHeader(header)
	table.AppendBulk(data)
	table.Render()
	return fmt.Errorf(buf.String())
}

func formatAsCodeError(response http.Response) error {
	var asCodeResponse = processAsCodeResponse(response)

	if asCodeResponse.Errors.Document != nil {
		return fmt.Errorf("\nMalformed YAML document.\nProblematic field: [%s], problem: %s", asCodeResponse.Errors.Document.Field, asCodeResponse.Errors.Document.Problem)
	}

	if asCodeResponse.Errors.Validation != nil && len(*asCodeResponse.Errors.Validation) > 0 {
		var data = make([][]string, len(*asCodeResponse.Errors.Validation))
		for i, validation := range *asCodeResponse.Errors.Validation {
			data[i] = []string{validation.CiId, validation.PropertyName, validation.Message}
		}
		return printTable("Validation failed for the following CIs:", []string{"ID", "Property", "Problem"}, data)
	}

	if asCodeResponse.Errors.Permission != nil && len(*asCodeResponse.Errors.Permission) > 0 {
		var data = make([][]string, len(*asCodeResponse.Errors.Permission))
		for i, permission := range *asCodeResponse.Errors.Permission {
			data[i] = []string{permission.CiId, permission.Permission}
		}
		return printTable("Following permissions are not granted to you:", []string{"ID", "Permission"}, data)
	}

	return fmt.Errorf("\nUnexpected response: %s", *asCodeResponse.Errors.Generic)
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
		return nil, formatAsCodeError(*response)
	}

	return response, nil
}

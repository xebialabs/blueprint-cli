package xl

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"

	"github.com/mholt/archiver"
	"github.com/olekukonko/tablewriter"
	"github.com/xebialabs/xl-cli/pkg/util"
)

type HTTPServer interface {
	TaskInfo(path string) (map[string]interface{}, error)
	PostYamlDoc(path string, yamlDocBytes []byte) (*Changes, error)
	PostYamlZip(path string, yamlZipFilename string) (*Changes, error)
	DownloadSchema(resource string) ([]byte, error)
	GenerateYamlDoc(generateFilename string, path string, override bool) error
}

type SimpleHTTPServer struct {
	Url      url.URL
	Username string
	Password string
}

var client = &http.Client{}

func buildHeaders(contentType string, acceptType string) map[string]string {
	result := make(map[string]string)
	if contentType != "" {
		result["Content-Type"] = contentType
	}
	if acceptType != "" {
		result["Accept"] = acceptType
	}
	return result
}

func (server *SimpleHTTPServer) GenerateYamlDoc(generateFilename string, requestUrl string, override bool) error {
	if override == false {
		if _, err := os.Stat(generateFilename); !os.IsNotExist(err) {
			return fmt.Errorf("file `%s` already exists. Use -o flag to overwrite it.", generateFilename)
		}
	}

	response, err := server.doRequest("GET", requestUrl, nil, nil)
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

	indexFilePath, err := filepath.Abs(generateFilename)
	if err != nil {
		return err
	}

	destinationDir := filepath.Dir(indexFilePath)
	err = archiver.Zip.Open(tempArchivePath, destinationDir)
	err = os.Rename(filepath.Join(destinationDir, "index.yaml"), filepath.Join(destinationDir, filepath.Base(generateFilename)))
	if err != nil {
		return err
	}

	return nil
}

func processAsCodeResponse(response http.Response) (*AsCodeResponse, error) {
	var resp AsCodeResponse
	bodyText, err := ioutil.ReadAll(response.Body)
	resp.RawBody = string(bodyText)
	if err != nil {
		return nil, err
	}
	uerr := json.Unmarshal(bodyText, &resp)
	if uerr != nil {
		return nil, uerr
	}
	return &resp, nil
}

func (server *SimpleHTTPServer) DownloadSchema(resource string) ([]byte, error) {
	response, err := server.doRequest("GET", resource, buildHeaders("", "application/json"), nil)
	if err != nil {
		return nil, err
	}
	bodyText, err := ioutil.ReadAll(response.Body)
	if err != nil {
		return nil, err
	}
	return bodyText, nil
}

func (server *SimpleHTTPServer) TaskInfo(resource string) (map[string]interface{}, error) {
	response, err := server.doRequest("GET", resource, buildHeaders("application/json", "application/json"), nil)
	if err != nil {
		return nil, err
	}
	var js map[string]interface{}
	bodyText, err := ioutil.ReadAll(response.Body)
	if err != nil {
		return nil, err
	}
	uerr := json.Unmarshal(bodyText, &js)
	if uerr != nil {
		return nil, uerr
	}
	return js, nil
}

func (server *SimpleHTTPServer) PostYamlDoc(resource string, yamlDocBytes []byte) (*Changes, error) {
	response, err := server.doRequest("POST", resource, buildHeaders("text/vnd.yaml", ""), bytes.NewReader(yamlDocBytes))
	if err != nil {
		return nil, err
	}
	asCodeResponse, e := processAsCodeResponse(*response)
	if e != nil {
		return nil, e
	}
	return asCodeResponse.Changes, nil
}

func (server *SimpleHTTPServer) PostYamlZip(resource string, yamlZipFilename string) (*Changes, error) {
	f, err := os.Open(yamlZipFilename)
	if err != nil {
		return nil, err
	}

	defer f.Close()

	response, err2 := server.doRequest("POST", resource, buildHeaders("application/zip", ""), f)
	if err2 != nil {
		return nil, err2
	}
	asCodeResponse, e := processAsCodeResponse(*response)
	if e != nil {
		return nil, e
	}
	return asCodeResponse.Changes, nil
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
	asCodeResponse, e := processAsCodeResponse(response)
	if e != nil {
		return e
	}

	if asCodeResponse.Errors == nil {
		util.Verbose("Unexpected response: %s \n", asCodeResponse.RawBody)
		return fmt.Errorf("Unexpected server problem. Please contact your system administrator. Run with verbose flag for more details")
	}

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

func (server *SimpleHTTPServer) doRequest(method string, path string, headers map[string]string, body io.Reader) (*http.Response, error) {
	maybeSlash := ""
	if !strings.HasSuffix(server.Url.String(), "/") {
		maybeSlash = "/"
	}
	theUrl := server.Url.String() + maybeSlash + path

	request, err := http.NewRequest(method, theUrl, body)
	if err != nil {
		return nil, err
	}

	if headers != nil {
		for header, value := range headers {
			request.Header.Set(header, value)
		}
	}

	request.SetBasicAuth(server.Username, server.Password)
	response, err := client.Do(request)

	if err != nil {
		return nil, err
	}

	if response.StatusCode == 401 {
		return nil, fmt.Errorf("401 Request unauthorized. Please check your credentials.")
	} else if response.StatusCode == 402 {
		baseURL := server.Url.Scheme + "://" + server.Url.Host
		return nil, fmt.Errorf("402 License invalid. Please renew you license at %s/productregistration ", baseURL)
	} else if response.StatusCode == 403 {
		return nil, fmt.Errorf("403 Request forbidden. Please check your permissions on the server.")
	} else if response.StatusCode == 404 {
		return nil, fmt.Errorf("404 Not found. Please specify the correct url.")
	} else if response.StatusCode >= 400 {
		return nil, formatAsCodeError(*response)
	}

	return response, nil
}

func translateHTTPStatusCodeErrors(statusCode int, urlVal string) error {
	switch {
	case statusCode == 401:
		return fmt.Errorf("401 Request unauthorized. Please check your credentials for %s", urlVal)
	case statusCode == 403:
		return fmt.Errorf("403 Request forbidden. Please check your permissions for %s", urlVal)
	case statusCode == 404:
		return fmt.Errorf("404 Not found. Please specify the correct url. Provided was %s", urlVal)
	case statusCode >= 400:
		return fmt.Errorf("error: StatusCode %d for URL %s", statusCode, urlVal)
	}
	return nil
}

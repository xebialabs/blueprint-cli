package xl

import (
	"archive/zip"
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/xebialabs/xl-cli/pkg/auth"
	"github.com/xebialabs/xl-cli/pkg/models"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"

	"bufio"
	"github.com/mholt/archiver"
	"github.com/olekukonko/tablewriter"
	"github.com/xebialabs/xl-cli/pkg/util"
	"time"
)

type HTTPServer interface {
	TaskInfo(path string) (map[string]interface{}, error)
	ApplyYamlDoc(path string, yamlDocBytes []byte, scmInfo *SCMInfo) (*Changes, error)
	ApplyYamlZip(path string, yamlZipFilename string, scmInfo *SCMInfo) (*Changes, error)
	PreviewYamlDoc(path string, yamlDocBytes []byte) (*models.PreviewResponse, error)
	DownloadSchema(resource string) ([]byte, error)
	GenerateYamlDoc(generateFilename string, path string, override bool) error
}

type AuthType int

type SimpleHTTPServer struct {
	Url        url.URL
	AuthMethod string
	Username   string
	Password   string
	Product    models.Product
}

var client = &http.Client{}

func firstLine(s string) string {
	if s == "" {
		return ""
	}
	scanner := bufio.NewScanner(strings.NewReader(s))
	scanner.Scan()
	line := scanner.Text()
	if scanner.Err() != nil {
		return ""
	} else {
		return line
	}
}

func buildHeaders(contentType string, acceptType string, scmInfo *SCMInfo) map[string]string {
	result := make(map[string]string)
	if contentType != "" {
		result["Content-Type"] = contentType
	}
	if acceptType != "" {
		result["Accept"] = acceptType
	}

	if scmInfo != nil {
		if scmInfo.scmType != "" {
			result["X-Xebialabs-Scm-Type"] = string(scmInfo.scmType)
		}
		if scmInfo.filename != "" {
			result["X-Xebialabs-Scm-Filename"] = scmInfo.filename
		}
		if scmInfo.commit != "" {
			result["X-Xebialabs-Scm-Commit"] = scmInfo.commit
		}
		if scmInfo.author != "" {
			result["X-Xebialabs-Scm-Author"] = scmInfo.author
		}
		if scmInfo.date != (time.Time{}) {
			result["X-Xebialabs-Scm-Date"] = scmInfo.date.UTC().Format(time.RFC3339)
		}
		if scmInfo.message != "" {
			result["X-Xebialabs-Scm-Message"] = firstLine(scmInfo.message) // Todo sanitize more? base64?
		}
		if scmInfo.remote != "" {
			result["X-Xebialabs-Scm-Remote"] = scmInfo.remote
		}
	}

	return result
}

func checkForEmptyFile(filePath string) error {
	if _, err := os.Stat(filePath); !os.IsNotExist(err) {
		return fmt.Errorf("file `%s` already exists. Use -o flag to overwrite it.", filePath)
	}

	return nil
}

func (server *SimpleHTTPServer) GenerateYamlDoc(generateFilename string, requestUrl string, override bool) error {
	if override == false {
		err := checkForEmptyFile(generateFilename)
		if err != nil {
			return err
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
	if override == false {
		err := archiver.DefaultZip.Walk(tempArchivePath, func(f archiver.File) error {
			header := f.Header.(zip.FileHeader)
			return checkForEmptyFile(filepath.Join(destinationDir, header.Name))
		})

		if err != nil {
			return err
		}
	}
	archiver.DefaultZip.OverwriteExisting = override
	err = archiver.DefaultZip.Unarchive(tempArchivePath, destinationDir)
	if err != nil {
		return err
	}
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

func processPreviewResponse(response http.Response) (*models.PreviewResponse, error) {
	var resp models.PreviewResponse
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
	response, err := server.doRequest("GET", resource, buildHeaders("", "application/json", nil), nil)
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
	response, err := server.doRequest("GET", resource, buildHeaders("application/json", "application/json", nil), nil)
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

func (server *SimpleHTTPServer) ApplyYamlDoc(resource string, yamlDocBytes []byte, scmInfo *SCMInfo) (*Changes, error) {
	response, err := server.doRequest("POST", resource, buildHeaders("text/vnd.yaml", "", scmInfo), bytes.NewReader(yamlDocBytes))
	if err != nil {
		return nil, err
	}
	asCodeResponse, e := processAsCodeResponse(*response)
	if e != nil {
		return nil, e
	}

	if asCodeResponse.Changes != nil {
		asCodeResponse.Changes.Server = server
	}

	return asCodeResponse.Changes, nil
}

func (server *SimpleHTTPServer) ApplyYamlZip(resource string, yamlZipFilename string, scmInfo *SCMInfo) (*Changes, error) {
	f, err := os.Open(yamlZipFilename)
	if err != nil {
		return nil, err
	}

	defer f.Close()

	response, err2 := server.doRequest("POST", resource, buildHeaders("application/zip", "", scmInfo), f)
	if err2 != nil {
		return nil, err2
	}
	asCodeResponse, e := processAsCodeResponse(*response)
	if e != nil {
		return nil, e
	}

	if asCodeResponse.Changes != nil {
		asCodeResponse.Changes.Server = server
	}

	return asCodeResponse.Changes, nil
}

func (server *SimpleHTTPServer) PreviewYamlDoc(resource string, yamlDocBytes []byte) (*models.PreviewResponse, error) {
	response, err := server.doRequest("POST", resource, buildHeaders("text/vnd.yaml", "", nil), bytes.NewReader(yamlDocBytes))
	if err != nil {
		return nil, err
	}
	previewResponse, e := processPreviewResponse(*response)
	if e != nil {
		return nil, e
	}
	return previewResponse, nil
}

func printTable(initialString string, header []string, data [][]string) error {
	var buf = bytes.NewBufferString(fmt.Sprintf("\n%s%s\n\n%s", util.Indent1(), initialString, util.Indent1()))
	var table = tablewriter.NewWriter(buf)
	table.SetBorder(false)
	table.SetHeader(header)
	table.AppendBulk(data)
	table.SetNewLine(fmt.Sprintf("\n%s", util.Indent1()))
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

	genericError := "Unknown. Please check server logs."
	if asCodeResponse.Errors.Generic != nil {
	   genericError = *asCodeResponse.Errors.Generic
    }

	return fmt.Errorf("\nUnexpected response: %s", genericError)
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

	err = auth.Authenticate(request, server.Product, server.AuthMethod, server.Url.String(), server.Username, server.Password)
	if err != nil {
		return nil, err
	}

	if headers != nil {
		for header, value := range headers {
			request.Header.Set(header, value)
		}
	}

	response, err := client.Do(request)

	if err != nil {
		return nil, err
	}

	return handleServerResponse(response, server)
}

func handleServerResponse(response *http.Response, server *SimpleHTTPServer) (*http.Response, error) {
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

package lib

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"strings"
)

type HTTPServer interface {
	PostYaml(path string, body []byte) error
}

type SimpleHTTPServer struct {
	Url      url.URL
	Username string
	Password string
}

var client = &http.Client{}

func (server *SimpleHTTPServer) PostYaml(resource string, body []byte) error {
	buf := bytes.NewReader(body)

	maybeSlash := ""
	if !strings.HasSuffix(server.Url.String(), "/") {
		maybeSlash = "/"
	}
	theUrl := server.Url.String() + maybeSlash + resource
	request, err := http.NewRequest("POST", theUrl, buf)
	if err != nil {
		return err
	}

	request.Header.Set("Content-Type", "text/vnd.yaml")
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
	}

	return nil
}

package handle

import (
	"fmt"
	"github.com/xebialabs/xl-cli/internal/servers"
	"io/ioutil"
	"log"
	"net/http"
	"strings"
)

const (
	ContentTypeYaml = "text/vnd.yaml"
	MethodPost      = "POST"
)

func serverResponse(reqM string, resp *http.Response, err error) (string, string, error) {
	if err != nil {
		return "", "", fmt.Errorf("error sending %s request: %v", reqM, err)
	}

	defer func() {
		if resp != nil {
			err := resp.Body.Close()

			if err != nil {
				log.Println("error closing response body:", err)
			}
		}
	}()

	/*TODO: handle server response accordingly
	Remove log statements from this function and handle the return values in the calling functions,
	so this code can remain as an API
	*/
	if resp.StatusCode != 200 {
		body, err := ioutil.ReadAll(resp.Body)

		if err != nil {
			return "", "", fmt.Errorf("error reading response body: %v", err)
		}

		log.Println(resp.Status)
		log.Println(string(body))
		return resp.Status, string(body), nil
	}

	return "", "", nil
}

func NewServerRequest(srv *servers.Server, method string, body string, ctype string) (string, string, error) {
	resp, err := srv.NewRequest(method, strings.NewReader(body), ctype, len(body))
	return serverResponse(method, resp, err)
}

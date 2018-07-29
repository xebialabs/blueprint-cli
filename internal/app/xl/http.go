package xl

import (
	"fmt"
	"github.com/xebialabs/xl-cli/internal/platform/handle"
	"github.com/xebialabs/xl-cli/internal/servers"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"strings"
)

const (
	contentTypeYaml = "text/vnd.yaml"
	methodPost      = "POST"
)

func handleServerResponse(reqM string, resp *http.Response, err error) {
	//TODO: remove credentials from err
	handle.BasicError(fmt.Sprintf("error sending %s request", reqM), err)

	defer func() {
		if resp != nil {
			err := resp.Body.Close()

			if err != nil {
				log.Println("error closing response body:", err)
			}
		}
	}()

	if resp.StatusCode != 200 {
		body, err := ioutil.ReadAll(resp.Body)

		handle.BasicError("error reading response body", err)

		log.Println(resp.Status)
		log.Println(string(body))
	}
}

func newBasicServerRequest(srv *servers.Server, method string, body string, ctype string) {
	defer handle.BasicPanicLog()

	resp, err := srv.NewBasicRequest(method, strings.NewReader(body), ctype, len(body))

	handleServerResponse(method, resp, err)
}

func newBasicUrlRequest(u string, method string, body string, ctype string) {
	defer handle.BasicPanicLog()

	up, err := url.Parse(u)

	handle.BasicError("error parsing url", err)

	resp, err := servers.NewBasicUrlRequest(method, up, strings.NewReader(body), ctype, len(body))

	handleServerResponse(method, resp, err)
}

package xl

import (
	"github.com/xebialabs/xl-cli/internal/platform/handle"
	"github.com/xebialabs/xl-cli/internal/servers"
	"io/ioutil"
	"log"
	"strings"
)

func postStringUrlAuth(url string, body string, ctype string) {
	defer handle.BasicPanicAsLog()

	resp, err := servers.NewRequestUrlAuth("POST", url, strings.NewReader(body), ctype, len(body))

	handle.BasicError("error sending post request", err)

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

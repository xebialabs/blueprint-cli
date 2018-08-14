package lib

import (
	"encoding/base64"
	"github.com/stretchr/testify/assert"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
)

func TestHttp(t *testing.T) {
	t.Run("should post yaml with credentials and correct content type", func(t *testing.T) {

		handler := func(responseWriter http.ResponseWriter, request *http.Request) {
			// Useful for troubleshooting
			//requestDump, err := httputil.DumpRequest(request, true)
			//if err != nil {
			//	fmt.Println(err)
			//}
			//fmt.Println("REQUEST: \n" + string(requestDump))

			assert.Equal(t, "POST", request.Method)
			assert.Equal(t, "/ascode", request.URL.Path)
			assert.Equal(t, "text/vnd.yaml", request.Header.Get("Content-Type"))
			assert.Equal(t, "Basic "+base64.StdEncoding.EncodeToString([]byte("admin:admin")), request.Header.Get("Authorization"))
			body, err := ioutil.ReadAll(request.Body)
			assert.Nil(t, err)
			assert.Equal(t, "document body", string(body))
		}

		testServer := httptest.NewServer(http.HandlerFunc(handler))
		defer testServer.Close()

		res, _ := url.Parse(testServer.URL)
		server := SimpleHTTPServer{Url: *res, Username: "admin", Password: "admin"}

		error := server.PostYaml("ascode", []byte("document body"))
		assert.Nil(t, error)
	})
}

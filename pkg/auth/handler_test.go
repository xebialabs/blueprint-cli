package auth

import (
    "bytes"
    "fmt"
    "github.com/stretchr/testify/assert"
    "github.com/xebialabs/xl-cli/pkg/models"
    "net/http"
    "net/http/httptest"
    "testing"
)

func TestAuthHandler(t *testing.T) {
	XldToken := "ABC123"
	XlrToken := "XYZ987"

	t.Run("should authenticate to XLD using http", func(t *testing.T) {
		xldHandler := func(responseWriter http.ResponseWriter, request *http.Request) {
			assert.Equal(t, "POST", request.Method)
			assert.Equal(t, "/login", request.URL.Path)
			buf := new(bytes.Buffer)
			_, _ = buf.ReadFrom(request.Body)
			assert.Equal(t, "password=qwerty&username=admin", buf.String())

			assert.Equal(t, "application/x-www-form-urlencoded", request.Header.Get("Content-Type"))
			responseWriter.Header().Add("Set-Cookie", fmt.Sprintf("%s=%s", models.XLD_LOGIN_TOKEN, XldToken))
			_, _ = responseWriter.Write([]byte("{}"))
		}

		testServerXld := httptest.NewServer(http.HandlerFunc(xldHandler))
		defer testServerXld.Close()

		request, _ := http.NewRequest("POST", testServerXld.URL, nil)
		err := Authenticate(request, models.XLD, models.AuthMethodHttp, testServerXld.URL, "admin", "qwerty")
		assert.Nil(t, err)
		assert.Len(t, request.Header, 1)
		assert.Len(t, request.Cookies(), 1)
		authCookies := request.Cookies()[0]
		assert.Equal(t, models.XLD_LOGIN_TOKEN, authCookies.Name)
		assert.Equal(t, XldToken, authCookies.Value)

		assert.Len(t, Sessions, 1)
		xldSession := Sessions[models.XLD]
		assert.Equal(t, XldToken, xldSession.Token)
		assert.Equal(t, testServerXld.URL, xldSession.ServerUrl)
		delete(Sessions, models.XLD)
	})

	t.Run("should authenticate to XLD using basic", func(t *testing.T) {
		var requests []*http.Request
		xldHandler := func(responseWriter http.ResponseWriter, request *http.Request) {
			requests = append(requests, request)
			_, _ = responseWriter.Write([]byte("{}"))
		}

		testServerXld := httptest.NewServer(http.HandlerFunc(xldHandler))
		defer testServerXld.Close()

		request, _ := http.NewRequest("POST", testServerXld.URL, nil)
		err := Authenticate(request, models.XLD, models.AuthMethodBasic, testServerXld.URL, "admin", "qwerty")
		assert.Nil(t, err)
		assert.Empty(t, requests)
		assert.Len(t, request.Cookies(), 0)
		assert.Len(t, request.Header, 1)
		authHeader := request.Header.Get("Authorization")
		assert.Equal(t, "Basic YWRtaW46cXdlcnR5", authHeader)

		assert.Len(t, Sessions, 0)
	})

	t.Run("should authenticate to XLR using http", func(t *testing.T) {
		xlrHandler := func(responseWriter http.ResponseWriter, request *http.Request) {
			assert.Equal(t, "POST", request.Method)
			assert.Equal(t, "/login", request.URL.Path)
            buf := new(bytes.Buffer)
            _, _ = buf.ReadFrom(request.Body)
            assert.Equal(t, "password=qwerty&username=admin", buf.String())

            assert.Equal(t, "application/x-www-form-urlencoded", request.Header.Get("Content-Type"))
            responseWriter.Header().Add("Set-Cookie", fmt.Sprintf("%s=%s", models.XLR_LOGIN_TOKEN, XlrToken))
            _, _ = responseWriter.Write([]byte("{}"))
		}

		testServerXlr := httptest.NewServer(http.HandlerFunc(xlrHandler))
		defer testServerXlr.Close()

		request, _ := http.NewRequest("POST", testServerXlr.URL, nil)
		err := Authenticate(request, models.XLR, models.AuthMethodHttp, testServerXlr.URL, "admin", "qwerty")
		assert.Nil(t, err)
		assert.Len(t, request.Header, 1)
		assert.Len(t, request.Cookies(), 1)
		authCookies := request.Cookies()[0]
		assert.Equal(t, models.XLR_LOGIN_TOKEN, authCookies.Name)
		assert.Equal(t, XlrToken, authCookies.Value)

		assert.Len(t, Sessions, 1)
		xlrSession := Sessions[models.XLR]
		assert.Equal(t, XlrToken, xlrSession.Token)
		assert.Equal(t, testServerXlr.URL, xlrSession.ServerUrl)
		delete(Sessions, models.XLR)
	})

    t.Run("should authenticate to XLR using http with multiple Set-Cookie", func(t *testing.T) {
        xlrHandler := func(responseWriter http.ResponseWriter, request *http.Request) {
            assert.Equal(t, "POST", request.Method)
            assert.Equal(t, "/login", request.URL.Path)
            buf := new(bytes.Buffer)
            _, _ = buf.ReadFrom(request.Body)
            assert.Equal(t, "password=qwerty&username=admin", buf.String())

            assert.Equal(t, "application/x-www-form-urlencoded", request.Header.Get("Content-Type"))
            responseWriter.Header().Add("Set-Cookie", fmt.Sprintf("%s=%s", "RandomCookie", XlrToken))
            responseWriter.Header().Add("Set-Cookie", fmt.Sprintf("%s=%s", models.XLR_LOGIN_TOKEN, XlrToken))
            responseWriter.Header().Add("Set-Cookie", fmt.Sprintf("%s=%s", "AnotherOne", XlrToken))
            _, _ = responseWriter.Write([]byte("{}"))
        }

        testServerXlr := httptest.NewServer(http.HandlerFunc(xlrHandler))
        defer testServerXlr.Close()

        request, _ := http.NewRequest("POST", testServerXlr.URL, nil)
        err := Authenticate(request, models.XLR, models.AuthMethodHttp, testServerXlr.URL, "admin", "qwerty")
        assert.Nil(t, err)
        assert.Len(t, request.Header, 1)
        assert.Len(t, request.Cookies(), 1)
        authCookies := request.Cookies()[0]
        assert.Equal(t, models.XLR_LOGIN_TOKEN, authCookies.Name)
        assert.Equal(t, XlrToken, authCookies.Value)

        assert.Len(t, Sessions, 1)
        xlrSession := Sessions[models.XLR]
        assert.Equal(t, XlrToken, xlrSession.Token)
        assert.Equal(t, testServerXlr.URL, xlrSession.ServerUrl)
        delete(Sessions, models.XLR)
    })

	t.Run("should authenticate to XLR using basic", func(t *testing.T) {
		var requests []*http.Request
		xlrHandler := func(responseWriter http.ResponseWriter, request *http.Request) {
			requests = append(requests, request)
			_, _ = responseWriter.Write([]byte("{}"))
		}

		testServerXlr := httptest.NewServer(http.HandlerFunc(xlrHandler))
		defer testServerXlr.Close()

		request, _ := http.NewRequest("POST", testServerXlr.URL, nil)
		err := Authenticate(request, models.XLR, models.AuthMethodBasic, testServerXlr.URL, "admin", "qwerty")
		assert.Nil(t, err)
		assert.Empty(t, requests)
		assert.Len(t, request.Cookies(), 0)
		assert.Len(t, request.Header, 1)
		authHeader := request.Header.Get("Authorization")
		assert.Equal(t, "Basic YWRtaW46cXdlcnR5", authHeader)

		assert.Len(t, Sessions, 0)
	})
}

package auth

import (
    "github.com/xebialabs/xl-cli/pkg/models"
    "gopkg.in/errgo.v2/fmt/errors"
    "net/http"
    "net/url"
    "strings"
)

type session struct {
	Token     string
	ServerUrl string
}

var Sessions = map[models.Product]*session{}

func getToken(product models.Product) *string {
	if auth, authExists := Sessions[product]; authExists {
		return &auth.Token
	}
	return nil
}

func setSession(product models.Product, serverUrl string, token string) {
	Sessions[product] = &session{Token: token, ServerUrl: serverUrl}
}

type authModel struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

// disable redirects because when using oidc the client gets lost following links
var client = &http.Client{
    CheckRedirect: func(req *http.Request, via []*http.Request) error {
        return http.ErrUseLastResponse
    },
}

func doLogin(request *http.Request, cookieName string) (*string, error) {
	resp, err := client.Do(request)

	if err != nil {
		return nil, err
	}

	if resp.StatusCode >= 400 {
		return nil, errors.Newf("auth returned %d http code on login.", resp.StatusCode)
	}

	parsed, err := url.ParseQuery(resp.Header.Get("Set-Cookie"))

	if err != nil {
		return nil, err
	}

	token := parsed.Get(cookieName)

	return &token, nil
}

func getEnding(url string) string {
	if !strings.HasSuffix(url, "/") {
		return "/"
	} else {
		return ""
	}
}

func Logout() {
	for product, authentication := range Sessions {
		var logoutPath = authentication.ServerUrl + getEnding(authentication.ServerUrl) + "logout"
		logoutRequest, err := http.NewRequest("GET", logoutPath, nil)

		if err != nil {
			continue
		}

		cookie := http.Cookie{
			Name:  getProductCookieName(product),
			Value: authentication.Token,
		}

		logoutRequest.AddCookie(&cookie)

		_, _ = client.Do(logoutRequest)
	}
}

func getProductCookieName(product models.Product) string {
	if product == models.XLD {
		return models.XLD_LOGIN_TOKEN
	} else {
		return models.XLR_LOGIN_TOKEN
	}
}

func createLoginRequest(loginPath string, username string, password string) (*http.Request, error) {
	request, err := http.NewRequest("POST", loginPath, strings.NewReader(
		url.Values{"username": {username}, "password": {password}}.Encode()),
	)

	if err != nil {
		return nil, err
	}

	request.Header.Add("Content-Type", "application/x-www-form-urlencoded")
	return request, nil
}

func login(product models.Product, serverUrl string, username string, password string) (*string, error) {
	maybeSlash := getEnding(serverUrl)
	var loginPath = serverUrl + maybeSlash + "login"

	loginRequest := (*http.Request)(nil)
	err := (error)(nil)
	loginToken := ""

    loginRequest, err = createLoginRequest(loginPath, username, password)
    if product == models.XLR {
		loginToken = models.XLR_LOGIN_TOKEN
	} else {
		loginToken = models.XLD_LOGIN_TOKEN
	}
	if err != nil {
		return nil, err
	}

	return doLogin(loginRequest, loginToken)
}

func loginIfNeeded(product models.Product, authMethod string, url string, username string, password string) error {
	if authMethod == models.AuthMethodHttp {
		if getToken(product) == nil {
			token, err := login(product, url, username, password)
			if err != nil {
				return err
			}
			setSession(product, url, *token)
		}
	}
	return nil
}

func addAuthInformation(request *http.Request, product models.Product, authMethod string, username string, password string) {
	if authMethod == models.AuthMethodBasic {
		request.SetBasicAuth(username, password)
	} else if authMethod == models.AuthMethodHttp {
		cookie := http.Cookie{
			Name:  getProductCookieName(product),
			Value: *getToken(product),
		}
		request.AddCookie(&cookie)
	}
}

func Authenticate(request *http.Request, product models.Product, authMethod string, url string, username string, password string) error {
	err := loginIfNeeded(product, authMethod, url, username, password)
	if err != nil {
		return err
	}
	addAuthInformation(request, product, authMethod, username, password)
	return nil
}

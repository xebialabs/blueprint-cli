package servers

import (
	"fmt"
	"io"
	"net/http"
	"strconv"
)

type Server struct {
	ApiVersion string
	Name       string
	Url        string
	Username   string
	password   string
}

var servers = make(map[string]Server)

func init() {
	xld := Server{
		ApiVersion: "xl-deploy/v1",
		Name:       "XL Deploy",
		Url:        "http://admin:admin@localhost:4516/deployit/ascode",
		Username:   "admin",
		password:   "admin",
	}

	xlda := Server{
		ApiVersion: "xl-deploy/v1alpha1",
		Name:       "XL Deploy",
		Url:        "http://admin:admin@localhost:4516/deployit/ascode",
		Username:   "admin",
		password:   "admin",
	}

	xlr := Server{
		ApiVersion: "xl-release/v1",
		Name:       "XL Release",
		Url:        "http://admin:admin@localhost:5516/ascode",
		Username:   "admin",
		password:   "admin",
	}

	xlra := Server{
		ApiVersion: "xl-release/v1alpha1",
		Name:       "XL Release",
		Url:        "http://admin:admin@localhost:5516/ascode",
		Username:   "admin",
		password:   "admin",
	}

	servers[xld.ApiVersion] = xld
	servers[xlr.ApiVersion] = xlr
	servers[xlda.ApiVersion] = xlda
	servers[xlra.ApiVersion] = xlra
}

func FromApiVersion(api string) (*Server, error) {
	if s, exists := servers[api]; exists {
		return &s, nil
	} else {
		return &Server{}, fmt.Errorf("no server found for apiVersion %v", api)
	}
}

func NewRequestUrlAuth(method string, url string, body io.Reader, ctype string, clen int) (*http.Response, error) {
	client := &http.Client{}
	req, err := http.NewRequest(method, url, body)

	if err != nil {
		return &http.Response{}, err
	}

	req.Header.Add("Content-Type", ctype)
	req.Header.Add("Content-Length", strconv.Itoa(clen))
	resp, err := client.Do(req)

	if err != nil {
		return &http.Response{}, err
	}

	return resp, nil
}

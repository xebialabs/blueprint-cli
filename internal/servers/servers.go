package servers

import (
	"fmt"
	"github.com/spf13/viper"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

const (
	SrvCfgKey        = "xl.servers"
	XldId            = "xl-deploy"
	XlrId            = "xl-release"
	XldAppHomeDirKey = "Applications-home"
	XldCfgHomeDirKey = "Configuration-home"
	XldEnvHomeDirKey = "Environments-home"
	XldInfHomeDirKey = "Infrastructure-home"
	XlrHomeDirKey    = "home"
)

type Server struct {
	Name        string
	Type        string
	Host        string
	Port        int
	Username    string
	Password    string
	Ssl         bool
	ContextRoot string `mapstructure:"context"`
	HomeDir     string
	Metadata    map[string]string
}

//serverConfig is the configuration struct of a server.
//This struct has one difference compared to the Server struct.
//It does not contain a Type property, because the type is available as a key in the config file.
//Removing Type excludes duplicate values in the config file.
type serverConfig struct {
	Name        string
	Host        string
	Port        int
	Username    string
	Password    string
	Ssl         bool
	ContextRoot string `mapstructure:"context"`
	HomeDir     string
	Metadata    map[string]string
}

var (
	apiCtx    map[string]string
	cfgLoaded bool
	servers   map[string]map[string]*Server
)

var (
	DefaultXld = Server{
		Name:     "default",
		Type:     XldId,
		Host:     "localhost",
		Port:     4516,
		Username: "admin",
		Password: "admin",
		Ssl:      false,
		Metadata: map[string]string{
			XldAppHomeDirKey: "Applications",
			XldCfgHomeDirKey: "Configuration",
			XldEnvHomeDirKey: "Environments",
			XldInfHomeDirKey: "Infrastructure",
		},
	}

	//TODO: change home directory config for XLR
	DefaultXlr = Server{
		Name:     "default",
		Type:     XlrId,
		Host:     "localhost",
		Port:     5516,
		Username: "admin",
		Password: "admin",
		Ssl:      false,
		Metadata: map[string]string{
			XlrHomeDirKey: "",
		},
	}
)

func init() {
	apiCtx = map[string]string{
		XldId: "deployit/ascode",
		XlrId: "ascode",
	}

	cfgLoaded = false
	servers = make(map[string]map[string]*Server)
}

func FromApiVersionAndName(apiV string, name string) (*Server, error) {
	if !cfgLoaded {
		if err := LoadConfig(SrvCfgKey); err != nil {
			return &Server{}, err
		}
	}

	k := ParseApiVersion(apiV)

	if sm, exists := servers[k]; exists {
		if s, found := sm[name]; found {
			return s, nil
		}
	}

	return &Server{}, fmt.Errorf("no server found for apiVersion %s with name %s. Configure a new server with the login command", apiV, name)
}

func LoadConfig(cfgKey string) error {
	srvs := make(map[string][]*Server)
	err := viper.UnmarshalKey(cfgKey, &srvs)

	if err == nil {
		cfgLoaded = true

		for t, ss := range srvs {
			for _, s := range ss {
				s.Type = t
				servers[t] = map[string]*Server{
					s.Name: s,
				}
			}
		}

		return nil
	}

	return fmt.Errorf("error loading servers from config: %v", err)
}

func NewBasicUrlRequest(method string, url *url.URL, body io.Reader, ctype string, clen int) (*http.Response, error) {
	client := &http.Client{
		Timeout: 10 * time.Second,
	}

	req, err := http.NewRequest(method, url.String(), body)

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

func ParseApiVersion(apiV string) string {
	ks := strings.Split(apiV, "/")
	k := ""

	if len(ks) > 0 {
		k = ks[0]
	}

	return k
}

func (s *Server) Endpoint() string {
	if s.Port > 0 {
		return fmt.Sprintf("%s:%v", s.Host, s.Port)
	} else {
		return s.Host
	}
}

func (s *Server) NewBasicRequest(method string, body io.Reader, ctype string, clen int) (*http.Response, error) {
	u := &url.URL{
		Scheme: s.Scheme(),
		User:   url.UserPassword(s.Username, s.Password),
		Host:   s.Endpoint(),
		Path:   s.Pathname(),
	}

	return NewBasicUrlRequest(method, u, body, ctype, clen)
}

func (s *Server) Pathname() string {
	p := ""

	if s.ContextRoot != "" {
		p += fmt.Sprintf("%s/", s.ContextRoot)
	}

	if a, exists := apiCtx[s.Type]; exists {
		p += a
	}

	return p
}

func (s *Server) Save() error {
	if !cfgLoaded {
		if err := LoadConfig(SrvCfgKey); err != nil {
			return err
		}
	}

	if srv, exist := servers[s.Type]; exist {
		srv[s.Name] = s
	} else {
		srv = map[string]*Server{
			s.Name: s,
		}

		servers[s.Type] = srv
	}

	scm := make(map[string][]*serverConfig)

	for t, sm := range servers {
		for _, v := range sm {
			srvC := &serverConfig{
				Name:        v.Name,
				Host:        v.Host,
				Port:        v.Port,
				Username:    v.Username,
				Password:    v.Password,
				Ssl:         v.Ssl,
				ContextRoot: v.ContextRoot,
				HomeDir:     v.HomeDir,
				Metadata:    v.Metadata,
			}

			scm[t] = append(scm[t], srvC)
		}
	}

	viper.Set(SrvCfgKey, scm)

	if err := viper.WriteConfig(); err != nil {
		if swcErr := viper.SafeWriteConfig(); swcErr != nil {
			return swcErr
		}
	}

	return nil
}

func (s *Server) Scheme() string {
	if s.Ssl {
		return "https"
	} else {
		return "http"
	}
}

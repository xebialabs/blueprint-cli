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
	"github.com/xebialabs/xl-cli/internal/platform/obfuscrypter"
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
	err := loadConfig()
	if err != nil {
		return nil, err
	}

	k := ParseApiVersion(apiV)

	if sm, exists := servers[k]; exists {
		if s, found := sm[name]; found {
			return s, nil
		}
	}

	return nil, fmt.Errorf("no server found for apiVersion %s with name %s. Configure a new server with the login command", apiV, name)
}


func (s *Server) AddToConfig() error {
	if !cfgLoaded {
		if _, err := loadConfigAndCheckDirty(); err != nil {
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

	err := saveConfig()
	if err != nil {
		return err
	}

	return nil
}

func loadConfig() (error) {
	dirty, err := loadConfigAndCheckDirty();
	if err != nil {
		return err;
	}

	if dirty {
		err := saveConfig()
		if err != nil {
			return err;
		}
	}

	return nil;
}

func loadConfigAndCheckDirty() (bool, error) {
	if !cfgLoaded {
		srvs := make(map[string][]*Server)

		err := viper.UnmarshalKey(SrvCfgKey, &srvs)
		if err != nil {
			return false, fmt.Errorf("error loading servers from config: %v", err)
		}

		cfgLoaded = true
		var dirty= false

		for t, ss := range srvs {
			for _, s := range ss {
				s.Type = t

				deobfuscryptedPassword, err := obfuscrypter.Deobfuscrypt(s.Password)
				if err != nil {
					dirty = true
				} else {
					s.Password = deobfuscryptedPassword
				}

				if servers[t] == nil {
					servers[t] = make(map[string]*Server)
				}
				servers[t][s.Name] = s
			}
		}

		return dirty, nil
	} else {
		return false, nil
	}
}

func saveConfig() error {
	scm := make(map[string][]*serverConfig)

	for t, sm := range servers {
		for _, v := range sm {
			obfuscryptedPassword, err := obfuscrypter.Obfuscrypt(v.Password)
			if err != nil {
				panic(fmt.Sprintf("Cannot encrypt password for %s server %s: %s", v.Type, v.Name, err))
			}

			srvC := &serverConfig{
				Name:        v.Name,
				Host:        v.Host,
				Port:        v.Port,
				Username:    v.Username,
				Password:    obfuscryptedPassword,
				Ssl:         v.Ssl,
				ContextRoot: v.ContextRoot,
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

func (s *Server) NewRequest(method string, body io.Reader, ctype string, clen int) (*http.Response, error) {
	url := &url.URL{
		Scheme: s.Scheme(),
		Host:   s.Endpoint(),
		Path:   s.Pathname(),
	}

	client := &http.Client{
		Timeout: 10 * time.Second,
	}

	req, err := http.NewRequest(method, url.String(), body)

	if err != nil {
		return &http.Response{}, err
	}

	req.SetBasicAuth(s.Username, s.Password)
	req.Header.Add("Content-Type", ctype)
	req.Header.Add("Content-Length", strconv.Itoa(clen))
	resp, err := client.Do(req)

	if err != nil {
		return &http.Response{}, err
	}

	return resp, nil
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

func (s *Server) Scheme() string {
	if s.Ssl {
		return "https"
	} else {
		return "http"
	}
}

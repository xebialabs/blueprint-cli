package login

import (
	"errors"
	"fmt"
	"github.com/xebialabs/xl-cli/internal/servers"
	"reflect"
	"testing"
	"github.com/stretchr/testify/mock"
	"strings"
	"github.com/spf13/viper"
)

const (
	succeed = "\u2713"
	failed  = "\u2717"
)

// Default, valid and invalid server input/flags values used for testing
const (
	defSrvName     = "default"
	vldSrvType1    = "1"
	vldSrvType2    = "2"
	vldSrvType3    = "xl-deploy"
	vldSrvType4    = "xl-release"
	invSrvType1    = "3"
	defSrvHost     = "localhost"
	defXldPort     = "4516"
	defXlrPort     = "5516"
	invSrvPort1    = "port"
	invSrvPort2    = "0"
	defSrvUsername = "admin"
	defSrvPassword = "admin"
	defSrvSsl      = "http"
	vldSrvSsl1     = "1"
	vldSrvSsl2     = "2"
	vldSrvSsl3     = "https"
	invSrvSsl1     = "3"
	vldContextRoot = "root"
)

func TestParseServerFlags(t *testing.T) {
	tests := []struct {
		name           string
		when           string
		srvName        string
		srvType        string
		srvHost        string
		srvPort        string
		srvUsername    string
		srvPassword    string
		srvSsl         string
		srvContextRoot string
		server         *servers.Server
	}{
		{"noInput", "no input", "", "", "", "", "",
			"", "", "",
			&servers.Server{
				Name:     "",
				Type:     "",
				Host:     "",
				Port:     0,
				Username: "",
				Password: "",
				Ssl:      false,
				Metadata: map[string]string{},
			}},
		{"validInput", "valid input with type 1 and ssl 1", defSrvName, vldSrvType1,
			defSrvHost, defXldPort, defSrvUsername, defSrvPassword,
			vldSrvSsl1, "",
			&servers.Server{
				Name:     defSrvName,
				Type:     vldSrvType3,
				Host:     defSrvHost,
				Port:     4516,
				Username: defSrvUsername,
				Password: defSrvPassword,
				Ssl:      false,
				Metadata: map[string]string{
					servers.XldAppHomeDirKey: "",
					servers.XldCfgHomeDirKey: "",
					servers.XldEnvHomeDirKey: "",
					servers.XldInfHomeDirKey: "",
				},
			}},
		{"invalidInput", "invalid input", defSrvName, invSrvType1, defSrvHost,
			invSrvPort1, defSrvUsername, defSrvPassword, invSrvSsl1,
			"",
			&servers.Server{
				Name:     defSrvName,
				Type:     "",
				Host:     defSrvHost,
				Port:     0,
				Username: defSrvUsername,
				Password: defSrvPassword,
				Ssl:      false,
				Metadata: map[string]string{},
			}},
	}

	t.Log("Given the need to test the parsing of login flags")
	{
		for i, tt := range tests {
			tf := func(at *testing.T) {
				at.Logf("\tTest: %d\tWhen receiving %s as flags", i, tt.when)
				{
					srv := ParseServerFlags(tt.srvName, tt.srvType, tt.srvHost, tt.srvPort, tt.srvUsername, tt.srvPassword, tt.srvSsl, tt.srvContextRoot, "", "", "", "", "")

					if reflect.DeepEqual(srv, tt.server) {
						at.Logf("\t%s\tShould return the expected server %#v", succeed, tt.server)
					} else {
						at.Errorf("\t%s\tShould return server ||%#v|| : ||%#v||", failed, tt.server, srv)
					}
				}
			}

			t.Run(tt.name, tf)
		}
	}
}

func TestParseServerInput(t *testing.T) {
	const (
		srvPropName        = "name"
		srvPropType        = "type"
		srvPropHost        = "host"
		srvPropPort        = "port"
		srvPropUsername    = "username"
		srvPropPassword    = "password"
		srvPropSsl         = "ssl"
		srvPropContextRoot = "contextRoot"
	)

	tests := []struct {
		name   string
		prop   string
		defVal string
		inp    string
		pErr   error
		srv    *servers.Server
	}{
		{"invalidProperty", "unknown", "", "",
			errors.New("unknown is not a valid property to be parsed for server"),
			&servers.Server{}},
		{"emptyNameAndDefault", srvPropName, "", "",
			errors.New("server name is required"),
			&servers.Server{
				Name: "",
			}},
		{"emptyNameWithInvalidDefault", srvPropName, " ", "",
			errors.New("server name is required; default value error: server name is required"),
			&servers.Server{
				Name: "",
			}},
		{"emptyNameWithDefault", srvPropName, defSrvName, "",
			nil,
			&servers.Server{
				Name: defSrvName,
			}},
		{"validName", srvPropName, "", defSrvName,
			nil,
			&servers.Server{
				Name: defSrvName,
			}},
		{"emptyTypeAndDefault", srvPropType, "", "",
			errors.New("server type is required"),
			&servers.Server{
				Type: "",
			}},
		{"emptyTypeWithDefault", srvPropType, vldSrvType2, "",
			errors.New("server type is required"),
			&servers.Server{
				Type: "",
			}},
		{"validType", srvPropType, "", vldSrvType2,
			nil,
			&servers.Server{
				Type: vldSrvType4,
			}},
		{"emptyHostAndDefault", srvPropHost, "", "",
			errors.New("server host is required"),
			&servers.Server{
				Host: "",
			}},
		{"emptyHostWithInvalidDefault", srvPropHost, " ", "",
			errors.New("server host is required; default value error: server host is required"),
			&servers.Server{
				Host: "",
			}},
		{"emptyHostWithDefault", srvPropHost, defSrvHost, "",
			nil,
			&servers.Server{
				Host: defSrvHost,
			}},
		{"validHost", srvPropHost, "", defSrvHost,
			nil,
			&servers.Server{
				Host: defSrvHost,
			}},
		{"emptyPortAndDefault", srvPropPort, "", "",
			errors.New("server port is required"),
			&servers.Server{
				Port: 0,
			}},
		{"invalidPortAndDefault", srvPropPort, invSrvPort2, invSrvPort1,
			errors.New("server port must be a number"),
			&servers.Server{
				Port: 0,
			}},
		{"emptyPortWithInvalidDefault", srvPropPort, invSrvPort2, "",
			errors.New("server port is required; default value error: server port range is not valid"),
			&servers.Server{
				Port: 0,
			}},
		{"emptyPortWithDefault", srvPropPort, defXldPort, "",
			nil,
			&servers.Server{
				Port: 4516,
			}},
		{"validPort", srvPropPort, "", defXlrPort,
			nil,
			&servers.Server{
				Port: 5516,
			}},
		{"emptyUsernameAndDefault", srvPropUsername, "", "",
			errors.New("username is required"),
			&servers.Server{
				Username: "",
			}},
		{"emptyUsernameWithInvalidDefault", srvPropUsername, " ", "",
			errors.New("username is required; default value error: username is required"),
			&servers.Server{
				Username: "",
			}},
		{"emptyUsernameWithDefault", srvPropUsername, defSrvUsername, "",
			nil,
			&servers.Server{
				Username: defSrvUsername,
			}},
		{"validUsername", srvPropUsername, "", defSrvUsername,
			nil,
			&servers.Server{
				Username: defSrvUsername,
			}},
		{"emptyPasswordAndDefault", srvPropPassword, "", "",
			errors.New("password is required"),
			&servers.Server{
				Password: "",
			}},
		{"emptyPasswordWithInvalidDefault", srvPropPassword, " ", "",
			errors.New("password is required; default value error: password is required"),
			&servers.Server{
				Password: "",
			}},
		{"emptyPasswordWithDefault", srvPropPassword, defSrvPassword, "",
			nil,
			&servers.Server{
				Password: defSrvPassword,
			}},
		{"validPassword", srvPropPassword, "", defSrvPassword,
			nil,
			&servers.Server{
				Password: defSrvPassword,
			}},
		{"emptySslAndDefault", srvPropSsl, "", "",
			errors.New("server protocol is required"),
			&servers.Server{
				Ssl: false,
			}},
		{"emptySslWithInvalidDefault", srvPropSsl, invSrvSsl1, "",
			errors.New("server protocol is required; default value error: server protocol must be http or https"),
			&servers.Server{
				Ssl: false,
			}},
		{"emptySslWithDefault", srvPropSsl, vldSrvSsl3, "",
			nil,
			&servers.Server{
				Ssl: true,
			}},
		{"validSsl", srvPropSsl, vldSrvSsl2, defSrvSsl,
			nil,
			&servers.Server{
				Ssl: false,
			}},
		{"emptyContextRootAndDefault", srvPropContextRoot, "", "",
			nil,
			&servers.Server{
				ContextRoot: "",
			}},
		{"emptyContextRootWithDefault", srvPropContextRoot, vldContextRoot, "",
			nil,
			&servers.Server{
				ContextRoot: vldContextRoot,
			}},
		{"validContextRoot", srvPropContextRoot, "", vldContextRoot,
			nil,
			&servers.Server{
				ContextRoot: vldContextRoot,
			}},
		{"whiteSpacesWithContextRoot", srvPropContextRoot, "", fmt.Sprintf(" %s ", vldContextRoot),
			nil,
			&servers.Server{
				ContextRoot: vldContextRoot,
			}},
	}

	t.Log("Given the need to test the parsing of server input")
	{
		for i, tt := range tests {
			tf := func(at *testing.T) {
				at.Logf("\tTest: %d\tWhen receiving %#v as input and %#v as default for %s property", i, tt.inp, tt.defVal, tt.prop)
				{
					srv := &servers.Server{}
					err := ParseServerInput(srv, tt.prop, tt.defVal, tt.inp)

					if reflect.DeepEqual(srv, tt.srv) {
						at.Logf("\t%s\tShould expect the server %#v", succeed, tt.srv)
					} else {
						at.Errorf("\t%s\tShould expect the server ||%#v|| : ||%#v||", failed, tt.srv, srv)
					}

					if reflect.DeepEqual(err, tt.pErr) {
						at.Logf("\t%s\tShould return the expected error \"%v\"", succeed, tt.pErr)
					} else {
						at.Errorf("\t%s\tShould return error \"%v\" : \"%v\"", failed, tt.pErr, err)
					}
				}
			}

			t.Run(tt.name, tf)
		}
	}
}

func TestScanServerProp(t *testing.T) {

	pwdReader = MockedPwdReader{}
	inputReader = strings.NewReader("How many times?")

	tests := []struct {
		name    string
		srv     *servers.Server
		prop    string
		defVal  string
		out     string
		freq    int
		secScan bool
		pErr    error
	}{
		{"Test 1", &servers.Server{}, "unknown", "", "", 1, false,
			errors.New("unknown is not a valid property to be parsed for server"),
		},
		{"Test 2", &servers.Server{}, "host", "localhost", "localhost", 1, false,
			nil,
		},
		{"Test 3", &servers.Server{}, "host", "localhost", "localhost", 1, true,
			nil,
		},
	}

	t.Log("Given the need to test the scanning of server prop")
	{
		for i, tt := range tests {
			tf := func(at *testing.T) {
				at.Logf("\tTest: %d\tWhen scanning %#v as property and %#v as default", i, tt.prop, tt.defVal)
				{
					err := ScanServerProp(tt.srv, tt.prop, tt.defVal, tt.out, tt.freq, tt.secScan)

					if reflect.DeepEqual(err, tt.pErr) {
						at.Logf("\t%s\tShould return the expected error \"%v\"", succeed, tt.pErr)
					} else {
						at.Errorf("\t%s\tShould return error \"%v\" : \"%v\"", failed, tt.pErr, err)
					}
				}
			}

			t.Run(tt.name, tf)
		}
	}
}

type MockedPwdReader struct {
	mock.Mock
}

func (MockedPwdReader) ReadPassword(fd int) ([]byte, error) {
	return make([]byte, 5), nil
}

func TestExecuteServer(t *testing.T) {

	pwdReader = MockedPwdReader{}
	inputReader = strings.NewReader("1")

	tests := []struct {
		name       string
		skipO      bool
		n          string
		t          string
		host       string
		p          string
		u          string
		pwd        string
		ssl        string
		ctx        string
		xldAppHome string
		xldCfgHome string
		xldEnvHome string
		xldInfHome string
		xlrHome string
		pErr       error
	}{
		{"Test with nothing predefined for xld", false, "", "1", "", "", "",
			"", "", "", "", "", "", "",
			"", viper.ConfigFileNotFoundError{},
		},
		{"Test with nothing predefined for xld and skipO", true, "", "1", "", "", "",
			"", "", "", "", "", "", "",
			"",  viper.ConfigFileNotFoundError{},
		},
		{"Test with everything defined for xld", false, "ascode", "1", "localhost", "21", "adsf",
			"afds", "1", "fda", "Applications", "Configuration", "Environments", "Infrastructure",
			"",  viper.ConfigFileNotFoundError{},
		},
		{"Test with nothing predefined for xlr", false, "", "2", "", "", "",
			"", "", "", "", "", "", "",
			"", viper.ConfigFileNotFoundError{},
		},
		{"Test with nothing predefined for xlr and skipO", true, "", "2", "", "", "",
			"", "", "", "", "", "", "",
			"", viper.ConfigFileNotFoundError{},
		},
		{"Test with everything defined for xlr", false, "fds", "2", "fa", "3", "as",
			"fdsa", "2", "dsaf", "", "", "", "",
			"fdsa", viper.ConfigFileNotFoundError{},
		},
	}

	t.Log("Given the need to test the scanning of server prop")
	{
		for i, tt := range tests {
			tf := func(at *testing.T) {
				at.Logf("\tTest: %d\tExecuting login command with default properties", i)
				{
					err := ExecuteServer(tt.skipO, tt.n, tt.t, tt.host, tt.p, tt.u, tt.pwd, tt.ssl, tt.ctx,
						tt.xldAppHome, tt.xldCfgHome, tt.xldEnvHome, tt.xldInfHome, tt.xlrHome)

					if reflect.TypeOf(err) == reflect.TypeOf(tt.pErr) {
						at.Logf("\t%s\tShould return the expected error \"%v\"", succeed, tt.pErr)
					} else {
						at.Errorf("\t%s\tShould return error \"%v\" : \"%v\"", failed, tt.pErr, err)
					}
				}
			}

			t.Run(tt.name, tf)
		}
	}
}

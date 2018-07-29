package login_test

import (
	"errors"
	"fmt"
	"github.com/xebialabs/xl-cli/internal/app/xl/login"
	"github.com/xebialabs/xl-cli/internal/servers"
	"reflect"
	"testing"
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
			}},
	}

	t.Log("Given the need to test the parsing of login flags")
	{
		for i, tt := range tests {
			tf := func(at *testing.T) {
				at.Logf("\tTest: %d\tWhen receiving %s as flags", i, tt.when)
				{
					srv := login.ParseServerFlags(tt.srvName, tt.srvType, tt.srvHost, tt.srvPort, tt.srvUsername, tt.srvPassword, tt.srvSsl, tt.srvContextRoot)

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
					err := login.ParseServerInput(srv, tt.prop, tt.defVal, tt.inp)

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

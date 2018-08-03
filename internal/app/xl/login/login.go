package login

import (
	"bufio"
	"fmt"
	"github.com/xebialabs/xl-cli/internal/servers"
	"golang.org/x/crypto/ssh/terminal"
	"os"
	"strconv"
	"strings"
	"syscall"
	"github.com/xebialabs/xl-cli/internal/platform/pwdreader"
	"io"
)

const (
	srvPropName                  = "name"
	srvPropType                  = "type"
	srvPropHost                  = "host"
	srvPropPort                  = "port"
	srvPropUsername              = "username"
	srvPropPassword              = "password"
	srvPropSsl                   = "ssl"
	srvPropContextRoot           = "contextRoot"
	srvPropApplicationsHomeXld   = "applicationsHomeXld"
	srvPropConfigurationHomeXld  = "configurationHomeXld"
	srvPropEnvironmentsHomeXld   = "environmentsHomeXld"
	srvPropInfrastructureHomeXld = "infrastructureHomeXld"
	srvPropXlrHome               = "xlrHome"
)

type TerminalPwdReader struct {
}

func (TerminalPwdReader) ReadPassword(fd int) ([]byte, error) {
	return terminal.ReadPassword(fd)
}

var pwdReader pwdreader.PasswordReader = TerminalPwdReader{}
var inputReader io.Reader = os.Stdin

func checkRequired(inp string, prop string) (string, error) {
	if strings.TrimSpace(inp) == "" {
		return "", fmt.Errorf("%s is required", prop)
	}

	return inp, nil
}

func ExecuteServer(skipO bool, n string, t string, host string, p string, u string, pwd string, ssl string, ctx string, xldAppHome string, xldCfgHome string, xldEnvHome string, xldInfHome string, xlrHome string) error {
	const freq = 3
	srv := ParseServerFlags(n, t, host, p, u, pwd, ssl, ctx, xldAppHome, xldCfgHome, xldEnvHome, xldInfHome, xlrHome)

	if srv.Type == "" {
		if err := ScanServerProp(srv, srvPropType, "", "Server type (1: xl-deploy or 2: xl-release)", freq, false); err != nil {
			return err
		}
	}

	var defSrv servers.Server

	if srv.Type == servers.XldId {
		defSrv = servers.DefaultXld
	} else {
		defSrv = servers.DefaultXlr
	}

	if srv.Name == "" {
		if err := ScanServerProp(srv, srvPropName, defSrv.Name, fmt.Sprintf("Server name (identifier) [%v]", defSrv.Name), freq, false); err != nil {
			return err
		}
	}

	if srv.Host == "" {
		if err := ScanServerProp(srv, srvPropHost, defSrv.Host, fmt.Sprintf("Server hostname or IP address [%v]", defSrv.Host), freq, false); err != nil {
			return err
		}
	}

	if p == "" {
		if err := ScanServerProp(srv, srvPropPort, strconv.Itoa(defSrv.Port), fmt.Sprintf("Server port [%v]", defSrv.Port), freq, false); err != nil {
			return err
		}
	}

	if srv.Username == "" {
		if err := ScanServerProp(srv, srvPropUsername, defSrv.Username, fmt.Sprintf("Username [%v]", defSrv.Username), freq, false); err != nil {
			return err
		}
	}

	if srv.Password == "" {
		if err := ScanServerProp(srv, srvPropPassword, "", "Password", freq, true); err != nil {
			return err
		}
	}

	if ssl == "" {
		var defSsl string

		if defSrv.Ssl {
			defSsl = "https"
		} else {
			defSsl = "http"
		}

		if err := ScanServerProp(srv, srvPropSsl, defSsl, fmt.Sprintf("Server protocol (1: http or 2: https) [%v]", defSsl), freq, false); err != nil {
			return err
		}
	}

	if !skipO && srv.ContextRoot == "" {
		if err := ScanServerProp(srv, srvPropContextRoot, "", "Application context root (optional)", freq, false); err != nil {
			return err
		}
	}

	if srv.Type == servers.XldId {
		if srv.Metadata[servers.XldAppHomeDirKey] == "" {
			if !skipO {
				if err := ScanServerProp(srv, srvPropApplicationsHomeXld, defSrv.Metadata[servers.XldAppHomeDirKey], fmt.Sprintf("Applications home directory for XL Deploy [%v]", defSrv.Metadata[servers.XldAppHomeDirKey]), freq, false); err != nil {
					return err
				}
			} else {
				srv.Metadata[servers.XldAppHomeDirKey] = defSrv.Metadata[servers.XldAppHomeDirKey]
			}
		}

		if srv.Metadata[servers.XldCfgHomeDirKey] == "" {
			if !skipO {
				if err := ScanServerProp(srv, srvPropConfigurationHomeXld, defSrv.Metadata[servers.XldCfgHomeDirKey], fmt.Sprintf("Configuration home directory for XL Deploy [%v]", defSrv.Metadata[servers.XldCfgHomeDirKey]), freq, false); err != nil {
					return err
				}
			} else {
				srv.Metadata[servers.XldCfgHomeDirKey] = defSrv.Metadata[servers.XldCfgHomeDirKey]
			}
		}

		if srv.Metadata[servers.XldEnvHomeDirKey] == "" {
			if !skipO {
				if err := ScanServerProp(srv, srvPropEnvironmentsHomeXld, defSrv.Metadata[servers.XldEnvHomeDirKey], fmt.Sprintf("Environments home directory for XL Deploy [%v]", defSrv.Metadata[servers.XldEnvHomeDirKey]), freq, false); err != nil {
					return err
				}
			} else {
				srv.Metadata[servers.XldEnvHomeDirKey] = defSrv.Metadata[servers.XldEnvHomeDirKey]
			}
		}

		if srv.Metadata[servers.XldInfHomeDirKey] == "" {
			if !skipO {
				if err := ScanServerProp(srv, srvPropInfrastructureHomeXld, defSrv.Metadata[servers.XldInfHomeDirKey], fmt.Sprintf("Infrastructure home directory for XL Deploy [%v]", defSrv.Metadata[servers.XldInfHomeDirKey]), freq, false); err != nil {
					return err
				}
			} else {
				srv.Metadata[servers.XldInfHomeDirKey] = defSrv.Metadata[servers.XldInfHomeDirKey]
			}
		}
	} else if srv.Type == servers.XlrId {
		if srv.Metadata[servers.XlrHomeDirKey] == "" {
			if !skipO {
				if err := ScanServerProp(srv, srvPropXlrHome, defSrv.Metadata[servers.XlrHomeDirKey], "Home directory for XL Release (optional)", freq, false); err != nil {
					return err
				}
			}
		}
	}

	return srv.AddToConfig()
}

func ParseServerFlags(n string, t string, host string, p string, u string, pwd string, ssl string, ctx string, xldAppHome string, xldCfgHome string, xldEnvHome string, xldInfHome string, xlrHome string) *servers.Server {
	srv := &servers.Server{}
	srv.Name, _ = ParseServerName(n)
	srv.Type, _ = ParseServerType(t)
	srv.Host, _ = ParseServerHost(host)
	srv.Port, _ = ParseServerPort(p)
	srv.Username, _ = ParseServerUsername(u)
	srv.Password, _ = ParseServerPassword(pwd)
	srv.Ssl, _ = ParseServerSsl(ssl)
	srv.ContextRoot = strings.TrimSpace(ctx)
	srv.Metadata = make(map[string]string)

	if srv.Type == servers.XldId {
		srv.Metadata[servers.XldAppHomeDirKey], _ = ParseServerXldHomeDir(xldAppHome, servers.XldAppHomeDirKey, "applications")
		srv.Metadata[servers.XldCfgHomeDirKey], _ = ParseServerXldHomeDir(xldCfgHome, servers.XldCfgHomeDirKey, "configuration")
		srv.Metadata[servers.XldEnvHomeDirKey], _ = ParseServerXldHomeDir(xldEnvHome, servers.XldEnvHomeDirKey, "environments")
		srv.Metadata[servers.XldInfHomeDirKey], _ = ParseServerXldHomeDir(xldInfHome, servers.XldInfHomeDirKey, "infrastructure")
	} else if srv.Type == servers.XlrId {
		srv.Metadata[servers.XlrHomeDirKey] = strings.TrimSpace(xlrHome)
	}

	return srv
}

func ParseServerHost(inp string) (string, error) {
	return checkRequired(inp, "server host")
}

func ParseServerInput(srv *servers.Server, prop string, defVal string, inp string) error {
	const defErrFmt = "%v; default value error: %v"

	parseServerXldHomeDir := func(key string, dirType string) error {
		var s string

		if inp != "" {
			s = inp
		} else {
			s = defVal
		}

		if res, err := ParseServerXldHomeDir(s, key, dirType); err == nil {
			srv.Metadata[key] = res
			return nil
		} else {
			return err
		}
	}

	switch prop {
	case srvPropName:
		if res, err := ParseServerName(inp); err == nil {
			srv.Name = res
			return nil
		} else if inp == "" && defVal != "" {
			if defRes, defErr := ParseServerName(defVal); defErr == nil {
				srv.Name = defRes
				return nil
			} else {
				return fmt.Errorf(defErrFmt, err, defErr)
			}
		} else {
			return err
		}
	case srvPropType:
		if res, err := ParseServerType(inp); err == nil {
			srv.Type = res
			return nil
		} else {
			return err
		}
	case srvPropHost:
		if res, err := ParseServerHost(inp); err == nil {
			srv.Host = res
			return nil
		} else if inp == "" && defVal != "" {
			if defRes, defErr := ParseServerHost(defVal); defErr == nil {
				srv.Host = defRes
				return nil
			} else {
				return fmt.Errorf(defErrFmt, err, defErr)
			}
		} else {
			return err
		}
	case srvPropPort:
		if res, err := ParseServerPort(inp); err == nil {
			srv.Port = res
			return nil
		} else if inp == "" && defVal != "" {
			if defRes, defErr := ParseServerPort(defVal); defErr == nil {
				srv.Port = defRes
				return nil
			} else {
				return fmt.Errorf(defErrFmt, err, defErr)
			}
		} else {
			return err
		}
	case srvPropUsername:
		if res, err := ParseServerUsername(inp); err == nil {
			srv.Username = res
			return nil
		} else if inp == "" && defVal != "" {
			if defRes, defErr := ParseServerUsername(defVal); defErr == nil {
				srv.Username = defRes
				return nil
			} else {
				return fmt.Errorf(defErrFmt, err, defErr)
			}
		} else {
			return err
		}
	case srvPropPassword:
		if res, err := ParseServerPassword(inp); err == nil {
			srv.Password = res
			return nil
		} else if inp == "" && defVal != "" {
			if defRes, defErr := ParseServerPassword(defVal); defErr == nil {
				srv.Password = defRes
				return nil
			} else {
				return fmt.Errorf(defErrFmt, err, defErr)
			}
		} else {
			return err
		}
	case srvPropSsl:
		if res, err := ParseServerSsl(inp); err == nil {
			srv.Ssl = res
			return nil
		} else if inp == "" && defVal != "" {
			if defRes, defErr := ParseServerSsl(defVal); defErr == nil {
				srv.Ssl = defRes
				return nil
			} else {
				return fmt.Errorf(defErrFmt, err, defErr)
			}
		} else {
			return err
		}
	case srvPropContextRoot:
		if inp != "" {
			srv.ContextRoot = strings.TrimSpace(inp)
		} else {
			srv.ContextRoot = strings.TrimSpace(defVal)
		}

		return nil
	case srvPropApplicationsHomeXld:
		return parseServerXldHomeDir(servers.XldAppHomeDirKey, "applications")
	case srvPropConfigurationHomeXld:
		return parseServerXldHomeDir(servers.XldCfgHomeDirKey, "configuration")
	case srvPropEnvironmentsHomeXld:
		return parseServerXldHomeDir(servers.XldEnvHomeDirKey, "environments")
	case srvPropInfrastructureHomeXld:
		return parseServerXldHomeDir(servers.XldInfHomeDirKey, "infrastructure")
	case srvPropXlrHome:
		if inp != "" {
			srv.Metadata[servers.XlrHomeDirKey] = strings.TrimSpace(inp)
		} else {
			srv.Metadata[servers.XlrHomeDirKey] = strings.TrimSpace(defVal)
		}

		return nil
	}

	return fmt.Errorf("%s is not a valid property to be parsed for server", prop)
}

func ParseServerName(inp string) (string, error) {
	return checkRequired(inp, "server name")
}

func ParseServerPassword(inp string) (string, error) {
	return checkRequired(inp, "password")
}

func ParseServerPort(inp string) (int, error) {
	if inp == "" {
		return 0, fmt.Errorf("server port is required")
	}

	port, err := strconv.ParseInt(inp, 10, strconv.IntSize)

	if err != nil {
		return 0, fmt.Errorf("server port must be a number")
	}

	if port <= 0 || port > 65535 {
		return 0, fmt.Errorf("server port range is not valid")
	}

	return int(port), nil
}

func ParseServerSsl(inp string) (bool, error) {
	if inp == "" {
		return false, fmt.Errorf("server protocol is required")
	}

	vldInp := map[string]bool{
		"1":     false,
		"2":     true,
		"http":  false,
		"https": true,
	}

	if ssl, ok := vldInp[inp]; ok {
		return ssl, nil
	} else {
		return false, fmt.Errorf("server protocol must be http or https")
	}
}

func ParseServerType(inp string) (string, error) {
	if inp == "" {
		return "", fmt.Errorf("server type is required")
	}

	vldInp := map[string]string{
		"1":           servers.XldId,
		"2":           servers.XlrId,
		servers.XldId: servers.XldId,
		servers.XlrId: servers.XlrId,
	}

	if t, ok := vldInp[inp]; ok {
		return t, nil
	} else {
		return "", fmt.Errorf("server type must be %s or %s", servers.XldId, servers.XlrId)
	}
}

func ParseServerUsername(inp string) (string, error) {
	return checkRequired(inp, "username")
}

func ParseServerXldHomeDir(inp string, key string, dirType string) (string, error) {
	prefix := servers.DefaultXld.Metadata[key]
	vldInp := []string{fmt.Sprintf("%s/", prefix), prefix}
	s := strings.TrimSpace(inp)

	if strings.HasPrefix(s, vldInp[0]) || strings.HasPrefix(s, vldInp[1]) {
		return s, nil
	}

	return "", fmt.Errorf("%s home directory for XL Deploy must start with %s", dirType, prefix)
}

func scanDefault(r io.Reader) (string, error) {
	scnr := bufio.NewScanner(r)
	scnr.Scan()
	inp := scnr.Text()
	err := scnr.Err()
	return inp, err
}

func scanSecure(pr pwdreader.PasswordReader) (string, error) {
	pwdB, err := pr.ReadPassword(int(syscall.Stdin))
	pwd := string(pwdB)
	fmt.Println()
	return pwd, err
}

func ScanServerProp(srv *servers.Server, prop string, defVal string, out string, freq int, secScan bool) error {
	var err error

	for i := 0; i < freq; i++ {
		var inp string
		fmt.Printf("%s: ", out)

		if !secScan {
			inp, err = scanDefault(inputReader)
		} else {
			inp, err = scanSecure(pwdReader)
		}

		if err == nil {
			if inpErr := ParseServerInput(srv, prop, defVal, inp); inpErr == nil {
				return nil
			} else {
				err = inpErr
			}
		}

		fmt.Println(err)
	}

	return err
}

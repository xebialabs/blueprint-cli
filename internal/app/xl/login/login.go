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
)

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

func checkRequired(inp string, prop string) (string, error) {
	if strings.TrimSpace(inp) == "" {
		return "", fmt.Errorf("%s is required", prop)
	}

	return inp, nil
}

func ExecuteServer(n string, t string, host string, p string, u string, pwd string, ssl string, ctx string, skipO bool) error {
	const freq = 3
	srv := ParseServerFlags(n, t, host, p, u, pwd, ssl, ctx)

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
		if err := ScanServerProp(srv, srvPropPassword, defSrv.Password, fmt.Sprintf("Password [%v]", defSrv.Password), freq, true); err != nil {
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

	return srv.Save()
}

func ParseServerFlags(n string, t string, host string, p string, u string, pwd string, ssl string, ctx string) *servers.Server {
	srv := &servers.Server{}
	srv.Name, _ = ParseServerName(n)
	srv.Type, _ = ParseServerType(t)
	srv.Host, _ = ParseServerHost(host)
	srv.Port, _ = ParseServerPort(p)
	srv.Username, _ = ParseServerUsername(u)
	srv.Password, _ = ParseServerPassword(pwd)
	srv.Ssl, _ = ParseServerSsl(ssl)
	srv.ContextRoot = ctx
	return srv
}

func ParseServerHost(inp string) (string, error) {
	return checkRequired(inp, "server host")
}

func ParseServerInput(srv *servers.Server, prop string, defVal string, inp string) error {
	const defErrFmt = "%v; default value error: %v"

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

func scanDefault() (string, error) {
	scnr := bufio.NewScanner(os.Stdin)
	scnr.Scan()
	inp := scnr.Text()
	err := scnr.Err()
	return inp, err
}

func scanSecure() (string, error) {
	pwdB, err := terminal.ReadPassword(int(syscall.Stdin))
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
			inp, err = scanDefault()
		} else {
			inp, err = scanSecure()
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

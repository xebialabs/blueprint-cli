package osHelper

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"
)

const (
	_DefaultApiServerUrl = "_defaultapiserverurl"
	Os                   = "_operatingsystem"
	DateTime             = "_datetime"
	CertFileLocation     = "getcertfilelocation"
	KeyFileLocation      = "getkeyfilelocation"
)

type OSFnResult struct {
	kubeURL          string
	os               string
	certFileLocation string
	keyFileLocation  string
}

type OperatingSystem struct{}

func (s *OperatingSystem) getOs() string {
	return GetOperatingSystem()
}

type IOperatingSystem interface {
	getOs() string
}

func (result *OSFnResult) GetResult(module string, attr string, index int) (string, error) {
	switch module {
	case _DefaultApiServerUrl:
		return result.kubeURL, nil
	case Os:
		return result.os, nil
	case CertFileLocation:
		return result.certFileLocation, nil
	case KeyFileLocation:
		return result.keyFileLocation, nil
	default:
		return "", fmt.Errorf("%s is not a valid OS module", module)
	}
}

func GetOperatingSystem() string {
	return runtime.GOOS
}

func GetDateTime() string {
	currentTime := time.Now()
	return fmt.Sprintf("%04d%02d%02d-%02d%02d%02d", currentTime.Year(), currentTime.Month(), currentTime.Day(), currentTime.Hour(), currentTime.Minute(), currentTime.Second())
}

func DefaultApiServerUrl(ios IOperatingSystem) string {
	if ios.getOs() == "windows" || ios.getOs() == "darwin" {
		return "https://host.docker.internal:6443"
	}
	return ""
}

func GetLocation(file string) string {
	dir, err := os.Getwd()
	if err != nil {
		log.Fatal(err)
	}
	return filepath.Join(dir, file)
}

func GetPropertyByName(module string) (interface{}, error) {
	switch strings.ToLower(module) {
	case _DefaultApiServerUrl:
		return DefaultApiServerUrl(&OperatingSystem{}), nil
	case Os:
		return GetOperatingSystem(), nil
	case DateTime:
		return GetDateTime(), nil
	case CertFileLocation:
		return GetLocation("cert.crt"), nil
	case KeyFileLocation:
		return GetLocation("cert.key"), nil
	default:
		return nil, fmt.Errorf("%s is not a valid OS module", module)
	}
}

func ProcessCmdResult(cmd exec.Cmd) ([]byte, error) {
	return util.ProcessCmdResult(cmd)
}

func ProcessCmdResultWithoutLog(cmd exec.Cmd) ([]byte, error) {
	util.Verbose("\nExecuting command: %s\n", cmd.String())
	cmdOutput, err := cmd.CombinedOutput()
	return cmdOutput, err
}

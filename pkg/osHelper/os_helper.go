package osHelper

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"runtime"
	"strings"
)

const (
	_DefaultApiServerUrl = "_defaultapiserverurl"
	Os                   = "_operatingsystem"
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
	case CertFileLocation:
		return GetLocation("cert.crt"), nil
	case KeyFileLocation:
		return GetLocation("cert.key"), nil
	default:
		return nil, fmt.Errorf("%s is not a valid OS module", module)
	}
}

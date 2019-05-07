package osHelper

import (
    "fmt"
    "log"
    "os"
    "path/filepath"
    "runtime"
    "strings"

    "github.com/xebialabs/xl-cli/pkg/models"
)

const (
	_DefaultApiServerUrl = "_defaultapiserverurl"
	Os                   = "_operatingsystem"
    CertFileLocation     = "getcertfilelocation"
    KeyFileLocation      = "getkeyfilelocation"
)

type OSFnResult struct {
	kubeURL          []string
	os               []string
    certFileLocation []string
	keyFileLocation  []string
}

type OperatingSystem struct{}

func (s *OperatingSystem) getOs() string {
	return getOperatingSystem()
}

type IOperatingSystem interface {
	getOs() string
}

func (result *OSFnResult) GetResult(module string, attr string, index int) ([]string, error) {
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
		return nil, fmt.Errorf("%s is not a valid OS module", module)
	}
}

func getOperatingSystem() string {
	return runtime.GOOS
}

func defaultApiServerUrl(ios IOperatingSystem) ([]string, error) {
	if ios.getOs() == "windows" {
		return []string{"https://host.docker.internal:6445"}, nil
	} else if ios.getOs() == "darwin" {
		return []string{"https://host.docker.internal:6443"}, nil
	} else {
		return []string{""}, nil
	}
}

// CallOSFuncByName calls related OS module function with parameters provided
func CallOSFuncByName(module string, params ...string) (models.FnResult, error) {
	switch strings.ToLower(module) {
	case _DefaultApiServerUrl:
		url, _ := defaultApiServerUrl(&OperatingSystem{})
		return &OSFnResult{kubeURL: url}, nil
	case Os:
		return &OSFnResult{os: []string{getOperatingSystem()}}, nil
    case CertFileLocation:
        return &OSFnResult{certFileLocation: getLocation("cert.crt")}, nil
    case KeyFileLocation:
        return &OSFnResult{keyFileLocation: getLocation("cert.key")}, nil
	default:
		return nil, fmt.Errorf("%s is not a valid OS module", module)
	}
}

func getLocation(file string) []string {
    dir, err := os.Getwd()
    if err != nil {
        log.Fatal(err)
    }
    return []string{filepath.Join(dir, file)}
}

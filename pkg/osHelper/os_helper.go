package osHelper

import (
	"fmt"
	"runtime"
	"strings"

	"github.com/xebialabs/xl-cli/pkg/models"
)

const (
	_DefaultApiServerUrl = "_defaultapiserverurl"
	Os                   = "_operatingsystem"
)

type OSFnResult struct {
	kubeURL []string
	os      []string
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
	default:
		return nil, fmt.Errorf("%s is not a valid OS module", module)
	}
}

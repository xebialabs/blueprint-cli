package osHelper

import (
	"fmt"
	"runtime"
	"strings"

	"github.com/xebialabs/xl-cli/pkg/models"
)

const (
	DefaultApiServerUrl = "defaultapiserverurl"
)

type OSFnResult struct {
	kubeURL []string
}

type OperatingSystem struct{}

func (s *OperatingSystem) getOs() string {
	return runtime.GOOS
}

type IOperatingSystem interface {
	getOs() string
}

func (result *OSFnResult) GetResult(module string, attr string, index int) ([]string, error) {
	switch module {
	case DefaultApiServerUrl:
		return result.kubeURL, nil
	default:
		return nil, fmt.Errorf("%s is not a valid OS module", module)
	}
}

func defaultApiServerUrl(ios IOperatingSystem) ([]string, error) {
	if ios.getOs() == "windows" {
		return []string{"https://host.docker.internal:6445/"}, nil
	} else if ios.getOs() == "darwin" {
		return []string{"https://host.docker.internal:6443/"}, nil
	} else {
		return []string{"https://localhost:6443/"}, nil
	}
}

// CallOSFuncByName calls related OS module function with parameters provided
func CallOSFuncByName(module string, params ...string) (models.FnResult, error) {
	switch strings.ToLower(module) {
	case DefaultApiServerUrl:
		url, _ := defaultApiServerUrl(&OperatingSystem{})
		return &OSFnResult{kubeURL: url}, nil
	default:
		return nil, fmt.Errorf("%s is not a valid OS module", module)
	}
}

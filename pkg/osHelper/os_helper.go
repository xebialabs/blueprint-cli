package osHelper

import (
	"runtime"
)

type OperatingSystem struct{}

func (s *OperatingSystem) getOs() string {
	return runtime.GOOS
}

type IOperatingSystem interface {
	getOs() string
}

func FindOperatingSystem(ios IOperatingSystem) ([]string, error) {
	if ios.getOs() == "windows" {
		return []string{"https://host.docker.internal:6445/"}, nil
	} else if ios.getOs() == "darwin" || ios.getOs() == "linux" {
		return []string{"https://host.docker.internal:6443/"}, nil
	} else {
		return []string{"https://localhost:6443/"}, nil
	}
}

package osHelper

import (
    "runtime"
    "testing"

    "github.com/stretchr/testify/assert"
)

var currentOperatingSystem = "windows"
var ms = &MockOperatingSystem{
	mockGetOs: func() string {
		return currentOperatingSystem
	},
}

type MockOperatingSystem struct {
	mockGetOs func() string
}

func (s *MockOperatingSystem) getOs() string {
	return s.mockGetOs()
}

func testScenarios(t *testing.T, ms *MockOperatingSystem, apiServerURL string) {
	value := DefaultApiServerUrl(ms)
	assert.EqualValues(t, value, apiServerURL)
}

func TestApiServerUrlOnWindows(t *testing.T) {
	testScenarios(t, ms, "https://host.docker.internal:6443")
}

func TestApiServerUrlOnMacos(t *testing.T) {
	currentOperatingSystem = "darwin"
	testScenarios(t, ms, "https://host.docker.internal:6443")
}

func TestApiServerUrlOnLinux(t *testing.T) {
	currentOperatingSystem = "linux"
	testScenarios(t, ms, "")
}

func TestApiServerUrlOnOther(t *testing.T) {

	currentOperatingSystem = "other"
	testScenarios(t, ms, "")
}

func TestOperatingSystem(t *testing.T) {
	t.Run("should return the correct operating system", func(t *testing.T) {
		ms := OperatingSystem{}
		assert.EqualValues(t, ms.getOs(), runtime.GOOS)
	})
}

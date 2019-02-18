package osHelper

import (
	"runtime"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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
	value, err := FindOperatingSystem(ms)
	require.Nil(t, err)
	assert.NotEmpty(t, value)
	assert.Len(t, value, 1)
	assert.EqualValues(t, value[0], apiServerURL)
}

func TestApiServerUrlOnWindows(t *testing.T) {
	testScenarios(t, ms, "https://host.docker.internal:6445/")
}

func TestApiServerUrlOnMacos(t *testing.T) {
	currentOperatingSystem = "darwin"
	testScenarios(t, ms, "https://host.docker.internal:6443/")
}

func TestApiServerUrlOnLinux(t *testing.T) {
	currentOperatingSystem = "linux"
	testScenarios(t, ms, "https://localhost:6443/")
}

func TestApiServerUrlOnOther(t *testing.T) {

	currentOperatingSystem = "other"
	testScenarios(t, ms, "https://localhost:6443/")
}

func TestOperatingSystem(t *testing.T) {
	ms := OperatingSystem{}
	assert.EqualValues(t, ms.getOs(), runtime.GOOS)
}

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
	value, err := DefaultApiServerUrl(ms)
	require.Nil(t, err)
	assert.NotEmpty(t, value)
	assert.Len(t, value, 1)
	assert.EqualValues(t, value[0], apiServerURL)
}

func TestApiServerUrlOnWindows(t *testing.T) {
	testScenarios(t, ms, "https://host.docker.internal:6445")
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

func TestApiServerURL(t *testing.T) {
	t.Run("should return the URL based on the Operating System", func(t *testing.T) {
		result, err := CallOSFuncByName(_DefaultApiServerUrl)
		require.Nil(t, err)
		apiServerURL, err := result.GetResult(_DefaultApiServerUrl, "", -1)
		require.Nil(t, err)
		assert.Len(t, apiServerURL, 1)
	})

	t.Run("should error when the function is not available", func(t *testing.T) {
		_, err := CallOSFuncByName("CallSomeNonExistentFunction")
		require.NotNil(t, err)
	})

	t.Run("should return error when GetResult is called with non existent function", func(t *testing.T) {
		result, err := CallOSFuncByName(_DefaultApiServerUrl)
		require.Nil(t, err)
		_, err = result.GetResult("CallSomeNonExistentFunction", "", 0)
		require.NotNil(t, err)
	})
}

func TestOperatingSystem(t *testing.T) {
	t.Run("should return the correct operating system", func(t *testing.T) {
		ms := OperatingSystem{}
		assert.EqualValues(t, ms.getOs(), runtime.GOOS)
	})

	t.Run("should return the operating system in which it is running", func(t *testing.T) {
		result, err := CallOSFuncByName(Os)
		require.Nil(t, err)
		os, _ := result.GetResult(Os, "", 1)
		assert.EqualValues(t, os, []string{runtime.GOOS})
	})
}

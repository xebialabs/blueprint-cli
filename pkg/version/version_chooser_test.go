package version

import (
	"fmt"
	"reflect"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/xebialabs/xl-cli/pkg/models"
)

func Test_showVersions(t *testing.T) {
	tests := []struct {
		name    string
		params  []string
		want    []string
		wantErr bool
	}{
		{
			"show XLR versions",
			[]string{"xlr"},
			[]string{"9.0.2", "9.0.4", "9.0.6"},
			false,
		},
		{
			"show XLD versions",
			[]string{""},
			[]string{"9.0.2", "9.0.3", "9.0.5"},
			false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := showVersions(tt.params)
			if (err != nil) != tt.wantErr {
				t.Errorf("showVersions() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("showVersions() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_checkVersion(t *testing.T) {
	type TestCheckVersion struct {
		name    string
		params  []string
		want    bool
		wantErr bool
	}

	tests := []func() TestCheckVersion{
		func() TestCheckVersion {
			models.AvailableXlrVersion = ""
			return TestCheckVersion{
				"should error when invlaid number of params are passed",
				[]string{"8.0"},
				false,
				true,
			}
		},
		func() TestCheckVersion {
			models.AvailableXlrVersion = ""
			return TestCheckVersion{
				"should check valid versions for XLR",
				[]string{"xlr", "xlr:9.0.2"},
				true,
				false,
			}
		},
		func() TestCheckVersion {
			models.AvailableXlrVersion = ""
			return TestCheckVersion{
				"should check valid versions for XLD",
				[]string{"xld", "xld:9.0.5"},
				true,
				false,
			}
		},
		func() TestCheckVersion {
			models.AvailableXlrVersion = ""
			return TestCheckVersion{
				"should check invalid versions for XLR when current version is nil",
				[]string{"xlr", "xlr:8.9.2"},
				true,
				false,
			}
		},
		func() TestCheckVersion {
			models.AvailableXldVersion = ""
			return TestCheckVersion{
				"should check invalid versions for XLD when current version is nil",
				[]string{"xld", "xld:8.0"},
				true,
				false,
			}
		},
		func() TestCheckVersion {
			models.AvailableXlrVersion = "xlr:8.9.2"
			return TestCheckVersion{
				"should check valid versions for XLR when current version is set",
				[]string{"xlr", "xlr:8.9.2"},
				true,
				false,
			}
		},
		func() TestCheckVersion {
			models.AvailableXlrVersion = "xlr:8.9.2"
			return TestCheckVersion{
				"should check invalid versions for XLR when current version is set",
				[]string{"xlr", "xlr:8.9.1"},
				false,
				true,
			}
		},
		func() TestCheckVersion {
			models.AvailableXldVersion = "xld:8.9.2"
			return TestCheckVersion{
				"should check invalid versions for XLD when current version is set",
				[]string{"xld", "xld:8.9.1"},
				false,
				true,
			}
		},
	}
	for _, ttF := range tests {
		tt := ttF()
		t.Run(tt.name, func(t *testing.T) {
			got, err := checkVersion(tt.params)
			if (err != nil) != tt.wantErr {
				t.Errorf("checkVersion() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("checkVersion() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_getVersionFromTag(t *testing.T) {
	tests := []struct {
		name    string
		params  []string
		want    string
		wantErr bool
	}{
		{
			"should error when passing insufficient params",
			[]string{},
			"",
			true,
		},
		{
			"should return the version from the passed string",
			[]string{"xld:9.0.4"},
			"9.0.4",
			false,
		},
		{
			"should return the version from the passed string",
			[]string{"xld:9.0.4-beta.1"},
			"9.0.4-beta.1",
			false,
		},
		{
			"should return the version from the passed string",
			[]string{"xld:9.0"},
			"9.0.0",
			false,
		},
		{
			"should return the version from the passed string",
			[]string{"9.0.2"},
			"9.0.2",
			false,
		},
		{
			"should error when passing invlaid version",
			[]string{"xld:latest"},
			"",
			true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := getVersionFromTag(tt.params)
			if (err != nil) != tt.wantErr {
				t.Errorf("getVersionFromTag() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("getVersionFromTag() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_GetVersionFromImageTag(t *testing.T) {
	t.Run("should get version when Docker image tag version is semver", func(t *testing.T) {
		actual, _ := GetVersionFromImageTag("test:1.0.0")
		expected := "1.0.0"

		assert.Equal(t, expected, actual.String())

		actual, _ = GetVersionFromImageTag("test:8.5.1")
		expected = "8.5.1"
		assert.Equal(t, expected, actual.String())

		actual, _ = GetVersionFromImageTag("test:9.9.1000")
		expected = "9.9.1000"

		assert.Equal(t, expected, actual.String())

		actual, _ = GetVersionFromImageTag("test:8.6.0")
		expected = "8.6.0"

		assert.Equal(t, expected, actual.String())

		actual, _ = GetVersionFromImageTag("test:9999.9999.9999")
		expected = "9999.9999.9999"

		assert.Equal(t, expected, actual.String())

		actual, _ = GetVersionFromImageTag("test:9.0.5-beta1")
		expected = "9.0.5-beta1"

		assert.Equal(t, expected, actual.String())

		actual, _ = GetVersionFromImageTag("test:9.0.5-pre.1")
		expected = "9.0.5-pre.1"

		assert.Equal(t, expected, actual.String())

		actual, _ = GetVersionFromImageTag("test:9.0.5-1")
		expected = "9.0.5-1"

		assert.Equal(t, expected, actual.String())
	})

	t.Run("should throw an error when Docker image tag version is not semver", func(t *testing.T) {
		_, err := GetVersionFromImageTag("test:latest")
		expected := fmt.Errorf("Version tag latest is not valid: Invalid Semantic Version")

		assert.Equal(t, err, expected)

		_, err = GetVersionFromImageTag("test:beta")
		expected = fmt.Errorf("Version tag beta is not valid: Invalid Semantic Version")

		assert.Equal(t, err, expected)
	})

	t.Run("should throw an error when there is no Docker image tag version", func(t *testing.T) {
		_, err := GetVersionFromImageTag("test:")
		expected := fmt.Errorf("Version tag is missing")

		assert.Equal(t, err, expected)

		_, err = GetVersionFromImageTag("test")
		expected = fmt.Errorf("Version tag is missing")

		assert.Equal(t, err, expected)
	})
}

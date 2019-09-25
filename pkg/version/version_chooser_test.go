package version

import (
	"fmt"
	"reflect"
	"testing"

	"github.com/stretchr/testify/assert"
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
	type args struct {
		params []string
	}
	tests := []struct {
		name    string
		args    args
		want    bool
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := checkVersion(tt.args.params)
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
	type args struct {
		params []string
	}
	tests := []struct {
		name    string
		args    args
		want    string
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := getVersionFromTag(tt.args.params)
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

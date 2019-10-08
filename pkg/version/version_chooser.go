package version

import (
	"fmt"
	"strings"

	"github.com/Masterminds/semver"
	"github.com/xebialabs/xl-cli/pkg/models"
)

const (
	_ShowApplicableVersions = "_showapplicableversions"
	CheckVersion            = "checkversion"
	GetVersionFromTag       = "getversionfromtag"
)

// TODO find a better way to handle this...
var AvailableXlrVersions = []string{"9.0.2", "9.0.4", "9.0.6"}
var AvailableXldVersions = []string{"9.0.2", "9.0.3", "9.0.5"}

type VersionFnResult struct {
	versions []string
}

func (result *VersionFnResult) GetResult(module string, attr string, index int) ([]string, error) {
	switch module {
	case _ShowApplicableVersions:
		return result.versions, nil
	default:
		return nil, fmt.Errorf("%s is not a valid Version module", module)
	}
}

func showVersions(params []string) ([]string, error) {
	var application string
	if len(params) > 0 {
		application = params[0]
	}

	if application == "xlr" {
		return getVersionForApp(models.AvailableOfficialXlrVersion, AvailableXlrVersions)
	} else {
		return getVersionForApp(models.AvailableOfficialXldVersion, AvailableXldVersions)
	}

}

func getVersionForApp(availableOfficialAppVersion string, availableAppVersions []string) ([]string, error) {
	var currentAppVersion *semver.Version
	var err error

	if availableOfficialAppVersion != "" {
		currentAppVersion, err = semver.NewVersion(availableOfficialAppVersion)
		if err != nil {
			return nil, fmt.Errorf("Current version tag %s is not valid: %s", availableOfficialAppVersion, err)
		}
	}

	if currentAppVersion == nil {
		return availableAppVersions, nil
	}

	var applicableAppVersion []string

	for _, version := range availableAppVersions {
		appVersion, err := semver.NewVersion(version)
		if err != nil {
			return nil, fmt.Errorf("App version tag %s is not valid: %s", version, err)
		}
		if currentAppVersion.Compare(appVersion) <= 0 {
			applicableAppVersion = append(applicableAppVersion, version)
		}
	}

	return applicableAppVersion, nil
}

func getVersionFromConfigMap(application string) (*semver.Version, error) {
	if application == "xlr" && models.AvailableXlrVersion != "" {
		return GetVersionFromImageTag(models.AvailableXlrVersion)
	}

	if application == "xld" && models.AvailableXldVersion != "" {
		return GetVersionFromImageTag(models.AvailableXldVersion)
	}
	return nil, nil
}

func compareVersion(application, version string) (bool, error) {
	currentVersion, err := getVersionFromConfigMap(application)

	if err != nil {
		return false, fmt.Errorf("%s:%s provided in config map is not valid", application, version)
	}

	givenVersion, err := GetVersionFromImageTag(version)

	if err != nil {
		return false, fmt.Errorf("%s is not a valid version/tag", version)
	}

	if currentVersion != nil {
		if currentVersion.Compare(givenVersion) <= 0 {
			return true, nil
		}

		if application == "xlr" && models.AvailableXlrVersion != "" {
			return false, fmt.Errorf("cannot downgrade from %s to %s", models.AvailableXlrVersion, version)
		}

		if application == "xld" && models.AvailableXldVersion != "" {
			return false, fmt.Errorf("cannot downgrade from %s to %s", models.AvailableXldVersion, version)
		}
		//probably not needed
		if application == "xlr" && models.AvailableOfficialXlrVersion != "" {
			return false, fmt.Errorf("cannot downgrade from %s to %s", models.AvailableOfficialXlrVersion, version)
		}

		if application == "xld" && models.AvailableOfficialXldVersion != "" {
			return false, fmt.Errorf("cannot downgrade from %s to %s", models.AvailableOfficialXldVersion, version)
		}
	}

	return true, nil
}

func checkVersion(params []string) (bool, error) {
	if len(params) != 2 {
		return false, fmt.Errorf("invalid number of arguments sent in checkVersion")
	}

	application := params[0]
	version := params[1]

	return compareVersion(application, version)
}

func getVersionFromTag(params []string) (string, error) {
	if len(params) != 1 {
		return "", fmt.Errorf("invalid number of arguments sent in getVersionFromTag")
	}

	version, err := GetVersionFromImageTag(params[0])
	if err != nil {
		return "", fmt.Errorf("%s is not a valid version/tag", params[0])
	}
	return version.String(), nil

}

func GetPropertyByName(module string, params ...string) (interface{}, error) {
	switch strings.ToLower(module) {
	case _ShowApplicableVersions:
		return showVersions(params)
	case GetVersionFromTag:
		return getVersionFromTag(params)
	case CheckVersion:
		return checkVersion(params)
	default:
		return nil, fmt.Errorf("%s is not a valid UP helper module", module)
	}
}

func GetVersionFromImageTag(version string) (*semver.Version, error) {
	if strings.Contains(version, ":") {
		split := strings.Split(version, ":")
		version = split[len(split)-1]
	}
	if version != "" {
		vers, err := semver.NewVersion(version)
		if err != nil {
			return nil, fmt.Errorf("Version tag %s is not valid: %s", version, err)
		}

		return vers, nil
	}
	return nil, fmt.Errorf("Version tag is missing")
}

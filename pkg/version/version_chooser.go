package version

import (
	"fmt"
	"strings"

	"github.com/xebialabs/xl-cli/pkg/models"
	"github.com/xebialabs/xl-cli/pkg/util"
)

const (
	_ShowApplicableVersions = "_showapplicableversions"
	CheckVersion            = "checkversion"
	GetVersionFromTag       = "getversionfromtag"
)

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
	var currentVersion int64
	if models.AvailableVersion != "" {
		currentVersion = util.ParseVersion(models.AvailableVersion, 4)
	}

	// TODO find a better way to handle this...
	availableVersions := []string{"8.6.1", "9.0.2"}

	if currentVersion == int64(0) {
		return availableVersions, nil
	}

	var applicableVersion []string

	for _, version := range availableVersions {
		if currentVersion <= util.ParseVersion(version, 4) {
			applicableVersion = append(applicableVersion, version)
		}
	}

	return applicableVersion, nil
}

func versionFromDockerTag(version string) (int64, error) {
	version, err := util.GetVersionFromImageTag(version)
	if err != nil {
		return 0, err
	}

	return util.ParseVersion(version, 4), nil
}

func getVersionFromConfigMap(application string) (int64, error) {
	if application == "xlr" && models.AvailableXlrVersion != "" {
		return versionFromDockerTag(models.AvailableXlrVersion)
	}

	if application == "xld" && models.AvailableXldVersion != "" {
		return versionFromDockerTag(models.AvailableXldVersion)
	}
	return int64(0), nil
}

func compareVersion(application, version string) (bool, error) {
	currentVersion, err := getVersionFromConfigMap(application)

	if err != nil {
		return false, fmt.Errorf("%s:%s provided in config map is not valid", application, version)
	}

	givenVersion, err := versionFromDockerTag(version)

	if err != nil {
		return false, fmt.Errorf("%s is not a valid version/tag", version)
	}

	if currentVersion != int64(0) {
		if currentVersion <= givenVersion {
			return true, nil
		}

		if application == "xlr" && models.AvailableXlrVersion != "" {
			return false, fmt.Errorf("cannot downgrade from %s to %s", models.AvailableXlrVersion, version)
		}

		if application == "xld" && models.AvailableXldVersion != "" {
			return false, fmt.Errorf("cannot downgrade from %s to %s", models.AvailableXldVersion, version)
		}

		return false, fmt.Errorf("cannot downgrade from %s to %s", models.AvailableVersion, version)
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

	version := params[0]

	return util.GetVersionFromImageTag(version)
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

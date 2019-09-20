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
	application := params[0]

	if application == "xlr" {
		var currentXlrVersion int64

		if models.AvailableOfficialXlrVersion != "" {
			currentXlrVersion = util.ParseVersion(models.AvailableOfficialXlrVersion, 4)
		}
		// TODO find a better way to handle this...
		availableXlrVersions := []string{"9.0.2", "9.0.4", "9.0.6"}

		if currentXlrVersion == int64(0) {
			return availableXlrVersions, nil
		}

		var applicableXlrVersion []string

		for _, version := range availableXlrVersions {
			if currentXlrVersion <= util.ParseVersion(version, 4) {
				applicableXlrVersion = append(applicableXlrVersion, version)
			}
		}

		return applicableXlrVersion, nil

	} else {
		var currentXldVersion int64

		if models.AvailableOfficialXldVersion != "" {
			currentXldVersion = util.ParseVersion(models.AvailableOfficialXldVersion, 4)
		}
		// TODO find a better way to handle this...
		availableXldVersions := []string{"9.0.2", "9.0.3", "9.0.5"}

		if currentXldVersion == int64(0) {
			return availableXldVersions, nil
		}

		var applicableXldVersion []string

		for _, version := range availableXldVersions {
			if currentXldVersion <= util.ParseVersion(version, 4) {
				applicableXldVersion = append(applicableXldVersion, version)
			}
		}

		return applicableXldVersion, nil
	}

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

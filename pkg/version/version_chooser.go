package version

import (
	"fmt"
	"strings"

	"github.com/xebialabs/xl-cli/pkg/models"
	"github.com/xebialabs/xl-cli/pkg/util"
)

const (
	_ShowApplicableVersions = "_showapplicableversions"
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
	availableVersions := []string{"8.5.3", "8.6.1"}

	if currentVersion == int64(0) {
		return availableVersions, nil
	}

	var applicableVersion []string

	for _, version := range availableVersions {
		if currentVersion < util.ParseVersion(version, 4) {
			applicableVersion = append(applicableVersion, version)
		}
	}

	return applicableVersion, nil
}

func GetPropertyByName(module string, params ...string) (interface{}, error) {
    switch strings.ToLower(module) {
    case _ShowApplicableVersions:
        return showVersions(params)
    default:
        return nil, fmt.Errorf("%s is not a valid UP helper module", module)
    }
}

package up

import (
	"fmt"
	"io/ioutil"
	"os"
	"os/user"
	"path/filepath"
	"testing"

	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"k8s.io/client-go/kubernetes"
)

func TestGetLocalContext(t *testing.T) {
	currentUser, _ := user.Current()

	t.Run("should error when local up context path is invalid", func(t *testing.T) {
		c, _, err := getLocalContext("")
		require.NotNil(t, err)
		require.Nil(t, c)
	})
	t.Run("should return valid local up test context", func(t *testing.T) {
		c, templatePath, err := getLocalContext(filepath.Join(currentUser.HomeDir, "xlTest", "blueprints"))
		require.Nil(t, err)
		require.NotNil(t, c)
		assert.Equal(t, "cmd-arg", (*c.ActiveRepo).GetName())
		assert.Equal(t, "xlTest", templatePath)
	})
}

func TestGetRepo(t *testing.T) {
	t.Run("should return repo with a branch name", func(t *testing.T) {
		repo, err := getGitRepo("xl-up")
		require.Nil(t, err)
		fmt.Println(repo.GetInfo())
		assert.Equal(t, repo.GetName(), XlUpBlueprint)
		assert.Equal(t, repo.GetProvider(), "github")
		assert.Contains(t, repo.GetInfo(), "Branch: xl-up")
	})
}

func TestMergeMaps(t *testing.T) {

	t.Run("should return empty map when the maps are empty", func(t *testing.T) {
		autoMap := make(map[string]string)
		providedMap := make(map[string]string)

		mergedMap, isConflict := mergeMaps(autoMap, providedMap)

		assert.Equal(t, isConflict, false)
		assert.Equal(t, len(mergedMap), 0)
	})

	t.Run("should merge map when provided map is empty", func(t *testing.T) {
		autoMap := make(map[string]string)
		autoMap["one"] = "1"
		autoMap["two"] = "2"
		autoMap["three"] = "3"
		autoMap["four"] = "4"

		providedMap := make(map[string]string)

		mergedMap, isConflict := mergeMaps(autoMap, providedMap)

		assert.Equal(t, isConflict, false)
		assert.Equal(t, len(mergedMap), 4)
		assert.Equal(t, mergedMap["one"], "1")
		assert.Equal(t, mergedMap["two"], "2")
		assert.Equal(t, mergedMap["three"], "3")
		assert.Equal(t, mergedMap["four"], "4")
	})

	t.Run("should merge map when auto map is empty", func(t *testing.T) {
		autoMap := make(map[string]string)

		providedMap := make(map[string]string)
		providedMap["one"] = "1"
		providedMap["two"] = "2"
		providedMap["three"] = "3"
		providedMap["four"] = "4"

		mergedMap, isConflict := mergeMaps(autoMap, providedMap)

		assert.Equal(t, isConflict, false)
		assert.Equal(t, len(mergedMap), 4)
		assert.Equal(t, mergedMap["one"], "1")
		assert.Equal(t, mergedMap["two"], "2")
		assert.Equal(t, mergedMap["three"], "3")
		assert.Equal(t, mergedMap["four"], "4")
	})

	t.Run("should merge map when there is no overlap", func(t *testing.T) {
		autoMap := make(map[string]string)
		autoMap["two"] = "2"
		autoMap["four"] = "4"

		providedMap := make(map[string]string)
		providedMap["one"] = "1"
		providedMap["three"] = "3"

		mergedMap, isConflict := mergeMaps(autoMap, providedMap)

		assert.Equal(t, isConflict, false)
		assert.Equal(t, len(mergedMap), 4)
		assert.Equal(t, mergedMap["one"], "1")
		assert.Equal(t, mergedMap["two"], "2")
		assert.Equal(t, mergedMap["three"], "3")
		assert.Equal(t, mergedMap["four"], "4")
	})

	t.Run("should merge map when there is overlap in value", func(t *testing.T) {
		autoMap := make(map[string]string)
		autoMap["one"] = "1"
		autoMap["two"] = "2"

		providedMap := make(map[string]string)
		providedMap["one"] = "one"
		providedMap["two"] = "two"

		mergedMap, isConflict := mergeMaps(autoMap, providedMap)

		assert.Equal(t, isConflict, true)
		assert.Equal(t, len(mergedMap), 2)
		assert.Equal(t, mergedMap["one"], "1")
		assert.Equal(t, mergedMap["two"], "2")
	})

	t.Run("should merge map when there is overlap", func(t *testing.T) {
		autoMap := make(map[string]string)
		autoMap["one"] = "1"
		autoMap["two"] = "2"
		autoMap["three"] = "3"

		providedMap := make(map[string]string)
		providedMap["one"] = "one"
		providedMap["two"] = "two"
		providedMap["four"] = "four"

		mergedMap, isConflict := mergeMaps(autoMap, providedMap)

		assert.Equal(t, isConflict, true)
		assert.Equal(t, len(mergedMap), 4)
		assert.Equal(t, mergedMap["one"], "1")
		assert.Equal(t, mergedMap["two"], "2")
		assert.Equal(t, mergedMap["three"], "3")
		assert.Equal(t, mergedMap["four"], "four")
	})

}

func TestDecideVersionMatch(t *testing.T) {
	t.Run("should throw error when the new version number is less than the installed one", func(t *testing.T) {
		msg, err := decideVersionMatch("10.0.0", "9.9.9")

		assert.Equal(t, msg, "")
		assert.Equal(t, err.Error(), "cannot downgrade the deployment from 10.0.0 to 9.9.9")
	})

	t.Run("should accept when the new version number is greater than the installed one", func(t *testing.T) {
		msg, err := decideVersionMatch("9.9.9", "9.9.10")

		assert.Equal(t, msg, "upgrading from 9.9.9 to 9.9.10")
		assert.Equal(t, err, nil)

		msg, err = decideVersionMatch("9.10.9", "9.10.10")

		assert.Equal(t, msg, "upgrading from 9.10.9 to 9.10.10")
		assert.Equal(t, err, nil)

		msg, err = decideVersionMatch("10.10.9", "10.10.10")

		assert.Equal(t, msg, "upgrading from 10.10.9 to 10.10.10")
		assert.Equal(t, err, nil)
	})

	t.Run("should throw error when the new version number is less than the installed one", func(t *testing.T) {
		msg, err := decideVersionMatch("10.0.0", "10.0.0")

		assert.Equal(t, msg, "")
		assert.Equal(t, err.Error(), "the given version 10.0.0 already exists")
	})
}

func TestWriteConfig(t *testing.T) {
	v := viper.GetViper()
	v.Set(xlDeployUser, "admin")
	v.Set(xlReleaseUser, "admin")
	v.Set(xlReleasePassword, "1234")
	v.Set(xlDeployPassword, "12345")
	v.Set(xlDeployUrl, "http://teshost/xl-deploy")
	v.Set(xlReleaseUrl, "http://teshost/xl-release")
	t.Run("should write config to file", func(t *testing.T) {
		err := writeConfig(v, "config.yaml")
		assert.Nil(t, err)
		assert.FileExists(t, "config.yaml")
		config := GetFileContentSafe("config.yaml")
		assert.Contains(t, config, "admin")
		assert.Contains(t, config, "1234")
		assert.Contains(t, config, "12345")
		assert.Contains(t, config, "http://teshost/xl-deploy")
		assert.Contains(t, config, "http://teshost/xl-release")
	})
	err := os.Remove("config.yaml")
	assert.Nil(t, err)
}

func TestUpdateXebialabsConfig(t *testing.T) {
	var client *kubernetes.Clientset
	GetIp = func(client *kubernetes.Clientset) (string, error) {
		return "http://testhost", nil
	}
	tempWriteConfig := writeConfig
	writeConfig = func(v *viper.Viper, configPath string) error {
		return nil
	}
	t.Run("should update both when both present", func(t *testing.T) {
		v := viper.New()
		v.SetConfigType("yaml")
		answerAllPositive := map[string]string{
			"InstallXLD":   "true",
			"InstallXLR":   "true",
			"XlrAdminPass": "12345",
			"XldAdminPass": "1234",
		}
		err := updateXebialabsConfig(client, answerAllPositive, v)
		assert.Nil(t, err)
		assert.Equal(t, "http://testhost/xl-deploy", v.GetString(xlDeployUrl))
		assert.Equal(t, "http://testhost/xl-release", v.GetString(xlReleaseUrl))
		assert.Equal(t, "admin", v.GetString(xlDeployUser))
		assert.Equal(t, "admin", v.GetString(xlReleaseUser))
		assert.Equal(t, "1234", v.GetString(xlDeployPassword))
		assert.Equal(t, "12345", v.GetString(xlReleasePassword))
	})
	t.Run("should not update XLD config when XLD was not deployed", func(t *testing.T) {
		v := viper.New()
		v.SetConfigType("yaml")
		answerInstallXLD := map[string]string{
			"InstallXLD":   "true",
			"InstallXLR":   "false",
			"XldAdminPass": "1234",
		}
		err := updateXebialabsConfig(client, answerInstallXLD, v)
		assert.Nil(t, err)
		assert.Equal(t, "http://testhost/xl-deploy", v.GetString(xlDeployUrl))
		assert.Equal(t, "", v.GetString(xlReleaseUrl))
		assert.Equal(t, "admin", v.GetString(xlDeployUser))
		assert.Equal(t, "", v.GetString(xlReleaseUser))
		assert.Equal(t, "1234", v.GetString(xlDeployPassword))
		assert.Equal(t, "", v.GetString(xlReleasePassword))
	})

	t.Run("should not update XLR config when XLR was not deployed", func(t *testing.T) {
		v := viper.New()
		v.SetConfigType("yaml")
		answerInstallXLR := map[string]string{
			"InstallXLD":   "false",
			"InstallXLR":   "true",
			"XlrAdminPass": "12345",
		}
		err := updateXebialabsConfig(client, answerInstallXLR, v)
		assert.Nil(t, err)
		assert.Equal(t, "", v.GetString(xlDeployUrl))
		assert.Equal(t, "http://testhost/xl-release", v.GetString(xlReleaseUrl))
		assert.Equal(t, "", v.GetString(xlDeployUser))
		assert.Equal(t, "admin", v.GetString(xlReleaseUser))
		assert.Equal(t, "", v.GetString(xlDeployPassword))
		assert.Equal(t, "12345", v.GetString(xlReleasePassword))
	})
	t.Run("should not update XLR or XLR", func(t *testing.T) {
		v := viper.New()
		v.SetConfigType("yaml")
		answerNothingChanged := map[string]string{
			"InstallXLD": "false",
			"InstallXLR": "false",
		}
		err := updateXebialabsConfig(client, answerNothingChanged, v)
		assert.Nil(t, err)
		assert.Equal(t, "", v.GetString(xlDeployUrl))
		assert.Equal(t, "", v.GetString(xlReleaseUrl))
		assert.Equal(t, "", v.GetString(xlDeployUser))
		assert.Equal(t, "", v.GetString(xlReleaseUser))
		assert.Equal(t, "", v.GetString(xlDeployPassword))
		assert.Equal(t, "", v.GetString(xlReleasePassword))
	})
	writeConfig = tempWriteConfig
}

func GetFileContentSafe(filePath string) string {
	f, err := ioutil.ReadFile(filePath)
	if err != nil {
		fmt.Println("Error getting file contents", err.Error())
		return ""
	}

	return string(f)
}

func Test_shouldUpdateConfig(t *testing.T) {
	GetIp = func(client *kubernetes.Clientset) (string, error) {
		return "http://testhost", nil
	}
	tests := []struct {
		name    string
		answers map[string]string
		v       *viper.Viper
		want    bool
		wantErr bool
	}{
		{
			"should return false when XLD/XLR not installed",
			map[string]string{
				"InstallXLD": "false",
				"InstallXLR": "false",
			},
			viper.New(),
			false,
			false,
		},
		{
			"should return false when XLD/XLR is installed but config didn't change",
			map[string]string{
				"InstallXLD":   "true",
				"InstallXLR":   "true",
				"XlrAdminPass": "12345",
				"XldAdminPass": "1234",
			},
			func() *viper.Viper {
				v := viper.New()
				v.Set(xlReleaseUrl, "http://testhost/xl-release")
				v.Set(xlReleaseUser, "admin")
				v.Set(xlReleasePassword, "12345")
				v.Set(xlDeployUrl, "http://testhost/xl-deploy")
				v.Set(xlDeployUser, "admin")
				v.Set(xlDeployPassword, "1234")
				return v
			}(),
			false,
			false,
		},
		{
			"should return true when XLD/XLR is installed and config changed",
			map[string]string{
				"InstallXLD":   "true",
				"InstallXLR":   "true",
				"XlrAdminPass": "12345",
				"XldAdminPass": "1234",
			},
			func() *viper.Viper {
				v := viper.New()
				v.Set(xlReleaseUrl, "http://testhostold/xl-release")
				v.Set(xlReleaseUser, "admin")
				v.Set(xlReleasePassword, "12345")
				v.Set(xlDeployUrl, "http://testhost/xl-deploy")
				v.Set(xlDeployUser, "admin")
				v.Set(xlDeployPassword, "1234")
				return v
			}(),
			true,
			false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := shouldUpdateConfig(nil, tt.answers, tt.v)
			if (err != nil) != tt.wantErr {
				t.Errorf("shouldUpdateConfig() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("shouldUpdateConfig() = %v, want %v", got, tt.want)
			}
		})
	}
}

package xl

import (
	"errors"
	"fmt"
	"github.com/spf13/viper"
	"net/url"
)

func BuildContext(v *viper.Viper) (*Context, error) {
	var xlDeploy *XLDeployServer
	var xlRelease *XLReleaseServer

	serverConfig, xlDeployPasswordWasNotObfuscrypted, err := readServerConfig(v, "xl-deploy")
	if err != nil {
		return nil, err
	}
	if serverConfig != nil {
		xlDeploy = &XLDeployServer{Server: serverConfig}
		xlDeploy.ApplicationsHome = v.GetString("xl-deploy.applications-home")
		xlDeploy.ConfigurationHome = v.GetString("xl-deploy.configuration-home")
		xlDeploy.EnvironmentsHome = v.GetString("xl-deploy.environments-home")
		xlDeploy.InfrastructureHome = v.GetString("xl-deploy.infrastructure-home")
	}

	serverConfig, xlReleasePasswordWasNotObfuscrypted, err := readServerConfig(v, "xl-release")
	if err != nil {
		return nil, err
	}
	if serverConfig != nil {
		xlRelease = &XLReleaseServer{Server: serverConfig}
		xlRelease.Home = v.GetString("xl-release.home")
	}

	if xlDeployPasswordWasNotObfuscrypted || xlReleasePasswordWasNotObfuscrypted {
		writeObfuscryptPasswords(v, []string{"xl-deploy", "xl-release"})
	}

	return &Context{
		XLDeploy:  xlDeploy,
		XLRelease: xlRelease,
	}, nil
}

func readServerConfig(v *viper.Viper, prefix string) (*SimpleHTTPServer, bool, error) {
	urlstring := v.GetString(fmt.Sprintf("%s.url", prefix))
	if urlstring == "" {
		return nil, false, nil
	}

	var passwordWasNotObfuscrypted = false
	u, err := url.ParseRequestURI(urlstring)
	if err != nil {
		return nil, false, err
	}

	username := v.GetString(fmt.Sprintf("%s.username", prefix))
	if username == "" {
		return nil, false, errors.New(fmt.Sprintf("configuration property %s.username is required if %s.url is set", prefix, prefix))
	}

	password := v.GetString(fmt.Sprintf("%s.password", prefix))
	if password == "" {
		return nil, false, errors.New(fmt.Sprintf("configuration property %s.password is required if %s.url is set", prefix, prefix))
	}

	deobfuscrypted, err := Deobfuscrypt(password)
	if err == nil {
		password = deobfuscrypted
	} else {
		obfuscrypted, err := Obfuscrypt(password)
		if err != nil {
			return nil, false, err
		}

		v.Set(fmt.Sprintf("%s.password", prefix), obfuscrypted)
		passwordWasNotObfuscrypted = true
	}

	return &SimpleHTTPServer{
		Url:      *u,
		Username: username,
		Password: password,
	}, passwordWasNotObfuscrypted, nil
}

func writeObfuscryptPasswords(v *viper.Viper, prefixes []string) error {

	configFile := v.ConfigFileUsed()
	if configFile != "" {
		// read original config
		configOnDisk := viper.New()
		configOnDisk.SetConfigFile(configFile)
		configOnDisk.ReadInConfig()

		// copy obfuscrypted passwords
		configDirty := false
		for _, prefix := range prefixes {
			configDirty = copyObfuscryptedPassword(configOnDisk, v, fmt.Sprintf("%s.password", prefix)) || configDirty
		}

		// write config if dirty
		if configDirty {
			err := configOnDisk.WriteConfig()
			if err == nil {
				Info("Saved config file %s\n", v.ConfigFileUsed())
			} else {
				return err
			}
		}
	}

	return nil
}

func copyObfuscryptedPassword(to *viper.Viper, from *viper.Viper, key string) bool {
	fromPassword := from.GetString(key)
	if fromPassword != "" {
		toPassword := to.GetString(key)
		if toPassword != "" && toPassword != fromPassword {
			to.Set(key, fromPassword)
			return true
		}
	}
	return false
}

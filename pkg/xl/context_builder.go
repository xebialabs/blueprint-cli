package xl

import (
	"errors"
	"fmt"
	"github.com/spf13/viper"
	"net/url"
	"os"
	"strings"
)

func BuildContext(v *viper.Viper, valueOverrides *map[string]string, secretOverrides *map[string]string) (*Context, error) {
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

	values, _, err := readValues(v, "values", "XL_VALUE_", valueOverrides, false)
	if err != nil {
		return nil, err
	}

	secrets, secretsWereNotObfuscrypted, err := readValues(v, "secrets", "XL_SECRET_", secretOverrides, true)
	if err != nil {
		return nil, err
	}

	if xlDeployPasswordWasNotObfuscrypted || xlReleasePasswordWasNotObfuscrypted || secretsWereNotObfuscrypted {
		obfuscryptPasswordsOnDisk(v)
	}

	return &Context{
		XLDeploy:  xlDeploy,
		XLRelease: xlRelease,
		values:    values,
		secrets:   secrets,
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
		passwordWasNotObfuscrypted = true
	}

	return &SimpleHTTPServer{
		Url:      *u,
		Username: username,
		Password: password,
	}, passwordWasNotObfuscrypted, nil
}

func readValues(v *viper.Viper, configName string, envPrefix string, flagOverrides *map[string]string, deobfuscryptSecrets bool) (map[string]string, bool, error) {
	var secretsWereNotObfuscrypted = false

	m := v.GetStringMapString(configName)
	for key, value := range m {
		deobfuscrypted, err := Deobfuscrypt(value)
		if err == nil {
			m[key] = deobfuscrypted
		} else {
			secretsWereNotObfuscrypted = true
		}
	}

	for _, envOverride := range os.Environ() {
		eqPos := strings.Index(envOverride, "=")
		if eqPos == -1 {
			continue
		}
		key := envOverride[:eqPos]
		value := envOverride[eqPos+1:]

		if strings.HasPrefix(key, envPrefix) {
			m[key[len(envPrefix):]] = value
		}
	}

	if flagOverrides != nil {
		for k, v := range *flagOverrides {
			m[k] = v
		}
	}

	return m, secretsWereNotObfuscrypted, nil
}

func obfuscryptPasswordsOnDisk(v *viper.Viper) error {

	configFile := v.ConfigFileUsed()
	if configFile != "" {
		// read original config
		configOnDisk := viper.New()
		configOnDisk.SetConfigFile(configFile)
		configOnDisk.ReadInConfig()

		// obfuscryptIfNeeded unobfuscrypted passwords
		configDirty := false

		for _, prefix := range []string{"xl-deploy", "xl-release"} {
			key := fmt.Sprintf("%s.password", prefix)
			if configOnDisk.IsSet(key) {
				value := configOnDisk.GetString(key)

				dirty, obfuscrypted, err := obfuscryptIfNeeded(value)
				if err != nil {
					return err
				}
				if dirty {
					configOnDisk.Set(key, obfuscrypted)
					configDirty = true
				}
			}
		}

		if configOnDisk.IsSet("secrets") {
			secrets := configOnDisk.GetStringMapString("secrets")
			for key, value := range secrets {
				dirty, obfuscrypted, err := obfuscryptIfNeeded(value)
				if err != nil {
					return err
				}
				if dirty {
					secrets[key] = obfuscrypted
					configDirty = true
				}
			}
			// reset the secrets map to ensure it does not get unfolded
			configOnDisk.Set("secrets", secrets)
		}

		//
		// write config if dirty
		if configDirty {
			// ensure the values map does not get unfolded
			if configOnDisk.IsSet("values") {
				values := configOnDisk.GetStringMapString("values")
				configOnDisk.Set("values", values)
			}

			err := configOnDisk.WriteConfig()
			if err == nil {
				Info("Configuration file %s saved\n", v.ConfigFileUsed())
			} else {
				return err
			}
		}
	}

	return nil
}

func obfuscryptIfNeeded(value string) (bool, string, error){
	_, err := Deobfuscrypt(value)
	if err == nil {
		return false, value, nil
	} else {
		obfuscrypted, err := Obfuscrypt(value)
		if err != nil {
			return false, "", err
		}
		return true, obfuscrypted, nil
	}
}
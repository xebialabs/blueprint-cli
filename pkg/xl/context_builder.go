package xl

import (
	"errors"
	"fmt"
	"github.com/spf13/viper"
	"net/url"
	"os"
	"strings"
	"regexp"
	"github.com/magiconair/properties"
)

func BuildContext(v *viper.Viper, valueOverrides *map[string]string, valueFiles []string) (*Context, error) {
	var xlDeploy *XLDeployServer
	var xlRelease *XLReleaseServer

	serverConfig, err := readServerConfig(v, "xl-deploy")
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

	serverConfig2, err := readServerConfig(v, "xl-release")
	if err != nil {
		return nil, err
	}
	if serverConfig2 != nil {
		xlRelease = &XLReleaseServer{Server: serverConfig2}
		xlRelease.Home = v.GetString("xl-release.home")
	}

	values, err := mergeValues("XL_VALUE_", valueOverrides, valueFiles)
	if err != nil {
		return nil, err
	}

	return &Context{
		XLDeploy:  xlDeploy,
		XLRelease: xlRelease,
		values:    values,
	}, nil
}

func setEnvVariableIfNotPresent(key string, value string) {
	_, present := os.LookupEnv(key)
	if !present {
		os.Setenv(key, value)
	}
}

func processServerCredentials(serverKind string) error {
	credentialsEnvKey := fmt.Sprintf("XL_%s_CREDENTIALS", serverKind)
	usernameEnvKey := fmt.Sprintf("XL_%s_USERNAME", serverKind)
	passwordEnvKey := fmt.Sprintf("XL_%s_PASSWORD", serverKind)

	credentials, credentialsPresent := os.LookupEnv(credentialsEnvKey)
	if credentialsPresent {
		credentialsArray := strings.Split(credentials, ":")
		if len(credentialsArray) != 2 {
			return errors.New(fmt.Sprintf("environment variable %s has an invalid format. It must container a username and a password separated by a colon", credentialsEnvKey))
		}

		setEnvVariableIfNotPresent(usernameEnvKey, credentialsArray[0])
		setEnvVariableIfNotPresent(passwordEnvKey, credentialsArray[1])
	}
	return nil
}

func ProcessCredentials() error {
	err := processServerCredentials("DEPLOY")
	if err != nil {
		return err
	}
	return processServerCredentials("RELEASE")
}

func readServerConfig(v *viper.Viper, prefix string) (*SimpleHTTPServer, error) {
	urlstring := v.GetString(fmt.Sprintf("%s.url", prefix))
	if urlstring == "" {
		return nil, nil
	}

	u, err := url.ParseRequestURI(urlstring)
	if err != nil {
		return nil, err
	}

	username := v.GetString(fmt.Sprintf("%s.username", prefix))
	if username == "" {
		return nil, errors.New(fmt.Sprintf("configuration property %s.username is required if %s.url is set", prefix, prefix))
	}

	password := v.GetString(fmt.Sprintf("%s.password", prefix))
	if password == "" {
		return nil, errors.New(fmt.Sprintf("configuration property %s.password is required if %s.url is set", prefix, prefix))
	}

	return &SimpleHTTPServer{
		Url:      *u,
		Username: username,
		Password: password,
	}, nil
}

func mergeValues(envPrefix string, flagOverrides *map[string]string, valueFiles []string) (map[string]string, error) {

	m := properties.MustLoadFiles(valueFiles, properties.UTF8, false).Map()

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

	var validKeyRegex = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_]*$`)
	for k, _ := range m {
		if !validKeyRegex.MatchString(k) {
			return nil, errors.New(fmt.Sprintf("the name of the value %s is invalid. It must start with an alphabetical character or an underscore and be followed by zero or more alphanumerical characters or underscores", k))
		}
	}

	return m, nil
}

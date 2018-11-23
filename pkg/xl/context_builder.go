package xl

import (
	"fmt"
	"net/url"
	"os"
	"regexp"
	"strings"

	"github.com/xebialabs/xl-cli/pkg/models"
	"github.com/magiconair/properties"
	"github.com/spf13/viper"
	"github.com/spf13/cobra"
	"github.com/thoas/go-funk"
)

func PrepareRootCmdFlags(command *cobra.Command, cfgFile *string) {
	rootFlags := command.PersistentFlags()
	rootFlags.StringVar(cfgFile, "config", "", "config file (default: $HOME/.xebialabs/config.yaml)")
	rootFlags.BoolVarP(&IsQuiet, "quiet", "q", false, "suppress all output, except for errors")
	rootFlags.BoolVarP(&IsVerbose, "verbose", "v", false, "verbose output")
	rootFlags.String(models.FlagXldUrl, DefaultXlDeployUrl, "URL to access the XL Deploy server")
	rootFlags.String(models.FlagXldUser, DefaultXlDeployUsername, "Username to access the XL Deploy server")
	rootFlags.String(models.FlagXldPass, DefaultXlDeployPassword, "Password to access the XL Deploy server")
	viper.BindPFlag(models.ViperKeyXLdUrl, rootFlags.Lookup(models.FlagXldUrl))
	viper.BindPFlag(models.ViperKeyXLdUsername, rootFlags.Lookup(models.FlagXldUser))
	viper.BindPFlag(models.ViperKeyXLdPassword, rootFlags.Lookup(models.FlagXldPass))

	rootFlags.String(models.FlagXlrUrl, DefaultXlReleaseUrl, "URL to access the XL Release server")
	rootFlags.String(models.FlagXlrUser, DefaultXlReleaseUsername, "Username to access the XL Release server")
	rootFlags.String(models.FlagXlrPass, DefaultXlReleasePassword, "Password to access the XL Release server")
	viper.BindPFlag(models.ViperKeyXLrUrl, rootFlags.Lookup(models.FlagXlrUrl))
	viper.BindPFlag(models.ViperKeyXLrUsername, rootFlags.Lookup(models.FlagXlrUser))
	viper.BindPFlag(models.ViperKeyXLrPassword, rootFlags.Lookup(models.FlagXlrPass))
}

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

	// Get cobra flag values
	configDefaults := getServerConfigDefaults(v)

	values, err := mergeValues("XL_VALUE_", valueOverrides, valueFiles, configDefaults)
	if err != nil {
		return nil, err
	}

	templateRegistries, err := getTemplateRegistries(v)
	if err != nil {
		return nil, err
	}

	return &Context{
		XLDeploy:           xlDeploy,
		XLRelease:          xlRelease,
		values:             values,
		TemplateRegistries: templateRegistries,
	}, nil
}

func getTemplateRegistries(v *viper.Viper) ([]TemplateRegistry, error) {
	registries := []TemplateRegistry{}
	// the type of this is []interface{}{map[interface{}]interface{}, map[interface{}]interface{}}
	yamlVal := v.Get("template-registries")

	switch typeVal := yamlVal.(type) {
	case []interface{}:
		for _, v := range typeVal {
			registry := TemplateRegistry{}
			registryMap, ok := v.(map[interface{}]interface{})
			if ok {
				name, ok := registryMap["name"].(string)
				if ok {
					registry.Name = name
				}
				urlval, ok := registryMap["url"].(string)
				if ok {
					parsedurl, err := url.ParseRequestURI(urlval)
					if err != nil {
						return nil, err
					}
					registry.URL = *parsedurl
				} else {
					return nil, fmt.Errorf("invalid template registry configuration. URL is required")
				}
				username, ok := registryMap["username"].(string)
				if ok {
					registry.Username = username
				}
				password, ok := registryMap["password"].(string)
				if ok {
					registry.Password = password
				}
			}
			registries = append(registries, registry)

		}
	}
	return registries, nil
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
			return fmt.Errorf("environment variable %s has an invalid format. It must container a username and a password separated by a colon", credentialsEnvKey)
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
		return nil, fmt.Errorf("configuration property %s.username is required if %s.url is set", prefix, prefix)
	}

	password := v.GetString(fmt.Sprintf("%s.password", prefix))
	if password == "" {
		return nil, fmt.Errorf("configuration property %s.password is required if %s.url is set", prefix, prefix)
	}

	return &SimpleHTTPServer{
		Url:      *u,
		Username: username,
		Password: password,
	}, nil
}

func addCmdFlagValueToMap(cmd *cobra.Command, getOnlyChanged bool, flagName string, key string, m map[string]string) {
	flag := cmd.Flag(flagName)
	if flag == nil {
		return
	}
	if getOnlyChanged {
		if !flag.Changed {
			return
		}
	}
	m[key] = flag.Value.String()
}


func getServerConfigDefaults(v *viper.Viper) *map[string]string {
	m := make(map[string]string)
	m["XL_DEPLOY_URL"] = v.GetString(models.ViperKeyXLdUrl)
	m["XL_DEPLOY_USERNAME"] = v.GetString(models.ViperKeyXLdUsername)
	m["XL_DEPLOY_PASSWORD"] = v.GetString(models.ViperKeyXLdPassword)
	m["XL_RELEASE_URL"] = v.GetString(models.ViperKeyXLrUrl)
	m["XL_RELEASE_USERNAME"] = v.GetString(models.ViperKeyXLrUsername)
	m["XL_RELEASE_PASSWORD"] = v.GetString(models.ViperKeyXLrPassword)
	return &m
}

func mergeValues(envPrefix string, flagOverrides *map[string]string, valueFiles []string, configDefaults *map[string]string) (map[string]string, error) {
	/*
	Value merging priority list, first being least priority
	- GLOBAL CONFIG YAML
	- LOCAL VALUE FILES
	- ENV VARS
	- FLAG VALUES - VIPER OVERRIDES
	- COBRA CMD FLAGS - If changed
	*/
	m := make(map[string]string)

	// Add defaults for server configuration
	for k, v := range *configDefaults {
		m[k] = v
	}

	// Add all values files variables
	funk.ForEach(valueFiles, func(valueFile string) {
		Verbose("Using value file %s\n", valueFile)
	})
	valuesMap := properties.MustLoadFiles(valueFiles, properties.UTF8, false).Map()
	for k, v := range valuesMap {
		m[k] = v
	}

	// Add environment variable values
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

	// Add Viper flag overrides
	if flagOverrides != nil {
		for k, v := range *flagOverrides {
			m[k] = v
		}
	}

	// Validate keys
	var validKeyRegex = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_]*$`)
	for k := range m {
		if !validKeyRegex.MatchString(k) {
			return nil, fmt.Errorf("the name of the value %s is invalid. It must start with an alphabetical character or an underscore and be followed by zero or more alphanumerical characters or underscores", k)
		}
	}

	return m, nil
}

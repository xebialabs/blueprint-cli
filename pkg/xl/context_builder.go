package xl

import (
	"fmt"
	"net/url"
	"os"
	"regexp"
	"strings"

	"sort"

	"github.com/magiconair/properties"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/thoas/go-funk"
	"github.com/xebialabs/xl-cli/pkg/models"
)

func PrepareRootCmdFlags(command *cobra.Command, cfgFile *string) {
	rootFlags := command.PersistentFlags()
	rootFlags.StringVar(cfgFile, "config", "", "config file (default: $HOME/.xebialabs/config.yaml)")
	rootFlags.BoolVarP(&IsQuiet, "quiet", "q", false, "suppress all output, except for errors")
	rootFlags.BoolVarP(&IsVerbose, "verbose", "v", false, "verbose output")
	rootFlags.String(models.FlagXldUrl, models.DefaultXlDeployUrl, "URL to access the XL Deploy server")
	rootFlags.String(models.FlagXldUser, models.DefaultXlDeployUsername, "Username to access the XL Deploy server")
	rootFlags.String(models.FlagXldPass, models.DefaultXlDeployPassword, "Password to access the XL Deploy server")
	viper.BindPFlag(models.ViperKeyXLDUrl, rootFlags.Lookup(models.FlagXldUrl))
	viper.BindPFlag(models.ViperKeyXLDUsername, rootFlags.Lookup(models.FlagXldUser))
	viper.BindPFlag(models.ViperKeyXLDPassword, rootFlags.Lookup(models.FlagXldPass))

	rootFlags.String(models.FlagXlrUrl, models.DefaultXlReleaseUrl, "URL to access the XL Release server")
	rootFlags.String(models.FlagXlrUser, models.DefaultXlReleaseUsername, "Username to access the XL Release server")
	rootFlags.String(models.FlagXlrPass, models.DefaultXlReleasePassword, "Password to access the XL Release server")
	viper.BindPFlag(models.ViperKeyXLRUrl, rootFlags.Lookup(models.FlagXlrUrl))
	viper.BindPFlag(models.ViperKeyXLRUsername, rootFlags.Lookup(models.FlagXlrUser))
	viper.BindPFlag(models.ViperKeyXLRPassword, rootFlags.Lookup(models.FlagXlrPass))

	rootFlags.String(models.FlagBlueprintRepositoryUrl, models.DefaultBlueprintRepositoryUrl, "URL for the blueprint repository")
	rootFlags.String(models.FlagBlueprintRepositoryUsername, models.DefaultBlueprintRepositoryUsername, "Username for the blueprint repository")
	rootFlags.String(models.FlagBlueprintRepositoryPassword, models.DefaultBlueprintRepositoryPassword, "Password for the blueprint repository")
	viper.BindPFlag(models.ViperKeyBlueprintRepositoryUrl, rootFlags.Lookup(models.FlagBlueprintRepositoryUrl))
	viper.BindPFlag(models.ViperKeyBlueprintRepositoryUsername, rootFlags.Lookup(models.FlagBlueprintRepositoryUsername))
	viper.BindPFlag(models.ViperKeyBlueprintRepositoryPassword, rootFlags.Lookup(models.FlagBlueprintRepositoryPassword))
}

func BuildContext(v *viper.Viper, valueOverrides *map[string]string, valueFiles []string) (*Context, error) {
	var xlDeploy *XLDeployServer
	var xlRelease *XLReleaseServer
	var blueprintRepository BlueprintRepository

	xldServerConfig, err := readServerConfig(v, "xl-deploy", true)
	if err != nil {
		return nil, err
	}
	if xldServerConfig != nil {
		xlDeploy = &XLDeployServer{Server: xldServerConfig}
		xlDeploy.ApplicationsHome = v.GetString("xl-deploy.applications-home")
		xlDeploy.ConfigurationHome = v.GetString("xl-deploy.configuration-home")
		xlDeploy.EnvironmentsHome = v.GetString("xl-deploy.environments-home")
		xlDeploy.InfrastructureHome = v.GetString("xl-deploy.infrastructure-home")
	}

	xlrServerConfig, err := readServerConfig(v, "xl-release", true)
	if err != nil {
		return nil, err
	}
	if xlrServerConfig != nil {
		xlRelease = &XLReleaseServer{Server: xlrServerConfig}
		xlRelease.Home = v.GetString("xl-release.home")
	}

	blueprintRepositoryServer, err := readServerConfig(v, "blueprint-repository", false)
	if err != nil {
		return nil, err
	}
	if blueprintRepositoryServer != nil {
		blueprintRepository = BlueprintRepository{Server: *blueprintRepositoryServer}
	}

	// Get cobra flag values
	configDefaults := getServerConfigDefaults(v)

	values, err := mergeValues("XL_VALUE_", valueOverrides, valueFiles, configDefaults)
	if err != nil {
		return nil, err
	}

	return &Context{
		XLDeploy:            xlDeploy,
		XLRelease:           xlRelease,
		BlueprintRepository: blueprintRepository,
		values:              values,
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

func readServerConfig(v *viper.Viper, prefix string, credentialsRequired bool) (*SimpleHTTPServer, error) {
	urlString := v.GetString(fmt.Sprintf("%s.url", prefix))
	if urlString == "" {
		return nil, nil
	}

	u, err := url.ParseRequestURI(urlString)
	if err != nil {
		return nil, err
	}

	username := v.GetString(fmt.Sprintf("%s.username", prefix))
	if credentialsRequired && username == "" {
		return nil, fmt.Errorf("configuration property %s.username is required if %s.url is set", prefix, prefix)
	}

	password := v.GetString(fmt.Sprintf("%s.password", prefix))
	if credentialsRequired && password == "" {
		return nil, fmt.Errorf("configuration property %s.password is required if %s.url is set", prefix, prefix)
	}

	return &SimpleHTTPServer{
		Url:      *u,
		Username: username,
		Password: password,
	}, nil

	return &SimpleHTTPServer{
		Url: *u,
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
	m["XL_DEPLOY_URL"] = v.GetString(models.ViperKeyXLDUrl)
	m["XL_DEPLOY_USERNAME"] = v.GetString(models.ViperKeyXLDUsername)
	m["XL_DEPLOY_PASSWORD"] = v.GetString(models.ViperKeyXLDPassword)
	m["XL_RELEASE_URL"] = v.GetString(models.ViperKeyXLRUrl)
	m["XL_RELEASE_USERNAME"] = v.GetString(models.ViperKeyXLRUsername)
	m["XL_RELEASE_PASSWORD"] = v.GetString(models.ViperKeyXLRPassword)
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

	// print values without defaults
	if IsVerbose {
		printValues(m)
	}

	finalMap := make(map[string]string)

	// Add defaults for server configuration
	for k, v := range *configDefaults {
		finalMap[k] = v
	}

	// Reapply all values on top of final map
	for k, v := range m {
		finalMap[k] = v
	}

	return finalMap, nil
}

func printValues(values map[string]string) {
	var keys = funk.Keys(values).([]string)
	sort.Strings(keys)
	Verbose("Values:\n")
	funk.ForEach(keys, func(key string) {
		Verbose("%s: %s\n", key, values[key])
	})
}

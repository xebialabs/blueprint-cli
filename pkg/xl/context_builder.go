package xl

import (
	"fmt"
	"os"
	"regexp"
	"sort"
	"strings"

	"github.com/magiconair/properties"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/thoas/go-funk"
	"github.com/xebialabs/xl-cli/pkg/blueprint"
	"github.com/xebialabs/xl-cli/pkg/models"
	"github.com/xebialabs/xl-cli/pkg/util"
)

// root

func PrepareRootCmdFlags(command *cobra.Command, cfgFile *string) {
	rootFlags := command.PersistentFlags()
	rootFlags.StringVar(cfgFile, "config", "", "config file (default: $HOME/.xebialabs/config.yaml)")
	rootFlags.BoolVarP(&util.IsQuiet, "quiet", "q", false, "suppress all output, except for errors")
	rootFlags.BoolVarP(&util.IsVerbose, "verbose", "v", false, "verbose output")

	rootFlags.String(models.FlagXldUrl, models.DefaultXlDeployUrl, "URL to access the XL Deploy server")
	rootFlags.String(models.FlagXldUser, models.DefaultXlDeployUsername, "Username to access the XL Deploy server")
	rootFlags.String(models.FlagXldPass, models.DefaultXlDeployPassword, "Password to access the XL Deploy server")
	rootFlags.String(models.FlagXldAuthMethod, models.DefaultXlDeployAuthMethod, "Authentication method to access the XL Deploy server")
	viper.BindPFlag(models.ViperKeyXLDUrl, rootFlags.Lookup(models.FlagXldUrl))
	viper.BindPFlag(models.ViperKeyXLDUsername, rootFlags.Lookup(models.FlagXldUser))
	viper.BindPFlag(models.ViperKeyXLDPassword, rootFlags.Lookup(models.FlagXldPass))
	viper.BindPFlag(models.ViperKeyXLDAuthMethod, rootFlags.Lookup(models.FlagXldAuthMethod))

	rootFlags.String(models.FlagXlrUrl, models.DefaultXlReleaseUrl, "URL to access the XL Release server")
	rootFlags.String(models.FlagXlrUser, models.DefaultXlReleaseUsername, "Username to access the XL Release server")
	rootFlags.String(models.FlagXlrPass, models.DefaultXlReleasePassword, "Password to access the XL Release server")
	rootFlags.String(models.FlagXlrAuthMethod, models.DefaultXlReleaseAuthMethod, "Authentication method to access the XL Release server")
	viper.BindPFlag(models.ViperKeyXLRUrl, rootFlags.Lookup(models.FlagXlrUrl))
	viper.BindPFlag(models.ViperKeyXLRUsername, rootFlags.Lookup(models.FlagXlrUser))
	viper.BindPFlag(models.ViperKeyXLRPassword, rootFlags.Lookup(models.FlagXlrPass))
	viper.BindPFlag(models.ViperKeyXLDPassword, rootFlags.Lookup(models.FlagXldPass))
	viper.BindPFlag(models.ViperKeyXLRAuthMethod, rootFlags.Lookup(models.FlagXlrAuthMethod))

	blueprint.SetRootFlags(rootFlags)
}

// Blueprints

func BuildContext(v *viper.Viper, valueOverrides *map[string]string, valueFiles []string, scmInfo *SCMInfo, CLIVersion string) (*Context, error) {
	var blueprintContext *blueprint.BlueprintContext

	configPath, err := util.DefaultConfigfilePath()
	if err != nil {
		return nil, err
	}

	blueprintContext, err = blueprint.ConstructBlueprintContext(v, configPath, CLIVersion)
	if err != nil {
		return nil, err
	}

	values, err := mergeValues("XL_VALUE_", valueOverrides, valueFiles, nil)
	if err != nil {
		return nil, err
	}

	return &Context{
		BlueprintContext: blueprintContext,
		values:           values,
	}, nil
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
		util.Verbose("Using value file %s\n", valueFile)
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
	if util.IsVerbose {
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
	util.Verbose("%sValues:\n", util.Indent1())
	if funk.IsEmpty(keys) {
		util.Verbose("%sEMPTY\n", util.Indent2())
	} else {
		funk.ForEach(keys, func(key string) {
			util.Verbose("%s%s: %s\n", util.Indent2(), key, values[key])
		})
	}
	util.Verbose("\n")
}

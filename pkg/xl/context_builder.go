package xl

import (
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/xebialabs/xl-blueprint/pkg/blueprint"
	"github.com/xebialabs/xl-blueprint/pkg/util"
)

// root
func PrepareRootCmdFlags(command *cobra.Command, cfgFile *string) {
	rootFlags := command.PersistentFlags()
	rootFlags.StringVar(cfgFile, "config", "", "config file (default: $HOME/.xebialabs/config.yaml)")
	rootFlags.BoolVarP(&util.IsQuiet, "quiet", "q", false, "suppress all output, except for errors")
	rootFlags.BoolVarP(&util.IsVerbose, "verbose", "v", false, "verbose output")

	blueprint.SetRootFlags(rootFlags)
}

// Blueprints
func BuildContext(v *viper.Viper, CLIVersion string) (*Context, error) {
	var blueprintContext *blueprint.BlueprintContext

	configPath, err := util.DefaultConfigfilePath()
	if err != nil {
		return nil, err
	}

	blueprintContext, err = blueprint.ConstructBlueprintContext(v, configPath, CLIVersion)
	if err != nil {
		return nil, err
	}

	if err != nil {
		return nil, err
	}

	return &Context{
		BlueprintContext: blueprintContext,
	}, nil
}

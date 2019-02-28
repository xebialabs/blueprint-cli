package blueprint

import (
	"fmt"
	"github.com/spf13/pflag"
	"github.com/spf13/viper"
	"github.com/xebialabs/xl-cli/pkg/models"
)

const (
	ContextPrefix = "blueprint-repository"

	FlagBlueprintRepositoryProvider  = ContextPrefix + "-provider"
	FlagBlueprintRepositoryName      = ContextPrefix + "-name"
	FlagBlueprintRepositoryOwner     = ContextPrefix + "-owner"
	FlagBlueprintRepositoryBranch    = ContextPrefix + "-branch"
	FlagBlueprintRepositoryToken     = ContextPrefix + "-token"

	ViperKeyBlueprintRepositoryProvider  = ContextPrefix + ".provider"
	ViperKeyBlueprintRepositoryName      = ContextPrefix + ".name"
	ViperKeyBlueprintRepositoryOwner     = ContextPrefix + ".owner"
	ViperKeyBlueprintRepositoryBranch    = ContextPrefix + ".branch"
	ViperKeyBlueprintRepositoryToken     = ContextPrefix + ".token"
)

type BlueprintContext struct {
	Provider string
	Name     string
	Owner    string
	Token    string
	Branch   string
}

func SetRootFlags(rootFlags *pflag.FlagSet) {
	rootFlags.String(FlagBlueprintRepositoryProvider, models.DefaultBlueprintRepositoryProvider, "Provider for the blueprint repository")
	rootFlags.String(FlagBlueprintRepositoryName, models.DefaultBlueprintRepositoryName, "Name of the blueprint repository")
	rootFlags.String(FlagBlueprintRepositoryOwner, models.DefaultBlueprintRepositoryOwner, "Owner of the blueprint repository")
	rootFlags.String(FlagBlueprintRepositoryBranch, models.DefaultBlueprintRepositoryBranch, "Branch of the blueprint repository")
	rootFlags.String(FlagBlueprintRepositoryToken, models.DefaultBlueprintRepositoryToken, "API Token for the blueprint repository")
	viper.BindPFlag(ViperKeyBlueprintRepositoryProvider, rootFlags.Lookup(FlagBlueprintRepositoryProvider))
	viper.BindPFlag(ViperKeyBlueprintRepositoryName, rootFlags.Lookup(FlagBlueprintRepositoryName))
	viper.BindPFlag(ViperKeyBlueprintRepositoryOwner, rootFlags.Lookup(FlagBlueprintRepositoryOwner))
	viper.BindPFlag(ViperKeyBlueprintRepositoryBranch, rootFlags.Lookup(FlagBlueprintRepositoryBranch))
	viper.BindPFlag(ViperKeyBlueprintRepositoryToken, rootFlags.Lookup(FlagBlueprintRepositoryToken))
}

func ConstructBlueprintContext(v *viper.Viper) (*BlueprintContext, error) {
	repoProvider, err := models.GetRepoProvider(v.GetString(fmt.Sprintf("%s.provider", ContextPrefix)))
	if err != nil {
		return nil, err
	}

	name := v.GetString(fmt.Sprintf("%s.name", ContextPrefix))
	if name == "" {
		return nil, fmt.Errorf("blueprint repo name cannot be empty")
	}

	branch := v.GetString(fmt.Sprintf("%s.branch", ContextPrefix))
	if branch == "" {
		branch = "master"
	}

	return &BlueprintContext{
		Provider: repoProvider,
		Name:     name,
		Owner:    v.GetString(fmt.Sprintf("%s.owner", ContextPrefix)),
		Token:    v.GetString(fmt.Sprintf("%s.token", ContextPrefix)),
		Branch:   branch,
	}, nil
}

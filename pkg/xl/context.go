package xl

import (
	"github.com/xebialabs/blueprint-cli/pkg/blueprint"
	"github.com/xebialabs/blueprint-cli/pkg/util"
)

type Context struct {
	BlueprintContext *blueprint.BlueprintContext
	values           map[string]string
}

func (c *Context) PrintConfiguration() {
	util.Info("Active Blueprint Context:\n  %s\n", (*c.BlueprintContext.ActiveRepo).GetInfo())
}

package xl

import (
	"github.com/xebialabs/xl-cli/pkg/blueprint"
	"github.com/xebialabs/xl-cli/pkg/util"
)

type Context struct {
	BlueprintContext *blueprint.BlueprintContext
	values           map[string]string
}

func (c *Context) PrintConfiguration() {
	util.Info("Active Blueprint Context:\n  %s\n", (*c.BlueprintContext.ActiveRepo).GetInfo())
}

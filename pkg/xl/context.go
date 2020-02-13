package xl

import (
	"github.com/xebialabs/xl-blueprint/pkg/blueprint"
	"github.com/xebialabs/xl-blueprint/pkg/util"
)

type Context struct {
	BlueprintContext *blueprint.BlueprintContext
	values           map[string]string
}

func (c *Context) PrintConfiguration() {
	util.Info("Active Blueprint Context:\n  %s\n", (*c.BlueprintContext.ActiveRepo).GetInfo())
}

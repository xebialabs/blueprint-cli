package xl

import (
	"time"

	"github.com/xebialabs/xl-cli/pkg/blueprint"
	"github.com/xebialabs/xl-cli/pkg/util"
)

type ChangedIds struct {
	Kind    string
	Created *[]string
	Updated *[]string
}

type CiValidationError struct {
	CiId         string
	PropertyName string
	Message      string
}

type PermissionError struct {
	CiId       string
	Permission string
}

type DocumentFieldError struct {
	Field   string
	Problem string
}

type Errors struct {
	Validation *[]CiValidationError
	Permission *[]PermissionError
	Document   *DocumentFieldError
	Generic    *string
}

type TaskInfo struct {
	Id          string
	Description string
	Started     bool
}

type Changes struct {
	Ids  *[]ChangedIds
	Task *TaskInfo
}

type AsCodeResponse struct {
	Changes *Changes
	Errors  *Errors
	RawBody string
}

type Context struct {
	BlueprintContext *blueprint.BlueprintContext
	values           map[string]string
	scmInfo          *SCMInfo
}

type CurrentStep struct {
	Name      string
	State     string
	Automated bool
}

type TaskState struct {
	State        string
	CurrentSteps []CurrentStep
}

type SCMInfo struct {
	filename  string
	scmType   string
	remote    string
	commit    string
	author    string
	date      time.Time
	message   string
	localPath string
}

func (c *Context) PrintConfiguration() {
	util.Info("Active Blueprint Context:\n  %s\n", (*c.BlueprintContext.ActiveRepo).GetInfo())
}

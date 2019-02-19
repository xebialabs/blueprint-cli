package cmd

import (
	"fmt"
	"github.com/spf13/cobra"
	"github.com/xebialabs/xl-cli/pkg/util"
	"github.com/xebialabs/xl-cli/pkg/xl"
	"path/filepath"
	"time"
)

var applyFilenames []string
var applyValues map[string]string
var applyDetach bool

var applyCmd = &cobra.Command{
	Use:   "apply",
	Short: "Apply configuration changes",
	Long:  `Apply configuration changes`,
	Run: func(cmd *cobra.Command, args []string) {
		DoApply(applyFilenames)
	},
}

func printIds(op string, ids *[]string) {
	if ids != nil && len(*ids) > 0 {
		for _, id := range *ids {
			util.Info(fmt.Sprintf("%s %s\n", op, id))
		}
	}
}

func printChangedIds(entityName string, ids *xl.ChangedIds) {
	if ids != nil {
		printIds(fmt.Sprintf("Created %s:", entityName), ids.Created)
		printIds(fmt.Sprintf("Updated %s:", entityName), ids.Updated)
	}
}

func printTaskInfo(task *xl.TaskInfo) {
	if task != nil {
		util.Info(fmt.Sprintf("Task [%s] started (%s)\n", task.Description, task.Id))
	}
}

func printChanges(changes *xl.Changes) {
	if changes != nil {
		printTaskInfo(changes.Task)
		printChangedIds("ci", changes.Cis)
		printChangedIds("user", changes.Users)
		printChangedIds("permission", changes.Permissions)
		printChangedIds("role", changes.Roles)
	}
}

func requestTaskId(context *xl.Context, doc *xl.Document, taskId string) (*xl.TaskState, error) {
	server, err := context.GetDocumentHandlingServer(doc)
	if err != nil {
		return nil, err
	}

	util.Verbose("Checking task state... ")
	state, serr := server.GetTaskStatus(taskId)
	if serr != nil {
		return nil, serr
	}
	util.Verbose("[%s]\n", state.State)

	if util.IsVerbose {
		if len(state.CurrentSteps) > 0 {
			util.Verbose("### Currently active task steps:\n")
			for _, step := range state.CurrentSteps {
				util.Verbose("### %s [%s]\n", step.Name, step.State)
			}
		}
	} else {
		util.Info(".")
	}
	return state, nil
}

func waitForTasks(context *xl.Context, doc *xl.Document, changes *xl.Changes, shouldDetach bool) {
	if changes != nil && changes.Task != nil && !shouldDetach {
		util.Info("Waiting for task (%s)\n", changes.Task.Id)
		result, err := requestTaskId(context, doc, changes.Task.Id)
		for err == nil {
			switch result.State {
			case "COMPLETED":
				fallthrough
			case "DONE":
				util.Verbose("Done.")
				util.Info("\n")
				return

			case "IN_PROGRESS":
				for _, step := range result.CurrentSteps {
					if !step.Automated {
						util.Fatal("\nUnable to complete the task (%s) automatically as it's current active step is manual.\n", changes.Task.Id)
					}
				}

			case "FAILING":
				fallthrough
			case "CANCELING":
				fallthrough
			case "CANCELLED":
				fallthrough
			case "FAILED":
				fallthrough
			case "STOPPED":
				fallthrough
			case "ABORTED":
				util.Fatal("\nUnable to complete the task (%s) automatically as it's state became [%s]. The task will be rolled back.\n", changes.Task.Id, result.State)
			}
			time.Sleep(2 * time.Second)
			result, err = requestTaskId(context, doc, changes.Task.Id)
		}
		if err != nil {
			util.Fatal("\nError waiting for task %s, %s\n", changes.Task.Id, err)
		}
	}
}

func applyDocument(context *xl.Context, fileWithDocs xl.FileWithDocuments, doc *xl.Document, shouldDetach bool) {
	applyDir := filepath.Dir(fileWithDocs.FileName)
	changes, err := context.ProcessSingleDocument(doc, applyDir)
	printChanges(changes)
	waitForTasks(context, doc, changes, shouldDetach)
	if err != nil {
		xl.ReportFatalDocumentError(fileWithDocs.FileName, doc, err)
	}
}

func DoApply(applyFilenames []string) {
	xl.ForEachDocument("Applying", applyFilenames, applyValues, applyDetach, applyDocument)
}

func init() {
	rootCmd.AddCommand(applyCmd)

	applyFlags := applyCmd.Flags()
	applyFlags.StringArrayVarP(&applyFilenames, "file", "f", []string{}, "Path(s) to the file(s) to apply (required)")
	_ = applyCmd.MarkFlagRequired("file")
	applyFlags.StringToStringVar(&applyValues, "values", map[string]string{}, "Values")
	applyFlags.BoolVarP(&applyDetach, "detach", "d", false, "Detach the client at the moment of starting a deploy or release")
}

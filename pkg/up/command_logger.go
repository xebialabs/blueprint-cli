package up

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"os/signal"
	"strings"
	"sync"

	"github.com/briandowns/spinner"
	"github.com/xebialabs/xl-cli/pkg/models"
	"github.com/xebialabs/xl-cli/pkg/osSpecific"
	"github.com/xebialabs/xl-cli/pkg/util"
	"gopkg.in/AlecAivazis/survey.v1"
)

const (
	DEPLOYING = iota
	UNDEPLOYING
	UPDATING
	UNKNOWN
)

var currentTask = ""
var deploymentDesc = ""

var phase = UNKNOWN

// TODO a better way or to use the APIs available
var generatedPlan = "c.x.d.s.deployment.DeploymentService - Generated plan"
var phaseLogEnd = "on K8S"
var executedLog = "is completed with state [DONE]"
var failExecutedLog = "is completed with state [FAILED]"

func identifyPhase(log string) (phase int, start int) {
	switch {
	case strings.Contains(log, "# [Serial] Deploy"):
		return DEPLOYING, getIndexPlusLen(log, "# [Serial] Deploy")
	case strings.Contains(log, "* Deploy"):
		return DEPLOYING, getIndexPlusLen(log, "* Deploy")
	case strings.Contains(log, "# [Serial] Update"):
		return UPDATING, getIndexPlusLen(log, "# [Serial] Update")
	case strings.Contains(log, "* Update"):
		return UPDATING, getIndexPlusLen(log, "* Update")
	case strings.Contains(log, "# [Serial] Undeploy"):
		return UNDEPLOYING, getIndexPlusLen(log, "# [Serial] Undeploy")
	case strings.Contains(log, "* Undeploy"):
		return UNDEPLOYING, getIndexPlusLen(log, "* Undeploy")
	default:
		return UNKNOWN, -1
	}
}

func logCapture(data []byte, writeFn func(currentStage string)) {
	eventLog := string(data)
	if strings.Contains(eventLog, generatedPlan) {
		currentTask = getCurrentTask(eventLog, strings.Index(eventLog, generatedPlan))
		if currentTask != "" {
			var start int
			phase, start = identifyPhase(eventLog)
			end := strings.Index(eventLog, phaseLogEnd)

			if start >= 0 && end >= 0 {
				deploymentDesc = eventLog[start:end]
				writeCheck(getCurrentStage(false, phase), writeFn)
			}
		}
	}

	if strings.Contains(eventLog, failExecutedLog) {
		if phase == DEPLOYING || phase == UPDATING {
			writeCheck("Failed deployment for", writeFn)
			phase = UNDEPLOYING
			writeCheck(getCurrentStage(false, phase), writeFn)
		} else {
			writeCheck("Failed undeployment for", writeFn)
		}
	}

	if strings.Contains(eventLog, executedLog) {
		writeCheck(getCurrentStage(true, phase), writeFn)
	}
}

func getCurrentStage(isExecuted bool, phase int) string {
	var currentStage string

	switch phase {
	case DEPLOYING:
		currentStage = "Deploy"
	case UNDEPLOYING:
		currentStage = "Undeploy"
	case UPDATING:
		currentStage = "Updat" // isExecuted appends ed/ing
	default:
		currentStage = "Finish"
	}

	if isExecuted {
		currentStage += "ed"
	} else {
		currentStage += "ing"
	}

	return currentStage
}

var lastWritten = ""

func writeCheck(currentStage string, writeFn func(currentStage string)) {
	if phase == UNKNOWN || currentStage == lastWritten {
		return
	}

	if deploymentDesc != "" {
		lastWritten = currentStage
		message := ""
		for _, desc := range strings.Split(deploymentDesc, ",") {
			message += currentStage + desc + "\n\n"
		}
		writeFn(message)
	}
}

func getIndexPlusLen(eventLog string, ident string) int {
	index := strings.Index(eventLog, ident)
	if index >= 0 {
		return index + len(ident)
	}
	return index
}

func getCurrentTask(eventLog string, index int) string {
	start := index + len(generatedPlan)
	end := strings.Index(eventLog, "\n")

	if end > 0 && start > 0 {
		task := eventLog[start:end]
		words := strings.Split(task, " ")

		for _, word := range words {
			part := strings.Split(word, "-")
			if len(part) > 1 {
				return word
			}
		}
	}
	return ""
}

func copyAndCapture(writer io.Writer, reader io.Reader, spinner *spinner.Spinner) ([]byte, error) {
	var out []byte
	buf := make([]byte, 1024, 1024)
	for {
		n, err := reader.Read(buf[:])
		if err != nil {
			// Read returns io.EOF at the end of file, which is not an error for us
			if err == io.EOF {
				err = nil
			}
			// since it EOF we return here
			return out, err
		}
		if n > 0 {
			data := buf[:n]
			out = append(out, data...)

			if util.IsVerbose {
				_, err := writer.Write(data)
				if err != nil {
					return out, err
				}
			}

			logCapture(data, func(content string) {
				spinner.Stop()
				writer.Write([]byte(content))
				spinner.Start()
			})
		}
	}
}

func ExecuteCommandAndShowLogs(command models.Command, spinner *spinner.Spinner) (string, string, error) {
	cmd := exec.Command(command.Name, command.Args...)
	if !util.IsVerbose {
		spinner.Start()
	}

	cmd.SysProcAttr = osSpecific.GetSyscall()

	var stdout, stderr []byte
	var errStdout, errStderr error

	stdoutIn, err := cmd.StdoutPipe()
	if err != nil {
		return "", "", err
	}
	stderrIn, err := cmd.StderrPipe()
	if err != nil {
		return "", "", err
	}
	err = cmd.Start()
	if err != nil {
		return "", "", fmt.Errorf("cmd.Start() failed with '%s'", err)
	}

	var wg sync.WaitGroup
	wg.Add(1)

	done := make(chan bool)
	sig := make(chan os.Signal)
	signal.Notify(sig, os.Interrupt)

	go func() {
		for {
			select {
			case <-sig:
				spinner.Stop()
				cancel := false
				survey.AskOne(
					&survey.Confirm{
						Message: "Do you want to cancel the deployment, this will lead to corrupted kubernetes environment?",
						Default: false,
					}, &cancel, nil)
				if cancel {
					spinner.Stop()
					StopAndRemoveContainer(spinner)
					os.Exit(1)
				} else {
					spinner.Start()
				}
			case <-done:
				return
			}
		}
	}()

	go func() {
		stdout, errStdout = copyAndCapture(os.Stdout, stdoutIn, spinner)
		wg.Done()
		done <- true
	}()

	stderr, errStderr = copyAndCapture(os.Stderr, stderrIn, spinner)
	wg.Wait()

	err = cmd.Wait()

	outStr, errStr := string(stdout), string(stderr)

	if errStdout != nil || errStderr != nil {
		util.Info("failed to capture stdout or stderr\n")
	}

	if !util.IsVerbose {
		spinner.Stop()
	}

	return outStr, errStr, nil
}

func StopAndRemoveContainer(spinner *spinner.Spinner) {
	util.Verbose("stopping the container")

	stopContainer := models.Command{
		Name: "docker",
		Args: []string{"stop", "xl-seed"},
	}
	ExecuteCommandAndShowLogs(stopContainer, s)

	util.Verbose("removing the container")
	rmContainer := models.Command{
		Name: "docker",
		Args: []string{"rm", "xl-seed"},
	}
	ExecuteCommandAndShowLogs(rmContainer, spinner)
}

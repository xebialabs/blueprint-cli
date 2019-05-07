package util

import (
	"io"
	"os"
	"os/exec"
	"os/signal"
	"strings"
	"sync"

	"github.com/briandowns/spinner"
	"github.com/xebialabs/xl-cli/pkg/models"
	"github.com/xebialabs/xl-cli/pkg/osSpecific"
	"gopkg.in/AlecAivazis/survey.v1"
)

var currentTask = ""
var ctDesc = ""

var deploy = true

// TODO a better way or to use the APIs available
var generatedPlan = "c.x.d.s.deployment.DeploymentService - Generated plan"
var phaseLogEnd = "on K8S"
var executedLog = "is completed with state [DONE]"
var failExecutedLog = "is completed with state [FAILED]"

func logCapture(w io.Writer, d []byte, s *spinner.Spinner) {
	eventLog := string(d)

	if strings.Index(eventLog, generatedPlan) != -1 {
		currentTask = getCurrentTask(eventLog, strings.Index(eventLog, generatedPlan))
		if currentTask != "" {
			start := getIndexPlusLen(eventLog, "# [Serial] Deploy")
			end := strings.Index(eventLog, phaseLogEnd)
			if start < 0 {
				start = getIndexPlusLen(eventLog, "* Deploy")
			}

			if start < 0 {
                start = getIndexPlusLen(eventLog, "# [Serial] Update")
            }

            if start < 0 {
                start = getIndexPlusLen(eventLog, "* Update")
            }

			if start >= 0 && end >= 0 {
				s.Stop()
				ctDesc = eventLog[start:end]
				w.Write([]byte("Deploying " + ctDesc + "\n\n"))
				s.Start()
			}
		}
	}

	if strings.Index(eventLog, failExecutedLog) != -1 {
		s.Stop()
		if deploy {
			w.Write([]byte("Failed deploying for " + ctDesc + "\n\n"))
			w.Write([]byte("Undeploying " + ctDesc + "\n\n"))
			deploy = false
		} else {
			w.Write([]byte("Failed undeploying for " + ctDesc + "\n\n"))
		}
		s.Start()
	}

	if strings.Index(eventLog, executedLog) != -1 {
		s.Stop()
		if deploy {
			w.Write([]byte("Deployed " + ctDesc + "\n\n"))
		} else {
			w.Write([]byte("Undeployed " + ctDesc + "\n\n"))
		}
		s.Start()
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
			char := strings.Split(word, "-")
			if len(char) > 1 {
				return word
			}
		}
	}
	return ""
}

func copyAndCapture(w io.Writer, r io.Reader, s *spinner.Spinner) ([]byte, error) {
	var out []byte
	buf := make([]byte, 1024, 1024)
	for {
		n, err := r.Read(buf[:])
		if n > 0 {
			d := buf[:n]
			out = append(out, d...)

			if IsVerbose {
				_, err := w.Write(d)
				if err != nil {
					return out, err
				}
			}

			logCapture(w, d, s)

		}
		if err != nil {
			// Read returns io.EOF at the end of file, which is not an error for us
			if err == io.EOF {
				err = nil
			}

			return out, err
		}
	}
}

func ExecuteCommandAndShowLogs(command models.Command, s *spinner.Spinner) (string, string) {
	cmd := exec.Command(command.Name, command.Args...)
	if !IsVerbose {
		s.Start()
	}

	cmd.SysProcAttr = osSpecific.GetSyscall()

	var stdout, stderr []byte
	var errStdout, errStderr error

	stdoutIn, _ := cmd.StdoutPipe()
	stderrIn, _ := cmd.StderrPipe()

	err := cmd.Start()
	if err != nil {
		Fatal("cmd.Start() failed with '%s' \n", err)
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
				s.Stop()
				cancel := false
				survey.AskOne(
					&survey.Confirm{
						Message: "Do you want to cancel the deployment, this will lead to corrupted kubernetes environment?",
						Default: false,
					}, &cancel, nil)
				if cancel {
					s.Stop()
					os.Exit(1)
				} else {
					s.Start()
				}
			case <-done:
				return
			}
		}
	}()

	go func() {
		stdout, errStdout = copyAndCapture(os.Stdout, stdoutIn, s)
		wg.Done()
		done <- true
	}()

	stderr, errStderr = copyAndCapture(os.Stderr, stderrIn, s)
	wg.Wait()

	err = cmd.Wait()

	outStr, errStr := string(stdout), string(stderr)

	if errStdout != nil || errStderr != nil {
		Info("failed to capture stdout or stderr\n")
	}

	if !IsVerbose {
		s.Stop()
	}

	return outStr, errStr
}

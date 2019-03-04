package util

import (
	"github.com/briandowns/spinner"
	"io"
	"os"
	"os/exec"
	"strings"
	"sync"
	"time"

	"github.com/xebialabs/xl-cli/pkg/models"
)

var currentTask = ""
var s = spinner.New(spinner.CharSets[9], 100*time.Millisecond)
var deploy = true
// TODO a better way or to use the APIs available
var generatedPlan = "c.x.d.s.deployment.DeploymentService - Generated plan"
var phaseLogEnd = "on K8S"
var executedLog = "is completed with state [DONE]"


func logCapture(w io.Writer, d []byte){
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
				start = getIndexPlusLen(eventLog, "# [Serial] Undeploy")
				if start < 0 {
					start = getIndexPlusLen(eventLog, "* Undeploy")
				}
				deploy = false
			}

			if start >= 0 && end >= 0 {
				s.Stop()
				currentTask = eventLog[start:end]
				if deploy {
					w.Write([]byte("Deploying " + currentTask +"\n\n"))
				} else {
					w.Write([]byte("Undeploying " + currentTask + "\n\n"))
				}
				s.Start()
			}
		}
	}

	if strings.Index(eventLog, executedLog) != -1 {
		s.Stop()
		if deploy {
			w.Write([]byte("Deployed "+ currentTask +"\n\n"))
		} else {
			w.Write([]byte("Undeployed "+ currentTask +"\n\n"))
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

		for  _, word := range words {
			char := strings.Split(word, "-")
			if len(char) > 1 {
				return word
			}
 		}
	}
	return ""
}

func copyAndCapture(w io.Writer, r io.Reader) ([]byte, error) {
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

			logCapture(w, d)
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

func ExecuteCommandAndShowLogs(command models.Command) (string, string) {

	cmd := exec.Command(command.Name, command.Args...)
	if !IsVerbose {
		s.Start()
	}

	var stdout, stderr []byte
	var errStdout, errStderr error

	stdoutIn, _ := cmd.StdoutPipe()
	stderrIn, _ := cmd.StderrPipe()

	err := cmd.Start()
	if err != nil {
		Fatal("cmd.Start() failed with '%s'\n", err)
	}

	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		stdout, errStdout = copyAndCapture(os.Stdout, stdoutIn)
		wg.Done()
	}()

	stderr, errStderr = copyAndCapture(os.Stderr, stderrIn)

	wg.Wait()

	err = cmd.Wait()

	outStr, errStr := string(stdout), string(stderr)


	if err != nil {
		Fatal("Failed to run with %s\n", errStr)
		Fatal("cmd.Run() failed with %s\n", err)
	}
	if errStdout != nil || errStderr != nil {
		Fatal("failed to capture stdout or stderr\n")
	}

	if !IsVerbose {
		s.Stop()
	}

	return outStr, errStr
}

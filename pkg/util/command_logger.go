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

// TODO a better way or to use the APIs available
var generatedPlan = "c.x.d.s.deployment.DeploymentService - Generated plan for currentTask"
var phaseLogStart = "# [Plan phase] Deploy\n"
var phaseLogEnd = "on K8S\n"
var executingLog = "Publishing state change QUEUED -> EXECUTING"
var executedLog = "Publishing state change EXECUTED -> DONE"


func copyAndCapture(w io.Writer, r io.Reader) ([]byte, error) {
	var out []byte
	buf := make([]byte, 1024, 1024)
	for {
		n, err := r.Read(buf[:])
		if n > 0 {
			d := buf[:n]
			out = append(out, d...)

			eventLog := string(d)
			if strings.Index(eventLog, generatedPlan) != -1 {
				length := len(phaseLogStart)
				i:= strings.Index(eventLog, phaseLogStart)
				j:= strings.Index(eventLog, phaseLogEnd) - 1
				if i > 0 && j > 0 {
					currentTask = eventLog[i +length :j]
					currentTask = strings.Replace(currentTask, "* Deploy", "", -1)
					currentTask = strings.Replace(currentTask, "1.0.0", "", -1)
					currentTask = strings.TrimSpace(currentTask)
					s.Stop()
					w.Write([]byte("Starting deployment of "+ currentTask +"\n\n"))
					s.Start()
				}
			}

			if strings.Index(eventLog, executingLog) != -1 {
				s.Stop()
				w.Write([]byte("Deploying "+ currentTask +"\n\n"))
				s.Start()
			}

			if strings.Index(eventLog, executedLog) != -1 {
				s.Stop()
				w.Write([]byte("Deployed "+ currentTask +"\n\n"))
				s.Start()
			}

			if IsVerbose {
				_, err := w.Write(d)
				if err != nil {
					return out, err
				}
			}
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
	if err != nil {
		Fatal("cmd.Run() failed with %s\n", err)
	}
	if errStdout != nil || errStderr != nil {
		Fatal("failed to capture stdout or stderr\n")
	}
	outStr, errStr := string(stdout), string(stderr)

	s.Stop()

	return outStr, errStr
}

package util

import (
	"github.com/briandowns/spinner"
	"io"
	"os"
	"os/exec"
	"sync"
	"time"

	"github.com/xebialabs/xl-cli/pkg/models"
)

var isStart = false
var prevIndx = -1

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
		s := spinner.New(spinner.CharSets[4], 100*time.Millisecond)
		s.Start()
		defer s.Stop()
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

	return outStr, errStr
}

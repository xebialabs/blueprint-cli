package up

import (
	"fmt"
	"os"
	"strings"

	"github.com/xebialabs/xl-cli/pkg/models"
	"github.com/xebialabs/xl-cli/pkg/util"
)

func runAndCaptureResponse(cmd models.Command) error {

	completedTask := false
	outStr, errorStr, err := util.ExecuteCommandAndShowLogs(cmd, s)
	if err != nil {
		return err
	}
	if outStr != "" {
		err := createLogFile("xl-seed-log.txt", outStr)
		if err != nil {
			return err
		}
		logCaptureChars := "*=*=*=*=*=*=*=*"
		index := strings.Index(outStr, logCaptureChars)

		if index != -1 {
			// Flip the string, get the "first" instance of the *=*=*=*=*=*=*=* stuff, then get the position
			lastIndex := strings.LastIndex(outStr, logCaptureChars)
			completedTask = true
			util.Info(outStr[index:lastIndex+len(logCaptureChars)] + "\n")
		}
	}

	if errorStr != "" {
		err := createLogFile("xl-seed-error.txt", errorStr)
		if err != nil {
			return err
		}
		s.Stop()
		util.StopAndRemoveContainer(s)
		if !completedTask {
			return fmt.Errorf("Deployment was unsuccessful, please see xl-seed-error.txt and xl-seed-logs.txt for more details")
		}
	}
	return nil
}

func createLogFile(fileName string, contents string) error {
	f, err := os.Create(fileName)
	if err != nil {
		return fmt.Errorf("error creating a file %s", err)
	}
	_, err = f.WriteString(contents)
	if err != nil {
		return err
	}
	return f.Close()
}

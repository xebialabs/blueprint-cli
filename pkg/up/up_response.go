package up

import (
	"github.com/xebialabs/xl-cli/pkg/models"
	"github.com/xebialabs/xl-cli/pkg/util"
	"os"
	"strings"
)

func runAndCaptureResponse(cmd models.Command) {

	completedTask := false
	outStr, errorStr := util.ExecuteCommandAndShowLogs(cmd, s)

	if outStr != "" {
		createLogFile("xl-seed-log.txt", outStr)
		index := strings.Index(outStr, "***************")
		if index != -1 {
			completedTask = true
			util.Info(outStr[index:])
		}
	}

	if errorStr != "" {
		createLogFile("xl-seed-error.txt", errorStr)
		s.Stop()
		if !completedTask {
			util.Fatal("Error while running xl up: \n %s", errorStr)
		}
	}

}

func createLogFile(fileName string, contents string) {
	f, err := os.Create(fileName)
	if err != nil {
		util.Fatal(" Error creating a file %s \n", err)
	}
	f.WriteString(contents)
	f.Close()
}

package up

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/xebialabs/xl-cli/pkg/models"
	"github.com/xebialabs/xl-cli/pkg/util"
)

func Test_runAndCaptureResponse(t *testing.T) {
	defer os.Remove("xl-seed-log.txt")
	defer os.Remove("xl-seed-error.txt")

	tests := []struct {
		name    string
		cmd     models.Command
		check   func() bool
		wantErr bool
	}{
		{
			"throw error on invalid command",
			models.Command{
				Name: "lsssss",
				Args: []string{"-al"},
			},
			func() bool {
				return true
			},
			true,
		},
		{
			"capture output string to xl-seed-log",
			models.Command{
				Name: "ls",
				Args: []string{"-al"},
			},
			func() bool {
				return util.PathExists("xl-seed-log.txt", false)
			},
			false,
		},
		{
			"capture error output string to xl-error-log and fail",
			models.Command{
				Name: "ls",
				Args: []string{"2>"},
			},
			func() bool {
				return util.PathExists("xl-seed-error.txt", false)
			},
			true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := runAndCaptureResponse(tt.cmd)
			if (err != nil) != tt.wantErr {
				t.Errorf("runAndCaptureResponse() error = %v, wantErr %v", err, tt.wantErr)
			}
			assert.True(t, tt.check())
		})
	}
}

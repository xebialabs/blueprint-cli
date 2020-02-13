package cmd

import (
	"fmt"
	"runtime"

	"github.com/spf13/cobra"
)

var BuildVersion = "undefined"
var BuildGitCommit = "undefined"
var BuildDate = "undefined"

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Display version info",
	Long:  `Display version info`,
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Printf("CLI version:             %s\n", CliVersion)
		fmt.Printf("Git version:             %s\n", BuildVersion)
		fmt.Printf("Git commit:              %s\n", BuildGitCommit)
		fmt.Printf("Build date:              %s\n", BuildDate)
		fmt.Printf("GO version:              %s\n", runtime.Version())
		fmt.Printf("OS/Arch:                 %s/%s\n", runtime.GOOS, runtime.GOARCH)
	},
}

func init() {
	rootCmd.AddCommand(versionCmd)
}

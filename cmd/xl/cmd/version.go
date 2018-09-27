package cmd

import (
	"github.com/spf13/cobra"
	"fmt"
	"github.com/xebialabs/xl-cli/pkg/xl"
	"runtime"
)

var CliVersion = "undefined"
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
		fmt.Printf("API version XL Deploy:   %s\n", xl.XldApiVersion)
		fmt.Printf("API version XL Release:  %s\n", xl.XlrApiVersion)
		fmt.Printf("Git commit:              %s\n", BuildGitCommit)
		fmt.Printf("Build date:              %s\n", BuildDate)
		fmt.Printf("GO version:              %s\n", runtime.Version())
		fmt.Printf("OS/Arch:                 %s/%s\n", runtime.GOOS, runtime.GOARCH)
	},
}

func init() {
	rootCmd.AddCommand(versionCmd)
}

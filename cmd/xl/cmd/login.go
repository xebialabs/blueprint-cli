package cmd

import (
	"github.com/spf13/cobra"
	"github.com/xebialabs/xl-cli/internal/app/xl"
)

// loginCmd represents the login command
var loginCmd = &cobra.Command{
	Use:   "login",
	Short: "Configure this tool",
	Long: `Configure this tool either interactively or with flags.
The default or provided configuration file is used to save the changes.
Overriding occurs for an item with an existing identifier.
The tool asks for missing flag values or flag values that are set invalid.`,
	Run: func(cmd *cobra.Command, args []string) {
		xl.Login(srvName, srvType, srvHost, srvPort, srvUsername, srvPassword, srvSsl, srvContextRoot, skipOptional)
	},
}

func init() {
	rootCmd.AddCommand(loginCmd)

	setServerFlags(loginCmd.Flags())
	setSkipOptionalFlags(loginCmd.Flags())
}

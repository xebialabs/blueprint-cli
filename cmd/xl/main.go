package main

import (
	"github.com/xebialabs/xl-cli/cmd/xl/cmd"
	"github.com/xebialabs/xl-cli/pkg/auth"
)

func preExit() {
	auth.Logout()
}

func main() {
	defer preExit()
	cmd.Execute()
}

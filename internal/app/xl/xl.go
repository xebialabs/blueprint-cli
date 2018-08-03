package xl

import (
	"github.com/xebialabs/xl-cli/internal/app/xl/apply"
	"github.com/xebialabs/xl-cli/internal/app/xl/login"
	"github.com/xebialabs/xl-cli/internal/platform/handle"
)

func Apply(fs []string, xld string, xlr string) {
	defer handle.BasicPanicLog()
	handle.BasicError("error executing apply", apply.Execute(fs, xld, xlr))
}

func Login(skipO bool, n string, t string, host string, p string, u string, pwd string, ssl string, ctx string, xldAppHome string, xldCfgHome string, xldEnvHome string, xldInfHome string, xlrHome string) {
	defer handle.BasicPanicLog()
	handle.BasicError("error executing login", login.ExecuteServer(skipO, n, t, host, p, u, pwd, ssl, ctx, xldAppHome, xldCfgHome, xldEnvHome, xldInfHome, xlrHome))
}

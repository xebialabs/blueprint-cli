package xl

import (
	"github.com/xebialabs/xl-cli/internal/app/xl/login"
	"github.com/xebialabs/xl-cli/internal/platform/handle"
)

func Login(n string, t string, host string, p string, u string, pwd string, ssl string, ctx string, skipO bool) {
	defer handle.BasicPanicLog()
	handle.BasicError("error executing login", login.ExecuteServer(n, t, host, p, u, pwd, ssl, ctx, skipO))
}

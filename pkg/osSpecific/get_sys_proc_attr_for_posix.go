// +build darwin linux

package osSpecific

import "syscall"

func GetSyscall() *syscall.SysProcAttr {
	return &syscall.SysProcAttr{
		Setpgid: true,
	}
}

// +build windows

package osSpecific

import "syscall"

const (
    createNewProcessGroupFlag = 0x000000200
)

func GetSyscall() *syscall.SysProcAttr {
	return &syscall.SysProcAttr{
        CreationFlags: createNewProcessGroupFlag,
    }
}

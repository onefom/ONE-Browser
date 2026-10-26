//go:build windows

package backend

import (
	"fmt"
	"golang.org/x/sys/windows"
	"os"
	"os/exec"
	"syscall"
)

func configureCamoufoxPipe(cmd *exec.Cmd, read, write *os.File) error {
	for _, file := range []*os.File{read, write} {
		if err := windows.SetHandleInformation(windows.Handle(file.Fd()), windows.HANDLE_FLAG_INHERIT, windows.HANDLE_FLAG_INHERIT); err != nil {
			return err
		}
	}
	if cmd.SysProcAttr == nil {
		cmd.SysProcAttr = &syscall.SysProcAttr{}
	}
	cmd.SysProcAttr.AdditionalInheritedHandles = append(cmd.SysProcAttr.AdditionalInheritedHandles, syscall.Handle(read.Fd()), syscall.Handle(write.Fd()))
	cmd.Env = append(cmd.Env, fmt.Sprintf("PW_PIPE_READ=%d", read.Fd()), fmt.Sprintf("PW_PIPE_WRITE=%d", write.Fd()))
	return nil
}

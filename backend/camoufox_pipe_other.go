//go:build !windows

package backend

import (
	"os"
	"os/exec"
)

func configureCamoufoxPipe(cmd *exec.Cmd, read, write *os.File) error {
	cmd.ExtraFiles = []*os.File{read, write}
	return nil
}

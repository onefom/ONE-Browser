//go:build windows

package browser

import (
	"fmt"
	"golang.org/x/sys/windows"
	"unsafe"
)

// Read version resources without launching a process or trusting the folder name.
func coreExecutableVersion(baseDir string) string {
	executable, _, ok := FindCoreExecutable(baseDir)
	if !ok {
		return ""
	}
	var handle windows.Handle
	size, err := windows.GetFileVersionInfoSize(executable, &handle)
	if err != nil || size == 0 || size > 16*1024*1024 {
		return ""
	}
	data := make([]byte, size)
	if windows.GetFileVersionInfo(executable, 0, size, unsafe.Pointer(&data[0])) != nil {
		return ""
	}
	var info *windows.VS_FIXEDFILEINFO
	length := uint32(unsafe.Sizeof(windows.VS_FIXEDFILEINFO{}))
	if windows.VerQueryValue(unsafe.Pointer(&data[0]), `\`, unsafe.Pointer(&info), &length) != nil || info == nil || length < uint32(unsafe.Sizeof(*info)) {
		return ""
	}
	return fmt.Sprintf("%d.%d.%d.%d", info.FileVersionMS>>16, info.FileVersionMS&65535, info.FileVersionLS>>16, info.FileVersionLS&65535)
}

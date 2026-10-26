//go:build windows

package backend

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"unsafe"

	"golang.org/x/sys/windows"
)

func nativeProcessCommandLine(pid uint32) (string, error) {
	handle, err := windows.OpenProcess(windows.PROCESS_QUERY_INFORMATION|windows.PROCESS_VM_READ, false, pid)
	if err != nil {
		handle, err = windows.OpenProcess(windows.PROCESS_QUERY_LIMITED_INFORMATION, false, pid)
	}
	if err != nil {
		return "", err
	}
	defer windows.CloseHandle(handle)
	var size uint32
	_ = windows.NtQueryInformationProcess(handle, windows.ProcessCommandLineInformation, nil, 0, &size)
	if size < uint32(unsafe.Sizeof(windows.NTUnicodeString{})) || size > 1024*1024 {
		return "", fmt.Errorf("invalid command line buffer length: %d", size)
	}
	buffer := make([]byte, size)
	if err := windows.NtQueryInformationProcess(handle, windows.ProcessCommandLineInformation, unsafe.Pointer(&buffer[0]), size, &size); err != nil {
		return "", err
	}
	value := (*windows.NTUnicodeString)(unsafe.Pointer(&buffer[0]))
	start := uintptr(unsafe.Pointer(&buffer[0]))
	ptr := uintptr(unsafe.Pointer(value.Buffer))
	length := uintptr(value.Length)
	if length%2 != 0 || ptr < start || ptr-start > uintptr(len(buffer)) || length > uintptr(len(buffer))-(ptr-start) {
		return "", fmt.Errorf("invalid command line string bounds")
	}
	result := value.String()
	runtime.KeepAlive(buffer)
	return result, nil
}

func nativeBrowserProcessName(name string) bool {
	name = strings.ToLower(name)
	if name == "msedgewebview2.exe" {
		return false
	}
	return strings.Contains(name, "camoufox") || strings.Contains(name, "firefox") || strings.Contains(name, "chrome") || strings.Contains(name, "chromium") || strings.Contains(name, "msedge") || strings.Contains(name, "fingerprint")
}

// Query fresh process command lines without starting PowerShell. Unsupported
// systems and unreadable browser processes fall back to the original CIM scan.
func findBrowserUserDataProcessesNative(userDir string) ([]browserUserDataProcess, error) {
	if _, err := nativeProcessCommandLine(uint32(os.Getpid())); err != nil {
		return nil, err
	}
	target, err := filepath.Abs(userDir)
	if err != nil {
		return nil, err
	}
	target = strings.TrimRight(filepath.Clean(target), `\/`)
	snapshot, err := windows.CreateToolhelp32Snapshot(windows.TH32CS_SNAPPROCESS, 0)
	if err != nil {
		return nil, err
	}
	defer windows.CloseHandle(snapshot)
	entry := windows.ProcessEntry32{Size: uint32(unsafe.Sizeof(windows.ProcessEntry32{}))}
	if err := windows.Process32First(snapshot, &entry); err != nil {
		return nil, err
	}

	entries := []windows.ProcessEntry32{}
	parents := map[uint32]string{}
	for {
		entries = append(entries, entry)
		parents[entry.ProcessID] = strings.ToLower(windows.UTF16ToString(entry.ExeFile[:]))
		if err := windows.Process32Next(snapshot, &entry); err != nil {
			if errors.Is(err, windows.ERROR_NO_MORE_FILES) {
				break
			}
			return nil, err
		}
	}
	result := []browserUserDataProcess{}
	for _, entry := range entries {
		name := strings.ToLower(windows.UTF16ToString(entry.ExeFile[:]))
		if nativeBrowserProcessName(name) && parents[entry.ParentProcessID] == name {
			continue
		}
		if entry.ProcessID > 4 {
			command, readErr := nativeProcessCommandLine(entry.ProcessID)
			if readErr != nil && nativeBrowserProcessName(windows.UTF16ToString(entry.ExeFile[:])) {
				// An exited process is harmless; an inaccessible live browser needs CIM.
				handle, openErr := windows.OpenProcess(windows.SYNCHRONIZE, false, entry.ProcessID)
				if openErr == nil {
					state, _ := windows.WaitForSingleObject(handle, 0)
					windows.CloseHandle(handle)
					if state != windows.WAIT_OBJECT_0 {
						return nil, readErr
					}
				} else if !errors.Is(openErr, windows.ERROR_INVALID_PARAMETER) {
					return nil, readErr
				}
			}
			if readErr == nil && (strings.Contains(command, "--user-data-dir") || strings.Contains(command, "-profile")) {
				args, err := windows.DecomposeCommandLine(command)
				if err != nil {
					return nil, err
				}
				for i, arg := range args {
					path := ""
					if strings.HasPrefix(arg, "--user-data-dir=") {
						path = strings.TrimPrefix(arg, "--user-data-dir=")
					}
					if (arg == "--user-data-dir" || arg == "-profile" || arg == "--profile") && i+1 < len(args) {
						path = args[i+1]
					}
					if path == "" {
						continue
					}
					absolute, err := filepath.Abs(path)
					if err != nil {
						return nil, err
					}
					if strings.EqualFold(strings.TrimRight(filepath.Clean(absolute), `\/`), target) || strings.EqualFold(filepath.Clean(absolute), filepath.Join(target, "camoufox")) {
						result = append(result, browserUserDataProcess{PID: int(entry.ProcessID), DebugPort: parseRemoteDebuggingPort(command), CommandLine: command})
					}
					break
				}
			}
		}
	}
	return result, nil
}

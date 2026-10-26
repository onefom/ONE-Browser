//go:build windows
// +build windows

package backend

import (
	"golang.org/x/sys/windows"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"unsafe"
)

var mainApplicationWindowIconState struct {
	sync.Mutex
	resourcePath string
	bigIcon      uintptr
	smallIcon    uintptr
}

func ApplyMainApplicationWindowIcon(appRoot, expectedTitle string) {
	resourcePath := profileWindowMarkerIconResourcePath(appRoot)
	if strings.TrimSpace(resourcePath) == "" {
		resourcePath, _ = os.Executable()
	}
	if strings.TrimSpace(resourcePath) == "" {
		return
	}

	windowsForApp := findTopLevelWindowsForProfileMarker(uint32(os.Getpid()))
	window, ok := selectMainApplicationWindow(windowsForApp, expectedTitle)
	if !ok {
		return
	}

	mainApplicationWindowIconState.Lock()
	defer mainApplicationWindowIconState.Unlock()

	if mainApplicationWindowIconState.resourcePath != resourcePath ||
		mainApplicationWindowIconState.bigIcon == 0 ||
		mainApplicationWindowIconState.smallIcon == 0 {
		bigIcon, smallIcon := loadMainApplicationIcons(resourcePath)
		if bigIcon == 0 && smallIcon == 0 {
			return
		}
		if bigIcon == 0 {
			bigIcon = smallIcon
		}
		if smallIcon == 0 {
			smallIcon = bigIcon
		}
		mainApplicationWindowIconState.resourcePath = resourcePath
		mainApplicationWindowIconState.bigIcon = bigIcon
		mainApplicationWindowIconState.smallIcon = smallIcon
	}

	setProfileWindowMarkerIcon(window.hwnd, iconBig, mainApplicationWindowIconState.bigIcon)
	setProfileWindowMarkerIcon(window.hwnd, iconSmall, mainApplicationWindowIconState.smallIcon)
	clearProfileWindowMarkerTaskbarOverlay(window.hwnd)
	applyProfileWindowMarkerRelaunchIconResource(
		window.hwnd,
		resourcePath,
		&profileWindowMarkerWindowState{},
	)
	refreshProfileWindowMarkerIcon(window.hwnd)
}

func selectMainApplicationWindow(candidates []profileWindowMarkerWindow, expectedTitle string) (profileWindowMarkerWindow, bool) {
	expectedTitle = strings.TrimSpace(expectedTitle)
	for _, candidate := range candidates {
		if !candidate.visible || candidate.owned || !candidate.titled {
			continue
		}
		if expectedTitle != "" && strings.TrimSpace(getProfileWindowMarkerTitle(candidate.hwnd)) == expectedTitle {
			return candidate, true
		}
	}
	for _, candidate := range candidates {
		if candidate.visible && !candidate.owned && candidate.titled {
			return candidate, true
		}
	}
	return profileWindowMarkerWindow{}, false
}

// ICO files use LoadImageW directly; ExtractIconEx is reserved for executable resources.
func loadMainApplicationIcons(path string) (uintptr, uintptr) {
	if strings.EqualFold(filepath.Ext(path), ".ico") {
		if ptr, err := windows.UTF16PtrFromString(path); err == nil {
			load := user32WindowMarker.NewProc("LoadImageW")
			big, _, _ := load.Call(0, uintptr(unsafe.Pointer(ptr)), 1, 32, 32, 0x10)
			small, _, _ := load.Call(0, uintptr(unsafe.Pointer(ptr)), 1, 16, 16, 0x10)
			if big != 0 || small != 0 {
				return big, small
			}
		}
	}
	big, small := extractProfileWindowMarkerExecutableIcons(path)
	if big == 0 && small == 0 {
		if exe, err := os.Executable(); err == nil {
			return extractProfileWindowMarkerExecutableIcons(exe)
		}
	}
	return big, small
}

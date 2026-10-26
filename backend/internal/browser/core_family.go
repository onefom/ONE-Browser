package browser

import (
	"debug/pe"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
)

// CoreFamily uses the actual executable, never a user editable display name.
func (m *Manager) CoreFamily(core Core) string {
	exe, _, ok := FindCoreExecutable(m.ResolveRelativePath(core.CorePath))
	if ok && strings.HasPrefix(strings.ToLower(filepath.Base(exe)), "camoufox") {
		return "camoufox"
	}
	return "chromium"
}

func firefoxApplicationVersion(baseDir string) string {
	exe, _, ok := FindCoreExecutable(baseDir)
	if !ok || !strings.HasPrefix(strings.ToLower(filepath.Base(exe)), "camoufox") {
		return ""
	}
	data, err := os.ReadFile(filepath.Join(filepath.Dir(exe), "application.ini"))
	if err != nil {
		return ""
	}
	section := ""
	for _, line := range strings.Split(string(data), "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "[") {
			section = line
		}
		if section == "[App]" && strings.HasPrefix(line, "Version=") {
			return strings.TrimSpace(strings.TrimPrefix(line, "Version="))
		}
	}
	return ""
}

// Verify the actual PE machine, even when a 32-bit archive was renamed.
func ValidateCamoufoxArchitecture(dir string) error {
	exe, _, ok := FindCoreExecutable(dir)
	if !ok || !strings.HasPrefix(strings.ToLower(filepath.Base(exe)), "camoufox") || runtime.GOOS != "windows" {
		return nil
	}
	file, err := pe.Open(exe)
	if err != nil {
		return fmt.Errorf("Camoufox 可执行文件无效：%w", err)
	}
	defer file.Close()
	if file.Machine != pe.IMAGE_FILE_MACHINE_AMD64 {
		return fmt.Errorf("Camoufox 必须为 Windows x86_64 / x64 内核，所选可执行文件不是64位 x64")
	}
	return nil
}

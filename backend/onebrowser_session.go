package backend

import (
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
)

// Unknown/local versions are allowed through for validation by the updater.
func oneBrowserKernelNeedsUpdate(current, latest string) bool {
	current = strings.TrimPrefix(strings.TrimSpace(current), "v")
	latest = strings.TrimPrefix(strings.TrimSpace(latest), "v")
	if current != "" && current == latest {
		return false
	}
	parse := func(v string) ([]int, int, bool) {
		m := regexp.MustCompile(`^(\d+(?:\.\d+)*)(?:-beta\.(\d+))?$`).FindStringSubmatch(v)
		if len(m) == 0 {
			return nil, 0, false
		}
		parts := []int{}
		for _, p := range strings.Split(m[1], ".") {
			n, _ := strconv.Atoi(p)
			parts = append(parts, n)
		}
		beta := -1
		if m[2] != "" {
			beta, _ = strconv.Atoi(m[2])
		}
		return parts, beta, true
	}
	a, ba, oka := parse(current)
	b, bb, okb := parse(latest)
	if !oka || !okb {
		return true
	}
	n := len(a)
	if len(b) > n {
		n = len(b)
	}
	for i := 0; i < n; i++ {
		x, y := 0, 0
		if i < len(a) {
			x = a[i]
		}
		if i < len(b) {
			y = b[i]
		}
		if x != y {
			return x < y
		}
	}
	if ba == bb {
		return false
	}
	if ba < 0 {
		return false
	}
	if bb < 0 {
		return true
	}
	return ba < bb
}

func oneBrowserHasSavedSession(userDataDir string) bool {
	entries, _ := os.ReadDir(filepath.Join(userDataDir, "Default", "Sessions"))
	for _, entry := range entries {
		if !entry.IsDir() && (strings.HasPrefix(entry.Name(), "Session_") || strings.HasPrefix(entry.Name(), "Tabs_")) {
			if info, err := entry.Info(); err == nil && info.Size() > 0 {
				return true
			}
		}
	}
	for _, name := range []string{"Last Session", "Current Session"} {
		if info, err := os.Stat(filepath.Join(userDataDir, "Default", name)); err == nil && !info.IsDir() && info.Size() > 0 {
			return true
		}
	}
	return false
}

func oneBrowserSessionLaunchArgs(args []string) []string {
	result := make([]string, 0, len(args))
	for _, arg := range args {
		lower := strings.ToLower(strings.TrimSpace(arg))
		if strings.HasPrefix(lower, "--window-size=") || strings.HasPrefix(lower, "--window-position=") || lower == "--start-maximized" {
			continue
		}
		result = append(result, arg)
	}
	return result
}

func oneBrowserRestoreWindowArgs(args []string) []string {
	result := make([]string, 0, len(args))
	for _, arg := range args {
		if !strings.EqualFold(strings.TrimSpace(arg), "--new-window") {
			result = append(result, arg)
		}
	}
	return result
}

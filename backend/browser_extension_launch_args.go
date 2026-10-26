package backend

import (
	"path/filepath"
	"strings"
)

func appendBrowserExtensionLaunchArgs(args []string, dirs []string) []string {
	paths := make([]string, 0, len(dirs))
	for _, dir := range dirs {
		if strings.TrimSpace(dir) == "" {
			continue
		}
		if path, err := filepath.Abs(dir); err == nil {
			paths = append(paths, path)
		}
	}
	if len(paths) == 0 {
		return args
	}
	return append(args, "--load-extension="+strings.Join(paths, ","))
}

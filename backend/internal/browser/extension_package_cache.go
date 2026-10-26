package browser

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
)

type cachedExtensionPackage struct {
	info      os.FileInfo
	hash      string
	runtimeID string
}

var sharedExtensionPackages sync.Map

// Reuse verified metadata while the same package file is unchanged. A replace,
// size change or modification timestamp invalidates it immediately.
func (m *Manager) resolveSharedExtensionPackage(extension Extension, binary string) (string, string, string, error) {
	path := strings.TrimSpace(extension.PackagePath)
	if path != "" && !filepath.IsAbs(path) {
		path = m.ResolveRelativePath(path)
	}
	if path != "" {
		info, err := os.Stat(path)
		if err == nil && info.Mode().IsRegular() {
			if value, ok := sharedExtensionPackages.Load(path); ok {
				cached := value.(cachedExtensionPackage)
				if os.SameFile(cached.info, info) && cached.info.Size() == info.Size() && cached.info.ModTime().Equal(info.ModTime()) {
					return path, cached.hash, cached.runtimeID, nil
				}
			}
			data, err := os.ReadFile(path)
			if err == nil && isCRXExtensionPackage(data) {
				after, err := os.Stat(path)
				if err != nil {
					return "", "", "", err
				}
				if !os.SameFile(info, after) || info.Size() != after.Size() || !info.ModTime().Equal(after.ModTime()) {
					return "", "", "", fmt.Errorf("插件包在校验期间发生变化，请重试")
				}
				id := extensionIDFromCRX(data)
				if id == "" {
					id = NormalizeExtensionID(extension.ExtensionID)
				}
				if id == "" {
					return "", "", "", fmt.Errorf("无法识别插件运行时 ID")
				}
				hash := extensionPackageHash(data)
				sharedExtensionPackages.Store(path, cachedExtensionPackage{info: after, hash: hash, runtimeID: id})
				return path, hash, id, nil
			}
		}
	}
	path, hash, err := m.resolveExtensionPackage(extension, binary)
	if err != nil {
		return "", "", "", err
	}
	id, err := runtimeExtensionIDFromPackage(path, extension)
	return path, hash, id, err
}

func sharedExtensionRegistrationMatches(userDir, codePath, runtimeID, version string) bool {
	prefs, err := readProfileJSON(filepath.Join(userDir, "Default", "Secure Preferences"), false)
	if err != nil {
		return false
	}
	extensions, _ := prefs["extensions"].(map[string]any)
	settings, _ := extensions["settings"].(map[string]any)
	setting, _ := settings[runtimeID].(map[string]any)
	if setting == nil {
		return false
	}
	location, _ := setting["location"].(float64)
	state, _ := setting["state"].(float64)
	path, _ := setting["path"].(string)
	manifest, _ := setting["manifest"].(map[string]any)
	switch reasons := setting["disable_reasons"].(type) {
	case []any:
		if len(reasons) > 0 {
			return false
		}
	case float64:
		if reasons != 0 {
			return false
		}
	case nil:
	default:
		return false
	}
	return location == 3 && state == 1 && sameProfileExtensionPath(path, codePath) && manifest["version"] == version
}

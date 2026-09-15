package browser

import "path/filepath"

// EnableProfileExtensionDeveloperMode makes chrome://extensions open with
// Developer mode enabled for every isolated One Browser profile.
func EnableProfileExtensionDeveloperMode(userDataDir string) error {
	path := filepath.Join(userDataDir, "Default", "Preferences")
	root, err := readProfileJSON(path, true)
	if err != nil {
		return err
	}
	extensions, err := ensureProfileJSONMap(root, "extensions")
	if err != nil {
		return err
	}
	ui, err := ensureProfileJSONMap(extensions, "ui")
	if err != nil {
		return err
	}
	ui["developer_mode"] = true
	return writeProfileJSON(path, root)
}

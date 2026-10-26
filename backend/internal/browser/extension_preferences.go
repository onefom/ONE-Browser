package browser

import "path/filepath"

// EnableProfileExtensionDeveloperMode updates both preference stores while the
// profile is stopped. Chromium tracks this setting in Secure Preferences.
func EnableProfileExtensionDeveloperMode(userDataDir string) error {
	for _, name := range []string{"Preferences", "Secure Preferences"} {
		path := filepath.Join(userDataDir, "Default", name)
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
		// Remove only this managed setting's stale MAC; preserve all other prefs.
		if protection, _ := ensureProfileJSONMapIfPresent(root, "protection"); protection != nil {
			if macs, _ := ensureProfileJSONMapIfPresent(protection, "macs"); macs != nil {
				if ext, _ := ensureProfileJSONMapIfPresent(macs, "extensions"); ext != nil {
					if trackedUI, _ := ensureProfileJSONMapIfPresent(ext, "ui"); trackedUI != nil {
						delete(trackedUI, "developer_mode")
					}
				}
			}
		}
		if err := writeProfileJSON(path, root); err != nil {
			return err
		}
	}
	return nil
}

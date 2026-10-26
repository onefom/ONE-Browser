package browser

import "path/filepath"

// Continue where you left off also retains Chromium session cookies. Login
// databases and saved passwords stay in the existing isolated user-data-dir.
func EnableProfileSessionPersistence(userDataDir string) error {
	path := filepath.Join(userDataDir, "Default", "Preferences")
	root, err := readProfileJSON(path, true)
	if err != nil {
		return err
	}
	session, err := ensureProfileJSONMap(root, "session")
	if err != nil {
		return err
	}
	session["restore_on_startup"] = 1
	if _, exists := root["credentials_enable_service"]; !exists {
		root["credentials_enable_service"] = true
	}
	profile, err := ensureProfileJSONMap(root, "profile")
	if err != nil {
		return err
	}
	if _, exists := profile["password_manager_enabled"]; !exists {
		profile["password_manager_enabled"] = true
	}
	return writeProfileJSON(path, root)
}

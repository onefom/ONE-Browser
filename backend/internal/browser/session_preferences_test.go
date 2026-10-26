package browser

import (
	"os"
	"path/filepath"
	"testing"
)

func TestSessionPreferencesPreserveCredentialsAndOtherData(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "Default", "Preferences")
	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(`{"session":{"restore_on_startup":5},"credentials_enable_service":false,"profile":{"password_manager_enabled":true},"other":"keep"}`), 0600); err != nil {
		t.Fatal(err)
	}
	cookiePath := filepath.Join(root, "Default", "Cookies")
	if err := os.WriteFile(cookiePath, []byte("existing cookie database"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := EnableProfileSessionPersistence(root); err != nil {
		t.Fatal(err)
	}
	prefs, err := readProfileJSON(path, false)
	if err != nil {
		t.Fatal(err)
	}
	if prefs["session"].(map[string]any)["restore_on_startup"] != float64(1) {
		t.Fatal("session restore not enabled")
	}
	if prefs["other"] != "keep" || prefs["credentials_enable_service"] != false {
		t.Fatal("unrelated preferences overwritten")
	}
	data, _ := os.ReadFile(cookiePath)
	if string(data) != "existing cookie database" {
		t.Fatal("cookies modified")
	}
}

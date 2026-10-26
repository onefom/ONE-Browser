package browser

import (
	"path/filepath"
	"testing"
)

func TestEnableProfileExtensionDeveloperMode(t *testing.T) {
	userDataDir := t.TempDir()
	if err := EnableProfileExtensionDeveloperMode(userDataDir); err != nil {
		t.Fatalf("enable developer mode: %v", err)
	}
	root, err := readProfileJSON(filepath.Join(userDataDir, "Default", "Preferences"), false)
	if err != nil {
		t.Fatalf("read preferences: %v", err)
	}
	extensions, ok := root["extensions"].(map[string]any)
	if !ok {
		t.Fatal("extensions preferences missing")
	}
	ui, ok := extensions["ui"].(map[string]any)
	if !ok {
		t.Fatal("extensions ui preferences missing")
	}
	if enabled, ok := ui["developer_mode"].(bool); !ok || !enabled {
		t.Fatalf("developer_mode = %#v, want true", ui["developer_mode"])
	}
}

func TestDeveloperModeOverridesSecurePreference(t *testing.T) {
	root := t.TempDir()
	secure := filepath.Join(root, "Default", "Secure Preferences")
	if err := writeProfileJSON(secure, map[string]any{"extensions": map[string]any{"ui": map[string]any{"developer_mode": false}}, "other": "keep"}); err != nil {
		t.Fatal(err)
	}
	if err := EnableProfileExtensionDeveloperMode(root); err != nil {
		t.Fatal(err)
	}
	prefs, err := readProfileJSON(secure, false)
	if err != nil {
		t.Fatal(err)
	}
	if prefs["extensions"].(map[string]any)["ui"].(map[string]any)["developer_mode"] != true || prefs["other"] != "keep" {
		t.Fatal("secure developer mode not enabled or unrelated data overwritten")
	}
}

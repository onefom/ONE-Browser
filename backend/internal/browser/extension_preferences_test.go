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

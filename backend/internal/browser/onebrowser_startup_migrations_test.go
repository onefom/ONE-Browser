package browser

import (
	"ant-chrome/backend/internal/config"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLegacyBookmarkMigrationRunsOnceAndRetriesFailure(t *testing.T) {
	root := t.TempDir()
	url := "ant://fingerprint-check"
	if err := EnsureDefaultBookmarks(root, []config.BrowserBookmark{{Name: "指纹检测", URL: url}, {Name: "My bookmark", URL: "https://example.com"}}); err != nil {
		t.Fatal(err)
	}
	if err := MigrateOneBrowserLegacyBookmarks(root, url); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(root, "Default", "Bookmarks")
	marker := filepath.Join(root, ".onebrowser-startup-migrations.json")
	data, _ := os.ReadFile(path)
	if strings.Contains(string(data), url) || !strings.Contains(string(data), "https://example.com") {
		t.Fatal("wrong bookmarks removed")
	}
	before, _ := os.Stat(path)
	beforeMarker, _ := os.Stat(marker)
	if err := MigrateOneBrowserLegacyBookmarks(root, url); err != nil {
		t.Fatal(err)
	}
	after, _ := os.Stat(path)
	afterMarker, _ := os.Stat(marker)
	if !os.SameFile(before, after) || !before.ModTime().Equal(after.ModTime()) || !beforeMarker.ModTime().Equal(afterMarker.ModTime()) {
		t.Fatal("completed migration rewrote files")
	}
	broken := t.TempDir()
	_ = os.MkdirAll(filepath.Join(broken, "Default"), 0700)
	_ = os.WriteFile(filepath.Join(broken, "Default", "Bookmarks"), []byte("broken"), 0600)
	if err := MigrateOneBrowserLegacyBookmarks(broken, url); err == nil {
		t.Fatal("bad bookmark file accepted")
	}
	if _, err := os.Stat(filepath.Join(broken, ".onebrowser-startup-migrations.json")); !os.IsNotExist(err) {
		t.Fatal("failed migration was marked complete")
	}
	_ = os.Remove(filepath.Join(broken, "Default", "Bookmarks"))
	if err := MigrateOneBrowserLegacyBookmarks(broken, url); err != nil {
		t.Fatal("failed migration did not retry", err)
	}
}

func TestLegacyExtensionScanOnlyRunsForMigrationAndCatalogChange(t *testing.T) {
	root := t.TempDir()
	m := NewManager(config.DefaultConfig(), root)
	m.ExtensionDAO = newTestExtensionDAO(t, root)
	ext := sharedTestPackage(t, m, "1.0")
	profile := &Profile{ProfileId: "one", Keywords: []string{"one-browser-window:one"}}
	user := filepath.Join(root, "data", "one")
	if _, err := m.ExtensionDAO.SetProfileSettings(profile.ProfileId, nil, true); err != nil {
		t.Fatal(err)
	}
	if _, warnings := m.PrepareProfileExtensions(profile, "unused.exe", user, nil); len(warnings) != 0 {
		t.Fatal(warnings)
	}
	path := filepath.Join(user, "Default", "Secure Preferences")
	_ = os.MkdirAll(filepath.Dir(path), 0700)
	_ = os.WriteFile(path, []byte("broken"), 0600)
	// Unchanged opt-out catalog must not parse every old Secure Preferences file.
	if _, warnings := m.PrepareProfileExtensions(profile, "unused.exe", user, nil); len(warnings) != 0 {
		t.Fatal("repeated legacy scan", warnings)
	}
	ext.ExtensionID = "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
	ext.InstallDir = filepath.Join(root, "other-legacy")
	if err := m.ExtensionDAO.Upsert(ext); err != nil {
		t.Fatal(err)
	}
	if _, warnings := m.PrepareProfileExtensions(profile, "unused.exe", user, nil); len(warnings) == 0 {
		t.Fatal("changed catalog did not trigger migration")
	}
	_ = os.WriteFile(path, []byte(`{}`), 0600)
	if _, warnings := m.PrepareProfileExtensions(profile, "unused.exe", user, nil); len(warnings) != 0 {
		t.Fatal("catalog migration did not retry", warnings)
	}
	all, _ := m.ExtensionDAO.List()
	if !legacyExtensionMigrationDone(user, all) {
		t.Fatal("migration not committed")
	}
}

package backend

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"ant-chrome/backend/internal/config"
)

func TestPortableWebviewDataPathIsInsideData(t *testing.T) {
	root := t.TempDir()
	got := PortableWebviewDataPath(root)
	want := filepath.Join(root, "data", "webview2")
	if !samePath(got, want) {
		t.Fatalf("PortableWebviewDataPath() = %q, want %q", got, want)
	}
}

func TestPreparePortableWebviewDataMigratesLegacyStorageOnce(t *testing.T) {
	root := t.TempDir()
	legacy := filepath.Join(root, "legacy", "OneBrowser.exe")
	target := filepath.Join(root, "portable", "data", "webview2")
	legacyLocalStorage := filepath.Join(legacy, "EBWebView", "Default", "Local Storage", "leveldb")
	if err := os.MkdirAll(legacyLocalStorage, 0o755); err != nil {
		t.Fatal(err)
	}
	legacyFile := filepath.Join(legacyLocalStorage, "000003.log")
	if err := os.WriteFile(legacyFile, []byte("one-browser-accounts"), 0o600); err != nil {
		t.Fatal(err)
	}

	result, err := preparePortableWebviewDataFromCandidates(target, []string{legacy})
	if err != nil {
		t.Fatal(err)
	}
	if !result.Migrated || !samePath(result.Source, legacy) {
		t.Fatalf("migration result = %#v", result)
	}
	migratedFile := filepath.Join(target, "EBWebView", "Default", "Local Storage", "leveldb", "000003.log")
	data, err := os.ReadFile(migratedFile)
	if err != nil || string(data) != "one-browser-accounts" {
		t.Fatalf("migrated localStorage = %q, err=%v", data, err)
	}
	if _, err := os.Stat(legacyFile); err != nil {
		t.Fatalf("legacy recovery copy was removed: %v", err)
	}

	if err := os.WriteFile(migratedFile, []byte("portable-newer"), 0o600); err != nil {
		t.Fatal(err)
	}
	second, err := preparePortableWebviewDataFromCandidates(target, []string{legacy})
	if err != nil || second.Migrated {
		t.Fatalf("second migration = %#v, err=%v", second, err)
	}
	data, err = os.ReadFile(migratedFile)
	if err != nil || string(data) != "portable-newer" {
		t.Fatalf("portable data was overwritten: %q, err=%v", data, err)
	}
}

func TestPreparePortableWebviewDataReplacesEmptyTarget(t *testing.T) {
	root := t.TempDir()
	legacy := filepath.Join(root, "legacy", "OneBrowser.exe")
	target := filepath.Join(root, "portable", "data", "webview2")
	if err := os.MkdirAll(legacy, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(legacy, "state.db"), []byte("legacy-state"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(target, 0o755); err != nil {
		t.Fatal(err)
	}

	result, err := preparePortableWebviewDataFromCandidates(target, []string{legacy})
	if err != nil || !result.Migrated {
		t.Fatalf("migration = %#v, err=%v", result, err)
	}
	data, err := os.ReadFile(filepath.Join(target, "state.db"))
	if err != nil || string(data) != "legacy-state" {
		t.Fatalf("migrated state = %q, err=%v", data, err)
	}
}

func TestPreparePortableWebviewDataFreshPackageDoesNotImportLegacyState(t *testing.T) {
	root := t.TempDir()
	legacy := filepath.Join(root, "legacy", "OneBrowser.exe")
	target := filepath.Join(root, "portable", "data", "webview2")
	if err := os.MkdirAll(legacy, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(legacy, "old-window-state"), []byte("must-not-import"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(filepath.Dir(target), "README.md"), []byte("One Browser data"), 0o600); err != nil {
		t.Fatal(err)
	}

	result, err := preparePortableWebviewDataFromCandidates(target, []string{legacy})
	if err != nil || result.Migrated {
		t.Fatalf("fresh package preparation = %#v, err=%v", result, err)
	}
	if _, err := os.Stat(filepath.Join(target, "old-window-state")); !os.IsNotExist(err) {
		t.Fatalf("legacy data unexpectedly imported: %v", err)
	}
	if _, err := os.Stat(filepath.Join(filepath.Dir(target), portableWebviewMigrationMarker)); err != nil {
		t.Fatalf("fresh package marker not persisted: %v", err)
	}
}

func TestPortableDataLoggingPolicyForcesDataLogs(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.Logging.FileEnabled = false
	cfg.Logging.FilePath = "outside.log"
	cfg.Logging.Rotation.Enabled = false
	cfg.Logging.Rotation.MaxAge = 365
	applyPortableDataLoggingPolicy(cfg)
	if !cfg.Logging.FileEnabled || !cfg.Logging.Rotation.Enabled {
		t.Fatal("portable logging must be enabled with rotation")
	}
	if filepath.ToSlash(cfg.Logging.FilePath) != "data/logs/app.log" {
		t.Fatalf("log path = %q", cfg.Logging.FilePath)
	}
	if cfg.Logging.Rotation.MaxAge != 30 {
		t.Fatalf("log max age = %d, want 30", cfg.Logging.Rotation.MaxAge)
	}
	if strings.TrimSpace(cfg.Logging.FilePath) == "" {
		t.Fatal("log path is empty")
	}
}

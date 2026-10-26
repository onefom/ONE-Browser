package browser

import (
	"ant-chrome/backend/internal/config"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestSharedExtensionWarmPrepareDoesNotRewritePreferences(t *testing.T) {
	root := t.TempDir()
	m := NewManager(config.DefaultConfig(), root)
	m.ExtensionDAO = newTestExtensionDAO(t, root)
	extension := sharedTestPackage(t, m, "1.0")
	userDir := filepath.Join(root, "data", "private")
	p := &Profile{ProfileId: "private", Keywords: []string{"one-browser-window:test"}}
	dirs, warnings := m.PrepareProfileExtensions(p, "unused-browser.exe", userDir, nil)
	if len(dirs) != 1 || len(warnings) > 0 {
		t.Fatalf("%v %v", dirs, warnings)
	}
	prefs := filepath.Join(userDir, "Default", "Secure Preferences")
	old := time.Unix(1000000, 0)
	if err := os.Chtimes(prefs, old, old); err != nil {
		t.Fatal(err)
	}
	dirsAgain, warnings := m.PrepareProfileExtensions(p, "unused-browser.exe", userDir, nil)
	if len(warnings) > 0 || len(dirsAgain) != 1 || dirsAgain[0] != dirs[0] {
		t.Fatalf("%v %v", dirsAgain, warnings)
	}
	info, err := os.Stat(prefs)
	if err != nil || !info.ModTime().Equal(old) {
		t.Fatal("warm launch rewrote preferences", err)
	}
	packagePath, hash, id, err := m.resolveSharedExtensionPackage(extension, "")
	if err != nil {
		t.Fatal(err)
	}
	before, _ := sharedExtensionPackages.Load(packagePath)
	_, hashAgain, idAgain, err := m.resolveSharedExtensionPackage(extension, "")
	after, _ := sharedExtensionPackages.Load(packagePath)
	if err != nil || hashAgain != hash || idAgain != id || before.(cachedExtensionPackage).info != after.(cachedExtensionPackage).info {
		t.Fatal("verified metadata was not reused")
	}
	sharedTestPackage(t, m, "2.0")
	dirsNew, warnings := m.PrepareProfileExtensions(p, "unused-browser.exe", userDir, nil)
	if len(warnings) > 0 || len(dirsNew) != 1 || dirsNew[0] == dirs[0] {
		t.Fatal("changed package reused stale code")
	}
}

func TestProfilePreferencesNoOpKeepsFileIdentity(t *testing.T) {
	path := filepath.Join(t.TempDir(), "Preferences")
	data := map[string]any{"session": map[string]any{"restore_on_startup": 1}}
	if err := writeProfileJSON(path, data); err != nil {
		t.Fatal(err)
	}
	before, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if err := writeProfileJSON(path, data); err != nil {
		t.Fatal(err)
	}
	after, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if !os.SameFile(before, after) || !before.ModTime().Equal(after.ModTime()) {
		t.Fatal("identical preferences were replaced")
	}
}

func BenchmarkSharedExtensionPackageMetadata(b *testing.B) {
	root := b.TempDir()
	path := filepath.Join(root, "large.crx")
	// Representative package size; this benchmark measures metadata/hash I/O,
	// not ZIP extraction or total browser launch time.
	payload := make([]byte, 32<<20)
	copy(payload, []byte("Cr24"))
	if err := os.WriteFile(path, payload, 0600); err != nil {
		b.Fatal(err)
	}
	m := NewManager(config.DefaultConfig(), root)
	extension := Extension{ExtensionID: "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa", PackagePath: path}
	b.Run("cold", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			sharedExtensionPackages.Delete(path)
			if _, _, _, err := m.resolveSharedExtensionPackage(extension, ""); err != nil {
				b.Fatal(err)
			}
		}
	})
	b.Run("warm", func(b *testing.B) {
		if _, _, _, err := m.resolveSharedExtensionPackage(extension, ""); err != nil {
			b.Fatal(err)
		}
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			if _, _, _, err := m.resolveSharedExtensionPackage(extension, ""); err != nil {
				b.Fatal(err)
			}
		}
	})
}

func TestCompactChromiumPreferencesRemainUnchanged(t *testing.T) {
	path := filepath.Join(t.TempDir(), "Preferences")
	original := []byte(`{"session":{"restore_on_startup":1},"profile":{"password_manager_enabled":true}}`)
	if err := os.WriteFile(path, original, 0600); err != nil {
		t.Fatal(err)
	}
	before, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	root, err := readProfileJSON(path, false)
	if err != nil {
		t.Fatal(err)
	}
	root["session"].(map[string]any)["restore_on_startup"] = 1
	if err := writeProfileJSON(path, root); err != nil {
		t.Fatal(err)
	}
	after, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	bytes, err := os.ReadFile(path)
	if err != nil || string(bytes) != string(original) || !os.SameFile(before, after) || !before.ModTime().Equal(after.ModTime()) {
		t.Fatal("unchanged compact preferences were rewritten")
	}
}

func TestExtensionMessageIdentifiersIgnoreCase(t *testing.T) {
	if got := resolveExtensionMessage("__MSG_brandName__", map[string]string{"brandname": "Example Plugin"}); got != "Example Plugin" {
		t.Fatal(got)
	}
}

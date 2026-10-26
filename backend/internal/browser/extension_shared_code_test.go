package browser

import (
	"ant-chrome/backend/internal/config"
	"archive/zip"
	"bytes"
	"encoding/binary"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func sharedTestPackage(t *testing.T, m *Manager, version string) Extension {
	t.Helper()
	var archive bytes.Buffer
	z := zip.NewWriter(&archive)
	w, err := z.Create("manifest.json")
	if err != nil {
		t.Fatal(err)
	}
	_, err = w.Write([]byte(`{"name":"Shared Test","version":"` + version + `","manifest_version":3}`))
	if err != nil {
		t.Fatal(err)
	}
	w, err = z.Create("program.js")
	if err != nil {
		t.Fatal(err)
	}
	_, _ = w.Write([]byte("program version " + version))
	if err := z.Close(); err != nil {
		t.Fatal(err)
	}
	key := []byte("test-public-key")
	data := make([]byte, 16+len(key))
	copy(data, []byte("Cr24"))
	binary.LittleEndian.PutUint32(data[4:], 2)
	binary.LittleEndian.PutUint32(data[8:], uint32(len(key)))
	copy(data[16:], key)
	data = append(data, archive.Bytes()...)
	extension, err := m.InstallExtensionPackageBytes(extensionIDFromPublicKey(key), "shared test", data)
	if err != nil {
		t.Fatal(err)
	}
	if err := m.ExtensionDAO.SetDefaultInstall(extension.ExtensionID, true); err != nil {
		t.Fatal(err)
	}
	return extension
}

func TestSharedExtensionCodePreservesIndependentStorageAndPublishedVersions(t *testing.T) {
	root := t.TempDir()
	m := NewManager(config.DefaultConfig(), root)
	m.ExtensionDAO = newTestExtensionDAO(t, root)
	ext := sharedTestPackage(t, m, "1.0")
	users := []string{filepath.Join(root, "data", "window-a"), filepath.Join(root, "data", "window-b")}
	var firstCode string
	for i, user := range users {
		p := &Profile{ProfileId: filepath.Base(user), Keywords: []string{"one-browser-window:" + filepath.Base(user)}}
		packagePath, _, err := m.resolveExtensionPackage(ext, "no-browser.exe")
		if err != nil {
			t.Fatal(err)
		}
		id, err := installExtensionPackageIntoProfile(user, packagePath, ext)
		if err != nil {
			t.Fatal(err)
		}
		files := map[string]string{"Default/Network/Cookies": "cookies-" + p.ProfileId, "Default/Login Data": "password-" + p.ProfileId, "Default/Sessions/Session_1": "tabs-" + p.ProfileId, filepath.Join("Default", "Local Extension Settings", id, "000001.log"): "plugin-" + p.ProfileId}
		for name, value := range files {
			path := filepath.Join(user, name)
			if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(path, []byte(value), 0600); err != nil {
				t.Fatal(err)
			}
		}
		dirs, warnings := m.PrepareProfileExtensions(p, "no-browser.exe", user, nil)
		if len(warnings) > 0 || len(dirs) != 1 {
			t.Fatalf("%v %v", dirs, warnings)
		}
		if i == 0 {
			firstCode = dirs[0]
		} else if dirs[0] != firstCode {
			t.Fatal("code not shared")
		}
		if _, err := os.Stat(filepath.Join(user, "Default", "Extensions", id)); !os.IsNotExist(err) {
			t.Fatal("obsolete code copy survived")
		}
		for name, value := range files {
			data, err := os.ReadFile(filepath.Join(user, name))
			if err != nil || string(data) != value {
				t.Fatalf("profile storage changed: %s %v", name, err)
			}
		}
		data, err := os.ReadFile(filepath.Join(user, "Default", "Secure Preferences"))
		if err != nil {
			t.Fatal(err)
		}
		var prefs map[string]any
		if err := json.Unmarshal(data, &prefs); err != nil {
			t.Fatal(err)
		}
		setting := prefs["extensions"].(map[string]any)["settings"].(map[string]any)[id].(map[string]any)
		if setting["path"] != firstCode || setting["location"] != float64(3) {
			t.Fatal("shared unpacked registration missing", setting)
		}
	}
	oldCode, err := os.ReadFile(filepath.Join(firstCode, "program.js"))
	if err != nil {
		t.Fatal(err)
	}
	sharedTestPackage(t, m, "2.0")
	dirs, warnings := m.PrepareProfileExtensions(&Profile{ProfileId: "window-b", Keywords: []string{"one-browser-window:b"}}, "no-browser.exe", users[1], nil)
	if len(warnings) > 0 || len(dirs) != 1 || dirs[0] == firstCode {
		t.Fatalf("new version reused published path: %v %v", dirs, warnings)
	}
	oldAfter, err := os.ReadFile(filepath.Join(firstCode, "program.js"))
	if err != nil || !bytes.Equal(oldAfter, oldCode) {
		t.Fatal("running version overwritten")
	}
	// Removing one window's entire data cannot remove shared program files.
	if err := os.RemoveAll(users[1]); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(firstCode, "manifest.json")); err != nil {
		t.Fatal("shared code deleted with window")
	}
}

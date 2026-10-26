package backend

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestSettingsConfigurationRoundTripAndInvalidFiles(t *testing.T) {
	snapshot := map[string]interface{}{"format": "One Browser data", "version": 1, "localStorage": map[string]string{"one-browser-environments": "[]", "one-browser-accounts": "[]", "online-brand": "{\"name\":\"测试\"}"}}
	payload, err := oneBrowserConfigurationPayload(snapshot)
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(t.TempDir(), "config.json")
	if err = os.WriteFile(path, payload, 0600); err != nil {
		t.Fatal(err)
	}
	read, err := oneBrowserReadConfiguration(path)
	if err != nil || read["format"] != "One Browser data" {
		t.Fatal(err, read)
	}
	for _, body := range []string{`{"format":"other","version":1,"localStorage":{}}`, `{"format":"One Browser data","version":1,"localStorage":{"x":1}}`, `{"format":"One Browser data","version":1,"localStorage":{"one-browser-environments":"[null]"}}`, `{"format":"One Browser data","version":1,"localStorage":{"one-browser-accounts":"{}"}}`, `{"format":"One Browser data","version":1,"localStorage":{"one-browser-environments":"[{}]"}}`, `{bad`, strings.Repeat("x", 16*1024*1024+1)} {
		os.WriteFile(path, []byte(body), 0600)
		if _, err := oneBrowserReadConfiguration(path); err == nil {
			t.Fatal("accepted invalid config")
		}
	}
}
func TestSettingsClearChromeAndFirefoxCachesPreservesSites(t *testing.T) {
	a, p, _ := deletableWindow(t)
	p.Tags = []string{"One Browser"}
	root := a.browserMgr.ResolveUserDataDir(p)
	keep := putDirectoryTestData(t, root)
	for _, relative := range []string{"Default/Cache/Cache_Data", "Default/Code Cache", "cache2/entries", "startupCache"} {
		dir := filepath.Join(root, relative)
		os.MkdirAll(dir, 0700)
		os.WriteFile(filepath.Join(dir, "test"), []byte("cache"), 0600)
	}
	count, err := a.OneBrowserClearCache()
	if err != nil || count != 4 {
		t.Fatal(count, err)
	}
	checkDirectoryTestData(t, root, keep)
}
func TestSettingsInitializeRejectsLiveWindowsThenKeepsCore(t *testing.T) {
	a, p, _ := deletableWindow(t)
	a.config = a.browserMgr.Config
	p.Tags = []string{"One Browser"}
	root := a.browserMgr.ResolveUserDataDir(p)
	core := filepath.Join(a.appRoot, "chrome", "test-core", "core.bin")
	os.MkdirAll(filepath.Dir(core), 0700)
	os.WriteFile(core, []byte("keep-core"), 0600)
	p.Running = true
	if err := a.OneBrowserInitializeSystem(); err == nil {
		t.Fatal("reset accepted running browser")
	}
	if _, err := os.Stat(root); err != nil {
		t.Fatal("live data changed", err)
	}
	p.Running = false
	backup := filepath.Join(a.appRoot, "data", "backups", "one-browser-data.json")
	os.MkdirAll(filepath.Dir(backup), 0700)
	os.WriteFile(backup, []byte("snapshot"), 0600)
	if err := a.OneBrowserInitializeSystem(); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(root); !os.IsNotExist(err) {
		t.Fatal("profile data survived", err)
	}
	if _, err := os.Stat(backup); !os.IsNotExist(err) {
		t.Fatal("snapshot survived", err)
	}
	if b, err := os.ReadFile(core); err != nil || string(b) != "keep-core" {
		t.Fatal("core changed", err)
	}
	if len(a.BrowserProfileList()) != 0 {
		t.Fatal("profile record survived")
	}
}

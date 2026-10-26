package backend

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

// A real subprocess verifies the inherited transport and persistent lifecycle;
// it intentionally does not claim website compatibility for a Windows binary.
func TestCamoufoxProcessHelper(t *testing.T) {
	if os.Getenv("ONEBROWSER_CAMOUFOX_HELPER") != "1" {
		return
	}
	dir := ""
	for i, arg := range os.Args {
		if arg == "-profile" && i+1 < len(os.Args) {
			dir = os.Args[i+1]
		}
	}
	if dir == "" {
		os.Exit(10)
	}
	var mask map[string]any
	if json.Unmarshal([]byte(os.Getenv("CAMOU_CONFIG_1")), &mask) != nil {
		os.Exit(11)
	}
	data, _ := json.Marshal(mask)
	os.WriteFile(filepath.Join(dir, "test-fingerprint.json"), data, 0600)
	reader := bufio.NewReader(os.NewFile(3, "read"))
	writer := os.NewFile(4, "write")
	for {
		data, err := reader.ReadBytes(0)
		if err != nil {
			os.Exit(0)
		}
		var req struct {
			ID     uint64 `json:"id"`
			Method string `json:"method"`
		}
		if json.Unmarshal(data[:len(data)-1], &req) != nil {
			os.Exit(12)
		}
		if req.Method == "Browser.close" {
			os.WriteFile(filepath.Join(dir, "sessionstore.jsonlz4"), []byte("saved-session"), 0600)
			os.Exit(0)
		}
		response := fmt.Sprintf(`{"id":%d,"result":{"version":"Firefox/158.0"}}`, req.ID)
		writer.Write(append([]byte(response), 0))
	}
}
func TestCamoufoxPersistentLifecycleAndIsolation(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("subprocess fixture uses a POSIX wrapper; Windows cross-build is checked separately")
	}
	t.Setenv("ONEBROWSER_CAMOUFOX_HELPER", "1")
	a, p, other := pluginTestApp(t)
	a.config = a.browserMgr.Config
	a.runtimeStopped = true
	coreDir := filepath.Join(t.TempDir(), "camoufox-test")
	os.MkdirAll(coreDir, 0700)
	exe, _ := os.Executable()
	script := "#!/bin/sh\nexec '" + strings.ReplaceAll(exe, "'", "'\\''") + "' -test.run '^TestCamoufoxProcessHelper$' -- \"$@\"\n"
	os.WriteFile(filepath.Join(coreDir, "camoufox-bin"), []byte(script), 0700)
	os.WriteFile(filepath.Join(coreDir, "application.ini"), []byte("[App]\nVersion=158.0\n"), 0600)
	a.browserMgr.Config.Browser.Cores = append(a.browserMgr.Config.Browser.Cores, BrowserCore{CoreId: "camoufox", CoreName: "test", CorePath: coreDir})
	p.CoreId = "camoufox"
	p.ProxyId = "__direct__"
	p.ProxyConfig = "direct://"
	p.LaunchArgs = []string{"--lang=en-US", "--timezone=America/New_York", "--window-size=900,680"}
	root := a.browserMgr.ResolveUserDataDir(p)
	oldData := putDirectoryTestData(t, root)
	otherData := putDirectoryTestData(t, a.browserMgr.ResolveUserDataDir(other))
	started, err := a.OneBrowserStart(OneBrowserStartRequest{WindowID: testWindowA, ProfileID: p.ProfileId, CoreID: "camoufox", Name: p.ProfileName, Language: "en-US · 英文", Timezone: "America/New_York", WindowSize: "900 × 680"})
	got := a.browserMgr.Profiles[started.ProfileID]
	if err != nil {
		t.Fatal(err)
	}
	defer a.BrowserInstanceStop(p.ProfileId)
	if !got.Running || got.Pid == 0 || got.DebugReady || got.DebugPort != 0 {
		t.Fatalf("invalid Gecko runtime: %+v", got)
	}
	if err = a.OneBrowserDeleteWindow(testWindowA, p.ProfileId); err == nil {
		t.Fatal("running Gecko profile deleted")
	}
	root = a.browserMgr.ResolveUserDataDir(p)
	before, _ := os.ReadFile(filepath.Join(root, "camoufox", "test-fingerprint.json"))
	var mask map[string]any
	json.Unmarshal(before, &mask)
	if !strings.Contains(mask["navigator.userAgent"].(string), "Firefox/158.0") || strings.Contains(mask["navigator.userAgent"].(string), "Chrome/") {
		t.Fatal("incoherent Firefox UA", mask)
	}
	if _, err = a.OneBrowserRestartWindow(testWindowA, p.ProfileId); err != nil {
		t.Fatal(err)
	}
	root = a.browserMgr.ResolveUserDataDir(p)
	after, _ := os.ReadFile(filepath.Join(root, "camoufox", "test-fingerprint.json"))
	if string(before) != string(after) {
		t.Fatal("fingerprint changed across restart")
	}
	configured, err := a.OneBrowserRestartConfiguredWindow(OneBrowserStartRequest{WindowID: testWindowA, ProfileID: p.ProfileId, CoreID: "camoufox", Name: p.ProfileName, OS: "macOS 15", Language: "en-US · English", Timezone: "America/New_York", WindowSize: "900 × 680"})
	if err != nil || configured.ProfileID != p.ProfileId || !configured.Running {
		t.Fatal("configured restart failed", configured, err)
	}
	data, _ := os.ReadFile(filepath.Join(a.browserMgr.ResolveUserDataDir(p), "camoufox", "test-fingerprint.json"))
	var updated map[string]any
	json.Unmarshal(data, &updated)
	if updated["navigator.platform"] != "MacIntel" || updated["audio:seed"] != mask["audio:seed"] {
		t.Fatal("new OS not applied or stable seed lost", updated)
	}
	if err = a.OneBrowserStop(p.ProfileId); err != nil {
		t.Fatal(err)
	}
	if !camoufoxHasSession(root) {
		t.Fatal("session not persisted")
	}
	prefs, _ := os.ReadFile(filepath.Join(root, "camoufox", "user.js"))
	if !strings.Contains(string(prefs), `"signon.rememberSignons", true`) {
		t.Fatal("password persistence disabled")
	}
	checkDirectoryTestData(t, root, oldData)
	checkDirectoryTestData(t, a.browserMgr.ResolveUserDataDir(other), otherData)
}
func TestCamoufoxConfigAndProxyValidation(t *testing.T) {
	t.Setenv("CAMOU_CONFIG_99", "stale")
	t.Setenv("CAMOU_CONFIG", "stale legacy")
	env, err := camoufoxEnvironment(map[string]any{"x": strings.Repeat("汉", 4000)})
	if err != nil {
		t.Fatal(err)
	}
	combined := ""
	for _, item := range env {
		if item == "CAMOU_CONFIG_99=stale" || strings.HasPrefix(item, "CAMOU_CONFIG=") {
			t.Fatal("inherited config leaked")
		}
		if strings.HasPrefix(item, "CAMOU_CONFIG_") {
			combined += strings.SplitN(item, "=", 2)[1]
		}
	}
	var config map[string]any
	if json.Unmarshal([]byte(combined), &config) != nil || len([]rune(config["x"].(string))) != 4000 {
		t.Fatal("config chunks corrupted")
	}
	dir := t.TempDir()
	for _, proxy := range []string{"socks5://127.0.0.1:1080", "http://127.0.0.1:7890", "direct://"} {
		if err = writeCamoufoxPrefs(dir, proxy, nil); err != nil {
			t.Fatal(err)
		}
	}
	for _, proxy := range []string{"http://user:secret@host:1234", "https://host:443", "bad"} {
		if err = writeCamoufoxPrefs(dir, proxy, nil); err == nil {
			t.Fatal("unsafe or unsupported proxy accepted", proxy)
		}
	}
	p := &BrowserProfile{ProfileId: "stable", LaunchArgs: []string{"--user-agent=Chrome/150", "--lang=zh-CN"}}
	config = camoufoxConfig(p, "158.0")
	if strings.Contains(config["navigator.userAgent"].(string), "Chrome") {
		t.Fatal("Chrome UA forwarded")
	}
}

package backend

import (
	"strings"
	"testing"
)

func TestGeneralWindowRestartGuardsAndRetainsProfileData(t *testing.T) {
	a, p, other := pluginTestApp(t)
	files := putDirectoryTestData(t, a.browserMgr.ResolveUserDataDir(p))
	if _, err := a.OneBrowserRestartWindow(testWindowB, p.ProfileId); err == nil {
		t.Fatal("foreign window restarted")
	}
	if _, err := a.OneBrowserRestartWindow(testWindowA, p.ProfileId); err == nil || !strings.Contains(err.Error(), "先启动") {
		t.Fatal("closed window restart accepted", err)
	}
	p.Running = true
	p.DebugPort = 0
	p.CoreId = "pinned-core"
	p.ProxyConfig = "http://proxy:1234"
	p.FingerprintArgs = []string{"--fingerprint=123"}
	if _, err := a.OneBrowserRestartWindow(testWindowA, p.ProfileId); err == nil || !strings.Contains(err.Error(), "取消重启") {
		t.Fatal("unresponsive browser restarted", err)
	}
	if !p.Running || p.CoreId != "pinned-core" || p.ProxyConfig != "http://proxy:1234" || p.FingerprintArgs[0] != "--fingerprint=123" {
		t.Fatal("failed restart rewrote profile")
	}
	other.UserDataDir = p.UserDataDir
	if _, err := a.OneBrowserRestartWindow(testWindowA, p.ProfileId); err == nil || !strings.Contains(err.Error(), "重复") {
		t.Fatal("restart bypassed isolation", err)
	}
	checkDirectoryTestData(t, a.browserMgr.ResolveUserDataDir(p), files)
}
func TestCoreImportExplainsEngineAndArchitecture(t *testing.T) {
	for _, tc := range []struct{ name, want string }{{"camoufox-158.0-beta.40-win.i686.zip", "32位"}, {"Firefox-146.zip", "尚未接入"}, {"chromium-win32.zip", "32位"}} {
		err := oneBrowserCoreImportCompatibility(tc.name)
		if err == nil || !strings.Contains(err.Error(), tc.want) {
			t.Fatalf("%s: %v", tc.name, err)
		}
	}
	if err := oneBrowserCoreImportCompatibility("camoufox-158.0-beta.40-win.x86_64.zip"); err != nil {
		t.Fatal(err)
	}
	if err := oneBrowserCoreImportCompatibility("ungoogled-chromium_150.0.7871.186-1.1_windows_x64.zip"); err != nil {
		t.Fatal(err)
	}
	a := NewApp(t.TempDir())
	if _, err := a.importLocalBrowserCoreArchive("camoufox-win.i686.zip"); err == nil || !strings.Contains(err.Error(), "32位") {
		t.Fatal("unsupported package was unpacked", err)
	}
}

package backend

import (
	"ant-chrome/backend/internal/browser"
	"ant-chrome/backend/internal/config"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func managedKernelFixture(t *testing.T) (*App, *BrowserProfile, string, string) {
	t.Helper()
	a, p := directoryTestApp(t)
	for _, v := range []string{"150", "148"} {
		dir := filepath.Join(a.appRoot, "chrome", v)
		if err := os.MkdirAll(dir, 0755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(dir, "manifest.json"), []byte(`{"version":"`+v+`.0.0.1"}`), 0600); err != nil {
			t.Fatal(err)
		}
		a.browserMgr.Config.Browser.Cores = append(a.browserMgr.Config.Browser.Cores, config.BrowserCore{CoreId: v, CoreName: v, CorePath: dir})
	}
	a.browserMgr.Profiles[p.ProfileId].CoreId = "150"
	return a, p, filepath.Join(a.appRoot, "chrome", "150"), filepath.Join(a.appRoot, "chrome", "148")
}
func TestCoreDeletionRequiresEveryWindowToDetach(t *testing.T) {
	a, p, oldDir, newDir := managedKernelFixture(t)
	if err := a.OneBrowserDeleteKernel("148", []OneBrowserKernelBinding{{WindowID: testWindowB, Name: "never-started", CoreID: "148"}}); err == nil {
		t.Fatal("UI-only window association ignored")
	}
	if err := a.OneBrowserDeleteKernel("150", nil); err == nil || !strings.Contains(err.Error(), "修改内核") {
		t.Fatalf("closed profile association ignored: %v", err)
	}
	files := putDirectoryTestData(t, a.browserMgr.ResolveUserDataDir(p))
	oldArgs := []string{"--fingerprint=123"}
	a.browserMgr.Profiles[p.ProfileId].FingerprintArgs = oldArgs
	if err := a.OneBrowserSetWindowKernel(testWindowB, p.ProfileId, "148"); err == nil {
		t.Fatal("foreign profile accepted")
	}
	a.browserMgr.Profiles[p.ProfileId].Running = true
	if err := a.OneBrowserSetWindowKernel(testWindowA, p.ProfileId, "148"); err == nil {
		t.Fatal("running core switched")
	}
	a.browserMgr.Profiles[p.ProfileId].Running = false
	if err := a.OneBrowserSetWindowKernel(testWindowA, p.ProfileId, "148"); err != nil {
		t.Fatal(err)
	}
	saved := a.browserMgr.Profiles[p.ProfileId]
	if saved.CoreId != "148" || len(saved.FingerprintArgs) != 1 || saved.FingerprintArgs[0] != oldArgs[0] {
		t.Fatal("kernel edit changed fingerprint data")
	}
	if err := a.OneBrowserDeleteKernel("150", nil); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(oldDir); !os.IsNotExist(err) {
		t.Fatal("managed core directory remains", err)
	}
	if _, err := os.Stat(newDir); err != nil {
		t.Fatal("other kernel removed", err)
	}
	checkDirectoryTestData(t, a.browserMgr.ResolveUserDataDir(p), files)
}
func TestExternalCoreDeletionOnlyUnregistersOriginalDirectory(t *testing.T) {
	a, _, _, _ := managedKernelFixture(t)
	external := t.TempDir()
	marker := filepath.Join(external, "keep")
	os.WriteFile(marker, []byte("original"), 0600)
	a.browserMgr.Config.Browser.Cores = append(a.browserMgr.Config.Browser.Cores, config.BrowserCore{CoreId: "external", CoreName: "external", CorePath: external})
	if err := a.OneBrowserDeleteKernel("external", nil); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(marker); err != nil {
		t.Fatal("external files were deleted", err)
	}
}
func TestImportedPluginDeleteClearsOnlyItsAssociations(t *testing.T) {
	a, first, second := pluginTestApp(t)
	id := "cccccccccccccccccccccccccccccccc"
	install := a.resolveAppPath("data/extensions/" + id)
	if err := os.MkdirAll(install, 0755); err != nil {
		t.Fatal(err)
	}
	ext := browser.Extension{ExtensionID: id, Name: "Imported", Version: "1.0", Enabled: true, InstallDir: install, InstallMode: browser.ExtensionInstallModePersistent}
	if err := a.browserMgr.ExtensionDAO.Upsert(ext); err != nil {
		t.Fatal(err)
	}
	if err := a.configureOneBrowserPlugins(first.ProfileId, []string{"adguard"}, []string{id}); err != nil {
		t.Fatal(err)
	}
	if err := a.configureOneBrowserPlugins(second.ProfileId, []string{"tampermonkey"}, nil); err != nil {
		t.Fatal(err)
	}
	a.browserMgr.Profiles[first.ProfileId].Running = true
	if err := a.OneBrowserDeleteImportedPlugin(id); err == nil {
		t.Fatal("running plugin deleted")
	}
	if _, err := a.browserMgr.ExtensionDAO.Get(id); err != nil {
		t.Fatal("failed deletion erased record")
	}
	a.browserMgr.Profiles[first.ProfileId].Running = false
	if err := a.OneBrowserDeleteImportedPlugin(id); err != nil {
		t.Fatal(err)
	}
	one, _ := a.browserMgr.ExtensionDAO.GetProfileSettings(first.ProfileId)
	two, _ := a.browserMgr.ExtensionDAO.GetProfileSettings(second.ProfileId)
	if len(one.ExtensionIDs) != 1 || one.ExtensionIDs[0] != oneBrowserBuiltinExtensions["adguard"] || len(two.ExtensionIDs) != 1 || two.ExtensionIDs[0] != oneBrowserBuiltinExtensions["tampermonkey"] {
		t.Fatal("other associations changed", one, two)
	}
	if _, err := os.Stat(install); !os.IsNotExist(err) {
		t.Fatal("plugin files remain", err)
	}
	if err := a.OneBrowserDeleteImportedPlugin(oneBrowserBuiltinExtensions["adguard"]); err == nil {
		t.Fatal("builtin deleted")
	}
}

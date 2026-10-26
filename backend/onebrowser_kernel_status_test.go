package backend

import (
	"os"
	"path/filepath"
	"testing"

	"ant-chrome/backend/internal/browser"
	"ant-chrome/backend/internal/config"
)

func TestOneBrowserKernelStatusUsesInstalledManifestVersion(t *testing.T) {
	root := t.TempDir()
	coreDir := filepath.Join(root, "chrome", "fingerprint-chromium-old")
	if err := os.MkdirAll(coreDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(coreDir, "manifest.json"), []byte(`{"version":"147.0.1"}`), 0o600); err != nil {
		t.Fatal(err)
	}
	cfg := config.DefaultConfig()
	cfg.Browser.Cores = []config.BrowserCore{{
		CoreId:   "fingerprint-core",
		CoreName: "fingerprint-chromium-old",
		CorePath: filepath.Join("chrome", "fingerprint-chromium-old"),
	}}
	app := NewApp(root)
	app.config = cfg
	app.browserMgr = browser.NewManager(cfg, root)

	status := app.OneBrowserKernelStatus()
	if !status.Installed || status.Version != "147.0.1" {
		t.Fatalf("kernel status = %#v", status)
	}
}

func TestOneBrowserMultipleKernelsAndExplicitSelection(t *testing.T) {
	root := t.TempDir()
	cfg := config.DefaultConfig()
	for _, item := range []struct{ id, version string }{{"core150", "150.0.7871.186"}, {"core148", "148.0.7778.215"}} {
		dir := filepath.Join(root, item.id)
		if err := os.MkdirAll(dir, 0755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(dir, "manifest.json"), []byte(`{"version":"`+item.version+`"}`), 0600); err != nil {
			t.Fatal(err)
		}
		cfg.Browser.Cores = append(cfg.Browser.Cores, config.BrowserCore{CoreId: item.id, CoreName: "arbitrary-folder", CorePath: dir})
	}
	app := NewApp(root)
	app.config = cfg
	app.browserMgr = browser.NewManager(cfg, root)
	if got := app.OneBrowserKernels(); len(got) != 2 || got[1].Version != "148.0.7778.215" {
		t.Fatalf("imported core missing: %#v", got)
	}
	selected, err := app.oneBrowserSelectedKernel("core148", "")
	if err != nil || selected.CoreID != "core148" {
		t.Fatalf("explicit selection: %#v %v", selected, err)
	}
	if app.OneBrowserKernelStatus().CoreID != "core150" {
		t.Fatal("importing older core changed default for new windows")
	}
	if _, err := app.oneBrowserSelectedKernel("removed", ""); err == nil {
		t.Fatal("removed core silently fell back")
	}
	profile, err := app.BrowserProfileCreate(BrowserProfileInput{ProfileName: "pinned", CoreId: "core148"})
	if err != nil {
		t.Fatal(err)
	}
	selected, err = app.oneBrowserSelectedKernel("", profile.ProfileId)
	if err != nil || selected.CoreID != "core148" {
		t.Fatalf("existing window not pinned: %#v %v", selected, err)
	}
}

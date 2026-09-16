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

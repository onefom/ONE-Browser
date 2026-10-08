package backend

import (
	"ant-chrome/backend/internal/config"
	"ant-chrome/backend/internal/logger"
	"ant-chrome/backend/internal/proxy"
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func TestRepairUnavailablePortableConnectorFallsBackToBundledXray(t *testing.T) {
	root := t.TempDir()
	binDir := filepath.Join(root, "bin")
	if err := os.MkdirAll(binDir, 0o755); err != nil {
		t.Fatal(err)
	}
	binaryName := "xray"
	if runtime.GOOS == "windows" {
		binaryName = "xray.exe"
	}
	if err := os.WriteFile(filepath.Join(binDir, binaryName), []byte("test runtime"), 0o755); err != nil {
		t.Fatal(err)
	}

	cfg := config.DefaultConfig()
	cfg.Browser.DefaultConnectorType = config.BrowserConnectorMihomo
	if err := cfg.Save(filepath.Join(root, "config.yaml")); err != nil {
		t.Fatal(err)
	}
	app := &App{
		appRoot:  root,
		config:   cfg,
		xrayMgr:  proxy.NewXrayManager(cfg, root),
		clashMgr: proxy.NewClashManager(cfg, root),
	}
	app.repairUnavailablePortableConnector(cfg, logger.New("test"))

	if cfg.Browser.DefaultConnectorType != config.BrowserConnectorXray {
		t.Fatalf("connector = %q, want xray", cfg.Browser.DefaultConnectorType)
	}
	reloaded, err := LoadConfig(filepath.Join(root, "config.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	if reloaded.Browser.DefaultConnectorType != config.BrowserConnectorXray {
		t.Fatalf("persisted connector = %q, want xray", reloaded.Browser.DefaultConnectorType)
	}
}

func TestRepairUnavailablePortableConnectorKeepsConfiguredMihomo(t *testing.T) {
	root := t.TempDir()
	binDir := filepath.Join(root, "bin")
	if err := os.MkdirAll(binDir, 0o755); err != nil {
		t.Fatal(err)
	}
	mihomoName := "mihomo"
	if runtime.GOOS == "windows" {
		mihomoName = "mihomo.exe"
	}
	if err := os.WriteFile(filepath.Join(binDir, mihomoName), []byte("test runtime"), 0o755); err != nil {
		t.Fatal(err)
	}
	cfg := config.DefaultConfig()
	cfg.Browser.DefaultConnectorType = config.BrowserConnectorMihomo
	app := &App{
		appRoot:  root,
		config:   cfg,
		xrayMgr:  proxy.NewXrayManager(cfg, root),
		clashMgr: proxy.NewClashManager(cfg, root),
	}
	app.repairUnavailablePortableConnector(cfg, logger.New("test"))
	if cfg.Browser.DefaultConnectorType != config.BrowserConnectorMihomo {
		t.Fatalf("connector = %q, want mihomo", cfg.Browser.DefaultConnectorType)
	}
}

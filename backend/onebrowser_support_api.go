package backend

import (
	"ant-chrome/backend/internal/browser"
	"ant-chrome/backend/internal/logger"
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	wailsruntime "github.com/wailsapp/wails/v2/pkg/runtime"
)

var oneBrowserBuiltinExtensions = map[string]string{
	"adguard":          "bgnkhhnnamicmpeenaelnjfhikgbkllg",
	"tampermonkey":     "dhdgffkkebhmkfjojejmpbldmpobfkfo",
	"google-translate": "aapbdbdomjkkjkaonfhkkikfgjllcleb",
}

// OneBrowserSyncBuiltinExtensions downloads missing enabled built-ins and marks
// them for persistent installation in every One Browser profile.
func (a *App) OneBrowserSyncBuiltinExtensions(enabledKeys []string) []string {
	if a == nil || a.browserMgr == nil || a.browserMgr.ExtensionDAO == nil || a.ctx == nil {
		return []string{"插件服务未初始化"}
	}
	desired := map[string]bool{}
	for _, key := range enabledKeys {
		desired[strings.TrimSpace(key)] = true
	}
	client, _, err := oneBrowserSystemProxyClient(browser.ExtensionDownloadTimeout())
	if err != nil {
		client = &http.Client{Timeout: browser.ExtensionDownloadTimeout()}
	}
	warnings := []string{}
	for key, extensionID := range oneBrowserBuiltinExtensions {
		extension, getErr := a.browserMgr.ExtensionDAO.Get(extensionID)
		if !desired[key] {
			if getErr == nil {
				_ = a.browserMgr.ExtensionDAO.SetDefaultInstall(extensionID, false)
			}
			continue
		}
		if getErr == sql.ErrNoRows {
			a.maintenanceMu.Lock()
			extension, getErr = a.browserMgr.InstallExtensionFromWebStoreWithHTTPClient(a.ctx, extensionID, client)
			a.maintenanceMu.Unlock()
		}
		if getErr != nil {
			warnings = append(warnings, key+": "+getErr.Error())
			continue
		}
		_ = a.browserMgr.ExtensionDAO.SetEnabled(extension.ExtensionID, true)
		if setErr := a.browserMgr.ExtensionDAO.SetDefaultInstall(extension.ExtensionID, true); setErr != nil {
			warnings = append(warnings, key+": "+setErr.Error())
		}
	}
	if len(warnings) == 0 {
		logger.New("Extension").Info("内置扩展已同步", logger.F("count", len(desired)))
	}
	return warnings
}

func (a *App) OneBrowserOpenManagedURL(profileID, targetURL string) error {
	opened, err := a.BrowserInstanceOpenUrl(strings.TrimSpace(profileID), strings.TrimSpace(targetURL))
	if err != nil {
		return err
	}
	if !opened {
		return fmt.Errorf("请先启动一个浏览器窗口")
	}
	return nil
}

func (a *App) OneBrowserSaveLocalData(snapshot map[string]interface{}) error {
	payload, err := json.MarshalIndent(snapshot, "", "  ")
	if err != nil {
		return err
	}
	dir := a.resolveAppPath(filepath.Join("data", "backups"))
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return err
	}
	target := filepath.Join(dir, "one-browser-data.json")
	return os.WriteFile(target, append(payload, '\n'), 0o600)
}

func (a *App) OneBrowserExportLogs() (string, error) {
	if a.ctx == nil {
		return "", fmt.Errorf("应用上下文未初始化")
	}
	entries := logger.GetMemoryWriter().GetEntries()
	files := map[string]string{}
	logDir := a.resolveAppPath(filepath.Join("data", "logs"))
	if diskEntries, readErr := os.ReadDir(logDir); readErr == nil {
		for _, entry := range diskEntries {
			if entry.IsDir() || !strings.HasSuffix(strings.ToLower(entry.Name()), ".log") {
				continue
			}
			if data, fileErr := os.ReadFile(filepath.Join(logDir, entry.Name())); fileErr == nil {
				files[entry.Name()] = string(data)
			}
		}
	}
	payload, err := json.MarshalIndent(map[string]interface{}{"exportedAt": time.Now().Format(time.RFC3339), "memory": entries, "files": files}, "", "  ")
	if err != nil {
		return "", err
	}
	name := "OneBrowser-logs-" + time.Now().Format("20060102-150405") + ".json"
	path, err := wailsruntime.SaveFileDialog(a.ctx, wailsruntime.SaveDialogOptions{Title: "导出 One Browser 日志", DefaultFilename: name, Filters: []wailsruntime.FileFilter{{DisplayName: "JSON 日志 (*.json)", Pattern: "*.json"}}})
	if err != nil {
		return "", err
	}
	if strings.TrimSpace(path) == "" {
		return "", nil
	}
	if err := os.WriteFile(path, append(payload, '\n'), 0o600); err != nil {
		return "", err
	}
	logger.New("Logs").Info("日志已导出", logger.F("path", filepath.Base(path)), logger.F("count", len(entries)))
	return path, nil
}

// OneBrowserClearOldLogs removes rotated log files older than the requested
// retention period and prunes the in-memory view without deleting recent logs.
func (a *App) OneBrowserClearOldLogs(days int) (int, error) {
	if days < 1 {
		days = 30
	}
	cutoff := time.Now().Add(-time.Duration(days) * 24 * time.Hour)
	logger.GetMemoryWriter().PruneBefore(cutoff)
	dir := a.resolveAppPath(filepath.Join("data", "logs"))
	entries, err := os.ReadDir(dir)
	if os.IsNotExist(err) {
		return 0, nil
	}
	if err != nil {
		return 0, err
	}
	removed := 0
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(strings.ToLower(entry.Name()), ".log") || entry.Name() == "app.log" {
			continue
		}
		info, infoErr := entry.Info()
		if infoErr == nil && info.ModTime().Before(cutoff) {
			if removeErr := os.Remove(filepath.Join(dir, entry.Name())); removeErr != nil {
				return removed, removeErr
			}
			removed++
		}
	}
	logger.New("Logs").Info("已清理过期日志", logger.F("days", days), logger.F("files", removed))
	return removed, nil
}

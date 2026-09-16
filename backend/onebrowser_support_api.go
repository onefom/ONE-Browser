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
	"sort"
	"strings"
	"time"

	wailsruntime "github.com/wailsapp/wails/v2/pkg/runtime"
)

// OneBrowserExportConfiguration writes the UI-owned configuration to a JSON
// file selected by the user. Browser profile folders are intentionally not
// embedded; the regular data backup remains responsible for full profiles.
func (a *App) OneBrowserExportConfiguration(snapshot map[string]interface{}) (string, error) {
	if a.ctx == nil {
		return "", fmt.Errorf("应用上下文未初始化")
	}
	payload, err := json.MarshalIndent(snapshot, "", "  ")
	if err != nil {
		return "", err
	}
	path, err := wailsruntime.SaveFileDialog(a.ctx, wailsruntime.SaveDialogOptions{
		Title: "导出 One Browser 配置", DefaultFilename: "OneBrowser-config-" + time.Now().Format("20060102-150405") + ".json",
		Filters: []wailsruntime.FileFilter{{DisplayName: "One Browser 配置 (*.json)", Pattern: "*.json"}},
	})
	if err != nil || strings.TrimSpace(path) == "" {
		return "", err
	}
	if err := os.WriteFile(path, append(payload, '\n'), 0o600); err != nil {
		return "", err
	}
	return path, nil
}

// OneBrowserImportConfiguration reads and validates a previously exported
// configuration. Applying it stays in the UI so localStorage is replaced as a
// single transaction before reloading.
func (a *App) OneBrowserImportConfiguration() (map[string]interface{}, error) {
	if a.ctx == nil {
		return nil, fmt.Errorf("应用上下文未初始化")
	}
	path, err := wailsruntime.OpenFileDialog(a.ctx, wailsruntime.OpenDialogOptions{
		Title:   "加载 One Browser 配置",
		Filters: []wailsruntime.FileFilter{{DisplayName: "One Browser 配置 (*.json)", Pattern: "*.json"}},
	})
	if err != nil || strings.TrimSpace(path) == "" {
		return nil, err
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var snapshot map[string]interface{}
	if err := json.Unmarshal(data, &snapshot); err != nil {
		return nil, fmt.Errorf("配置文件格式无效: %w", err)
	}
	if snapshot == nil || (snapshot["localStorage"] == nil && snapshot["format"] == nil) {
		return nil, fmt.Errorf("不是有效的 One Browser 配置文件")
	}
	return snapshot, nil
}

// OneBrowserClearCache removes only disposable cache directories belonging to
// One Browser profiles. Cookies, local storage and profile preferences remain.
func (a *App) OneBrowserClearCache() (int, error) {
	if a == nil || a.browserMgr == nil {
		return 0, fmt.Errorf("浏览器服务未初始化")
	}
	cacheNames := map[string]bool{"cache": true, "code cache": true, "gpucache": true, "shadercache": true, "grshadercache": true, "dawncache": true}
	paths := make([]string, 0)
	for _, profile := range a.BrowserProfileList() {
		if !oneBrowserProfile(profile) {
			continue
		}
		root := a.browserMgr.ResolveUserDataDir(&profile)
		_ = filepath.WalkDir(root, func(path string, entry os.DirEntry, walkErr error) error {
			if walkErr != nil || entry == nil || !entry.IsDir() {
				return nil
			}
			if cacheNames[strings.ToLower(entry.Name())] {
				paths = append(paths, path)
				return filepath.SkipDir
			}
			return nil
		})
	}
	sort.Slice(paths, func(i, j int) bool { return len(paths[i]) > len(paths[j]) })
	removed := 0
	for _, path := range paths {
		if err := os.RemoveAll(path); err != nil {
			return removed, err
		}
		removed++
	}
	return removed, nil
}

// OneBrowserInitializeSystem clears One Browser business data while retaining
// installed cores. The caller clears UI localStorage after this succeeds.
func (a *App) OneBrowserInitializeSystem() error {
	if a == nil || a.browserMgr == nil {
		return fmt.Errorf("浏览器服务未初始化")
	}
	for _, profile := range a.BrowserProfileList() {
		if !oneBrowserProfile(profile) {
			continue
		}
		_, _ = a.BrowserInstanceStop(profile.ProfileId)
		if err := a.BrowserProfileDelete(profile.ProfileId); err != nil {
			return err
		}
		if err := a.BrowserProfilePermanentlyDelete(profile.ProfileId); err != nil {
			return err
		}
	}
	if err := a.SaveBrowserProxies([]BrowserProxy{{ProxyId: "__direct__", ProxyName: "直连（不走代理）", ProxyConfig: "direct://"}}); err != nil {
		return err
	}
	for _, relative := range []string{filepath.Join("data", "workspaces"), filepath.Join("data", "backups")} {
		if err := os.RemoveAll(a.resolveAppPath(relative)); err != nil {
			return err
		}
	}
	logger.GetMemoryWriter().Clear()
	return nil
}

func oneBrowserProfile(profile BrowserProfile) bool {
	for _, tag := range profile.Tags {
		if strings.EqualFold(strings.TrimSpace(tag), "One Browser") {
			return true
		}
	}
	return false
}

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
		a.maintenanceMu.Lock()
		extension, getErr := a.browserMgr.ExtensionDAO.Get(extensionID)
		if !desired[key] {
			if getErr == nil {
				_ = a.browserMgr.ExtensionDAO.SetDefaultInstall(extensionID, false)
			}
			a.maintenanceMu.Unlock()
			continue
		}
		if getErr == sql.ErrNoRows {
			extension, getErr = a.browserMgr.InstallExtensionFromWebStoreWithHTTPClient(a.ctx, extensionID, client)
		}
		if getErr != nil {
			a.maintenanceMu.Unlock()
			warnings = append(warnings, key+": "+getErr.Error())
			continue
		}
		_ = a.browserMgr.ExtensionDAO.SetEnabled(extension.ExtensionID, true)
		if setErr := a.browserMgr.ExtensionDAO.SetDefaultInstall(extension.ExtensionID, true); setErr != nil {
			warnings = append(warnings, key+": "+setErr.Error())
		}
		a.maintenanceMu.Unlock()
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

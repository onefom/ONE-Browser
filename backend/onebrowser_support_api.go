package backend

import (
	"ant-chrome/backend/internal/browser"
	"ant-chrome/backend/internal/logger"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
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
	payload, err := oneBrowserConfigurationPayload(snapshot)
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
	snapshot, err := oneBrowserReadConfiguration(path)
	if err != nil {
		return nil, err
	}
	if err := a.oneBrowserRequireStoppedProfiles(); err != nil {
		return nil, err
	}
	return snapshot, nil
}
func oneBrowserConfigurationPayload(snapshot map[string]interface{}) ([]byte, error) {
	if err := oneBrowserValidateConfiguration(snapshot); err != nil {
		return nil, err
	}
	payload, err := json.MarshalIndent(snapshot, "", "  ")
	if err != nil {
		return nil, err
	}
	if len(payload)+1 > 16*1024*1024 {
		return nil, fmt.Errorf("配置文件超过 16 MB")
	}
	return payload, nil
}
func oneBrowserReadConfiguration(path string) (map[string]interface{}, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()
	data, err := io.ReadAll(io.LimitReader(file, 16*1024*1024+1))
	if len(data) > 16*1024*1024 {
		return nil, fmt.Errorf("配置文件超过 16 MB")
	}
	if err != nil {
		return nil, err
	}
	var snapshot map[string]interface{}
	if err := json.Unmarshal(data, &snapshot); err != nil {
		return nil, fmt.Errorf("配置文件格式无效: %w", err)
	}
	if err := oneBrowserValidateConfiguration(snapshot); err != nil {
		return nil, err
	}
	return snapshot, nil
}

// OneBrowserClearCache removes only disposable cache directories belonging to
// One Browser profiles. Cookies, local storage and profile preferences remain.
func (a *App) OneBrowserClearCache() (int, error) {
	if a == nil || a.browserMgr == nil {
		return 0, fmt.Errorf("浏览器服务未初始化")
	}
	a.oneBrowserWindowMu.Lock()
	defer a.oneBrowserWindowMu.Unlock()
	a.browserMgr.InitData()
	a.browserMgr.Mutex.Lock()
	defer a.browserMgr.Mutex.Unlock()
	profiles := []BrowserProfile{}
	for _, p := range a.browserMgr.Profiles {
		if !oneBrowserProfile(*p) {
			continue
		}
		if p.Running || a.browserMgr.BrowserProcesses[p.ProfileId] != nil {
			return 0, fmt.Errorf("请先关闭全部浏览器窗口再清理缓存")
		}
		active, err := findBrowserUserDataProcesses(a.browserMgr.ResolveUserDataDir(p))
		if err != nil {
			return 0, fmt.Errorf("无法确认浏览器已退出：%w", err)
		}
		if len(active) > 0 {
			return 0, fmt.Errorf("请先关闭全部浏览器窗口再清理缓存")
		}
		profiles = append(profiles, *p)
	}
	cacheNames := map[string]bool{"cache": true, "code cache": true, "gpucache": true, "shadercache": true, "grshadercache": true, "dawncache": true, "gpupersistentcache": true, "cache2": true, "startupcache": true}
	paths := make([]string, 0)
	for _, profile := range profiles {
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
	if a == nil || a.browserMgr == nil || a.config == nil {
		return fmt.Errorf("浏览器服务未初始化")
	}
	a.oneBrowserWindowMu.Lock()
	defer a.oneBrowserWindowMu.Unlock()
	if err := a.oneBrowserRequireStoppedProfiles(); err != nil {
		return err
	}
	for _, profile := range a.BrowserProfileList() {
		if !oneBrowserProfile(profile) {
			continue
		}
		if owner := oneBrowserWindowOwner(profile); owner != "" {
			if err := a.oneBrowserDeleteWindowLocked(owner, profile.ProfileId); err != nil {
				return err
			}
		} else {
			if err := a.BrowserProfileDelete(profile.ProfileId); err != nil {
				return err
			}
			if err := a.BrowserProfilePermanentlyDelete(profile.ProfileId); err != nil {
				return err
			}
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
	a.oneBrowserIPMu.Lock()
	if a.oneBrowserIPMonitor != nil {
		a.oneBrowserIPMonitor.pending = nil
		a.oneBrowserIPMonitor.routes = map[string]*oneBrowserIPRoute{}
	}
	a.oneBrowserIPMu.Unlock()
	logger.GetMemoryWriter().Clear()
	return nil
}

func oneBrowserValidateConfiguration(snapshot map[string]interface{}) error {
	if snapshot == nil || snapshot["format"] != "One Browser data" {
		return fmt.Errorf("不是有效的 One Browser 配置文件")
	}
	version, err := json.Marshal(snapshot["version"])
	if err != nil || string(version) != "1" {
		return fmt.Errorf("配置文件版本不受支持")
	}
	data, err := json.Marshal(snapshot["localStorage"])
	if err != nil {
		return fmt.Errorf("配置数据无效")
	}
	var stored map[string]string
	if json.Unmarshal(data, &stored) != nil || stored == nil {
		return fmt.Errorf("配置文件缺少有效的 localStorage 数据")
	}
	for _, key := range []string{"one-browser-environments", "one-browser-accounts", "one-browser-proxy-nodes"} {
		if raw, ok := stored[key]; ok {
			var entries []map[string]interface{}
			if json.Unmarshal([]byte(raw), &entries) != nil || entries == nil {
				return fmt.Errorf("配置文件中的列表数据无效")
			}
			for _, entry := range entries {
				field := "name"
				if key == "one-browser-accounts" {
					field = "service"
				}
				name, valid := entry[field].(string)
				if entry == nil || !valid || strings.TrimSpace(name) == "" {
					return fmt.Errorf("配置文件中的列表数据无效")
				}
			}
		}
	}
	return nil
}
func (a *App) oneBrowserRequireStoppedProfiles() error {
	if a.browserMgr == nil {
		return fmt.Errorf("浏览器服务未初始化")
	}
	for _, p := range a.BrowserProfileList() {
		if !oneBrowserProfile(p) {
			continue
		}
		a.browserMgr.Mutex.Lock()
		tracked := a.browserMgr.BrowserProcesses[p.ProfileId] != nil
		a.browserMgr.Mutex.Unlock()
		if p.Running || tracked {
			return fmt.Errorf("请先关闭全部浏览器窗口再操作")
		}
		active, err := findBrowserUserDataProcesses(a.browserMgr.ResolveUserDataDir(&p))
		if err != nil || len(active) > 0 {
			return fmt.Errorf("无法确认全部浏览器已退出，请关闭后重试")
		}
	}
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
	if len(enabledKeys) == 0 {
		return []string{}
	}
	if a == nil || a.browserMgr == nil || a.browserMgr.ExtensionDAO == nil || a.ctx == nil {
		return []string{"插件服务未初始化"}
	}
	trace := newBrowserStartupTrace("内置插件同步明细", "builtin")
	defer trace.finish()
	desired := map[string]bool{}
	for _, key := range enabledKeys {
		desired[strings.TrimSpace(key)] = true
	}
	client, _, err := oneBrowserSystemProxyClient(browser.ExtensionDownloadTimeout())
	if err != nil {
		client = &http.Client{Timeout: browser.ExtensionDownloadTimeout()}
	}
	warnings := []string{}
	trace.step("系统代理下载客户端")
	for key, extensionID := range oneBrowserBuiltinExtensions {
		a.maintenanceMu.Lock()
		extension, getErr := a.browserMgr.ExtensionDAO.Get(extensionID)
		if !desired[key] {
			if getErr == nil && extension.DefaultInstall {
				_ = a.browserMgr.ExtensionDAO.SetDefaultInstall(extensionID, false)
			}
			a.maintenanceMu.Unlock()
			trace.step("跳过未启用插件 " + key)
			continue
		}
		if getErr == sql.ErrNoRows {
			extension, getErr = a.browserMgr.InstallExtensionFromWebStoreWithHTTPClient(a.ctx, extensionID, client)
		}
		if getErr != nil {
			a.maintenanceMu.Unlock()
			warnings = append(warnings, key+": "+getErr.Error())
			trace.step("插件准备返回错误 " + key)
			continue
		}
		if !extension.Enabled {
			if err := a.browserMgr.ExtensionDAO.SetEnabled(extension.ExtensionID, true); err != nil {
				warnings = append(warnings, key+": "+err.Error())
			}
		}
		if extension.DefaultInstall {
			if setErr := a.browserMgr.ExtensionDAO.SetDefaultInstall(extension.ExtensionID, false); setErr != nil {
				warnings = append(warnings, key+": "+setErr.Error())
			}
		}
		a.maintenanceMu.Unlock()
		trace.step("准备已启用插件 " + key)
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

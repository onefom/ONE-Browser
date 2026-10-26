package backend

import (
	"ant-chrome/backend/internal/browser"
	"fmt"
	"slices"
	"strings"
	"time"
)

type OneBrowserWindowPluginResult struct {
	ProfileID string `json:"profileId"`
	Running   bool   `json:"running"`
}

func oneBrowserPluginIDs(keys, extraIDs []string) ([]string, error) {
	ids := []string{}
	for _, key := range keys {
		id, ok := oneBrowserBuiltinExtensions[key]
		if !ok {
			return nil, fmt.Errorf("未知内置插件：%s", key)
		}
		if !slices.Contains(ids, id) {
			ids = append(ids, id)
		}
	}
	for _, id := range extraIDs {
		if browser.NormalizeExtensionID(id) != id {
			return nil, fmt.Errorf("插件ID无效")
		}
		if !slices.Contains(ids, id) {
			ids = append(ids, id)
		}
	}
	return ids, nil
}

// Explicit profile settings prevent global defaults leaking into new windows.
// Only the DB association changes here; active Chromium preferences are untouched.
func (a *App) configureOneBrowserPlugins(profileID string, keys, extraIDs []string, removedIDs ...string) error {
	ids, err := oneBrowserPluginIDs(keys, extraIDs)
	if err != nil {
		return err
	}
	if a.browserMgr == nil || a.browserMgr.ExtensionDAO == nil {
		return fmt.Errorf("插件服务未初始化")
	}
	dao := a.browserMgr.ExtensionDAO
	previous, err := dao.GetProfileSettings(profileID)
	if err != nil {
		return err
	}
	// Preserve explicitly associated imported extensions unknown to an older UI.
	if previous.Configured {
		for _, id := range previous.ExtensionIDs {
			builtin := false
			for _, known := range oneBrowserBuiltinExtensions {
				if known == id {
					builtin = true
					break
				}
			}
			if !builtin && !slices.Contains(removedIDs, id) && !slices.Contains(ids, id) {
				ids = append(ids, id)
			}
		}
	}
	if !previous.Configured {
		states, stateErr := dao.ListProfileExtensionRuntime(profileID)
		if stateErr != nil {
			return stateErr
		}
		for _, state := range states {
			builtin := false
			for _, known := range oneBrowserBuiltinExtensions {
				if known == state.ExtensionID {
					builtin = true
					break
				}
			}
			if !builtin && !slices.Contains(removedIDs, state.ExtensionID) && state.Status == browser.ExtensionRuntimeStatusInstalled && !slices.Contains(ids, state.ExtensionID) {
				ids = append(ids, state.ExtensionID)
			}
		}
	}
	for _, id := range ids {
		extension, err := dao.Get(id)
		if err != nil {
			return fmt.Errorf("插件尚未准备：%s：%w", id, err)
		}
		if !extension.Enabled {
			if err := dao.SetEnabled(id, true); err != nil {
				return err
			}
		}
	}
	old := append([]string{}, previous.ExtensionIDs...)
	desired := append([]string{}, ids...)
	slices.Sort(old)
	slices.Sort(desired)
	if previous.Configured && slices.Equal(old, desired) {
		return nil
	}
	_, err = dao.SetProfileSettings(profileID, ids, true)
	return err
}

func (a *App) OneBrowserSetWindowPlugins(windowID, profileID string, keys, extraIDs []string) (OneBrowserWindowPluginResult, error) {
	return a.OneBrowserSetWindowPluginsWithRemoval(windowID, profileID, keys, extraIDs, nil)
}

// Removal is explicit so older interfaces cannot erase imported associations.
func (a *App) OneBrowserSetWindowPluginsWithRemoval(windowID, profileID string, keys, extraIDs, removedIDs []string) (OneBrowserWindowPluginResult, error) {
	if _, err := oneBrowserPluginIDs(nil, removedIDs); err != nil {
		return OneBrowserWindowPluginResult{}, err
	}
	for _, id := range removedIDs {
		if slices.Contains(extraIDs, id) {
			return OneBrowserWindowPluginResult{}, fmt.Errorf("插件不能同时关联和停用")
		}
	}

	if _, err := oneBrowserPluginIDs(keys, extraIDs); err != nil {
		return OneBrowserWindowPluginResult{}, err
	}
	id, err := oneBrowserWindowID(windowID)
	if err != nil {
		return OneBrowserWindowPluginResult{}, err
	}
	if a.browserMgr == nil {
		return OneBrowserWindowPluginResult{}, fmt.Errorf("浏览器服务未初始化")
	}
	// Validate ownership before any opt-in download, then revalidate afterwards.
	validate := func() (*BrowserProfile, error) {
		return resolveOneBrowserWindowProfile(id, strings.TrimSpace(profileID), a.BrowserProfileList())
	}
	profile, err := validate()
	if err != nil {
		return OneBrowserWindowPluginResult{}, err
	}
	if profile == nil && strings.TrimSpace(profileID) != "" {
		return OneBrowserWindowPluginResult{}, fmt.Errorf("窗口配置已删除")
	}
	if profile != nil && a.isCamoufoxProfile(profile) && (len(keys) > 0 || len(extraIDs) > 0) {
		return OneBrowserWindowPluginResult{}, fmt.Errorf("Firefox 请使用 Firefox 扩展，不能关联当前 Chrome 插件")
	}
	if warnings := a.OneBrowserSyncBuiltinExtensions(keys); len(warnings) > 0 {
		return OneBrowserWindowPluginResult{}, fmt.Errorf("插件准备失败：%s", strings.Join(warnings, "；"))
	}
	a.oneBrowserWindowMu.Lock()
	defer a.oneBrowserWindowMu.Unlock()
	profile, err = validate()
	if err != nil {
		return OneBrowserWindowPluginResult{}, err
	}
	if profile == nil {
		if profileID != "" {
			return OneBrowserWindowPluginResult{}, fmt.Errorf("窗口配置已删除")
		}
		return OneBrowserWindowPluginResult{}, nil // UI-only window: applied at first start.
	}
	if err := a.configureOneBrowserPlugins(profile.ProfileId, keys, extraIDs, removedIDs...); err != nil {
		return OneBrowserWindowPluginResult{}, err
	}
	return OneBrowserWindowPluginResult{ProfileID: profile.ProfileId, Running: profile.Running}, nil
}

// This action is offered only after an explicit restart choice in the picker.
// Require a graceful CDP close; never kill an unresponsive browser for a plugin.
func (a *App) OneBrowserRestartWindowForPlugins(windowID, profileID string) (OneBrowserWindowPluginResult, error) {
	result, err := a.oneBrowserRestartWindow(windowID, profileID)
	if err != nil {
		return result, fmt.Errorf("插件关联已保存；%w", err)
	}
	return result, nil
}

// Restart the same owned profile gracefully, retaining its core and user data.
func (a *App) OneBrowserRestartWindow(windowID, profileID string) (OneBrowserWindowPluginResult, error) {
	return a.oneBrowserRestartWindow(windowID, profileID)
}
func (a *App) OneBrowserRestartConfiguredWindow(request OneBrowserStartRequest) (OneBrowserWindowPluginResult, error) {
	if _, err := a.oneBrowserSelectedKernel(request.CoreID, request.ProfileID); err != nil {
		return OneBrowserWindowPluginResult{}, err
	}
	if _, err := oneBrowserPluginIDs(request.Extensions, request.ExtensionIDs); err != nil {
		return OneBrowserWindowPluginResult{}, err
	}
	return a.oneBrowserRestartWindowWithRequest(request.WindowID, request.ProfileID, &request)
}
func (a *App) oneBrowserRestartWindow(windowID, profileID string) (OneBrowserWindowPluginResult, error) {
	return a.oneBrowserRestartWindowWithRequest(windowID, profileID, nil)
}
func (a *App) oneBrowserRestartWindowWithRequest(windowID, profileID string, request *OneBrowserStartRequest) (OneBrowserWindowPluginResult, error) {
	if a.browserMgr == nil {
		return OneBrowserWindowPluginResult{}, fmt.Errorf("浏览器服务未初始化")
	}
	id, err := oneBrowserWindowID(windowID)
	if err != nil {
		return OneBrowserWindowPluginResult{}, err
	}
	a.oneBrowserWindowMu.Lock()
	profile, err := resolveOneBrowserWindowProfile(id, profileID, a.BrowserProfileList())
	a.oneBrowserWindowMu.Unlock()
	if err != nil {
		return OneBrowserWindowPluginResult{}, err
	}
	if profile == nil {
		return OneBrowserWindowPluginResult{}, fmt.Errorf("窗口配置不存在")
	}
	if !profile.Running {
		return OneBrowserWindowPluginResult{ProfileID: profile.ProfileId}, fmt.Errorf("窗口已关闭，请先启动窗口")
	}
	if err := a.validateOneBrowserDataIsolation(profile); err != nil {
		return OneBrowserWindowPluginResult{}, err
	}
	if a.isCamoufoxProfile(profile) {
		if _, _, err := a.stopCamoufox(profile.ProfileId); err != nil {
			return OneBrowserWindowPluginResult{}, err
		}
	} else if !tryCloseBrowserViaCDP(profile.DebugPort, 8*time.Second) {
		return OneBrowserWindowPluginResult{}, fmt.Errorf("浏览器未能正常关闭，已取消重启；请手动关闭后再启动")
	}
	deadline := time.Now().Add(5 * time.Second)
	for {
		processes, scanErr := findBrowserUserDataProcesses(a.browserMgr.ResolveUserDataDir(profile))
		if scanErr != nil {
			return OneBrowserWindowPluginResult{}, fmt.Errorf("无法确认浏览器正常退出，已取消重启：%w", scanErr)
		}
		if len(processes) == 0 {
			break
		}
		if time.Now().After(deadline) {
			return OneBrowserWindowPluginResult{}, fmt.Errorf("浏览器仍在正常退出中，请稍后手动启动")
		}
		time.Sleep(150 * time.Millisecond)
	}
	if _, err := a.BrowserInstanceStop(profile.ProfileId); err != nil {
		return OneBrowserWindowPluginResult{}, err
	}
	if request != nil {
		request.ProfileID = profile.ProfileId
		started, err := a.OneBrowserStart(*request)
		return OneBrowserWindowPluginResult{ProfileID: started.ProfileID, Running: started.Running}, err
	}
	a.oneBrowserWindowMu.Lock()
	defer a.oneBrowserWindowMu.Unlock()
	current, resolveErr := resolveOneBrowserWindowProfile(id, profile.ProfileId, a.BrowserProfileList())
	if resolveErr != nil {
		return OneBrowserWindowPluginResult{}, resolveErr
	}
	if current == nil {
		return OneBrowserWindowPluginResult{}, fmt.Errorf("窗口配置已删除")
	}
	if err := a.validateOneBrowserDataIsolation(current); err != nil {
		return OneBrowserWindowPluginResult{}, err
	}
	if _, err := a.syncOneBrowserDirectory(current.ProfileId, current.ProfileName, id); err != nil {
		return OneBrowserWindowPluginResult{}, err
	}
	started, err := a.BrowserInstanceStart(current.ProfileId)
	if err != nil {
		return OneBrowserWindowPluginResult{}, err
	}
	return OneBrowserWindowPluginResult{ProfileID: started.ProfileId, Running: started.Running}, nil
}

func (a *App) OneBrowserImportWindowExtension() (BrowserExtension, error) {
	extension, err := a.BrowserExtensionInstallLocalFile()
	if err != nil {
		return extension, err
	}
	if err := a.browserMgr.ExtensionDAO.SetDefaultInstall(extension.ExtensionID, false); err != nil {
		return extension, err
	}
	extension.DefaultInstall = false
	return extension, nil
}

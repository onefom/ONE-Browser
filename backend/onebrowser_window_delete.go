package backend

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// The same lock used by start and rename prevents deletion racing either action.
// A stale UI state is never sufficient evidence that Chromium has exited.
func (a *App) OneBrowserDeleteWindow(windowID, profileID string) error {
	id, err := oneBrowserWindowID(windowID)
	if err != nil {
		return err
	}
	if a.browserMgr == nil {
		return fmt.Errorf("浏览器服务未就绪")
	}
	a.oneBrowserWindowMu.Lock()
	defer a.oneBrowserWindowMu.Unlock()
	return a.oneBrowserDeleteWindowLocked(id, profileID)
}

func (a *App) oneBrowserDeleteWindowLocked(id, profileID string) error {
	m := a.browserMgr
	m.InitData()
	m.Mutex.Lock()
	defer m.Mutex.Unlock()
	var profile *BrowserProfile
	for _, p := range m.Profiles {
		if oneBrowserWindowOwner(*p) == id {
			if profile != nil {
				return fmt.Errorf("窗口存在重复绑定，未删除数据")
			}
			profile = p
		}
	}
	if profile == nil {
		if strings.TrimSpace(profileID) != "" {
			if m.Profiles[profileID] != nil {
				return fmt.Errorf("窗口配置归属不匹配")
			}
			return a.finishPendingOneBrowserDeleteLocked(id, profileID)
		}
		return nil // A window which has never been started has no browser data.
	}
	if profileID != "" && profile.ProfileId != profileID {
		return fmt.Errorf("窗口配置归属不匹配")
	}
	if profile.Running || m.BrowserProcesses[profile.ProfileId] != nil {
		return fmt.Errorf("浏览器仍在运行，请先关闭或停止该窗口后再删除")
	}
	target := m.ResolveUserDataDir(profile)
	stage := target + ".deleting-" + id
	if _, err := os.Lstat(target); os.IsNotExist(err) {
		if _, pendingErr := os.Lstat(stage); pendingErr == nil {
			marker, readErr := oneBrowserReadDirectoryOwner(stage)
			if readErr != nil || marker.ProfileID != profile.ProfileId || marker.WindowID != id {
				return fmt.Errorf("待删除目录归属不匹配")
			}
			active, scanErr := findBrowserUserDataProcesses(stage)
			if scanErr != nil || len(active) > 0 {
				return fmt.Errorf("待删除目录仍被占用，无法恢复")
			}
			if err := os.Rename(stage, target); err != nil {
				return fmt.Errorf("恢复中断删除失败：%w", err)
			}
		}
	}
	processes, err := findBrowserUserDataProcesses(target)
	if err != nil {
		return fmt.Errorf("无法确认浏览器已退出，已阻止删除：%w", err)
	}
	if len(processes) != 0 {
		return fmt.Errorf("浏览器仍在运行，请先关闭或停止该窗口后再删除")
	}
	root := strings.TrimSpace(m.Config.Browser.UserDataRoot)
	if root == "" {
		root = "data"
	}
	root = canonicalOneBrowserDataDir(m.ResolveRelativePath(root))
	canonical := canonicalOneBrowserDataDir(target)
	rel, err := filepath.Rel(root, canonical)
	if err != nil || rel == "." || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return fmt.Errorf("窗口目录超出数据根目录，已阻止删除")
	}
	for _, p := range m.Profiles {
		if p.ProfileId == profile.ProfileId {
			continue
		}
		other := canonicalOneBrowserDataDir(m.ResolveUserDataDir(p))
		if oneBrowserPathsOverlap(canonical, other) {
			return fmt.Errorf("目录与其他窗口重叠，已阻止删除")
		}
	}
	if m.ProfileDAO != nil {
		deleted, err := m.ProfileDAO.ListDeleted()
		if err != nil {
			return err
		}
		for _, p := range deleted {
			if oneBrowserPathsOverlap(canonical, canonicalOneBrowserDataDir(m.ResolveUserDataDir(p))) {
				return fmt.Errorf("目录与回收站窗口重叠，已阻止删除")
			}
		}
	}
	info, statErr := os.Lstat(target)
	if statErr != nil && !os.IsNotExist(statErr) {
		return statErr
	}
	if statErr == nil {
		if !info.IsDir() || info.Mode()&os.ModeSymlink != 0 || !strings.EqualFold(canonical, filepath.Join(canonicalOneBrowserDataDir(filepath.Dir(target)), filepath.Base(target))) {
			return fmt.Errorf("窗口目录为链接或路径异常，已阻止删除")
		}
		marker, err := oneBrowserReadDirectoryOwner(target)
		if err != nil || marker.ProfileID != profile.ProfileId || marker.WindowID != id {
			return fmt.Errorf("目录归属校验失败，未删除数据；请先重启程序同步目录")
		}
	}
	// Move first, so a persistence failure can restore the complete directory.
	moved := statErr == nil
	if moved {
		if _, err := os.Lstat(stage); !os.IsNotExist(err) {
			return fmt.Errorf("存在待处理删除目录，未覆盖数据")
		}
		if err := os.Rename(target, stage); err != nil {
			return fmt.Errorf("目录仍被占用或无法移动，窗口未删除：%w", err)
		}
	}
	delete(m.Profiles, profile.ProfileId)
	if m.ProfileDAO != nil {
		err = m.ProfileDAO.Delete(profile.ProfileId)
	} else {
		err = m.SaveProfiles()
	}
	if err != nil {
		m.Profiles[profile.ProfileId] = profile
		if moved {
			if rollbackErr := os.Rename(stage, target); rollbackErr != nil {
				return fmt.Errorf("保存失败：%v；数据保留于 %s：%w", err, stage, rollbackErr)
			}
		}
		return fmt.Errorf("删除配置失败，原窗口已保留：%w", err)
	}
	if moved {
		if err := os.RemoveAll(stage); err != nil {
			// Keep a retryable record even when Windows only permits partial removal.
			_ = oneBrowserWriteDirectoryOwner(stage, oneBrowserDirectoryOwner{ProfileID: profile.ProfileId, WindowID: id, ShortID: oneBrowserFolderID(*profile)})
			moveErr := os.Rename(stage, target)
			if moveErr != nil {
				profile.UserDataDir = stage
			}
			m.Profiles[profile.ProfileId] = profile
			saveErr := a.saveOneBrowserDirectoryProfileLocked(profile)
			return fmt.Errorf("目录清理未完成，请关闭占用程序后重试：%v（恢复路径：%v，保存：%v）", err, moveErr, saveErr)
		}
	}
	if m.ExtensionDAO != nil {
		_ = m.ExtensionDAO.DeleteProfileSettings(profile.ProfileId)
		_ = m.ExtensionDAO.DeleteProfileExtensionRuntimeForProfile(profile.ProfileId)
	}
	if m.CodeProvider != nil {
		_ = m.CodeProvider.Remove(profile.ProfileId)
	}
	_ = os.Remove(a.resolveAppPath(filepath.Join("data", "workspaces", profile.ProfileId+".html")))
	return nil
}

// If persistence committed immediately before an application interruption, only
// remove a staged directory whose full identities match this explicit retry.
func (a *App) finishPendingOneBrowserDeleteLocked(windowID, profileID string) error {
	m := a.browserMgr
	root := strings.TrimSpace(m.Config.Browser.UserDataRoot)
	if root == "" {
		root = "data"
	}
	root = m.ResolveRelativePath(root)
	entries, err := os.ReadDir(root)
	if os.IsNotExist(err) {
		return nil
	}
	if err != nil {
		return err
	}
	for _, entry := range entries {
		if !strings.HasSuffix(entry.Name(), ".deleting-"+windowID) || !entry.IsDir() {
			continue
		}
		path := filepath.Join(root, entry.Name())
		info, err := os.Lstat(path)
		if err != nil {
			return err
		}
		if info.Mode()&os.ModeSymlink != 0 {
			return fmt.Errorf("待删除目录为链接")
		}
		marker, err := oneBrowserReadDirectoryOwner(path)
		if err != nil || marker.WindowID != windowID || marker.ProfileID != profileID {
			return fmt.Errorf("待删除目录归属不匹配")
		}
		for _, p := range m.Profiles {
			if oneBrowserPathsOverlap(canonicalOneBrowserDataDir(path), canonicalOneBrowserDataDir(m.ResolveUserDataDir(p))) {
				return fmt.Errorf("待删除目录仍有配置引用")
			}
		}
		for _, scanPath := range []string{path, strings.TrimSuffix(path, ".deleting-"+windowID)} {
			active, err := findBrowserUserDataProcesses(scanPath)
			if err != nil || len(active) > 0 {
				return fmt.Errorf("待删除目录仍被占用或无法检查")
			}
		}
		if err := os.RemoveAll(path); err != nil {
			return err
		}
	}
	return nil
}

func oneBrowserPathsOverlap(a, b string) bool {
	return a == b || strings.HasPrefix(a, b+string(filepath.Separator)) || strings.HasPrefix(b, a+string(filepath.Separator))
}

type OneBrowserDetachedWindow struct {
	WindowID  string `json:"windowId"`
	ProfileID string `json:"profileId"`
	Name      string `json:"name"`
	Running   bool   `json:"running"`
}

// Listing is read-only. Old frontend-only deletions require explicit review;
// absence from localStorage alone must never trigger automatic data removal.
func (a *App) OneBrowserDetachedWindows(windowIDs []string) ([]OneBrowserDetachedWindow, error) {
	if a.browserMgr == nil {
		return nil, fmt.Errorf("浏览器服务未就绪")
	}
	keep := map[string]bool{}
	for _, raw := range windowIDs {
		id, err := oneBrowserWindowID(raw)
		if err != nil {
			return nil, err
		}
		keep[id] = true
	}
	a.oneBrowserWindowMu.Lock()
	defer a.oneBrowserWindowMu.Unlock()
	a.browserMgr.InitData()
	a.browserMgr.Mutex.Lock()
	defer a.browserMgr.Mutex.Unlock()
	result := []OneBrowserDetachedWindow{}
	for _, profile := range a.browserMgr.Profiles {
		owner := oneBrowserWindowOwner(*profile)
		if owner == "" || keep[owner] {
			continue
		}
		processes, err := findBrowserUserDataProcesses(a.browserMgr.ResolveUserDataDir(profile))
		if err != nil {
			return nil, fmt.Errorf("检测遗留窗口进程失败：%w", err)
		}
		result = append(result, OneBrowserDetachedWindow{WindowID: owner, ProfileID: profile.ProfileId, Name: profile.ProfileName, Running: profile.Running || a.browserMgr.BrowserProcesses[profile.ProfileId] != nil || len(processes) > 0})
	}
	return result, nil
}

package backend

import (
	"database/sql"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

type OneBrowserKernelBinding struct {
	WindowID  string `json:"windowId"`
	ProfileID string `json:"profileId"`
	Name      string `json:"name"`
	CoreID    string `json:"coreId"`
}

func (a *App) OneBrowserWindowKernelBindings() []OneBrowserKernelBinding {
	result := []OneBrowserKernelBinding{}
	if a.browserMgr == nil {
		return result
	}
	for _, p := range a.BrowserProfileList() {
		if oneBrowserWindowOwner(p) != "" {
			result = append(result, OneBrowserKernelBinding{WindowID: oneBrowserWindowOwner(p), ProfileID: p.ProfileId, Name: p.ProfileName, CoreID: p.CoreId})
		}
	}
	return result
}

// Persist only the chosen core, without rewriting fingerprint, proxy or session data.
func (a *App) OneBrowserSetWindowKernel(windowID, profileID, coreID string) error {
	if strings.TrimSpace(coreID) == "" {
		return fmt.Errorf("请选择已安装内核版本")
	}
	id, err := oneBrowserWindowID(windowID)
	if err != nil {
		return err
	}
	if a.browserMgr == nil {
		return fmt.Errorf("内核服务未初始化")
	}
	a.oneBrowserWindowMu.Lock()
	defer a.oneBrowserWindowMu.Unlock()
	if _, err := a.oneBrowserSelectedKernel(coreID, ""); err != nil {
		return err
	}
	existing, err := resolveOneBrowserWindowProfile(id, strings.TrimSpace(profileID), a.BrowserProfileList())
	if err != nil {
		return err
	}
	if existing == nil {
		return nil
	}
	m := a.browserMgr
	m.Mutex.Lock()
	defer m.Mutex.Unlock()
	p := m.Profiles[existing.ProfileId]
	if p == nil {
		return fmt.Errorf("窗口配置已删除")
	}
	if p.CoreId == coreID {
		return nil
	}
	if p.Running || m.BrowserProcesses[p.ProfileId] != nil {
		return fmt.Errorf("请先停止该窗口，再修改内核版本")
	}
	processes, err := findBrowserUserDataProcesses(m.ResolveUserDataDir(p))
	if err != nil || len(processes) > 0 {
		return fmt.Errorf("无法确认窗口已退出，请先关闭浏览器再修改内核")
	}
	old := p.CoreId
	p.CoreId = coreID
	if m.ProfileDAO != nil {
		err = m.ProfileDAO.Upsert(p)
	} else {
		err = m.SaveProfiles()
	}
	if err != nil {
		p.CoreId = old
		return err
	}
	return nil
}

// UI owns never-started windows. Recheck them together with persisted profiles
// under the same lock used by start and window editing.
func (a *App) OneBrowserDeleteKernel(coreID string, windows []OneBrowserKernelBinding) error {
	if a.browserMgr == nil {
		return fmt.Errorf("内核服务未初始化")
	}
	coreID = strings.TrimSpace(coreID)
	if coreID == "" {
		return fmt.Errorf("请选择内核版本")
	}
	a.oneBrowserWindowMu.Lock()
	defer a.oneBrowserWindowMu.Unlock()
	for _, w := range windows {
		if w.CoreID == coreID {
			return fmt.Errorf("窗口“%s”仍关联此内核，请在窗口齿轮 → 编辑窗口中修改内核版本后再删除", w.Name)
		}
	}
	m := a.browserMgr
	defaultID := a.OneBrowserKernelStatus().CoreID
	for _, p := range a.BrowserProfileList() {
		if p.CoreId == coreID || (p.CoreId == "" && defaultID == coreID) {
			return fmt.Errorf("窗口“%s”仍使用此内核，请编辑窗口修改内核版本后再删除", p.ProfileName)
		}
	}
	var target *BrowserCore
	cores := m.ListCores()
	for i := range cores {
		if cores[i].CoreId == coreID {
			target = &cores[i]
			break
		}
	}
	if target == nil {
		return fmt.Errorf("内核已移除，请刷新列表")
	}
	// Only application-managed directories are removed. External imports are
	// unregistered without deleting the user's original directory.
	root := strings.TrimSpace(m.Config.Browser.CoreRoot)
	if root == "" {
		root = "chrome"
	}
	root = canonicalOneBrowserDataDir(m.ResolveRelativePath(root))
	path := canonicalOneBrowserDataDir(m.ResolveRelativePath(target.CorePath))
	rel, err := filepath.Rel(root, path)
	managed := err == nil && rel != "." && rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator))
	stage := ""
	if managed {
		for _, c := range cores {
			if c.CoreId != coreID {
				other := canonicalOneBrowserDataDir(m.ResolveRelativePath(c.CorePath))
				sub, e := filepath.Rel(path, other)
				if e == nil && (sub == "." || (sub != ".." && !strings.HasPrefix(sub, ".."+string(filepath.Separator)))) {
					return fmt.Errorf("此目录仍被其他内核记录共用，不能删除")
				}
			}
		}
		if _, err := os.Lstat(path); err == nil {
			stage = path + fmt.Sprintf(".deleting-%d", time.Now().UnixNano())
			if err = os.Rename(path, stage); err != nil {
				return fmt.Errorf("内核文件仍被占用或不可写，未删除：%w", err)
			}
		} else if !os.IsNotExist(err) {
			return err
		}
	}
	if err := m.DeleteCore(coreID); err != nil {
		if stage != "" {
			if restoreErr := os.Rename(stage, path); restoreErr != nil {
				return fmt.Errorf("删除配置失败：%v；恢复文件失败：%w", err, restoreErr)
			}
		}
		return err
	}
	if stage != "" {
		if err := os.RemoveAll(stage); err != nil {
			return fmt.Errorf("内核已移除，暂存文件清理失败：%w", err)
		}
	}
	return nil
}

// Share the start lock so a plugin cannot begin loading while it is deleted.
func (a *App) OneBrowserDeleteImportedPlugin(extensionID string) error {
	for _, id := range oneBrowserBuiltinExtensions {
		if id == extensionID {
			return fmt.Errorf("内置插件请使用停用功能")
		}
	}
	if _, err := oneBrowserPluginIDs(nil, []string{extensionID}); err != nil {
		return err
	}
	a.oneBrowserWindowMu.Lock()
	defer a.oneBrowserWindowMu.Unlock()
	if a.browserMgr == nil || a.browserMgr.ExtensionDAO == nil {
		return fmt.Errorf("插件服务未初始化")
	}
	for _, p := range a.BrowserProfileList() {
		settings, err := a.browserMgr.ExtensionDAO.GetProfileSettings(p.ProfileId)
		if err != nil {
			return err
		}
		linked := false
		for _, id := range settings.ExtensionIDs {
			if id == extensionID {
				linked = true
				break
			}
		}
		if !linked {
			continue
		}
		if p.Running {
			return fmt.Errorf("窗口“%s”正在使用此插件，请先停止该窗口", p.ProfileName)
		}
		processes, err := findBrowserUserDataProcesses(a.browserMgr.ResolveUserDataDir(&p))
		if err != nil || len(processes) > 0 {
			return fmt.Errorf("请先关闭使用此插件的浏览器窗口")
		}
	}
	if _, err := a.browserMgr.ExtensionDAO.Get(extensionID); errors.Is(err, sql.ErrNoRows) {
		return a.browserMgr.ExtensionDAO.Delete(extensionID)
	} else if err != nil {
		return err
	}
	return a.BrowserExtensionDelete(extensionID)
}

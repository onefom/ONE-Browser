package backend

import (
	"ant-chrome/backend/internal/logger"
	"fmt"
	"time"
)

func (a *App) BrowserInstanceStop(profileId string) (*BrowserProfile, error) {
	log := logger.New("Browser")
	a.browserMgr.Mutex.Lock()
	profile, exists := a.browserMgr.Profiles[profileId]
	if !exists {
		a.browserMgr.Mutex.Unlock()
		return nil, fmt.Errorf("profile not found")
	}
	cmd := a.browserMgr.BrowserProcesses[profileId]
	debugPort := profile.DebugPort
	userDataDir := a.browserMgr.ResolveUserDataDir(profile)
	a.browserMgr.Mutex.Unlock()

	cdpRequested := tryCloseBrowserViaCDP(debugPort, 5*time.Second)
	terminatedByDir, terminateErr := terminateBrowserProcessesByUserDataDir(userDataDir, 5*time.Second)

	// Process discovery is the authoritative check on Windows. If it could not
	// find/close the tree, fall back to the tracked root process and its children.
	if !terminatedByDir && cmd != nil && cmd.Process != nil {
		if err := a.stopBrowserProcess(cmd); err != nil {
			log.Error("实例停止失败", logger.F("profile_id", profileId), logger.F("error", err))
			a.browserMgr.Mutex.Lock()
			profile.LastError = err.Error()
			a.browserMgr.Mutex.Unlock()
			return profile, err
		}
	}

	remaining, verifyErr := findBrowserUserDataProcesses(userDataDir)
	if verifyErr == nil && len(remaining) > 0 {
		err := fmt.Errorf("实例停止失败：仍有 %d 个浏览器进程未退出", len(remaining))
		log.Error("实例停止失败", logger.F("profile_id", profileId), logger.F("reason", err.Error()))
		a.browserMgr.Mutex.Lock()
		profile.LastError = err.Error()
		a.browserMgr.Mutex.Unlock()
		return profile, err
	}
	if terminateErr != nil && verifyErr != nil {
		err := fmt.Errorf("实例停止失败：%v", terminateErr)
		log.Error("实例停止失败", logger.F("profile_id", profileId), logger.F("error", err))
		a.browserMgr.Mutex.Lock()
		profile.LastError = err.Error()
		a.browserMgr.Mutex.Unlock()
		return profile, err
	}

	if debugPort > 0 && canConnectDebugPort(debugPort, 250*time.Millisecond) {
		err := fmt.Errorf("实例停止失败：浏览器仍在运行（调试端口 %d 仍可访问）", debugPort)
		log.Error("实例停止失败", logger.F("profile_id", profileId), logger.F("debug_port", debugPort), logger.F("reason", err.Error()))
		a.browserMgr.Mutex.Lock()
		profile.LastError = err.Error()
		a.browserMgr.Mutex.Unlock()
		return profile, err
	}

	a.browserMgr.Mutex.Lock()
	a.markProfileStoppedLocked(profileId, profile)
	a.browserMgr.Mutex.Unlock()
	log.Info("实例停止", logger.F("profile_id", profileId), logger.F("cdp_requested", cdpRequested), logger.F("process_tree_cleaned", terminatedByDir))
	return profile, nil
}

func (a *App) BrowserInstanceRestart(profileId string) (*BrowserProfile, error) {
	if _, err := a.BrowserInstanceStop(profileId); err != nil {
		return nil, err
	}
	return a.BrowserInstanceStart(profileId)
}

package backend

import (
	"ant-chrome/backend/internal/logger"
	"time"
)

func (a *App) BrowserInstanceStart(profileId string) (*BrowserProfile, error) {
	return a.browserInstanceStartInternal(profileId, nil, nil, false, false, false, "", "")
}

func shouldPreferVisibleWindowForStartWithParams(startURLs []string) bool {
	return len(normalizeNonEmptyStrings(startURLs)) > 0
}

// BrowserInstanceStartDirect 仅本次启动走直连，不落库修改实例代理配置。
func (a *App) BrowserInstanceStartDirect(profileId string) (*BrowserProfile, error) {
	return a.browserInstanceStartInternal(profileId, nil, nil, false, false, true, "", "")
}

// BrowserInstanceStartWithParams 通过额外参数启动实例（仅本次启动生效，不落库）
func (a *App) BrowserInstanceStartWithParams(profileId string, extraLaunchArgs []string, startURLs []string, skipDefaultStartURLs bool) (*BrowserProfile, error) {
	preferVisibleWindow := shouldPreferVisibleWindowForStartWithParams(startURLs)
	return a.browserInstanceStartInternal(profileId, extraLaunchArgs, startURLs, skipDefaultStartURLs, preferVisibleWindow, false, "", "")
}

func (a *App) browserInstanceStartInternal(profileId string, extraLaunchArgs []string, startURLs []string, skipDefaultStartURLs bool, preferVisibleWindow bool, forceDirectProxy bool, proxyId string, proxyConfig string) (*BrowserProfile, error) {
	input := newBrowserStartInput(profileId, extraLaunchArgs, startURLs, skipDefaultStartURLs, preferVisibleWindow, forceDirectProxy, proxyId, proxyConfig)
	trace := newBrowserStartupTrace("浏览器启动入口明细", profileId)
	defer trace.finish()
	a.browserMgr.Mutex.Lock()
	defer a.browserMgr.Mutex.Unlock()
	trace.step("等待浏览器管理锁")

	if profile := a.browserMgr.Profiles[profileId]; profile != nil && a.isCamoufoxProfile(profile) {
		return a.startCamoufoxLocked(input, profile)
	}
	profile, handled, err := a.resolveBrowserStartProfile(input)
	if err != nil || handled {
		return profile, err
	}

	trace.step("实例状态及是否复用检查")
	preparingAt := time.Now()
	plan, err := a.prepareBrowserStartPlan(input, profile)
	if err == errBrowserStartHandledByRecoveredRuntime {
		a.emitBrowserInstanceStarted(profile, true)
		return profile, nil
	}
	if err != nil {
		return profile, err
	}
	defer plan.releaseBridgeIfNeeded(a)

	trace.step("准备启动计划")
	preparedAt := time.Now()
	started, err := a.startBrowserProfileWithPlan(input, plan)
	trace.step("启动进程及返回状态")
	logger.New("Browser").Info("浏览器启动耗时", logger.F("profile_id", profileId), logger.F("prepare_ms", preparedAt.Sub(preparingAt).Milliseconds()), logger.F("ready_ms", time.Since(preparedAt).Milliseconds()), logger.F("success", err == nil))
	return started, err
}

package backend

import (
	"ant-chrome/backend/internal/browser"
	"ant-chrome/backend/internal/logger"
	"ant-chrome/backend/internal/proxy"
	"bytes"
	"context"
	"fmt"
	"html/template"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

// Chromium 在线发布源；Camoufox 测试版通过本地 Windows x64 包导入。
// 内核不会打入初始便携包；用户点击下载后才写入 chrome/ 目录。
const (
	fingerprintChromiumRepo            = "adryfish/fingerprint-chromium"
	fingerprintChromiumReleasesURL     = "https://github.com/adryfish/fingerprint-chromium/releases"
	fingerprintChromiumFallbackVersion = "150.0.7871.186"
	fingerprintChromiumFallbackWinX64  = "https://github.com/adryfish/fingerprint-chromium/releases/download/150.0.7871.186/ungoogled-chromium_150.0.7871.186-1.1_windows_x64.zip"
)

type OneBrowserKernelRelease struct {
	Version     string `json:"version"`
	DownloadURL string `json:"downloadUrl"`
	ReleasesURL string `json:"releasesUrl"`
	AssetName   string `json:"assetName"`
}

type OneBrowserKernelStatus struct {
	Installed bool   `json:"installed"`
	Version   string `json:"version"`
	CoreID    string `json:"coreId"`
	Family    string `json:"family"`
	Integrity string `json:"integrity"`
}

// OneBrowserStartRequest is intentionally small: the existing One Browser UI
// owns the presentation state while Ant Browser owns the persisted profile,
// browser executable and proxy bridge.
type OneBrowserStartRequest struct {
	CoreID         string   `json:"coreId"`
	WindowID       string   `json:"windowId"`
	ProfileID      string   `json:"profileId"`
	Name           string   `json:"name"`
	ProxyID        string   `json:"proxyId"`
	ProxyName      string   `json:"proxyName"`
	Account        string   `json:"account"`
	OS             string   `json:"os"`
	Language       string   `json:"language"`
	Timezone       string   `json:"timezone"`
	UserAgent      string   `json:"userAgent"`
	WindowSize     string   `json:"windowSize"`
	WindowPosition string   `json:"windowPosition"`
	Extensions     []string `json:"extensions"`
	ExtensionIDs   []string `json:"extensionIds"`
}

type OneBrowserStartResult struct {
	ProfileID string `json:"profileId"`
	Running   bool   `json:"running"`
	Warning   string `json:"warning,omitempty"`
}

// OneBrowserNetworkInfo is the same route-aware network snapshot shown by the
// managed browser workspace.  The manager requests it with the profile ID so
// the selected node/system proxy cannot drift from the launched window.
type OneBrowserNetworkInfo struct {
	OK       bool   `json:"ok"`
	IP       string `json:"ip"`
	Country  string `json:"country"`
	Region   string `json:"region"`
	City     string `json:"city"`
	Location string `json:"location"`
	Error    string `json:"error"`
}

func (a *App) OneBrowserGetNetworkInfo(profileID string) OneBrowserNetworkInfo {
	profileID = strings.TrimSpace(profileID)
	proxies := a.getLatestProxies()
	probeID := ""
	for _, item := range a.BrowserProfileList() {
		if item.ProfileId != profileID {
			continue
		}
		probeID = strings.TrimSpace(item.ProxyId)
		if probeID == "" && strings.TrimSpace(item.ProxyConfig) != "" {
			probeID = "__one_browser_profile__"
			proxies = append(proxies, BrowserProxy{ProxyId: probeID, ProxyName: "One Browser 当前窗口", ProxyConfig: item.ProxyConfig})
		}
		break
	}
	if probeID == "" {
		if systemProxy, err := oneBrowserSystemProxyAddress(); err == nil && strings.TrimSpace(systemProxy) != "" {
			probeID = "__one_browser_system__"
			proxies = append(proxies, BrowserProxy{ProxyId: probeID, ProxyName: "Windows 系统代理", ProxyConfig: systemProxy})
		} else {
			probeID = "__one_browser_direct__"
			proxies = append(proxies, BrowserProxy{ProxyId: probeID, ProxyName: "本地直连", ProxyConfig: "direct://"})
		}
	}
	data, err := proxy.FetchIPHealthInfo(probeID, proxies, a.xrayMgr, a.singboxMgr, a.clashMgr, a.defaultProxyConnectorType(), a.proxyIPHealthConfig())
	if err != nil {
		return OneBrowserNetworkInfo{Error: "网络位置服务暂不可用，请稍后重试"}
	}
	result := OneBrowserNetworkInfo{
		OK: true, IP: mapString(data, "ip"), Country: mapString(data, "country"),
		Region: mapString(data, "region"), City: mapString(data, "city"),
	}
	parts := make([]string, 0, 3)
	for _, value := range []string{result.Country, result.Region, result.City} {
		value = strings.TrimSpace(value)
		if value != "" && (len(parts) == 0 || !strings.EqualFold(parts[len(parts)-1], value)) {
			parts = append(parts, value)
		}
	}
	result.Location = strings.Join(parts, " · ")
	if result.IP == "" {
		result.OK = false
		result.Error = "未获取到出口 IP"
	}
	return result
}

// OneBrowserKernels includes every registered local/downloaded version.
func (a *App) OneBrowserKernels() []OneBrowserKernelStatus {
	result := []OneBrowserKernelStatus{}
	if a == nil || a.browserMgr == nil {
		return result
	}
	for _, core := range a.browserMgr.ListCores() {
		version := strings.TrimSpace(a.browserMgr.GetChromeVersion(core.CorePath))
		if version == "" {
			version = core.CoreName
		}
		result = append(result, OneBrowserKernelStatus{Installed: true, Version: version, CoreID: core.CoreId, Family: a.browserMgr.CoreFamily(core), Integrity: oneBrowserCoreIntegrity(a.browserMgr.ResolveRelativePath(core.CorePath))})
	}
	return result
}

// Chromium update status must not compare Firefox versions against Chrome releases.
func (a *App) OneBrowserChromiumStatus() OneBrowserKernelStatus {
	best := OneBrowserKernelStatus{}
	for _, item := range a.OneBrowserKernels() {
		if item.Family != "camoufox" && (!best.Installed || oneBrowserKernelNeedsUpdate(best.Version, item.Version)) {
			best = item
		}
	}
	return best
}
func (a *App) OneBrowserKernelStatus() OneBrowserKernelStatus {
	items := a.OneBrowserKernels()
	if len(items) == 0 {
		return OneBrowserKernelStatus{}
	}
	chromium := []OneBrowserKernelStatus{}
	for _, item := range items {
		if item.Family != "camoufox" {
			chromium = append(chromium, item)
		}
	}
	if len(chromium) > 0 {
		items = chromium
	}
	best := items[0]
	for _, item := range items[1:] {
		if oneBrowserKernelNeedsUpdate(best.Version, item.Version) {
			best = item
		}
	}
	return best
}
func (a *App) oneBrowserSelectedKernel(coreID, profileID string) (OneBrowserKernelStatus, error) {
	coreID = strings.TrimSpace(coreID)
	if coreID == "" && profileID != "" {
		for _, p := range a.BrowserProfileList() {
			if p.ProfileId == profileID {
				coreID = p.CoreId
				break
			}
		}
	}
	if coreID == "" {
		return a.OneBrowserKernelStatus(), nil
	}
	for _, item := range a.OneBrowserKernels() {
		if item.CoreID == coreID {
			return item, nil
		}
	}
	return OneBrowserKernelStatus{}, fmt.Errorf("所选内核已移除，请编辑窗口重新选择内核版本")
}

// Manual/weekly checks surface network failures instead of claiming latest.
func (a *App) OneBrowserCheckKernelUpdate() (OneBrowserKernelRelease, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	return latestFingerprintChromiumRelease(ctx)
}

func fallbackFingerprintChromiumRelease() OneBrowserKernelRelease {
	return OneBrowserKernelRelease{
		Version:     fingerprintChromiumFallbackVersion,
		DownloadURL: fingerprintChromiumFallbackWinX64,
		ReleasesURL: fingerprintChromiumReleasesURL,
		AssetName:   filepath.Base(fingerprintChromiumFallbackWinX64),
	}
}

func fingerprintChromiumWindowsX64Asset(release githubRelease) (githubReleaseAsset, bool) {
	for _, asset := range release.Assets {
		name := strings.ToLower(strings.TrimSpace(asset.Name))
		if !strings.HasSuffix(name, ".zip") || !strings.Contains(name, "windows") {
			continue
		}
		if !strings.Contains(name, "x64") && !strings.Contains(name, "x86_64") && !strings.Contains(name, "amd64") {
			continue
		}
		if strings.TrimSpace(asset.BrowserDownloadURL) == "" {
			continue
		}
		return asset, true
	}
	return githubReleaseAsset{}, false
}

func oneBrowserReleaseClients(timeout time.Duration) []*http.Client {
	clients := make([]*http.Client, 0, 2)
	if client, _, err := oneBrowserSystemProxyClient(timeout); err == nil && client != nil {
		clients = append(clients, client)
	}
	clients = append(clients, &http.Client{Timeout: timeout, Transport: &http.Transport{Proxy: http.ProxyFromEnvironment}})
	return clients
}

func latestFingerprintChromiumRelease(ctx context.Context) (OneBrowserKernelRelease, error) {
	var lastErr error
	for _, client := range oneBrowserReleaseClients(12 * time.Second) {
		release, err := fetchGitHubRelease(ctx, client, fingerprintChromiumRepo, "latest")
		if err != nil {
			lastErr = err
			continue
		}
		asset, ok := fingerprintChromiumWindowsX64Asset(release)
		if !ok {
			lastErr = fmt.Errorf("最新版本没有 Windows x64 ZIP")
			continue
		}
		version := strings.TrimSpace(release.TagName)
		if version == "" {
			lastErr = fmt.Errorf("最新版本号为空")
			continue
		}
		return OneBrowserKernelRelease{
			Version:     version,
			DownloadURL: strings.TrimSpace(asset.BrowserDownloadURL),
			ReleasesURL: fingerprintChromiumReleasesURL,
			AssetName:   strings.TrimSpace(asset.Name),
		}, nil
	}
	return OneBrowserKernelRelease{}, lastErr
}

// OneBrowserLatestFingerprintChromium reads the current GitHub release using
// the Windows system proxy first, then a direct/environment-proxy fallback.
// The verified bundled release keeps the UI and downloader usable offline.
func (a *App) OneBrowserLatestFingerprintChromium() OneBrowserKernelRelease {
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	if release, err := latestFingerprintChromiumRelease(ctx); err == nil {
		return release
	}
	return fallbackFingerprintChromiumRelease()
}

// OneBrowserDownloadFingerprintChromium starts Ant Browser's verified archive
// downloader. Completion is reported through the existing download:progress event.
func (a *App) OneBrowserDownloadFingerprintChromium() error {
	release := a.OneBrowserLatestFingerprintChromium()
	return a.BrowserCoreDownload("Chrome-"+release.Version, release.DownloadURL, "__system__")
}

// OneBrowserUpdateFingerprintChromium replaces the verified installed core in
// place. The old directory remains available until the new archive passes
// extraction and executable validation.
func (a *App) OneBrowserUpdateFingerprintChromium() error {
	release := a.OneBrowserLatestFingerprintChromium()
	status := a.OneBrowserKernelStatus()
	if status.Installed && !oneBrowserKernelNeedsUpdate(status.Version, release.Version) {
		a.emitRuntimeEvent("download:progress", map[string]interface{}{"phase": "done", "progress": 100, "message": "当前内核已是最新版本，无需重新下载"})
		return nil
	}
	if strings.TrimSpace(status.CoreID) == "" {
		return a.BrowserCoreDownload("Chrome-"+release.Version, release.DownloadURL, "__system__")
	}
	return a.BrowserCoreDownload("Chrome-"+release.Version, release.DownloadURL, "__system__")
}

// OneBrowserImportClash downloads a subscription using Ant Browser's hardened
// fetcher and persists every node. Kernel selection follows the active
// connection stack instead of forcing imported nodes onto Mihomo.
func (a *App) OneBrowserImportClash(rawURL string) ([]BrowserProxy, error) {
	result, err := a.browserProxyFetchClashByURL(rawURL, "")
	if err != nil {
		return nil, err
	}
	content, _ := result["content"].(string)
	group, _ := result["suggestedGroup"].(string)
	proxies, err := oneBrowserClashNodes(content, rawURL, group)
	if err != nil {
		return nil, err
	}
	all := a.getLatestProxies()
	for _, proxy := range proxies {
		all = append(all, proxy)
	}
	if err := a.SaveBrowserProxies(all); err != nil {
		return nil, err
	}
	return proxies, nil
}

// OneBrowserParseClashText supports the manual YAML field in the existing UI.
func (a *App) OneBrowserParseClashText(content string) ([]BrowserProxy, error) {
	proxies, err := oneBrowserClashNodes(content, "", "手动导入")
	if err != nil {
		return nil, err
	}
	all := a.getLatestProxies()
	all = append(all, proxies...)
	if err := a.SaveBrowserProxies(all); err != nil {
		return nil, err
	}
	return proxies, nil
}

// OneBrowserProxyBatchTestSpeed repairs the kernel pin written by versions up
// to 1.8.13, then measures nodes through the currently selected connection
// stack. This keeps xray/sing-box and Mihomo isolated while making old One
// Browser imports usable without requiring the user to import them again.
func (a *App) OneBrowserProxyBatchTestSpeed(proxyIDs []string, concurrency int) []ProxyTestResult {
	a.oneBrowserUseActiveProxyStack(proxyIDs)
	return a.BrowserProxyBatchTestSpeed(proxyIDs, concurrency)
}

func (a *App) oneBrowserUseActiveProxyStack(proxyIDs []string) {
	if len(proxyIDs) == 0 {
		return
	}
	wanted := make(map[string]struct{}, len(proxyIDs))
	for _, proxyID := range proxyIDs {
		if proxyID = strings.TrimSpace(proxyID); proxyID != "" {
			wanted[proxyID] = struct{}{}
		}
	}
	if len(wanted) == 0 {
		return
	}
	items := a.getLatestProxies()
	changed := false
	for index := range items {
		if _, ok := wanted[strings.TrimSpace(items[index].ProxyId)]; !ok {
			continue
		}
		if strings.EqualFold(strings.TrimSpace(items[index].PreferredKernel), "mihomo") {
			items[index].PreferredKernel = ""
			changed = true
		}
	}
	if changed {
		_ = a.SaveBrowserProxies(items)
	}
}

// OneBrowserDeleteProxy keeps the UI list and the real Ant Browser proxy
// store in sync. A node that is removed here cannot be selected at launch.
func (a *App) OneBrowserDeleteProxy(proxyID string) error {
	proxyID = strings.TrimSpace(proxyID)
	if proxyID == "" || proxyID == "__direct__" || proxyID == "__system__" {
		return fmt.Errorf("不能删除直连配置")
	}
	proxies := a.getLatestProxies()
	filtered := make([]BrowserProxy, 0, len(proxies))
	found := false
	for _, item := range proxies {
		if item.ProxyId == proxyID {
			found = true
			continue
		}
		filtered = append(filtered, item)
	}
	if !found {
		// The UI may still contain a cached node after an interrupted save. Treat
		// deletion as idempotent so it can repair that stale local state.
		return nil
	}
	return a.SaveBrowserProxies(filtered)
}

// OneBrowserStart creates (or updates) the corresponding Ant Browser profile
// and opens it with a local workspace page plus Google. The browser process is
// always the downloaded fingerprint-chromium executable, never the OS default
// browser.
func (a *App) OneBrowserStart(request OneBrowserStartRequest) (OneBrowserStartResult, error) {
	trace := newBrowserStartupTrace("One Browser 启动阶段耗时", request.WindowID)
	defer trace.finish()
	phase := trace.step

	windowID, err := oneBrowserWindowID(request.WindowID)
	if err != nil {
		return OneBrowserStartResult{}, err
	}
	if _, validationErr := oneBrowserPluginIDs(request.Extensions, request.ExtensionIDs); validationErr != nil {
		return OneBrowserStartResult{}, validationErr
	}
	a.oneBrowserWindowMu.Lock()
	defer a.oneBrowserWindowMu.Unlock()
	phase("启动队列锁")
	status, kernelErr := a.oneBrowserSelectedKernel(request.CoreID, request.ProfileID)
	if kernelErr != nil {
		return OneBrowserStartResult{}, kernelErr
	}
	if !status.Installed || strings.TrimSpace(status.CoreID) == "" {
		return OneBrowserStartResult{}, fmt.Errorf("请先下载并完成 Chrome 内核安装")
	}
	phase("内核状态及版本")
	if status.Family == "camoufox" && (len(request.Extensions) > 0 || len(request.ExtensionIDs) > 0) {
		return OneBrowserStartResult{}, fmt.Errorf("Firefox 不兼容当前 Chrome 插件，请先在插件中心取消此窗口的插件关联；Firefox 扩展请在浏览器中安装")
	}
	if warnings := a.OneBrowserSyncBuiltinExtensions(request.Extensions); len(warnings) > 0 {
		return OneBrowserStartResult{}, fmt.Errorf("插件准备失败：%s", strings.Join(warnings, "；"))
	}

	phase("内置插件同步")
	profileID := strings.TrimSpace(request.ProfileID)
	name := strings.TrimSpace(request.Name)
	if name == "" {
		name = "One Browser 窗口"
	}
	proxyID := strings.TrimSpace(request.ProxyID)
	proxyConfig := ""
	if proxyID == "" || proxyID == "__system__" {
		proxyID = ""
		if systemProxy, systemErr := oneBrowserSystemProxyAddress(); systemErr == nil && strings.TrimSpace(systemProxy) != "" {
			proxyConfig = systemProxy
		} else {
			proxyConfig = "direct://"
		}
	} else {
		a.oneBrowserUseActiveProxyStack([]string{proxyID})
	}

	phase("选择代理及连接栈")
	var profile *BrowserProfile
	existing, err := resolveOneBrowserWindowProfile(windowID, profileID, a.BrowserProfileList())
	if err != nil {
		return OneBrowserStartResult{}, err
	}
	if existing == nil && profileID != "" {
		return OneBrowserStartResult{}, fmt.Errorf("窗口配置已删除，请刷新窗口列表")
	}
	if existing != nil && strings.TrimSpace(request.CoreID) == "" {
		status, err = a.oneBrowserSelectedKernel("", existing.ProfileId)
		if err != nil {
			return OneBrowserStartResult{}, err
		}
	}
	if status.Family == "camoufox" && (len(request.Extensions) > 0 || len(request.ExtensionIDs) > 0) {
		return OneBrowserStartResult{}, fmt.Errorf("Firefox 不兼容当前 Chrome 插件，请先取消此窗口的插件关联")
	}
	if existing != nil {
		item := *existing
		profileID = item.ProfileId
		input := BrowserProfileInput{
			ProfileName:        name,
			UserDataDir:        item.UserDataDir,
			CoreId:             status.CoreID,
			RestoreLastSession: browser.RestoreLastSessionEnabled,
			FingerprintArgs:    oneBrowserConfiguredFingerprintArgs(item.FingerprintArgs, request),
			ProxyId:            proxyID,
			ProxyConfig:        proxyConfig,
			MemoryLimitMB:      item.MemoryLimitMB,
			LaunchArgs:         oneBrowserWindowLaunchArgs(item.LaunchArgs, request),
			Tags:               item.Tags,
			Keywords:           oneBrowserIPModeKeywords(oneBrowserWindowKeywords(item.Keywords, windowID), request.ProxyID),
			GroupId:            item.GroupId,
		}
		if oneBrowserHasSavedSession(a.browserMgr.ResolveUserDataDir(&item)) {
			input.LaunchArgs = oneBrowserSessionLaunchArgs(input.LaunchArgs)
		}
		updated, err := a.BrowserProfileUpdate(profileID, input)
		if err != nil {
			return OneBrowserStartResult{}, err
		}
		profile = updated
	}
	if profile == nil {
		created, err := a.BrowserProfileCreate(BrowserProfileInput{
			ProfileName:        name,
			RestoreLastSession: browser.RestoreLastSessionEnabled,
			CoreId:             status.CoreID,
			FingerprintArgs:    oneBrowserConfiguredFingerprintArgs(a.config.Browser.DefaultFingerprintArgs, request),
			ProxyId:            proxyID,
			ProxyConfig:        proxyConfig,
			Tags:               []string{"One Browser"},
			Keywords:           oneBrowserIPModeKeywords(oneBrowserWindowKeywords(nil, windowID), request.ProxyID),
			LaunchArgs:         oneBrowserWindowLaunchArgs(nil, request),
		})
		if err != nil {
			return OneBrowserStartResult{}, err
		}
		profile = created
	}
	if status.Family == "camoufox" {
		args := []string{}
		for _, arg := range profile.LaunchArgs {
			if !strings.HasPrefix(arg, "--lang=") && !strings.HasPrefix(arg, "--timezone=") {
				args = append(args, arg)
			}
		}
		lang := strings.TrimSpace(strings.Split(request.Language, " · ")[0])
		if lang != "" {
			args = append(args, "--lang="+lang)
		}
		if request.Timezone != "" {
			args = append(args, "--timezone="+request.Timezone)
		}
		profile.LaunchArgs = args
		if err := a.browserMgr.SaveProfiles(); err != nil {
			return OneBrowserStartResult{}, err
		}
	}
	if err := a.configureOneBrowserPlugins(profile.ProfileId, request.Extensions, request.ExtensionIDs); err != nil {
		return OneBrowserStartResult{}, err
	}
	phase("查找及保存窗口配置")
	if err := a.validateOneBrowserDataIsolation(profile); err != nil {
		return OneBrowserStartResult{}, err
	}
	phase("窗口数据隔离校验")
	if _, err := a.syncOneBrowserDirectory(profile.ProfileId, name, windowID); err != nil {
		return OneBrowserStartResult{}, err
	}
	phase("目录归属及改名同步")
	if !profile.Running && status.Family != "camoufox" {
		if migrationErr := browser.MigrateOneBrowserLegacyBookmarks(a.browserMgr.ResolveUserDataDir(profile), fingerprintCheckBookmarkURL); migrationErr != nil {
			logger.New("Browser").Warn("旧书签迁移未完成，下次重试", logger.F("error", migrationErr.Error()))
		}
	}

	phase("清理旧指纹书签")
	workspaceURL, err := a.oneBrowserWorkspaceURL(profile.ProfileId, profile.ProfileName, request, status, proxyConfig)
	if err != nil {
		return OneBrowserStartResult{}, err
	}
	startURLs := []string{workspaceURL, "https://www.google.com/"}
	// A saved session is authoritative. Reopening must not append workspace/Google
	// tabs on every launch or override Chromium's saved placement.
	if (status.Family != "camoufox" && oneBrowserHasSavedSession(a.browserMgr.ResolveUserDataDir(profile))) || (status.Family == "camoufox" && camoufoxHasSession(a.browserMgr.ResolveUserDataDir(profile))) {
		startURLs = nil
	}
	phase("工作台及会话判断")
	started, err := a.BrowserInstanceStartWithParams(profile.ProfileId, nil, startURLs, true)
	if err != nil {
		return OneBrowserStartResult{}, err
	}
	phase("浏览器准备启动并就绪")
	a.oneBrowserWakeIPMonitor()
	return OneBrowserStartResult{ProfileID: started.ProfileId, Running: started.Running, Warning: started.RuntimeWarning}, nil
}

func (a *App) OneBrowserStop(profileID string) error {
	profileID = strings.TrimSpace(profileID)
	if profileID == "" {
		return fmt.Errorf("窗口配置不存在")
	}
	_, err := a.BrowserInstanceStop(profileID)
	return err
}

func ensureOneBrowserLaunchArgs(items []string) []string {
	result := append([]string{}, items...)
	for _, required := range []string{"--no-first-run", "--no-default-browser-check"} {
		found := false
		for _, item := range result {
			if strings.EqualFold(strings.TrimSpace(item), required) {
				found = true
				break
			}
		}
		if !found {
			result = append(result, required)
		}
	}
	return result
}

func oneBrowserWindowLaunchArgs(items []string, request OneBrowserStartRequest) []string {
	result := make([]string, 0, len(items)+4)
	for _, item := range items {
		value := strings.TrimSpace(item)
		lower := strings.ToLower(value)
		if value == "" || lower == "--start-maximized" || strings.HasPrefix(lower, "--window-size") || strings.HasPrefix(lower, "--window-position") || strings.HasPrefix(lower, "--user-agent") || strings.HasPrefix(lower, "--onebrowser-os=") {
			continue
		}
		result = append(result, value)
	}
	result = ensureOneBrowserLaunchArgs(result)
	hasCacheLimit := false
	for _, arg := range result {
		if strings.HasPrefix(strings.ToLower(arg), "--disk-cache-size=") {
			hasCacheLimit = true
		}
	}
	if !hasCacheLimit {
		result = append(result, "--disk-cache-size=67108864")
	}

	width, height := 900, 680
	if _, err := fmt.Sscanf(strings.NewReplacer("×", "x", "X", "x", " ", "").Replace(request.WindowSize), "%dx%d", &width, &height); err != nil || width < 360 || height < 480 {
		width, height = 900, 680
	}
	result = append(result, fmt.Sprintf("--window-size=%d,%d", width, height), "--onebrowser-os="+request.OS)
	// Use a stable top-left origin so Chromium does not apply a half-screen
	// placement heuristic and the One Browser manager remains visible.
	result = append(result, "--window-position=12,12")
	if userAgent := strings.TrimSpace(request.UserAgent); userAgent != "" {
		result = append(result, "--user-agent="+userAgent)
	}
	return result
}

func (a *App) oneBrowserWorkspaceURL(profileID, profileName string, request OneBrowserStartRequest, status OneBrowserKernelStatus, systemProxy string) (string, error) {
	if strings.TrimSpace(profileID) == "" {
		return "", fmt.Errorf("窗口配置不存在")
	}
	dir := a.resolveAppPath(filepath.Join("data", "workspaces"))
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", fmt.Errorf("创建工作区页面失败: %w", err)
	}
	filePath := filepath.Join(dir, profileID+".html")
	proxyName := strings.TrimSpace(request.ProxyName)
	if proxyName == "" {
		if systemProxy != "" && systemProxy != "direct://" {
			proxyName = "Windows 系统代理"
		} else {
			proxyName = "直连（未检测到系统代理）"
		}
	}
	data := struct{ Name, Account, Proxy, OS, Language, Timezone, UserAgent, WindowSize, Version, Extensions, WorkspaceID, Family string }{
		Name: profileName, Account: request.Account, Proxy: proxyName, OS: request.OS, Family: "Chrome",
		Language: request.Language, Timezone: request.Timezone, UserAgent: request.UserAgent,
		WindowSize: request.WindowSize, Version: status.Version,
		Extensions: fmt.Sprintf("%d 个已启用", len(request.Extensions)+len(request.ExtensionIDs)), WorkspaceID: profileID,
	}
	if strings.TrimSpace(data.Account) == "" {
		data.Account = "未关联账号"
	}
	if strings.TrimSpace(data.OS) == "" {
		data.OS = "Windows 11"
	}
	if strings.TrimSpace(data.Language) == "" {
		data.Language = "跟随窗口配置"
	}
	if strings.TrimSpace(data.Timezone) == "" {
		data.Timezone = "跟随窗口配置"
	}
	if status.Family == "camoufox" {
		data.Family = "Firefox"
		config := camoufoxConfig(&BrowserProfile{ProfileId: profileID, LaunchArgs: oneBrowserWindowLaunchArgs(nil, request)}, status.Version)
		if ua, ok := config["navigator.userAgent"].(string); ok {
			data.UserAgent = ua
		}
	} else if strings.TrimSpace(data.UserAgent) == "" {
		chromeVersion := strings.TrimSpace(data.Version)
		if chromeVersion == "" {
			chromeVersion = fingerprintChromiumFallbackVersion
		}
		data.UserAgent = oneBrowserChromiumUA(data.OS, chromeVersion)
	}
	if strings.TrimSpace(data.WindowSize) == "" {
		data.WindowSize = "自适应"
	}
	pageTemplate := compactOneBrowserWorkspacePage
	t, err := template.New("workspace").Parse(pageTemplate)
	if err != nil {
		return "", err
	}
	var output bytes.Buffer
	if err := t.Execute(&output, data); err != nil {
		return "", err
	}
	if err := os.WriteFile(filePath, output.Bytes(), 0o644); err != nil {
		return "", fmt.Errorf("写入工作区页面失败: %w", err)
	}
	return "file:///" + filepath.ToSlash(filePath), nil
}

func oneBrowserClashNodes(content, sourceURL, group string) ([]BrowserProxy, error) {
	var document struct {
		Proxies []map[string]interface{} `yaml:"proxies"`
	}
	if err := yaml.Unmarshal([]byte(strings.TrimSpace(content)), &document); err != nil {
		return nil, fmt.Errorf("Clash YAML 解析失败: %w", err)
	}
	if len(document.Proxies) == 0 {
		return nil, fmt.Errorf("订阅中未找到 proxies 节点")
	}
	result := make([]BrowserProxy, 0, len(document.Proxies))
	for index, node := range document.Proxies {
		name := strings.TrimSpace(fmt.Sprint(node["name"]))
		if name == "" || name == "<nil>" {
			name = fmt.Sprintf("Clash 节点 %d", index+1)
		}
		payload, err := yaml.Marshal(map[string]interface{}{"proxies": []map[string]interface{}{node}})
		if err != nil {
			return nil, fmt.Errorf("节点 %q 序列化失败: %w", name, err)
		}
		result = append(result, BrowserProxy{
			ProxyId:           generateUUID(),
			ProxyName:         name,
			ProxyConfig:       string(payload),
			PreferredKernel:   "",
			GroupName:         strings.TrimSpace(group),
			SourceID:          strings.TrimSpace(sourceURL),
			SourceURL:         strings.TrimSpace(sourceURL),
			SourceAutoRefresh: false,
			SortOrder:         index,
		})
	}
	return result, nil
}

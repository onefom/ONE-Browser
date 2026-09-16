package backend

import (
	"ant-chrome/backend/internal/browser"
	"bytes"
	"fmt"
	"html/template"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

// One Browser 第一阶段只维护 fingerprint-chromium 的 Windows x64 发行包。
// 内核不会打入初始便携包；用户点击下载后才写入 chrome/ 目录。
const (
	fingerprintChromiumVersion = "148.0.7778.215"
	fingerprintChromiumWinX64  = "https://github.com/adryfish/fingerprint-chromium/releases/download/148.0.7778.215/ungoogled-chromium_148.0.7778.215-1.1_windows_x64.zip"
)

type OneBrowserKernelStatus struct {
	Installed bool   `json:"installed"`
	Version   string `json:"version"`
	CoreID    string `json:"coreId"`
}

// OneBrowserStartRequest is intentionally small: the existing One Browser UI
// owns the presentation state while Ant Browser owns the persisted profile,
// browser executable and proxy bridge.
type OneBrowserStartRequest struct {
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
}

type OneBrowserStartResult struct {
	ProfileID string `json:"profileId"`
	Running   bool   `json:"running"`
}

// OneBrowserKernelStatus reports only verified Ant Browser core records.
func (a *App) OneBrowserKernelStatus() OneBrowserKernelStatus {
	if a == nil || a.browserMgr == nil {
		return OneBrowserKernelStatus{}
	}
	cores := a.browserMgr.ListCores()
	for _, core := range cores {
		if strings.Contains(strings.ToLower(core.CoreName+" "+core.CorePath), "fingerprint-chromium") {
			version := strings.TrimSpace(a.browserMgr.GetChromeVersion(core.CorePath))
			if version == "" {
				version = strings.TrimPrefix(strings.TrimSpace(core.CoreName), "fingerprint-chromium-")
			}
			if version == "" || version == core.CoreName {
				version = fingerprintChromiumVersion
			}
			return OneBrowserKernelStatus{Installed: true, Version: version, CoreID: core.CoreId}
		}
	}
	// A manually imported and validated directory may use its original folder
	// name (for example chrome-win). Treat the default/first registered core as
	// available so local imports are immediately usable from One Browser.
	if len(cores) > 0 {
		core := cores[0]
		for _, item := range cores {
			if item.IsDefault {
				core = item
				break
			}
		}
		version := strings.TrimSpace(a.browserMgr.GetChromeVersion(core.CorePath))
		if version == "" {
			version = "本地版本"
		}
		return OneBrowserKernelStatus{Installed: true, Version: version, CoreID: core.CoreId}
	}
	return OneBrowserKernelStatus{}
}

// OneBrowserDownloadFingerprintChromium starts Ant Browser's verified archive
// downloader. Completion is reported through the existing download:progress event.
func (a *App) OneBrowserDownloadFingerprintChromium() error {
	return a.BrowserCoreDownload("fingerprint-chromium-"+fingerprintChromiumVersion, fingerprintChromiumWinX64, "__system__")
}

// OneBrowserImportClash downloads a subscription using Ant Browser's hardened
// fetcher, persists every node, and marks nodes for the Mihomo bridge.
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
	status := a.OneBrowserKernelStatus()
	if !status.Installed || strings.TrimSpace(status.CoreID) == "" {
		return OneBrowserStartResult{}, fmt.Errorf("请先下载并完成 fingerprint-chromium 内核安装")
	}
	if warnings := a.OneBrowserSyncBuiltinExtensions(request.Extensions); len(warnings) > 0 {
		return OneBrowserStartResult{}, fmt.Errorf("插件准备失败：%s", strings.Join(warnings, "；"))
	}

	profileID := strings.TrimSpace(request.ProfileID)
	name := strings.TrimSpace(request.Name)
	if name == "" {
		name = "One Browser 窗口"
	}
	proxyID := strings.TrimSpace(request.ProxyID)
	proxyConfig := ""
	if proxyID == "" || proxyID == "__system__" {
		proxyID = ""
		if systemProxy, systemErr := oneBrowserSystemProxyAddress(); systemErr == nil {
			proxyConfig = systemProxy
		} else {
			proxyID = "__direct__"
		}
	}

	var profile *BrowserProfile
	for _, item := range a.BrowserProfileList() {
		if item.ProfileId == profileID {
			input := BrowserProfileInput{
				ProfileName:        name,
				UserDataDir:        item.UserDataDir,
				CoreId:             status.CoreID,
				RestoreLastSession: item.RestoreLastSession,
				FingerprintArgs:    item.FingerprintArgs,
				ProxyId:            proxyID,
				ProxyConfig:        proxyConfig,
				MemoryLimitMB:      item.MemoryLimitMB,
				LaunchArgs:         oneBrowserWindowLaunchArgs(item.LaunchArgs, request),
				Tags:               item.Tags,
				Keywords:           item.Keywords,
				GroupId:            item.GroupId,
			}
			updated, err := a.BrowserProfileUpdate(profileID, input)
			if err != nil {
				return OneBrowserStartResult{}, err
			}
			profile = updated
			break
		}
	}
	if profile == nil {
		created, err := a.BrowserProfileCreate(BrowserProfileInput{
			ProfileName: name,
			CoreId:      status.CoreID,
			ProxyId:     proxyID,
			ProxyConfig: proxyConfig,
			Tags:        []string{"One Browser"},
			LaunchArgs:  oneBrowserWindowLaunchArgs(nil, request),
		})
		if err != nil {
			return OneBrowserStartResult{}, err
		}
		profile = created
	}
	_, _ = browser.RemoveBookmarkURL(a.browserMgr.ResolveUserDataDir(profile), fingerprintCheckBookmarkURL)
	_, _ = browser.RemoveBookmarkName(a.browserMgr.ResolveUserDataDir(profile), "指纹检测")
	_, _ = browser.RemoveBookmarkName(a.browserMgr.ResolveUserDataDir(profile), "Ant 指纹检测")

	workspaceURL, err := a.oneBrowserWorkspaceURL(profile.ProfileId, profile.ProfileName, request, status, proxyConfig)
	if err != nil {
		return OneBrowserStartResult{}, err
	}
	started, err := a.BrowserInstanceStartWithParams(profile.ProfileId, nil, []string{workspaceURL, "https://www.google.com/"}, true)
	if err != nil {
		return OneBrowserStartResult{}, err
	}
	return OneBrowserStartResult{ProfileID: started.ProfileId, Running: started.Running}, nil
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
		if value == "" || lower == "--start-maximized" || strings.HasPrefix(lower, "--window-size") || strings.HasPrefix(lower, "--window-position") || strings.HasPrefix(lower, "--user-agent") {
			continue
		}
		result = append(result, value)
	}
	result = ensureOneBrowserLaunchArgs(result)

	width, height := 900, 680
	if _, err := fmt.Sscanf(strings.NewReplacer("×", "x", "X", "x", " ", "").Replace(request.WindowSize), "%dx%d", &width, &height); err != nil || width < 360 || height < 480 {
		width, height = 900, 680
	}
	result = append(result, fmt.Sprintf("--window-size=%d,%d", width, height))
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
		if systemProxy != "" {
			proxyName = "Windows 系统代理"
		} else {
			proxyName = "直连（未检测到系统代理）"
		}
	}
	data := struct{ Name, Account, Proxy, OS, Language, Timezone, UserAgent, WindowSize, Version, Extensions, WorkspaceID string }{
		Name: profileName, Account: request.Account, Proxy: proxyName, OS: request.OS,
		Language: request.Language, Timezone: request.Timezone, UserAgent: request.UserAgent,
		WindowSize: request.WindowSize, Version: status.Version,
		Extensions: fmt.Sprintf("%d 个已启用", len(request.Extensions)), WorkspaceID: profileID,
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
	if strings.TrimSpace(data.UserAgent) == "" {
		chromeVersion := strings.TrimSpace(data.Version)
		if chromeVersion == "" {
			chromeVersion = "148.0.7778.215"
		}
		data.UserAgent = "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/" + chromeVersion + " Safari/537.36"
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
			PreferredKernel:   "mihomo",
			GroupName:         strings.TrimSpace(group),
			SourceID:          strings.TrimSpace(sourceURL),
			SourceURL:         strings.TrimSpace(sourceURL),
			SourceAutoRefresh: false,
			SortOrder:         index,
		})
	}
	return result, nil
}

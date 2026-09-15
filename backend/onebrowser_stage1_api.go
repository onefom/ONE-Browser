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
	ProfileID  string   `json:"profileId"`
	Name       string   `json:"name"`
	ProxyID    string   `json:"proxyId"`
	ProxyName  string   `json:"proxyName"`
	Account    string   `json:"account"`
	OS         string   `json:"os"`
	Language   string   `json:"language"`
	Timezone   string   `json:"timezone"`
	UserAgent  string   `json:"userAgent"`
	WindowSize string   `json:"windowSize"`
	Extensions []string `json:"extensions"`
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
	for _, core := range a.browserMgr.ListCores() {
		if strings.Contains(strings.ToLower(core.CoreName+" "+core.CorePath), "fingerprint-chromium") {
			return OneBrowserKernelStatus{Installed: true, Version: fingerprintChromiumVersion, CoreID: core.CoreId}
		}
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
				LaunchArgs:         ensureOneBrowserLaunchArgs(item.LaunchArgs),
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
			LaunchArgs:  ensureOneBrowserLaunchArgs(nil),
		})
		if err != nil {
			return OneBrowserStartResult{}, err
		}
		profile = created
	}
	_, _ = browser.RemoveBookmarkURL(a.browserMgr.ResolveUserDataDir(profile), fingerprintCheckBookmarkURL)
	_, _ = browser.RemoveBookmarkName(a.browserMgr.ResolveUserDataDir(profile), "指纹检测")
	_, _ = browser.RemoveBookmarkName(a.browserMgr.ResolveUserDataDir(profile), "Ant 指纹检测")

	if warnings := a.OneBrowserSyncBuiltinExtensions(request.Extensions); len(warnings) > 0 {
		fmt.Fprintf(os.Stderr, "One Browser extensions: %s\n", strings.Join(warnings, "; "))
	}
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
	data := struct{ Name, Account, Proxy, OS, Language, Timezone, UserAgent, WindowSize, Version string }{
		Name: profileName, Account: request.Account, Proxy: proxyName, OS: request.OS,
		Language: request.Language, Timezone: request.Timezone, UserAgent: request.UserAgent,
		WindowSize: request.WindowSize, Version: status.Version,
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
		data.UserAgent = "Fingerprint Chromium · Windows"
	}
	if strings.TrimSpace(data.WindowSize) == "" {
		data.WindowSize = "自适应"
	}
	const page = `<!doctype html><html lang="zh-CN"><head><meta charset="utf-8"><meta name="viewport" content="width=device-width,initial-scale=1"><title>工作台 · {{.Name}}</title><style>:root{color-scheme:light}*{box-sizing:border-box}body{margin:0;font-family:"PingFang SC","Microsoft YaHei","Segoe UI",sans-serif;background:linear-gradient(145deg,#f2f5fc,#eaf0fb);color:#232940;min-height:100vh;padding:42px}main{max-width:1040px;margin:auto}.top{display:flex;justify-content:space-between;align-items:end;margin-bottom:24px}.top h1{margin:0;font-size:28px;font-weight:600}.top p{margin:8px 0 0;color:#7d859d;font-size:14px}.badge{padding:8px 13px;border-radius:999px;background:#e7f7f1;color:#35a37d;font-size:12px}.grid{display:grid;grid-template-columns:repeat(3,minmax(0,1fr));gap:14px}.card{background:rgba(255,255,255,.88);border:1px solid rgba(255,255,255,.95);border-radius:17px;padding:19px 20px;box-shadow:0 9px 28px rgba(66,81,120,.08)}.card span{display:block;color:#8a92a8;font-size:12px;margin-bottom:9px}.card b{font-size:15px;font-weight:550;overflow-wrap:anywhere}.wide{grid-column:span 2}.network-pair{display:grid;grid-template-columns:1fr 1fr;gap:18px}.network-pair div+div{border-left:1px solid #e8eaf1;padding-left:18px}.ip b{color:#655cd0}.footer{margin-top:18px;color:#8a92a8;font-size:12px}@media(max-width:720px){body{padding:22px}.grid{grid-template-columns:1fr}.wide{grid-column:auto}.top{align-items:start;gap:12px}.network-pair{grid-template-columns:1fr}.network-pair div+div{border-left:0;border-top:1px solid #e8eaf1;padding:14px 0 0}}</style></head><body><main><div class="top"><div><h1>{{.Name}}</h1><p>当前窗口配置与网络状态</p></div><span class="badge">独立工作区</span></div><section class="grid"><div class="card ip wide network-pair"><div><span>出口 IP</span><b id="publicIp">正在检测…</b></div><div><span>网络位置</span><b id="networkLocation">正在检测…</b></div></div><div class="card"><span>代理</span><b>{{.Proxy}}</b></div><div class="card"><span>账号</span><b>{{.Account}}</b></div><div class="card"><span>操作系统</span><b>{{.OS}}</b></div><div class="card"><span>语言</span><b>{{.Language}}</b></div><div class="card"><span>时区</span><b>{{.Timezone}}</b></div><div class="card"><span>浏览器内核</span><b>Fingerprint Chromium</b></div><div class="card"><span>内核版本</span><b>{{.Version}}</b></div><div class="card"><span>窗口尺寸</span><b>{{.WindowSize}}</b></div><div class="card wide"><span>User-Agent</span><b>{{.UserAgent}}</b></div></section><p class="footer">Google 已在相邻标签页打开。出口信息由当前窗口网络实时检测。</p></main><script>Promise.allSettled([fetch('https://api.ipify.org?format=json').then(r=>r.json()).then(v=>publicIp.textContent=v.ip||'检测失败'),fetch('https://ipwho.is/').then(r=>r.json()).then(v=>networkLocation.textContent=[v.country,v.city].filter(Boolean).join(' · ')||'检测失败')]).then(results=>results.forEach((r,i)=>{if(r.status==='rejected')(i?networkLocation:publicIp).textContent='检测失败'}));</script></body></html>`
	t, err := template.New("workspace").Parse(page)
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

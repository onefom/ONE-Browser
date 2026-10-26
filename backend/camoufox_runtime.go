package backend

import (
	"encoding/json"
	"fmt"
	"hash/fnv"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

type camoufoxRuntime struct {
	pipe    *camoufoxPipe
	monitor *browserProcessMonitor
}

func (a *App) isCamoufoxProfile(p *BrowserProfile) bool {
	if a == nil || a.browserMgr == nil || p == nil {
		return false
	}
	core, ok := a.browserMgr.GetCore(p.CoreId)
	return ok && a.browserMgr.CoreFamily(core) == "camoufox"
}
func camoufoxHasSession(root string) bool {
	for _, name := range []string{"sessionstore.jsonlz4", "sessionstore-backups/recovery.jsonlz4", "sessionstore-backups/previous.jsonlz4"} {
		if s, e := os.Stat(filepath.Join(root, "camoufox", filepath.FromSlash(name))); e == nil && s.Size() > 0 {
			return true
		}
	}
	return false
}
func camoufoxArg(args []string, key string) string {
	for _, arg := range args {
		if strings.HasPrefix(arg, key+"=") {
			return strings.TrimPrefix(arg, key+"=")
		}
	}
	return ""
}

// Keep the Gecko identity coherent with its version and selected target OS.
// Do not pass Chrome UA, command-line flags or profile databases to Firefox.
func camoufoxConfig(profile *BrowserProfile, version string) map[string]any {
	args := append(append([]string{}, profile.FingerprintArgs...), profile.LaunchArgs...)
	major := strings.Split(version, ".")[0]
	if _, err := strconv.Atoi(major); err != nil {
		major = ""
	}
	config := map[string]any{"allowAddonNewtab": false}
	if major != "" {
		_, _, token, platform, oscpu := oneBrowserSystemIdentity(camoufoxArg(profile.LaunchArgs, "--onebrowser-os"))
		if platform == "MacIntel" {
			token = "Macintosh; Intel Mac OS X 10.15"
		}
		ua := "Mozilla/5.0 (" + token + "; rv:" + major + ".0) Gecko/20100101 Firefox/" + major + ".0"
		config["navigator.userAgent"] = ua
		config["headers.User-Agent"] = ua
		config["navigator.platform"] = platform
		config["navigator.oscpu"] = oscpu
		appOS := "Windows"
		if platform == "MacIntel" {
			appOS = "Macintosh"
		} else if platform == "Linux x86_64" {
			appOS = "X11"
		}
		config["navigator.appVersion"] = "5.0 (" + appOS + ")"
	}
	if ua := camoufoxArg(args, "--user-agent"); strings.Contains(ua, "Firefox/") {
		config["navigator.userAgent"] = ua
		config["headers.User-Agent"] = ua
	}
	lang := camoufoxArg(args, "--lang")
	if lang != "" {
		config["navigator.language"] = lang
		config["headers.Accept-Language"] = lang
		parts := strings.Split(lang, "-")
		config["locale:language"] = parts[0]
		if len(parts) > 1 {
			config["locale:region"] = parts[len(parts)-1]
		}
	}
	if zone := camoufoxArg(args, "--timezone"); zone != "" {
		config["timezone"] = zone
	}
	hash := fnv.New32a()
	hash.Write([]byte(profile.ProfileId))
	config["audio:seed"] = hash.Sum32()
	return config
}
func camoufoxEnvironment(config map[string]any) ([]string, error) {
	data, err := json.Marshal(config)
	if err != nil {
		return nil, err
	}
	env := []string{}
	for _, v := range os.Environ() {
		key := strings.ToUpper(strings.SplitN(v, "=", 2)[0])
		if strings.HasPrefix(key, "CAMOU_CONFIG_") || key == "CAMOU_CONFIG" || key == "PW_PIPE_READ" || key == "PW_PIPE_WRITE" {
			continue
		}
		env = append(env, v)
	}
	var ascii string
	// Split at rune boundaries, matching the official CAMOU_CONFIG transport.
	runes := []rune(string(data))
	for i, n := 0, 1; i < len(runes); i, n = i+1800, n+1 {
		end := i + 1800
		if end > len(runes) {
			end = len(runes)
		}
		ascii = string(runes[i:end])
		env = append(env, fmt.Sprintf("CAMOU_CONFIG_%d=%s", n, ascii))
	}
	return env, nil
}
func writeCamoufoxPrefs(dir, proxy string, args []string) error {
	prefs := map[string]any{
		"taskbar.grouping.useprofile": true, "browser.startup.blankWindow": false,
		"browser.startup.page": 3, "browser.sessionstore.resume_from_crash": true,
		"browser.sessionstore.interval": 15000, "browser.sessionstore.max_resumed_crashes": -1,
		"browser.privatebrowsing.autostart": false, "privacy.sanitize.sanitizeOnShutdown": false,
		"network.cookie.lifetimePolicy": 0, "signon.rememberSignons": true, "signon.autofillForms": true,
		"browser.formfill.enable": true, "browser.shell.checkDefaultBrowser": false,
		"browser.startup.homepage_override.mstone": "ignore", "startup.homepage_welcome_url": "",
		"startup.homepage_welcome_url.additional": "", "browser.aboutwelcome.enabled": false,
		"browser.tabs.warnOnClose": false, "browser.warnOnQuit": false,
		"media.peerconnection.enabled": false, "network.proxy.no_proxies_on": "localhost, 127.0.0.1, ::1",
		"network.proxy.type": 0,
	}
	if proxy != "" && proxy != "direct://" {
		raw := proxy
		if !strings.Contains(raw, "://") {
			raw = "http://" + raw
		}
		u, err := url.Parse(raw)
		if err != nil || u.Hostname() == "" || u.Port() == "" {
			return fmt.Errorf("Firefox 代理地址无效")
		}
		if u.User != nil {
			return fmt.Errorf("Firefox 代理需先经本地连接栈转换，不能直接传入含账号的代理地址")
		}
		port, err := strconv.Atoi(u.Port())
		if err != nil || port < 1 || port > 65535 {
			return fmt.Errorf("Firefox 代理端口无效")
		}
		prefs["network.proxy.type"] = 1
		switch strings.ToLower(u.Scheme) {
		case "socks", "socks5", "socks5h":
			prefs["network.proxy.socks"] = u.Hostname()
			prefs["network.proxy.socks_port"] = port
			prefs["network.proxy.socks_version"] = 5
			prefs["network.proxy.socks_remote_dns"] = true
		case "http":
			for _, scheme := range []string{"http", "ssl"} {
				prefs["network.proxy."+scheme] = u.Hostname()
				prefs["network.proxy."+scheme+"_port"] = port
			}
		default:
			return fmt.Errorf("Firefox 暂不支持此代理传输：%s，请选择本地 HTTP / SOCKS5 节点", u.Scheme)
		}
	}
	if lang := camoufoxArg(args, "--lang"); lang != "" {
		prefs["intl.accept_languages"] = lang
	}
	var content strings.Builder
	for key, value := range prefs {
		k, _ := json.Marshal(key)
		v, _ := json.Marshal(value)
		fmt.Fprintf(&content, "user_pref(%s, %s);\n", k, v)
	}
	return os.WriteFile(filepath.Join(dir, "user.js"), []byte(content.String()), 0600)
}
func (a *App) startCamoufoxLocked(input browserStartInput, profile *BrowserProfile) (*BrowserProfile, error) {
	if value, ok := a.camoufoxRuntimes.Load(profile.ProfileId); ok {
		rt := value.(*camoufoxRuntime)
		if !rt.monitor.HasExited() {
			a.emitBrowserInstanceStarted(profile, true)
			return profile, nil
		}
	}
	root := a.browserMgr.ResolveUserDataDir(profile)
	processes, err := findBrowserUserDataProcesses(root)
	if err != nil {
		return profile, err
	}
	if len(processes) > 0 {
		return profile, fmt.Errorf("此窗口的浏览器仍在运行，请先手动关闭后再启动")
	}
	binary, err := a.browserMgr.ResolveChromeBinary(profile)
	if err != nil {
		return profile, err
	}
	dir := filepath.Join(root, "camoufox")
	if err = os.MkdirAll(dir, 0700); err != nil {
		return profile, err
	}
	effective, bridge, release, err := a.resolveBrowserStartProxy(input, profile)
	if err != nil {
		return profile, err
	}
	defer func() {
		if release {
			a.releaseProxyBridgeRef(bridge)
		}
	}()
	if err = writeCamoufoxPrefs(dir, effective, profile.LaunchArgs); err != nil {
		return profile, err
	}
	core, _ := a.browserMgr.GetCore(profile.CoreId)
	config := camoufoxConfig(profile, a.browserMgr.GetChromeVersion(core.CorePath))
	env, err := camoufoxEnvironment(config)
	if err != nil {
		return profile, err
	}
	args := []string{"-no-remote", "-new-instance", "-profile", dir, "-juggler-pipe"}
	if size := camoufoxArg(profile.LaunchArgs, "--window-size"); size != "" {
		parts := strings.Split(size, ",")
		if len(parts) == 2 {
			args = append(args, "-width", parts[0], "-height", parts[1])
		}
	}
	if !camoufoxHasSession(root) {
		targets := input.StartURLs
		if len(targets) == 0 {
			targets = []string{"https://www.google.com/"}
		}
		args = append(args, targets...)
	}
	cmd := exec.Command(binary, args...)
	cmd.Dir = filepath.Dir(binary)
	cmd.Env = env
	pipe, err := newCamoufoxPipe(cmd)
	if err != nil {
		return profile, err
	}
	monitor, err := newBrowserProcessMonitor(cmd)
	if err != nil {
		pipe.close()
		pipe.closeChild()
		return profile, err
	}
	if err = cmd.Start(); err != nil {
		pipe.close()
		pipe.closeChild()
		monitor.stderr.Close()
		return profile, fmt.Errorf("Firefox 启动失败：%w", err)
	}
	pipe.closeChild()
	monitor.Start()
	go pipe.readLoop()
	// Register before waiting, so a failed readiness check never creates an
	// untracked process which could permit deletion of an active profile.
	a.markProfileRunningLocked(profile.ProfileId, profile, cmd, cmd.Process.Pid, 0, false, "")
	a.setBrowserProcessMonitorLocked(profile.ProfileId, monitor)
	rt := &camoufoxRuntime{pipe: pipe, monitor: monitor}
	a.camoufoxRuntimes.Store(profile.ProfileId, rt)
	if bridge.valid() {
		a.bindProfileProxyBridge(profile.ProfileId, bridge)
		release = false
	}
	a.markProfileLastLaunchArgsLocked(profile, args)
	go func() {
		err := monitor.Wait()
		pipe.close()
		a.camoufoxRuntimes.CompareAndDelete(profile.ProfileId, rt)
		a.browserMgr.Mutex.Lock()
		current := a.browserProcessMonitorIsCurrentLocked(profile.ProfileId, monitor)
		if current {
			a.markProfileStoppedLocked(profile.ProfileId, profile)
			if err != nil {
				profile.LastError = "Firefox 异常退出：" + err.Error() + "；" + monitor.Result().StderrTail
			}
		}
		a.browserMgr.Mutex.Unlock()
		if current {
			a.emitRuntimeEvent("browser:instance:stopped", profile.ProfileId)
		}
	}()
	_, err = pipe.call("Browser.getInfo", map[string]any{}, 15*time.Second)
	if err != nil {
		if monitor.HasExited() {
			return profile, fmt.Errorf("Firefox 启动失败：%s", monitor.Result().StderrTail)
		}
		profile.RuntimeWarning = "Firefox 进程已启动，控制接口尚未就绪；请检查窗口是否已显示"
		a.emitBrowserInstanceStarted(profile, false)
		return profile, nil
	}
	a.emitBrowserInstanceStarted(profile, false)
	return profile, nil
}
func (a *App) stopCamoufox(profileID string) (*BrowserProfile, bool, error) {
	if a.browserMgr == nil {
		return nil, false, nil
	}
	a.browserMgr.Mutex.Lock()
	profile := a.browserMgr.Profiles[profileID]
	isCamoufox := a.isCamoufoxProfile(profile)
	a.browserMgr.Mutex.Unlock()
	if !isCamoufox {
		return nil, false, nil
	}
	value, ok := a.camoufoxRuntimes.Load(profileID)
	if !ok {
		processes, err := findBrowserUserDataProcesses(a.browserMgr.ResolveUserDataDir(profile))
		if err != nil || len(processes) > 0 {
			return profile, true, fmt.Errorf("Firefox 控制连接已断开，请手动关闭该浏览器窗口后重试")
		}
		a.browserMgr.Mutex.Lock()
		a.markProfileStoppedLocked(profileID, profile)
		a.browserMgr.Mutex.Unlock()
		return profile, true, nil
	}
	rt := value.(*camoufoxRuntime)
	if !rt.monitor.HasExited() {
		if err := rt.pipe.send(rt.pipe.seq.Add(1), "Browser.close", map[string]any{}); err != nil {
			return profile, true, fmt.Errorf("Firefox 正常关闭失败，请手动关闭：%w", err)
		}
	}
	timer := time.NewTimer(12 * time.Second)
	defer timer.Stop()
	select {
	case <-rt.monitor.Done():
	case <-timer.C:
		return profile, true, fmt.Errorf("Firefox 仍在保存并关闭，请稍后重试；未强制结束进程")
	}
	// wait for Gecko children to flush profile locks before permitting restart/delete.
	deadline := time.Now().Add(5 * time.Second)
	for {
		processes, err := findBrowserUserDataProcesses(a.browserMgr.ResolveUserDataDir(profile))
		if err != nil {
			return profile, true, err
		}
		if len(processes) == 0 {
			break
		}
		if time.Now().After(deadline) {
			return profile, true, fmt.Errorf("Firefox 后台进程仍在退出，请稍后重试")
		}
		time.Sleep(100 * time.Millisecond)
	}
	a.browserMgr.Mutex.Lock()
	if current := a.browserMgr.Profiles[profileID]; current != nil && a.browserProcessMonitorIsCurrentLocked(profileID, rt.monitor) {
		a.markProfileStoppedLocked(profileID, current)
	}
	a.browserMgr.Mutex.Unlock()
	return profile, true, nil
}

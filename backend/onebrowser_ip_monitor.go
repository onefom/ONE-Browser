package backend

import (
	"ant-chrome/backend/internal/logger"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/wailsapp/wails/v2/pkg/runtime"
)

type OneBrowserIPSnapshot struct {
	IP      string `json:"ip"`
	Country string `json:"country"`
}
type OneBrowserIPWindow struct {
	WindowID  string `json:"windowId"`
	Name      string `json:"name"`
	ProfileID string `json:"-"`
	PID       int    `json:"-"`
	Started   string `json:"-"`
}
type OneBrowserIPChange struct {
	ID      string               `json:"id"`
	Before  OneBrowserIPSnapshot `json:"before"`
	After   OneBrowserIPSnapshot `json:"after"`
	Windows []OneBrowserIPWindow `json:"windows"`
	route   string
}
type oneBrowserIPRoute struct {
	baseline  OneBrowserIPSnapshot
	candidate string
	failed    bool
}
type oneBrowserIPMonitor struct {
	running bool
	wake    chan struct{}
	routes  map[string]*oneBrowserIPRoute
	pending *OneBrowserIPChange
}

// Two consecutive successful observations are required. Errors never replace a
// known baseline and a failed confirmation breaks a candidate sequence.
func (s *oneBrowserIPRoute) observe(ip string, err error) bool {
	if err != nil || net.ParseIP(ip) == nil {
		s.candidate = ""
		return false
	}
	if s.baseline.IP == "" {
		s.baseline.IP = ip
		s.candidate = ""
		return false
	}
	if s.baseline.IP == ip {
		s.candidate = ""
		return false
	}
	if s.candidate != ip {
		s.candidate = ip
		return false
	}
	s.candidate = ""
	return true
}

const oneBrowserIPModePrefix = "one-browser-ip-mode:"

func oneBrowserIPModeKeywords(keywords []string, proxyID string) []string {
	result := []string{}
	for _, keyword := range keywords {
		if !strings.HasPrefix(keyword, oneBrowserIPModePrefix) {
			result = append(result, keyword)
		}
	}
	mode := "custom"
	if proxyID == "" || proxyID == "__system__" {
		mode = "system"
	}
	return append(result, oneBrowserIPModePrefix+mode)
}
func oneBrowserUsesSystemProxy(p BrowserProfile) bool {
	for _, keyword := range p.Keywords {
		if strings.HasPrefix(keyword, oneBrowserIPModePrefix) {
			return keyword == oneBrowserIPModePrefix+"system"
		}
	}
	return p.ProxyId == "" || p.ProxyId == "__system__"
}

// Snapshot live state only: do not initialise data, generate launch codes or scan
// profiles on every network tick.
func (a *App) oneBrowserIPProfiles() []BrowserProfile {
	if a.browserMgr == nil {
		return nil
	}
	a.browserMgr.Mutex.Lock()
	defer a.browserMgr.Mutex.Unlock()
	profiles := make([]BrowserProfile, 0, len(a.browserMgr.Profiles))
	for _, p := range a.browserMgr.Profiles {
		if p != nil {
			copy := *p
			copy.Keywords = append([]string{}, p.Keywords...)
			profiles = append(profiles, copy)
		}
	}
	return profiles
}
func oneBrowserIPGroups(profiles []BrowserProfile) map[string][]OneBrowserIPWindow {
	groups := map[string][]OneBrowserIPWindow{}
	for _, p := range profiles {
		owner := oneBrowserWindowOwner(p)
		if !p.Running || owner == "" || !oneBrowserUsesSystemProxy(p) {
			continue
		}
		// The launch-bound route is authoritative. Never test a custom node or fall
		// back to direct when a configured system proxy fails.
		route := strings.TrimSpace(p.ProxyConfig)
		if route == "" {
			route = "direct://"
		}
		groups[route] = append(groups[route], OneBrowserIPWindow{WindowID: owner, Name: p.ProfileName, ProfileID: p.ProfileId, PID: p.Pid, Started: p.LastStartAt})
	}
	return groups
}
func oneBrowserIPClient(route string) (*http.Client, error) {
	transport := &http.Transport{DisableKeepAlives: true}
	if route != "direct://" {
		u, err := url.Parse(route)
		if err != nil || u.Host == "" || (u.Scheme != "http" && u.Scheme != "https" && u.Scheme != "socks5" && u.Scheme != "socks5h") {
			return nil, fmt.Errorf("无效的系统代理")
		}
		transport.Proxy = http.ProxyURL(u)
	}
	return &http.Client{Transport: transport, Timeout: 8 * time.Second, CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}, nil
}
func oneBrowserIPJSON(ctx context.Context, client *http.Client, address string, out interface{}) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, address, nil)
	if err != nil {
		return err
	}
	req.Header.Set("Cache-Control", "no-cache")
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("IP 服务暂不可用")
	}
	data, err := io.ReadAll(io.LimitReader(resp.Body, 16*1024+1))
	if err != nil {
		return err
	}
	if len(data) > 16*1024 {
		return fmt.Errorf("IP 响应过大")
	}
	return json.Unmarshal(data, out)
}
func oneBrowserProbeIP(ctx context.Context, client *http.Client) (string, error) {
	var data struct {
		IP string `json:"ip"`
	}
	err := oneBrowserIPJSON(ctx, client, "https://api.ipify.org?format=json", &data)
	ip := net.ParseIP(strings.TrimSpace(data.IP))
	if err != nil {
		return "", err
	}
	if ip == nil {
		return "", fmt.Errorf("IP 响应无效")
	}
	return ip.String(), nil
}
func oneBrowserIPCountry(ctx context.Context, client *http.Client, ip string) string {
	if net.ParseIP(ip) == nil {
		return "未知地区"
	}
	var data struct {
		Success bool   `json:"success"`
		Country string `json:"country"`
	}
	// Country is queried only for a baseline or a confirmed change, not every tick.
	if oneBrowserIPJSON(ctx, client, "https://ipwho.is/"+url.PathEscape(ip)+"?lang=zh-CN", &data) != nil || !data.Success || strings.TrimSpace(data.Country) == "" {
		return "未知地区"
	}
	return data.Country
}
func (a *App) OneBrowserStartIPMonitor() error {
	a.oneBrowserIPMu.Lock()
	defer a.oneBrowserIPMu.Unlock()
	if a.oneBrowserIPMonitor == nil {
		a.oneBrowserIPMonitor = &oneBrowserIPMonitor{routes: map[string]*oneBrowserIPRoute{}, wake: make(chan struct{}, 1)}
	}
	if a.oneBrowserIPMonitor.running {
		return nil
	}
	if a.oneBrowserIPMonitor.wake == nil {
		a.oneBrowserIPMonitor.wake = make(chan struct{}, 1)
	}
	wake := a.oneBrowserIPMonitor.wake
	a.oneBrowserIPMonitor.running = true
	err := a.startBackgroundTask(func(ctx context.Context) {
		defer func() { a.oneBrowserIPMu.Lock(); a.oneBrowserIPMonitor.running = false; a.oneBrowserIPMu.Unlock() }()
		timer := time.NewTimer(time.Second)
		defer timer.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-timer.C:
				a.oneBrowserPollIP(ctx)
				timer.Reset(60 * time.Second)
			case <-wake:
				a.oneBrowserPollIP(ctx)
				timer.Reset(60 * time.Second)
			}
		}
	})
	if err != nil {
		a.oneBrowserIPMonitor.running = false
	}
	return err
}
func (a *App) oneBrowserWakeIPMonitor() {
	if err := a.OneBrowserStartIPMonitor(); err != nil {
		logger.New("IPMonitor").Warn("IP 变更检测未能启动，请重新打开客户端")
		return
	}
	a.oneBrowserIPMu.Lock()
	wake := a.oneBrowserIPMonitor.wake
	needed := false
	for route := range oneBrowserIPGroups(a.oneBrowserIPProfiles()) {
		state := a.oneBrowserIPMonitor.routes[route]
		if state == nil || state.baseline.IP == "" {
			needed = true
			break
		}
	}
	a.oneBrowserIPMu.Unlock()
	if !needed {
		return
	}
	select {
	case wake <- struct{}{}:
	default:
	}
}
func (a *App) oneBrowserPollIP(ctx context.Context) {
	a.oneBrowserPollIPWith(ctx, oneBrowserProbeIP, oneBrowserIPCountry, 2*time.Second, true)
}
func (a *App) oneBrowserPollIPWith(ctx context.Context, probe func(context.Context, *http.Client) (string, error), countryFor func(context.Context, *http.Client, string) string, confirmationDelay time.Duration, notify bool) {
	if a.browserMgr == nil {
		return
	}
	groups := oneBrowserIPGroups(a.oneBrowserIPProfiles())
	a.oneBrowserIPMu.Lock()
	m := a.oneBrowserIPMonitor
	if m == nil {
		a.oneBrowserIPMu.Unlock()
		return
	}
	for route := range m.routes {
		if len(groups[route]) == 0 {
			delete(m.routes, route)
		}
	}
	if m.pending != nil {
		a.oneBrowserIPMu.Unlock()
		return
	}
	a.oneBrowserIPMu.Unlock()
	for route, windows := range groups {
		if ctx.Err() != nil {
			return
		}
		client, err := oneBrowserIPClient(route)
		if err != nil {
			continue
		}
		ip, probeErr := probe(ctx, client)
		a.oneBrowserIPMu.Lock()
		state := m.routes[route]
		if state == nil {
			state = &oneBrowserIPRoute{}
			m.routes[route] = state
		}
		wasFailed := state.failed
		state.failed = probeErr != nil
		old := state.baseline
		changed := state.observe(ip, probeErr)
		initial := old.IP == "" && state.baseline.IP != ""
		a.oneBrowserIPMu.Unlock()
		if probeErr != nil && !wasFailed {
			logger.New("IPMonitor").Warn("出口 IP 检测失败，保留上次基准，60 秒后重试")
		}
		if probeErr == nil && wasFailed {
			logger.New("IPMonitor").Info("出口 IP 检测已恢复")
		}
		if probeErr == nil && old.IP != "" && ip != old.IP && !changed {
			timer := time.NewTimer(confirmationDelay)
			select {
			case <-ctx.Done():
				timer.Stop()
				return
			case <-timer.C:
			}
			confirmed, confirmErr := probe(ctx, client)
			a.oneBrowserIPMu.Lock()
			changed = state.observe(confirmed, confirmErr)
			state.failed = confirmErr != nil
			a.oneBrowserIPMu.Unlock()
			if confirmErr != nil && !wasFailed {
				logger.New("IPMonitor").Warn("IP 变更复核失败，保留上次基准，60 秒后重试")
			}
			ip = confirmed
		}
		if initial {
			country := countryFor(ctx, client, ip)
			a.oneBrowserIPMu.Lock()
			state.baseline.Country = country
			a.oneBrowserIPMu.Unlock()
		}
		if initial {
			logger.New("IPMonitor").Info("IP 变更检测已建立基准", logger.F("windows", len(windows)))
		}
		if !changed {
			continue
		}
		country := countryFor(ctx, client, ip)
		// Revalidate after network I/O; only windows from the observed launch survive.
		current := oneBrowserIPGroups(a.oneBrowserIPProfiles())[route]
		affected := []OneBrowserIPWindow{}
		for _, observed := range windows {
			for _, now := range current {
				if oneBrowserSameIPWindow(observed, now) {
					affected = append(affected, now)
					break
				}
			}
		}
		after := OneBrowserIPSnapshot{IP: ip, Country: country}
		a.oneBrowserIPMu.Lock()
		state.baseline = after
		if len(affected) == 0 {
			a.oneBrowserIPMu.Unlock()
			continue
		}
		change := &OneBrowserIPChange{ID: uuid.NewString(), Before: old, After: after, Windows: affected, route: route}
		m.pending = change
		a.oneBrowserIPMu.Unlock()
		logger.New("IPMonitor").Info("出口 IP 变化已确认，等待用户处理", logger.F("windows", len(affected)))
		if !notify {
			return
		}
		a.emitRuntimeEvent("one-browser:ip-change", change)
		a.runtimeMu.RLock()
		appContext := a.ctx
		stopped := a.runtimeStopped
		a.runtimeMu.RUnlock()
		if appContext != nil && !stopped {
			runtime.WindowShow(appContext)
			runtime.WindowUnminimise(appContext)
		}
		return // one unresolved dialog across all windows
	}
}
func oneBrowserSameIPWindow(a, b OneBrowserIPWindow) bool {
	return a.ProfileID == b.ProfileID && a.WindowID == b.WindowID && a.PID == b.PID && a.Started == b.Started
}
func (a *App) OneBrowserPendingIPChange() *OneBrowserIPChange {
	a.oneBrowserIPMu.Lock()
	defer a.oneBrowserIPMu.Unlock()
	if a.oneBrowserIPMonitor == nil || a.oneBrowserIPMonitor.pending == nil {
		return nil
	}
	value := *a.oneBrowserIPMonitor.pending
	value.Windows = append([]OneBrowserIPWindow{}, value.Windows...)
	return &value
}
func (a *App) OneBrowserResolveIPChange(id string, closeWindows bool) error {
	a.oneBrowserIPMu.Lock()
	if a.oneBrowserIPMonitor == nil || a.oneBrowserIPMonitor.pending == nil || a.oneBrowserIPMonitor.pending.ID != id {
		a.oneBrowserIPMu.Unlock()
		return fmt.Errorf("提醒已更新，请重新查看")
	}
	change := *a.oneBrowserIPMonitor.pending
	a.oneBrowserIPMu.Unlock()
	if closeWindows {
		a.oneBrowserWindowMu.Lock()
		defer a.oneBrowserWindowMu.Unlock()
		current := oneBrowserIPGroups(a.oneBrowserIPProfiles())[change.route]
		for _, target := range change.Windows {
			for _, now := range current {
				if oneBrowserSameIPWindow(target, now) {
					if err := a.OneBrowserStop(now.ProfileID); err != nil {
						return fmt.Errorf("部分窗口未能关闭，请重试或手动停止")
					}
					break
				}
			}
		}
	}
	a.oneBrowserIPMu.Lock()
	defer a.oneBrowserIPMu.Unlock()
	if a.oneBrowserIPMonitor.pending != nil && a.oneBrowserIPMonitor.pending.ID == id {
		a.oneBrowserIPMonitor.pending = nil
	}
	return nil
}

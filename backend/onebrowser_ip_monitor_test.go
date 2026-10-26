package backend

import (
	"ant-chrome/backend/internal/browser"
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestIPMonitorConfirmationAndFailures(t *testing.T) {
	s := oneBrowserIPRoute{}
	if s.observe("192.0.2.1", nil) || s.baseline.IP != "192.0.2.1" {
		t.Fatal("baseline")
	}
	if s.observe("198.51.100.2", nil) {
		t.Fatal("single observation")
	}
	if s.observe("", errors.New("offline")) {
		t.Fatal("failure")
	}
	if s.observe("198.51.100.2", nil) {
		t.Fatal("failure must break confirmation")
	}
	if !s.observe("198.51.100.2", nil) {
		t.Fatal("confirmed change")
	}
	s.baseline.IP = "198.51.100.2"
	if s.observe("198.51.100.2", nil) {
		t.Fatal("repeat")
	}
	if s.observe("203.0.113.3", nil) || s.observe("198.51.100.2", nil) || s.observe("203.0.113.3", nil) {
		t.Fatal("flap")
	}
}
func TestIPMonitorGroupsOnlyRunningOwnedSystemWindows(t *testing.T) {
	profile := func(id, proxy, route string, running bool) BrowserProfile {
		return BrowserProfile{ProfileId: id, ProfileName: id, ProxyId: proxy, ProxyConfig: route, Running: running, Keywords: []string{oneBrowserWindowOwnerPrefix + id}, Pid: 42, LastStartAt: "now"}
	}
	groups := oneBrowserIPGroups([]BrowserProfile{profile("a", "__system__", "http://127.0.0.1:8888", true), profile("b", "__system__", "http://127.0.0.1:8888", true), profile("c", "custom", "http://127.0.0.1:8888", true), profile("d", "__system__", "direct://", false), {Running: true, ProxyId: "__system__"}})
	if len(groups) != 1 || len(groups["http://127.0.0.1:8888"]) != 2 {
		t.Fatalf("%+v", groups)
	}
	a := groups["http://127.0.0.1:8888"][0]
	b := a
	b.PID++
	if oneBrowserSameIPWindow(a, b) {
		t.Fatal("restart must invalidate close target")
	}
	b = a
	b.Started = "later"
	if oneBrowserSameIPWindow(a, b) {
		t.Fatal("new launch must be excluded")
	}
	client, err := oneBrowserIPClient("http://127.0.0.1:8888")
	if err != nil || client.Transport.(*http.Transport).Proxy == nil {
		t.Fatal("missing proxy")
	}
	client, err = oneBrowserIPClient("direct://")
	if err != nil || client.Transport.(*http.Transport).Proxy != nil {
		t.Fatal("direct must ignore environment proxy")
	}
	if _, err = oneBrowserIPClient("bad-route"); err == nil {
		t.Fatal("invalid route must not fall back")
	}
}
func TestIPMonitorNetworkValidation(t *testing.T) {
	for _, tt := range []struct {
		body   string
		status int
		valid  bool
	}{{`{"ip":"192.0.2.1"}`, 200, true}, {`{"ip":`, 200, false}, {`{"ip":"192.0.2.1"}`, 429, false}} {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(tt.status); w.Write([]byte(tt.body)) }))
		var out map[string]interface{}
		err := oneBrowserIPJSON(context.Background(), server.Client(), server.URL, &out)
		server.Close()
		if (err == nil) != tt.valid {
			t.Fatalf("%s %v", tt.body, err)
		}
	}
	a := &App{}
	if err := a.OneBrowserResolveIPChange("invalid", true); err == nil {
		t.Fatal("unknown alert must reject")
	}
	if a.OneBrowserPendingIPChange() != nil {
		t.Fatal("unexpected alert")
	}
}

func TestIPMonitorActualPollSharedRouteAndDelivery(t *testing.T) {
	route := "http://127.0.0.1:8888"
	profile := func(id string) *BrowserProfile {
		return &BrowserProfile{ProfileId: id, ProfileName: id, ProxyId: "matched-pool-node", ProxyConfig: route, Running: true, Pid: 1, LastStartAt: "first", Keywords: oneBrowserIPModeKeywords(oneBrowserWindowKeywords(nil, id), "__system__")}
	}
	a := &App{browserMgr: &browser.Manager{Profiles: map[string]*browser.Profile{"a": profile("a"), "b": profile("b")}}, oneBrowserIPMonitor: &oneBrowserIPMonitor{routes: map[string]*oneBrowserIPRoute{}}}
	current := "192.0.2.1"
	calls := 0
	countryCalls := 0
	probe := func(context.Context, *http.Client) (string, error) { calls++; return current, nil }
	country := func(context.Context, *http.Client, string) string { countryCalls++; return "测试国家" }
	a.oneBrowserPollIPWith(context.Background(), probe, country, 0, false)
	if calls != 1 || countryCalls != 1 || a.OneBrowserPendingIPChange() != nil {
		t.Fatal("shared initial baseline")
	}
	a.oneBrowserPollIPWith(context.Background(), probe, country, 0, false)
	if calls != 2 || countryCalls != 1 {
		t.Fatal("unchanged IP must not query country")
	}
	current = "198.51.100.2"
	a.oneBrowserPollIPWith(context.Background(), probe, country, 0, false)
	event := a.OneBrowserPendingIPChange()
	if event == nil || event.Before.IP != "192.0.2.1" || event.After.IP != current || len(event.Windows) != 2 || calls != 4 {
		t.Fatalf("missing event %+v calls %d", event, calls)
	}
	a.oneBrowserPollIPWith(context.Background(), probe, country, 0, false)
	if calls != 4 {
		t.Fatal("pending dialog must pause duplicate probes")
	}
	if err := a.OneBrowserResolveIPChange(event.ID, false); err != nil {
		t.Fatal(err)
	}
	a.oneBrowserPollIPWith(context.Background(), probe, country, 0, false)
	if a.OneBrowserPendingIPChange() != nil {
		t.Fatal("same IP repeated")
	}
	// A custom assignment remains excluded even if its pool ID was cleared later.
	p := *profile("custom")
	p.ProxyId = ""
	p.Keywords = oneBrowserIPModeKeywords(p.Keywords, "custom-node")
	if oneBrowserUsesSystemProxy(p) {
		t.Fatal("lost custom-node ID must not opt into shared monitor")
	}
	a.browserMgr.Profiles["a"].Running = false
	a.browserMgr.Profiles["b"].Running = false
	a.oneBrowserPollIPWith(context.Background(), probe, country, 0, false)
	if calls != 5 || len(a.oneBrowserIPMonitor.routes) != 0 {
		t.Fatal("idle windows must issue no requests and discard baseline")
	}
}

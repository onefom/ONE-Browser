package backend

import (
	"ant-chrome/backend/internal/browser"
	"ant-chrome/backend/internal/config"
	"os"
	"path/filepath"
	"testing"
)

const testWindowA = "11111111-1111-4111-8111-111111111111"
const testWindowB = "22222222-2222-4222-8222-222222222222"

func TestPermanentWindowOwnershipNeverUsesAnotherProfile(t *testing.T) {
	profiles := []BrowserProfile{
		{ProfileId: "a", ProfileName: "same-name", Keywords: oneBrowserWindowKeywords(nil, testWindowA)},
		{ProfileId: "b", ProfileName: "same-name", Keywords: oneBrowserWindowKeywords(nil, testWindowB)},
		{ProfileId: "legacy", ProfileName: "old"},
	}
	for _, tc := range []struct {
		window, claim, want string
		bad                 bool
	}{
		{testWindowA, "a", "a", false}, {testWindowA, "", "a", false},
		{testWindowA, "b", "", true}, {testWindowB, "a", "", true},
		{"33333333-3333-4333-8333-333333333333", "a", "", true},
		{"33333333-3333-4333-8333-333333333333", "legacy", "legacy", false},
	} {
		p, err := resolveOneBrowserWindowProfile(tc.window, tc.claim, profiles)
		if tc.bad {
			if err == nil {
				t.Fatalf("shared profile accepted: %+v", tc)
			}
			continue
		}
		if err != nil || p == nil || p.ProfileId != tc.want {
			t.Fatalf("resolve %+v: %v %v", tc, p, err)
		}
	}
	p, err := resolveOneBrowserWindowProfile("33333333-3333-4333-8333-333333333333", "", profiles)
	if err != nil || p != nil {
		t.Fatal("new window must create a new profile")
	}
}

func TestWindowIDRequiredAndOwnershipReplacesOnlyInternalKeyword(t *testing.T) {
	for _, id := range []string{"", "0", "1", "00000000-0000-0000-0000-000000000000"} {
		if _, err := oneBrowserWindowID(id); err == nil {
			t.Fatalf("accepted unstable id %q", id)
		}
	}
	if _, err := oneBrowserWindowID(testWindowA); err != nil {
		t.Fatal(err)
	}
	keys := oneBrowserWindowKeywords([]string{"keep", oneBrowserWindowOwnerPrefix + testWindowA}, testWindowB)
	if len(keys) != 2 || keys[0] != "keep" || keys[1] != oneBrowserWindowOwnerPrefix+testWindowB {
		t.Fatal(keys)
	}
}

func TestIndependentProfileDirectoriesAndCookies(t *testing.T) {
	root := t.TempDir()
	cfg := config.DefaultConfig()
	mgr := browser.NewManager(cfg, root)
	first, err := mgr.Create(BrowserProfileInput{ProfileName: "same", Keywords: oneBrowserWindowKeywords(nil, testWindowA)})
	if err != nil {
		t.Fatal(err)
	}
	second, err := mgr.Create(BrowserProfileInput{ProfileName: "same", Keywords: oneBrowserWindowKeywords(nil, testWindowB)})
	if err != nil {
		t.Fatal(err)
	}
	if first.ProfileId == second.ProfileId || mgr.ResolveUserDataDir(first) == mgr.ResolveUserDataDir(second) {
		t.Fatal("profile directories shared")
	}
	app := &App{browserMgr: mgr}
	if err := app.validateOneBrowserDataIsolation(first); err != nil {
		t.Fatal(err)
	}
	cookie := filepath.Join(mgr.ResolveUserDataDir(first), "Default", "Cookies")
	if err := os.MkdirAll(filepath.Dir(cookie), 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(cookie, []byte("first-window-only"), 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(mgr.ResolveUserDataDir(second), "Default", "Cookies")); !os.IsNotExist(err) {
		t.Fatal("second profile inherited cookies")
	}
	firstArgs := buildBrowserLaunchArgs(mgr.ResolveUserDataDir(first), 9222, "direct://", nil, nil, nil, nil, true)
	secondArgs := buildBrowserLaunchArgs(mgr.ResolveUserDataDir(second), 9223, "direct://", nil, nil, nil, nil, true)
	if firstArgs[0] == secondArgs[0] || firstArgs[1] == secondArgs[1] {
		t.Fatal("launch arguments share data dir or debug port")
	}
	second.UserDataDir = first.UserDataDir
	if err := app.validateOneBrowserDataIsolation(second); err == nil {
		t.Fatal("duplicate data dir was accepted")
	}
	data, _ := os.ReadFile(cookie)
	if string(data) != "first-window-only" {
		t.Fatal("existing cookies modified")
	}
}

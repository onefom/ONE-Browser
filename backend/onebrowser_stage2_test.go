package backend

import (
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
)

func TestEnsureOneBrowserLaunchArgsDisablesDefaultBrowserPrompt(t *testing.T) {
	args := ensureOneBrowserLaunchArgs([]string{"--disable-sync", "--no-first-run"})
	joined := strings.Join(args, " ")
	if !strings.Contains(joined, "--no-default-browser-check") {
		t.Fatalf("args = %v", args)
	}
	if strings.Count(joined, "--no-first-run") != 1 {
		t.Fatalf("duplicate launch args: %v", args)
	}
}

func TestOneBrowserWorkspaceContainsConfiguration(t *testing.T) {
	app := NewApp(t.TempDir())
	url, err := app.oneBrowserWorkspaceURL("profile-1", "窗口一", OneBrowserStartRequest{ProxyName: "节点 A", Account: "Admin", OS: "Windows 11", Language: "de-DE", Timezone: "Europe/Berlin", WindowSize: "1440 × 900"}, OneBrowserKernelStatus{Version: "148.0"}, "")
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.FromSlash(strings.TrimPrefix(url, "file:///"))
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	text := string(data)
	for _, want := range []string{
		"窗口一", "节点 A", "出口 IP", "Windows 11", "内核类型", "Chrome",
		"User Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64)", "148.0",
		"自定义 · de-DE", "基于 IP 匹配 · Europe/Berlin", "地理位置提示",
		"字体指纹", "跟随系统", "WebRTC", "禁止",
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("workspace missing %q", want)
		}
	}
}

func TestOneBrowserProfilesUseIndependentLaunchWindowsAndDirectories(t *testing.T) {
	first := buildBrowserLaunchArgs("profile-one", 9222, "direct://", nil, nil, nil, []string{"file:///workspace-one.html"}, false)
	second := buildBrowserLaunchArgs("profile-two", 9223, "direct://", nil, nil, nil, []string{"file:///workspace-two.html"}, false)
	for name, args := range map[string][]string{"first": first, "second": second} {
		if !slices.Contains(args, "--new-window") {
			t.Fatalf("%s args = %#v, missing --new-window", name, args)
		}
	}
	if slices.Contains(first, "--user-data-dir=profile-two") || slices.Contains(second, "--user-data-dir=profile-one") {
		t.Fatalf("profile launch directories crossed: first=%#v second=%#v", first, second)
	}
}

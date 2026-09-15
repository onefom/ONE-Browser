package backend

import (
	"os"
	"path/filepath"
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
	for _, want := range []string{"窗口一", "节点 A", "出口 IP", "Windows 11", "Fingerprint Chromium", "148.0", "Europe/Berlin"} {
		if !strings.Contains(text, want) {
			t.Fatalf("workspace missing %q", want)
		}
	}
}

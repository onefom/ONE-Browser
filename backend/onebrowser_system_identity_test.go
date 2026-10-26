package backend

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestSelectedSystemIsAppliedToBothEnginesAndWorkspace(t *testing.T) {
	a, p, _ := pluginTestApp(t)
	a.config = a.browserMgr.Config
	a.runtimeStopped = true
	for _, tc := range []struct{ os, platform, nav, token string }{{"macOS 15", "macos", "MacIntel", "Macintosh"}, {"Ubuntu 24.04", "linux", "Linux x86_64", "Linux"}, {"Windows 11", "windows", "Win32", "Windows NT"}} {
		request := OneBrowserStartRequest{OS: tc.os, Language: "en-US · English", Timezone: "America/New_York"}
		args := oneBrowserConfiguredFingerprintArgs([]string{"--fingerprint=1234", "--fingerprint-platform=windows", "--lang=de-DE", "--disable-spoofing=canvas"}, request)
		if camoufoxArg(args, "--fingerprint-platform") != tc.platform || camoufoxArg(args, "--lang") != "en-US" {
			t.Fatal(tc, args)
		}
		if !strings.Contains(strings.Join(args, " "), "--fingerprint=1234") || !strings.Contains(strings.Join(args, " "), "--disable-spoofing=canvas") {
			t.Fatal("seed or user stability flag changed", args)
		}
		profile := &BrowserProfile{ProfileId: p.ProfileId, FingerprintArgs: args, LaunchArgs: oneBrowserWindowLaunchArgs(nil, request)}
		config := camoufoxConfig(profile, "158.0-beta.40")
		if config["navigator.platform"] != tc.nav || !strings.Contains(config["navigator.userAgent"].(string), tc.token) || config["navigator.userAgent"] != config["headers.User-Agent"] {
			t.Fatal(tc, config)
		}
		for _, family := range []string{"camoufox", "chromium"} {
			status := OneBrowserKernelStatus{Version: "158.0-beta.40", Family: family}
			if _, err := a.oneBrowserWorkspaceURL(p.ProfileId, "Window", request, status, "direct://"); err != nil {
				t.Fatal(err)
			}
			page, _ := os.ReadFile(filepath.Join(a.appRoot, "data", "workspaces", p.ProfileId+".html"))
			text := string(page)
			want := "Chrome"
			if family == "camoufox" {
				want = "Firefox"
			}
			if !strings.Contains(text, tc.os) || !strings.Contains(text, want) || !strings.Contains(text, tc.token) {
				t.Fatal("workspace diverged", tc, family)
			}
		}
	}
}

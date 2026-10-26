package backend

import "strings"

// These are webpage-visible identities, not a change to the host OS.
func oneBrowserSystemIdentity(os string) (platform, version, token, navigatorPlatform, oscpu string) {
	lower := strings.ToLower(os)
	switch {
	case strings.Contains(lower, "mac") || strings.Contains(lower, "苹果"):
		return "macos", "15.0.0", "Macintosh; Intel Mac OS X 10_15_7", "MacIntel", "Intel Mac OS X 10.15"
	case strings.Contains(lower, "ubuntu") || strings.Contains(lower, "linux"):
		return "linux", "6.8.0", "X11; Linux x86_64", "Linux x86_64", "Linux x86_64"
	default:
		version = "13.0.0"
		if strings.Contains(lower, "10") {
			version = "10.0.0"
		}
		return "windows", version, "Windows NT 10.0; Win64; x64", "Win32", "Windows NT 10.0; Win64; x64"
	}
}
func oneBrowserConfiguredFingerprintArgs(args []string, request OneBrowserStartRequest) []string {
	out := []string{}
	for _, arg := range args {
		if strings.HasPrefix(arg, "--fingerprint-platform=") || strings.HasPrefix(arg, "--fingerprint-platform-version=") || strings.HasPrefix(arg, "--lang=") || strings.HasPrefix(arg, "--timezone=") {
			continue
		}
		out = append(out, arg)
	}
	platform, version, _, _, _ := oneBrowserSystemIdentity(request.OS)
	out = append(out, "--fingerprint-platform="+platform, "--fingerprint-platform-version="+version)
	if lang := strings.TrimSpace(strings.Split(request.Language, " · ")[0]); lang != "" {
		out = append(out, "--lang="+lang)
	}
	if request.Timezone != "" {
		out = append(out, "--timezone="+request.Timezone)
	}
	return out
}
func oneBrowserChromiumUA(os, version string) string {
	_, _, token, _, _ := oneBrowserSystemIdentity(os)
	return "Mozilla/5.0 (" + token + ") AppleWebKit/537.36 (KHTML, like Gecko) Chrome/" + version + " Safari/537.36"
}

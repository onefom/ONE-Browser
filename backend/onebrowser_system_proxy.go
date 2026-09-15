package backend

import (
	"fmt"
	"net/url"
	"strings"
)

// normalizeOneBrowserSystemProxy accepts the WinINet forms used by Clash and
// FlClash, including "host:port" and "http=...;https=...;socks=...".
func normalizeOneBrowserSystemProxy(raw string) (string, error) {
	proxyAddress := strings.TrimSpace(raw)
	if proxyAddress == "" {
		return "", fmt.Errorf("Windows 系统代理地址为空")
	}
	selectedKind := ""
	for _, item := range strings.Split(proxyAddress, ";") {
		parts := strings.SplitN(strings.TrimSpace(item), "=", 2)
		if len(parts) != 2 {
			continue
		}
		kind := strings.ToLower(strings.TrimSpace(parts[0]))
		if kind == "https" || kind == "http" || kind == "socks" {
			proxyAddress = strings.TrimSpace(parts[1])
			selectedKind = kind
			break
		}
	}
	if !strings.Contains(proxyAddress, "://") {
		if selectedKind == "socks" {
			proxyAddress = "socks5://" + proxyAddress
		} else {
			proxyAddress = "http://" + proxyAddress
		}
	}
	parsed, err := url.Parse(proxyAddress)
	if err != nil || parsed.Hostname() == "" || parsed.Port() == "" {
		return "", fmt.Errorf("Windows 系统代理格式无效")
	}
	return parsed.String(), nil
}

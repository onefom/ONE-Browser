//go:build windows

package backend

import (
	"fmt"
	"net/http"
	"net/url"
	"time"

	"golang.org/x/sys/windows/registry"
)

func oneBrowserSystemProxyAddress() (string, error) {
	key, err := registry.OpenKey(registry.CURRENT_USER, `Software\Microsoft\Windows\CurrentVersion\Internet Settings`, registry.QUERY_VALUE)
	if err != nil {
		return "", err
	}
	defer key.Close()
	enabled, _, err := key.GetIntegerValue("ProxyEnable")
	if err != nil || enabled == 0 {
		return "", fmt.Errorf("Windows 系统代理未启用")
	}
	raw, _, err := key.GetStringValue("ProxyServer")
	if err != nil {
		return "", fmt.Errorf("Windows 系统代理地址为空")
	}
	return normalizeOneBrowserSystemProxy(raw)
}

// oneBrowserSystemProxyClient reads the WinINet proxy set by Clash/FlClash.
func oneBrowserSystemProxyClient(timeout time.Duration) (*http.Client, string, error) {
	proxyAddress, err := oneBrowserSystemProxyAddress()
	if err != nil {
		return nil, "", err
	}
	parsed, err := url.Parse(proxyAddress)
	if err != nil {
		return nil, "", fmt.Errorf("Windows 系统代理格式无效")
	}
	return &http.Client{Timeout: timeout, Transport: &http.Transport{Proxy: http.ProxyURL(parsed)}}, proxyAddress, nil
}

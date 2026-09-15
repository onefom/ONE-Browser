//go:build !windows

package backend

import (
	"fmt"
	"net/http"
	"time"
)

func oneBrowserSystemProxyClient(timeout time.Duration) (*http.Client, string, error) {
	return nil, "", fmt.Errorf("当前平台不支持读取 Windows 系统代理")
}

func oneBrowserSystemProxyAddress() (string, error) {
	return "", fmt.Errorf("当前平台不支持读取 Windows 系统代理")
}

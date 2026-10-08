package proxy

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"ant-chrome/backend/internal/config"
)

const DefaultIPHealthURL = "https://my.ippure.com/v1/info"

type ipHealthFallbackTarget struct {
	URL    string
	Source string
	Parser string
}

var defaultIPHealthFallbacks = []ipHealthFallbackTarget{
	{URL: "https://ipwho.is/", Source: "ipwho.is", Parser: "json"},
	{URL: "https://ipinfo.io/json", Source: "ipinfo.io", Parser: "json"},
	{URL: "https://api.ipify.org?format=json", Source: "ipify", Parser: "json"},
}

type IPHealthConfig struct {
	URL     string
	Source  string
	Parser  string
	Timeout time.Duration
}

// FetchDefaultIPHealthInfo 使用传入的检测目标查询出口 IP 健康信息。
// 返回值为第三方接口原始 JSON（map 形式），不做本地评分计算。
func FetchDefaultIPHealthInfo(
	proxyId string,
	proxies []config.BrowserProxy,
	xrayMgr *XrayManager,
	singboxMgr *SingBoxManager,
) (map[string]interface{}, error) {
	return FetchIPHealthInfo(proxyId, proxies, xrayMgr, singboxMgr, nil, config.BrowserConnectorXray, nil)
}

func FetchIPHealthInfo(
	proxyId string,
	proxies []config.BrowserProxy,
	xrayMgr *XrayManager,
	singboxMgr *SingBoxManager,
	clashMgr *ClashManager,
	connectorType string,
	cfg *IPHealthConfig,
) (map[string]interface{}, error) {
	if cfg == nil {
		cfg = &IPHealthConfig{}
	}
	targetURL := strings.TrimSpace(cfg.URL)
	if targetURL == "" {
		targetURL = DefaultIPHealthURL
	}
	timeout := cfg.Timeout
	if timeout <= 0 {
		timeout = 20 * time.Second
	}
	source := resolveIPHealthSource(cfg, targetURL)
	parser := resolveIPHealthParser(cfg.Parser)
	meta := map[string]interface{}{
		"_source":    source,
		"_targetUrl": targetURL,
		"_parser":    parser,
	}
	if targetURL == "" {
		meta["error"] = "IP 健康检测目标 URL 为空"
		return meta, fmt.Errorf("IP 健康检测目标 URL 为空")
	}

	src := resolveProxyConfig("", proxies, proxyId)
	if src == "" {
		meta["error"] = "未找到代理配置"
		return meta, fmt.Errorf("未找到代理配置")
	}

	client, err := buildIPHealthHTTPClient(src, proxyId, proxies, xrayMgr, singboxMgr, clashMgr, connectorType, timeout)
	if err != nil {
		meta["error"] = err.Error()
		return meta, fmt.Errorf("创建 IP 健康检测客户端失败（source=%s）: %w", source, err)
	}

	result, primaryErr := fetchIPHealthTarget(client, targetURL, cfg.Parser, source)
	if primaryErr == nil {
		return result, nil
	}

	// The bundled public endpoint can reject desktop traffic temporarily. Only
	// the default target uses fallbacks; an administrator's custom target keeps
	// its configured semantics. All fallbacks use the same route-aware client.
	if strings.EqualFold(strings.TrimSpace(targetURL), DefaultIPHealthURL) {
		for _, fallback := range defaultIPHealthFallbacks {
			if result, err = fetchIPHealthTarget(client, fallback.URL, fallback.Parser, fallback.Source); err == nil {
				result["_fallbackFrom"] = source
				return result, nil
			}
		}
	}

	meta["error"] = "公网 IP 服务暂不可用"
	return meta, fmt.Errorf("公网 IP 服务暂不可用（source=%s）: %w", source, primaryErr)
}

func fetchIPHealthTarget(client *http.Client, targetURL string, parser string, source string) (map[string]interface{}, error) {
	req, err := http.NewRequest(http.MethodGet, targetURL, nil)
	if err != nil {
		return nil, fmt.Errorf("创建请求失败: %w", err)
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", "OneBrowser/1.8")
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("请求失败: %w", err)
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return nil, fmt.Errorf("读取响应失败: %w", err)
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("HTTP %d", resp.StatusCode)
	}
	result, err := parseIPHealthBody(body, parser)
	if err != nil {
		return nil, fmt.Errorf("响应解析失败: %w", err)
	}
	if err := normalizeIPHealthPayload(result, source); err != nil {
		return nil, err
	}
	result["_source"] = source
	result["_targetUrl"] = targetURL
	result["_parser"] = resolveIPHealthParser(parser)
	return result, nil
}

func normalizeIPHealthPayload(result map[string]interface{}, source string) error {
	if result == nil {
		return fmt.Errorf("响应为空")
	}
	if strings.EqualFold(source, "ipwho.is") && result["success"] == false {
		return fmt.Errorf("服务返回失败")
	}
	if location, ok := result["location"].(map[string]interface{}); ok {
		copyIPHealthField(result, location, "country", "country")
		copyIPHealthField(result, location, "region", "state")
		copyIPHealthField(result, location, "city", "city")
		copyIPHealthField(result, location, "countryCode", "country_code")
	}
	if strings.EqualFold(source, "ipinfo.io") {
		if code := strings.TrimSpace(mapString(result, "country")); code != "" {
			result["countryCode"] = code
		}
	}
	if strings.TrimSpace(mapString(result, "ip")) == "" {
		return fmt.Errorf("响应缺少出口 IP")
	}
	return nil
}

func copyIPHealthField(target map[string]interface{}, source map[string]interface{}, targetKey string, sourceKey string) {
	if strings.TrimSpace(mapString(target, targetKey)) != "" {
		return
	}
	if value := strings.TrimSpace(mapString(source, sourceKey)); value != "" {
		target[targetKey] = value
	}
}

func parseIPHealthBody(body []byte, parser string) (map[string]interface{}, error) {
	if strings.EqualFold(strings.TrimSpace(parser), "cloudflare_trace") {
		result := map[string]interface{}{}
		for _, line := range strings.Split(string(body), "\n") {
			key, value, ok := strings.Cut(strings.TrimSpace(line), "=")
			if ok && strings.TrimSpace(key) != "" {
				result[strings.TrimSpace(key)] = strings.TrimSpace(value)
			}
		}
		if ip := mapString(result, "ip"); ip != "" {
			result["ip"] = ip
		}
		return result, nil
	}
	var result map[string]interface{}
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, err
	}
	return result, nil
}

func mapString(data map[string]interface{}, key string) string {
	value, ok := data[key]
	if !ok || value == nil {
		return ""
	}
	if text, ok := value.(string); ok {
		return text
	}
	return fmt.Sprint(value)
}

func buildIPHealthHTTPClient(
	src string,
	proxyId string,
	proxies []config.BrowserProxy,
	xrayMgr *XrayManager,
	singboxMgr *SingBoxManager,
	clashMgr *ClashManager,
	connectorType string,
	timeout time.Duration,
) (*http.Client, error) {
	return buildProxyHTTPClient(src, proxyId, proxies, xrayMgr, singboxMgr, clashMgr, connectorType, timeout)
}

func resolveIPHealthSource(cfg *IPHealthConfig, targetURL string) string {
	if cfg != nil {
		if source := strings.TrimSpace(cfg.Source); source != "" {
			return source
		}
		if parser := strings.TrimSpace(cfg.Parser); parser != "" {
			return parser
		}
	}
	if DefaultIPHealthURL != "" && strings.EqualFold(strings.TrimSpace(targetURL), DefaultIPHealthURL) {
		return "ip_health"
	}
	if parsed, err := url.Parse(strings.TrimSpace(targetURL)); err == nil {
		if host := strings.ToLower(strings.TrimSpace(parsed.Hostname())); host != "" {
			return host
		}
	}
	return "ip_health"
}

func resolveIPHealthParser(parser string) string {
	normalized := strings.TrimSpace(parser)
	if normalized == "" {
		return "json"
	}
	return normalized
}

func bodySnippet(body []byte, max int) string {
	s := strings.TrimSpace(string(body))
	if len(s) <= max {
		return s
	}
	return s[:max] + "..."
}

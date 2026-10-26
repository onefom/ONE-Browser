package backend

import (
	"ant-chrome/backend/internal/logger"
	"encoding/json"
	"strings"
	"time"
)

func oneBrowserUnwantedStartupTab(raw string) bool {
	value := strings.ToLower(strings.TrimRight(strings.TrimSpace(raw), "/"))
	return value == "chrome://newtab" || value == "chrome://new-tab-page" || value == "chrome://apps" || value == "chrome-search://local-ntp/local-ntp.html"
}

// Remove only Chromium's internal new-tab/apps pages. Websites and extension
// onboarding tabs are not classified by title or closed accidentally.
func cleanOneBrowserStartupTabs(debugPort int) error {
	data, err := cdpGetEndpointBody(debugPort, "/json/list")
	if err != nil {
		return err
	}
	var targets []struct {
		ID   string `json:"id"`
		URL  string `json:"url"`
		Type string `json:"type"`
	}
	if err := json.Unmarshal(data, &targets); err != nil {
		return err
	}
	unwanted := []string{}
	keepers := 0
	for _, target := range targets {
		if target.Type != "page" {
			continue
		}
		if oneBrowserUnwantedStartupTab(target.URL) {
			unwanted = append(unwanted, target.ID)
		} else {
			keepers++
		}
	}
	if len(unwanted) == 0 {
		return nil
	}
	// Chromium exits when the last page closes; replace a sole internal page.
	if keepers == 0 {
		if err := createBrowserStartTarget(debugPort, "https://www.google.com/"); err != nil {
			return err
		}
	}
	for _, target := range unwanted {
		if _, err := cdpBrowserCallResult(debugPort, "Target.closeTarget", map[string]any{"targetId": target}); err != nil {
			return err
		}
	}
	return nil
}

func (a *App) cleanOneBrowserStartupTabsForMonitor(profileID string, debugPort int, monitor *browserProcessMonitor) {
	if !a.browserProcessMonitorIsCurrent(profileID, monitor) {
		return
	}
	for attempt := 0; attempt < 2; attempt++ {
		if attempt > 0 {
			time.Sleep(800 * time.Millisecond)
		}
		if !a.browserProcessMonitorIsCurrent(profileID, monitor) {
			return
		}
		if err := cleanOneBrowserStartupTabs(debugPort); err != nil {
			logger.New("Browser").Warn("清理浏览器内部启动标签未完成", logger.F("profile_id", profileID), logger.F("error", err.Error()))
			return
		}
	}
}

package backend

import (
	"fmt"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/google/uuid"
)

const oneBrowserWindowOwnerPrefix = "one-browser-window:"

func oneBrowserWindowID(raw string) (string, error) {
	id, err := uuid.Parse(strings.TrimSpace(raw))
	if err != nil || id == uuid.Nil {
		return "", fmt.Errorf("窗口独立标识无效，请重新打开窗口管理页")
	}
	return id.String(), nil
}

func oneBrowserWindowOwner(profile BrowserProfile) string {
	for _, keyword := range profile.Keywords {
		if strings.HasPrefix(keyword, oneBrowserWindowOwnerPrefix) {
			return strings.TrimPrefix(keyword, oneBrowserWindowOwnerPrefix)
		}
	}
	return ""
}

func oneBrowserWindowKeywords(keywords []string, windowID string) []string {
	result := make([]string, 0, len(keywords)+1)
	for _, keyword := range keywords {
		if !strings.HasPrefix(keyword, oneBrowserWindowOwnerPrefix) {
			result = append(result, keyword)
		}
	}
	return append(result, oneBrowserWindowOwnerPrefix+windowID)
}

// Resolve by permanent UI identity, never by row position or browser name.
// Unclaimed legacy profiles may be adopted once; claimed ones cannot be reused.
func resolveOneBrowserWindowProfile(windowID, claimedID string, profiles []BrowserProfile) (*BrowserProfile, error) {
	var owned, claimed *BrowserProfile
	for _, profile := range profiles {
		p := profile
		if oneBrowserWindowOwner(p) == windowID {
			if owned != nil {
				return nil, fmt.Errorf("该窗口存在重复浏览器绑定，已阻止启动")
			}
			owned = &p
		}
		if p.ProfileId == claimedID {
			claimed = &p
		}
	}
	if owned != nil {
		if claimedID != "" && claimedID != owned.ProfileId {
			return nil, fmt.Errorf("窗口与浏览器配置不匹配，已阻止复用其他窗口")
		}
		return owned, nil
	}
	if claimed != nil {
		if owner := oneBrowserWindowOwner(*claimed); owner != "" && owner != windowID {
			return nil, fmt.Errorf("浏览器数据已属于另一个窗口，已阻止共用")
		}
		return claimed, nil
	}
	return nil, nil
}

func canonicalOneBrowserDataDir(path string) string {
	if absolute, err := filepath.Abs(path); err == nil {
		path = absolute
	}
	if resolved, err := filepath.EvalSymlinks(path); err == nil {
		path = resolved
	}
	path = filepath.Clean(path)
	if runtime.GOOS == "windows" {
		path = strings.ToLower(path)
	}
	return path
}

func (a *App) validateOneBrowserDataIsolation(profile *BrowserProfile) error {
	if profile == nil {
		return fmt.Errorf("窗口配置不存在")
	}
	current := canonicalOneBrowserDataDir(a.browserMgr.ResolveUserDataDir(profile))
	for _, other := range a.BrowserProfileList() {
		if other.ProfileId != profile.ProfileId && canonicalOneBrowserDataDir(a.browserMgr.ResolveUserDataDir(&other)) == current {
			return fmt.Errorf("窗口数据目录与另一窗口重复，已阻止启动以保护独立数据")
		}
	}
	return nil
}

package backend

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
	"time"
	"unicode"

	"ant-chrome/backend/internal/logger"
)

const oneBrowserFolderIDPrefix = "one-browser-folder-id:"
const oneBrowserDirectoryMarker = ".onebrowser-window.json"

var oneBrowserShortIDPattern = regexp.MustCompile(`^[0-9a-f]{8}$`)

type OneBrowserDirectoryResult struct {
	ProfileID string `json:"profileId"`
	Directory string `json:"directory"`
	Deferred  bool   `json:"deferred"`
}

type oneBrowserDirectoryOwner struct {
	ProfileID string `json:"profileId"`
	WindowID  string `json:"windowId"`
	ShortID   string `json:"shortId"`
}

func oneBrowserSafeDirectoryName(name string) string {
	var result strings.Builder
	units := 0
	for _, r := range strings.TrimSpace(name) {
		if unicode.IsControl(r) || strings.ContainsRune(`<>:"/\|?*`, r) {
			r = '_'
		}
		width := 1
		if r > 0xffff {
			width = 2
		}
		if units+width > 48 {
			break
		}
		units += width
		result.WriteRune(r)
	}
	value := strings.TrimRight(strings.TrimSpace(result.String()), ". ")
	if value == "" {
		value = "未命名窗口"
	}
	return value
}

func oneBrowserFolderID(profile BrowserProfile) string {
	for _, key := range profile.Keywords {
		if strings.HasPrefix(key, oneBrowserFolderIDPrefix) {
			value := strings.TrimPrefix(key, oneBrowserFolderIDPrefix)
			if oneBrowserShortIDPattern.MatchString(value) {
				return value
			}
		}
	}
	return ""
}

func oneBrowserAllocateFolderID(profile BrowserProfile, profiles map[string]*BrowserProfile, preferred string) (string, error) {
	owner := oneBrowserWindowOwner(profile)
	used := map[string]bool{}
	for _, other := range profiles {
		if other.ProfileId != profile.ProfileId {
			if id := oneBrowserFolderID(*other); id != "" {
				used[id] = true
			}
		}
	}
	if id := oneBrowserFolderID(profile); id != "" {
		if used[id] {
			return "", fmt.Errorf("窗口短标识重复，已停止目录迁移")
		}
		return id, nil
	}
	if oneBrowserShortIDPattern.MatchString(preferred) && !used[preferred] {
		return preferred, nil
	}
	compact := strings.ReplaceAll(owner, "-", "")
	for attempt := 0; attempt < 65536; attempt++ {
		candidate := ""
		if attempt == 0 && len(compact) >= 8 {
			candidate = compact[:8]
		} else {
			sum := sha256.Sum256([]byte(fmt.Sprintf("%s:%d", owner, attempt)))
			candidate = hex.EncodeToString(sum[:4])
		}
		if oneBrowserShortIDPattern.MatchString(candidate) && !used[candidate] {
			return candidate, nil
		}
	}
	return "", fmt.Errorf("无法分配独立目录短标识")
}

func oneBrowserReadDirectoryOwner(path string) (oneBrowserDirectoryOwner, error) {
	var owner oneBrowserDirectoryOwner
	data, err := os.ReadFile(filepath.Join(path, oneBrowserDirectoryMarker))
	if err != nil {
		return owner, err
	}
	err = json.Unmarshal(data, &owner)
	return owner, err
}

// Recover the original data after an interruption between the folder move and
// metadata save. Never adopt a directory without a matching full identity.
func oneBrowserRecoverDirectory(parent string, profileID, windowID string) (string, oneBrowserDirectoryOwner, error) {
	var found string
	var owner oneBrowserDirectoryOwner
	entries, err := os.ReadDir(parent)
	if os.IsNotExist(err) {
		return "", owner, nil
	}
	if err != nil {
		return "", owner, err
	}
	for _, entry := range entries {
		if !entry.IsDir() || !strings.HasPrefix(entry.Name(), "OneBrowser-") {
			continue
		}
		path := filepath.Join(parent, entry.Name())
		record, readErr := oneBrowserReadDirectoryOwner(path)
		if readErr == nil && record.ProfileID == profileID && record.WindowID == windowID {
			if found != "" {
				return "", owner, fmt.Errorf("发现多个相同窗口的数据目录，已停止自动迁移")
			}
			found = path
			owner = record
		}
	}
	return found, owner, nil
}

func oneBrowserWriteDirectoryOwner(path string, owner oneBrowserDirectoryOwner) error {
	payload, err := json.Marshal(owner)
	if err != nil {
		return err
	}
	file, err := os.CreateTemp(path, ".onebrowser-directory-*.tmp")
	if err != nil {
		return err
	}
	temp := file.Name()
	defer os.Remove(temp)
	if _, err = file.Write(payload); err != nil {
		file.Close()
		return err
	}
	if err = file.Sync(); err != nil {
		file.Close()
		return err
	}
	if err = file.Close(); err != nil {
		return err
	}
	return os.Rename(temp, filepath.Join(path, oneBrowserDirectoryMarker))
}

func (a *App) saveOneBrowserDirectoryProfileLocked(profile *BrowserProfile) error {
	if a.browserMgr.ProfileDAO != nil {
		return a.browserMgr.ProfileDAO.Upsert(profile)
	}
	return a.browserMgr.SaveProfiles()
}

// The caller holds oneBrowserWindowMu. BrowserProcesses and persistence are
// protected by the manager mutex for the complete move/save transaction.
func (a *App) syncOneBrowserDirectory(profileID, name, windowID string) (OneBrowserDirectoryResult, error) {
	a.browserMgr.InitData()
	a.browserMgr.Mutex.Lock()
	defer a.browserMgr.Mutex.Unlock()
	profile := a.browserMgr.Profiles[profileID]
	if profile == nil {
		return OneBrowserDirectoryResult{}, fmt.Errorf("窗口配置不存在")
	}
	if owner := oneBrowserWindowOwner(*profile); owner != "" && owner != windowID {
		return OneBrowserDirectoryResult{}, fmt.Errorf("该数据目录属于其他窗口")
	}
	oldName, oldDir, oldKeys, oldUpdated := profile.ProfileName, profile.UserDataDir, append([]string{}, profile.Keywords...), profile.UpdatedAt
	source := a.browserMgr.ResolveUserDataDir(profile)
	for _, other := range a.browserMgr.Profiles {
		if other.ProfileId != profileID && canonicalOneBrowserDataDir(a.browserMgr.ResolveUserDataDir(other)) == canonicalOneBrowserDataDir(source) {
			return OneBrowserDirectoryResult{}, fmt.Errorf("窗口数据目录重复，无法同步名称")
		}
	}
	parent := filepath.Dir(source)
	sourceInfo, statErr := os.Stat(source)
	var record oneBrowserDirectoryOwner
	if statErr == nil && !sourceInfo.IsDir() {
		return OneBrowserDirectoryResult{}, fmt.Errorf("窗口数据路径不是文件夹")
	}
	if statErr != nil && !os.IsNotExist(statErr) {
		return OneBrowserDirectoryResult{}, statErr
	}
	if os.IsNotExist(statErr) {
		recovered, owner, err := oneBrowserRecoverDirectory(parent, profileID, windowID)
		if err != nil {
			return OneBrowserDirectoryResult{}, err
		}
		if recovered != "" {
			source = recovered
			record = owner
			sourceInfo, statErr = os.Stat(source)
		}
	} else {
		owner, err := oneBrowserReadDirectoryOwner(source)
		if err == nil {
			record = owner
			if owner.ProfileID != profileID || owner.WindowID != windowID {
				return OneBrowserDirectoryResult{}, fmt.Errorf("数据目录归属校验不匹配")
			}
		}
		if err != nil && !os.IsNotExist(err) {
			return OneBrowserDirectoryResult{}, fmt.Errorf("读取目录标识失败：%w", err)
		}
	}
	shortID, err := oneBrowserAllocateFolderID(*profile, a.browserMgr.Profiles, record.ShortID)
	if err != nil {
		return OneBrowserDirectoryResult{}, err
	}
	folder := "OneBrowser-" + oneBrowserSafeDirectoryName(name) + "-" + shortID
	target := filepath.Join(parent, folder)
	targetRef := target
	if !filepath.IsAbs(oldDir) {
		targetRef = filepath.Join(filepath.Dir(oldDir), folder)
		if oldDir == "" {
			targetRef = folder
		}
	}
	keys := oneBrowserWindowKeywords(profile.Keywords, windowID)
	kept := make([]string, 0, len(keys)+1)
	for _, key := range keys {
		if !strings.HasPrefix(key, oneBrowserFolderIDPrefix) {
			kept = append(kept, key)
		}
	}
	kept = append(kept, oneBrowserFolderIDPrefix+shortID)
	result := OneBrowserDirectoryResult{ProfileID: profileID, Directory: filepath.Base(source)}
	if statErr == nil && filepath.Clean(source) == filepath.Clean(target) && record.ProfileID == profileID && record.WindowID == windowID && record.ShortID == shortID && oldName == name && oldDir == targetRef && slices.Equal(oldKeys, kept) {
		return result, nil
	}

	// A tracked running window must retain the exact launch path until exit.
	active := profile.Running
	if !active && statErr == nil && filepath.Clean(source) != filepath.Clean(target) {
		processes, scanErr := findBrowserUserDataProcesses(source)
		if scanErr != nil {
			return result, fmt.Errorf("无法确认浏览器已关闭，目录未移动：%w", scanErr)
		}
		active = len(processes) > 0
	}
	profile.ProfileName = name
	profile.Keywords = kept
	profile.UpdatedAt = time.Now().Format(time.RFC3339)
	rollbackProfile := func() {
		profile.ProfileName = oldName
		profile.UserDataDir = oldDir
		profile.Keywords = oldKeys
		profile.UpdatedAt = oldUpdated
	}
	if active {
		// Recovery may have found a live directory after an interrupted move.
		profile.UserDataDir = source
		if !filepath.IsAbs(oldDir) {
			profile.UserDataDir = filepath.Join(filepath.Dir(oldDir), filepath.Base(source))
		}
		if err := a.saveOneBrowserDirectoryProfileLocked(profile); err != nil {
			rollbackProfile()
			return result, err
		}
		result.Deferred = filepath.Clean(source) != filepath.Clean(target)
		return result, nil
	}
	moved, created := false, false
	if filepath.Clean(source) != filepath.Clean(target) {
		if targetInfo, targetErr := os.Stat(target); targetErr == nil {
			if statErr != nil || !os.SameFile(sourceInfo, targetInfo) {
				rollbackProfile()
				return result, fmt.Errorf("目标文件夹已存在，未覆盖或合并任何窗口数据")
			}
		} else if !os.IsNotExist(targetErr) {
			rollbackProfile()
			return result, targetErr
		}
	}
	// Store a non-secret full-identity marker before moving any existing data.
	markerDir := source
	if os.IsNotExist(statErr) {
		markerDir = target
		if err := os.MkdirAll(markerDir, 0700); err != nil {
			rollbackProfile()
			return result, err
		}
		created = true
	}
	if err := oneBrowserWriteDirectoryOwner(markerDir, oneBrowserDirectoryOwner{ProfileID: profileID, WindowID: windowID, ShortID: shortID}); err != nil {
		rollbackProfile()
		return result, err
	}
	if !created && filepath.Clean(source) != filepath.Clean(target) {
		if err := os.Rename(source, target); err != nil {
			rollbackProfile()
			return result, fmt.Errorf("数据目录改名失败，原数据已保留：%w", err)
		}
		moved = true
	}
	profile.UserDataDir = targetRef
	if err := a.saveOneBrowserDirectoryProfileLocked(profile); err != nil {
		rollbackProfile()
		if moved {
			if rollbackErr := os.Rename(target, source); rollbackErr != nil {
				return result, fmt.Errorf("保存路径失败：%v；目录保留于 %s，可根据完整标识恢复：%w", err, target, rollbackErr)
			}
		}
		if created {
			_ = os.Remove(filepath.Join(target, oneBrowserDirectoryMarker))
			_ = os.Remove(target)
		}
		return result, fmt.Errorf("保存目录路径失败，已回退：%w", err)
	}
	result.Directory = folder
	return result, nil
}

func (a *App) OneBrowserRenameWindow(windowID, profileID, name string) (OneBrowserDirectoryResult, error) {
	id, err := oneBrowserWindowID(windowID)
	if err != nil {
		return OneBrowserDirectoryResult{}, err
	}
	name = strings.TrimSpace(name)
	if name == "" {
		return OneBrowserDirectoryResult{}, fmt.Errorf("窗口名称不能为空")
	}
	if a.browserMgr == nil {
		return OneBrowserDirectoryResult{}, fmt.Errorf("浏览器服务未就绪")
	}
	a.oneBrowserWindowMu.Lock()
	defer a.oneBrowserWindowMu.Unlock()
	profile, err := resolveOneBrowserWindowProfile(id, strings.TrimSpace(profileID), a.BrowserProfileList())
	if err != nil {
		return OneBrowserDirectoryResult{}, err
	}
	if profile == nil {
		if strings.TrimSpace(profileID) != "" {
			return OneBrowserDirectoryResult{}, fmt.Errorf("窗口配置不存在，未改动原数据目录")
		}
		return OneBrowserDirectoryResult{}, nil
	}
	return a.syncOneBrowserDirectory(profile.ProfileId, name, id)
}

func (a *App) syncStoppedOneBrowserDirectory(profileID string) {
	a.oneBrowserWindowMu.Lock()
	defer a.oneBrowserWindowMu.Unlock()
	if a.browserMgr == nil {
		return
	}
	a.browserMgr.Mutex.Lock()
	p := a.browserMgr.Profiles[profileID]
	if p == nil || p.Running {
		a.browserMgr.Mutex.Unlock()
		return
	}
	name, owner := p.ProfileName, oneBrowserWindowOwner(*p)
	a.browserMgr.Mutex.Unlock()
	if owner == "" {
		return
	}
	if _, err := a.syncOneBrowserDirectory(profileID, name, owner); err != nil {
		logger.New("Browser").Warn("窗口目录名称等待同步", logger.F("profile_id", profileID), logger.F("error", err.Error()))
	}
}

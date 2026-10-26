package backend

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"unicode/utf16"

	"ant-chrome/backend/internal/browser"
	"ant-chrome/backend/internal/config"
)

type failingDirectoryDAO struct{ browser.ProfileDAO }

func (d failingDirectoryDAO) Upsert(*browser.Profile) error {
	return errors.New("simulated persistence failure")
}

func directoryTestApp(t *testing.T) (*App, *BrowserProfile) {
	t.Helper()
	cfg := config.DefaultConfig()
	root := t.TempDir()
	mgr := browser.NewManager(cfg, root)
	p, err := mgr.Create(BrowserProfileInput{ProfileName: "旧名称", Keywords: oneBrowserWindowKeywords(nil, testWindowA)})
	if err != nil {
		t.Fatal(err)
	}
	original := findBrowserUserDataProcesses
	findBrowserUserDataProcesses = func(string) ([]browserUserDataProcess, error) { return nil, nil }
	t.Cleanup(func() { findBrowserUserDataProcesses = original })
	return &App{browserMgr: mgr, appRoot: root}, p
}

func putDirectoryTestData(t *testing.T, root string) map[string]string {
	t.Helper()
	files := map[string]string{"Default/Network/Cookies": "cookie-bytes", "Default/Login Data": "encrypted-password-bytes", "Default/Sessions/Session_1": "SNSS-session-bytes", "Default/Local Storage/leveldb/000001.log": "site-data", "Default/Local Extension Settings/test/000001.log": "plugin-data"}
	for name, data := range files {
		path := filepath.Join(root, filepath.FromSlash(name))
		if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(data), 0600); err != nil {
			t.Fatal(err)
		}
	}
	return files
}
func checkDirectoryTestData(t *testing.T, root string, files map[string]string) {
	t.Helper()
	for name, want := range files {
		data, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(name)))
		if err != nil || string(data) != want {
			t.Fatalf("data changed %s: %v %q", name, err, data)
		}
	}
}

func TestWindowDirectoryMigratesAndFollowsNameWithoutChangingIdentity(t *testing.T) {
	app, p := directoryTestApp(t)
	old := app.browserMgr.ResolveUserDataDir(p)
	files := putDirectoryTestData(t, old)
	result, err := app.OneBrowserRenameWindow(testWindowA, p.ProfileId, "工作窗口")
	if err != nil {
		t.Fatal(err)
	}
	if result.Deferred || result.Directory != "OneBrowser-工作窗口-11111111" {
		t.Fatal(result)
	}
	if _, err := os.Stat(old); !os.IsNotExist(err) {
		t.Fatal("old UUID folder was not moved")
	}
	checkDirectoryTestData(t, app.browserMgr.ResolveUserDataDir(p), files)
	originalID, short := p.ProfileId, oneBrowserFolderID(*p)
	result, err = app.OneBrowserRenameWindow(testWindowA, p.ProfileId, "新的窗口名")
	if err != nil {
		t.Fatal(err)
	}
	if result.Directory != "OneBrowser-新的窗口名-11111111" || p.ProfileId != originalID || oneBrowserFolderID(*p) != short {
		t.Fatal("permanent identity changed")
	}
	checkDirectoryTestData(t, app.browserMgr.ResolveUserDataDir(p), files)
	cfg, err := config.Load(app.browserMgr.ResolveRelativePath("config.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	if len(cfg.Browser.Profiles) != 1 || cfg.Browser.Profiles[0].UserDataDir != p.UserDataDir {
		t.Fatal("directory not persisted")
	}
}

func TestRunningDirectoryRenameIsDeferredUntilStopped(t *testing.T) {
	app, p := directoryTestApp(t)
	old := app.browserMgr.ResolveUserDataDir(p)
	files := putDirectoryTestData(t, old)
	p.Running = true
	result, err := app.OneBrowserRenameWindow(testWindowA, p.ProfileId, "运行中改名")
	if err != nil || !result.Deferred {
		t.Fatalf("%v %v", result, err)
	}
	if app.browserMgr.ResolveUserDataDir(p) != old || p.ProfileName != "运行中改名" {
		t.Fatal("running data path moved or name not saved")
	}
	checkDirectoryTestData(t, old, files)
	p.Running = false
	app.syncStoppedOneBrowserDirectory(p.ProfileId)
	if filepath.Base(p.UserDataDir) != "OneBrowser-运行中改名-11111111" {
		t.Fatal("stopped directory not synchronized")
	}
	checkDirectoryTestData(t, app.browserMgr.ResolveUserDataDir(p), files)
}

func TestDirectoryDoesNotMoveForUntrackedActiveBrowser(t *testing.T) {
	app, p := directoryTestApp(t)
	old := app.browserMgr.ResolveUserDataDir(p)
	files := putDirectoryTestData(t, old)
	findBrowserUserDataProcesses = func(string) ([]browserUserDataProcess, error) { return []browserUserDataProcess{{PID: 123}}, nil }
	result, err := app.OneBrowserRenameWindow(testWindowA, p.ProfileId, "等待关闭")
	if err != nil || !result.Deferred || app.browserMgr.ResolveUserDataDir(p) != old {
		t.Fatalf("%v %v", result, err)
	}
	checkDirectoryTestData(t, old, files)
}

func TestDirectoryMoveRollsBackWhenPersistenceFails(t *testing.T) {
	app, p := directoryTestApp(t)
	old := app.browserMgr.ResolveUserDataDir(p)
	files := putDirectoryTestData(t, old)
	oldName := p.ProfileName
	app.browserMgr.ProfileDAO = failingDirectoryDAO{}
	if _, err := app.OneBrowserRenameWindow(testWindowA, p.ProfileId, "不会保存"); err == nil {
		t.Fatal("save failure ignored")
	}
	if app.browserMgr.ResolveUserDataDir(p) != old || p.ProfileName != oldName {
		t.Fatal("profile not rolled back")
	}
	checkDirectoryTestData(t, old, files)
}

func TestDirectoryShortCollisionAndOccupiedTargetAreNeverMerged(t *testing.T) {
	app, p := directoryTestApp(t)
	other := &BrowserProfile{ProfileId: "another", UserDataDir: "another", Keywords: []string{oneBrowserFolderIDPrefix + "11111111"}}
	app.browserMgr.Profiles[other.ProfileId] = other
	id, err := oneBrowserAllocateFolderID(*p, app.browserMgr.Profiles, "")
	if err != nil || id == "11111111" || !oneBrowserShortIDPattern.MatchString(id) {
		t.Fatalf("id=%q err=%v", id, err)
	}
	old := app.browserMgr.ResolveUserDataDir(p)
	files := putDirectoryTestData(t, old)
	target := filepath.Join(filepath.Dir(old), "OneBrowser-已占用-"+id)
	if err := os.MkdirAll(target, 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(target, "sentinel"), []byte("other-data"), 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := app.OneBrowserRenameWindow(testWindowA, p.ProfileId, "已占用"); err == nil {
		t.Fatal("occupied directory merged")
	}
	checkDirectoryTestData(t, old, files)
	data, _ := os.ReadFile(filepath.Join(target, "sentinel"))
	if string(data) != "other-data" {
		t.Fatal("foreign data modified")
	}
}

func TestDirectoryRecoversInterruptedMoveByFullIdentity(t *testing.T) {
	app, p := directoryTestApp(t)
	old := app.browserMgr.ResolveUserDataDir(p)
	files := putDirectoryTestData(t, old)
	if err := oneBrowserWriteDirectoryOwner(old, oneBrowserDirectoryOwner{ProfileID: p.ProfileId, WindowID: testWindowA, ShortID: "11111111"}); err != nil {
		t.Fatal(err)
	}
	moved := filepath.Join(filepath.Dir(old), "OneBrowser-中断时名称-11111111")
	if err := os.Rename(old, moved); err != nil {
		t.Fatal(err)
	}
	result, err := app.OneBrowserRenameWindow(testWindowA, p.ProfileId, "恢复后名称")
	if err != nil || result.Directory != "OneBrowser-恢复后名称-11111111" {
		t.Fatalf("%v %v", result, err)
	}
	checkDirectoryTestData(t, app.browserMgr.ResolveUserDataDir(p), files)
}

func TestDirectoryNameSanitizesWindowsCharactersAndPreservesChinese(t *testing.T) {
	if got := oneBrowserSafeDirectoryName("  中文:/测试*?. "); got != "中文__测试__" {
		t.Fatal(got)
	}
	if got := oneBrowserSafeDirectoryName(" . "); got != "未命名窗口" {
		t.Fatal(got)
	}
	if got := oneBrowserSafeDirectoryName(strings.Repeat("窗口😀", 40)); len(utf16.Encode([]rune(got))) > 48 {
		t.Fatal("name too long")
	}
}

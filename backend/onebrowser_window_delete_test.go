package backend

import (
	"ant-chrome/backend/internal/browser"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

func deletableWindow(t *testing.T) (*App, *BrowserProfile, map[string]string) {
	t.Helper()
	a, p := directoryTestApp(t)
	files := putDirectoryTestData(t, a.browserMgr.ResolveUserDataDir(p))
	if _, err := a.OneBrowserRenameWindow(testWindowA, p.ProfileId, "删除测试"); err != nil {
		t.Fatal(err)
	}
	return a, p, files
}

func TestDeleteWindowRemovesOwnedDataAndRecordOnly(t *testing.T) {
	a, p, _ := deletableWindow(t)
	path := a.browserMgr.ResolveUserDataDir(p)
	other, err := a.browserMgr.Create(BrowserProfileInput{ProfileName: "保留", Keywords: oneBrowserWindowKeywords(nil, testWindowB)})
	if err != nil {
		t.Fatal(err)
	}
	keep := putDirectoryTestData(t, a.browserMgr.ResolveUserDataDir(other))
	shared := filepath.Join(a.appRoot, "data", "shared", "extension-code", "keep")
	if err := os.MkdirAll(shared, 0700); err != nil {
		t.Fatal(err)
	}
	if err := a.OneBrowserDeleteWindow(testWindowA, p.ProfileId); err != nil {
		t.Fatal(err)
	}
	if _, ok := a.browserMgr.Profiles[p.ProfileId]; ok {
		t.Fatal("record survived")
	}
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Fatal("data survived", err)
	}
	checkDirectoryTestData(t, a.browserMgr.ResolveUserDataDir(other), keep)
	if _, err := os.Stat(shared); err != nil {
		t.Fatal("shared code deleted")
	}
}

func TestDeleteWindowRejectsRunningActualProcessesAndScanFailure(t *testing.T) {
	for _, mode := range []string{"tracked", "command", "actual", "scan-error"} {
		t.Run(mode, func(t *testing.T) {
			a, p, files := deletableWindow(t)
			switch mode {
			case "tracked":
				p.Running = true
			case "command":
				a.browserMgr.BrowserProcesses[p.ProfileId] = &exec.Cmd{}
			case "actual":
				findBrowserUserDataProcesses = func(string) ([]browserUserDataProcess, error) { return []browserUserDataProcess{{PID: 123}}, nil }
			case "scan-error":
				findBrowserUserDataProcesses = func(string) ([]browserUserDataProcess, error) { return nil, errors.New("scan failed") }
			}
			if err := a.OneBrowserDeleteWindow(testWindowA, p.ProfileId); err == nil {
				t.Fatal("running delete allowed")
			}
			checkDirectoryTestData(t, a.browserMgr.ResolveUserDataDir(p), files)
			if a.browserMgr.Profiles[p.ProfileId] == nil {
				t.Fatal("record lost")
			}
		})
	}
}

type failWindowDeleteDAO struct{ browser.ProfileDAO }

func (d failWindowDeleteDAO) ListDeleted() ([]*browser.Profile, error) { return nil, nil }
func (d failWindowDeleteDAO) Delete(string) error                      { return errors.New("database unavailable") }

func TestDeleteWindowPersistenceFailureRollsBackDirectory(t *testing.T) {
	a, p, files := deletableWindow(t)
	a.browserMgr.ProfileDAO = failWindowDeleteDAO{}
	if err := a.OneBrowserDeleteWindow(testWindowA, p.ProfileId); err == nil {
		t.Fatal("failure hidden")
	}
	checkDirectoryTestData(t, a.browserMgr.ResolveUserDataDir(p), files)
	if a.browserMgr.Profiles[p.ProfileId] == nil {
		t.Fatal("record lost")
	}
}

func TestDeleteWindowRejectsForeignMarkerAndSharedDirectory(t *testing.T) {
	for _, mode := range []string{"foreign", "shared", "outside", "wrong-profile"} {
		t.Run(mode, func(t *testing.T) {
			a, p, files := deletableWindow(t)
			original := a.browserMgr.ResolveUserDataDir(p)
			claim := p.ProfileId
			switch mode {
			case "foreign":
				if err := oneBrowserWriteDirectoryOwner(original, oneBrowserDirectoryOwner{ProfileID: "foreign", WindowID: testWindowB}); err != nil {
					t.Fatal(err)
				}
			case "shared":
				_, err := a.browserMgr.Create(BrowserProfileInput{ProfileName: "重叠", UserDataDir: p.UserDataDir, Keywords: oneBrowserWindowKeywords(nil, testWindowB)})
				if err != nil {
					t.Fatal(err)
				}
			case "outside":
				p.UserDataDir = t.TempDir()
			case "wrong-profile":
				claim = "another-profile"
			}
			if err := a.OneBrowserDeleteWindow(testWindowA, claim); err == nil {
				t.Fatal("unsafe delete allowed")
			}
			checkDirectoryTestData(t, original, files)
		})
	}
}

func TestDetachedWindowListingNeverDeletesData(t *testing.T) {
	a, p, files := deletableWindow(t)
	entries, err := a.OneBrowserDetachedWindows([]string{testWindowB})
	if err != nil || len(entries) != 1 || entries[0].WindowID != testWindowA {
		t.Fatalf("%v %v", entries, err)
	}
	checkDirectoryTestData(t, a.browserMgr.ResolveUserDataDir(p), files)
	p.Running = true
	entries, err = a.OneBrowserDetachedWindows(nil)
	if err != nil || len(entries) != 1 || !entries[0].Running {
		t.Fatalf("%v %v", entries, err)
	}
	entries, err = a.OneBrowserDetachedWindows([]string{testWindowA})
	if err != nil || len(entries) != 0 {
		t.Fatalf("current window treated as detached: %v %v", entries, err)
	}
}

func TestDeleteWindowRecoversInterruptedMove(t *testing.T) {
	a, p, _ := deletableWindow(t)
	path := a.browserMgr.ResolveUserDataDir(p)
	stage := path + ".deleting-" + testWindowA
	if err := os.Rename(path, stage); err != nil {
		t.Fatal(err)
	}
	if err := a.OneBrowserDeleteWindow(testWindowA, p.ProfileId); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(stage); !os.IsNotExist(err) {
		t.Fatal("staged data survived")
	}
}

func TestDeleteWindowRetryFinishesCommittedRemoval(t *testing.T) {
	a, p, _ := deletableWindow(t)
	path := a.browserMgr.ResolveUserDataDir(p)
	stage := path + ".deleting-" + testWindowA
	if err := os.Rename(path, stage); err != nil {
		t.Fatal(err)
	}
	delete(a.browserMgr.Profiles, p.ProfileId)
	if err := a.browserMgr.SaveProfiles(); err != nil {
		t.Fatal(err)
	}
	if err := a.OneBrowserDeleteWindow(testWindowA, p.ProfileId); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(stage); !os.IsNotExist(err) {
		t.Fatal("committed removal not completed")
	}
}

func TestOneBrowserCacheLimitKeepsExplicitOverrides(t *testing.T) {
	args := oneBrowserWindowLaunchArgs(nil, OneBrowserStartRequest{})
	found := false
	for _, arg := range args {
		if arg == "--disk-cache-size=67108864" {
			found = true
		}
	}
	if !found {
		t.Fatal("default cache limit absent")
	}
	args = oneBrowserWindowLaunchArgs([]string{"--disk-cache-size=33554432"}, OneBrowserStartRequest{})
	count := 0
	for _, arg := range args {
		if arg == "--disk-cache-size=33554432" || arg == "--disk-cache-size=67108864" {
			count++
		}
	}
	if count != 1 {
		t.Fatal("explicit cache size overridden")
	}
}

func TestDeleteWindowPersistsRemovalInSQLite(t *testing.T) {
	a, p, _ := deletableWindow(t)
	db := newProfilePackageDatabase(t, a.appRoot)
	a.browserMgr.ProfileDAO = browser.NewSQLiteProfileDAO(db.GetConn())
	if err := a.browserMgr.ProfileDAO.Upsert(p); err != nil {
		t.Fatal(err)
	}
	if err := a.OneBrowserDeleteWindow(testWindowA, p.ProfileId); err != nil {
		t.Fatal(err)
	}
	list, err := a.browserMgr.ProfileDAO.List()
	if err != nil || len(list) != 0 {
		t.Fatalf("database removal failed: %v %v", list, err)
	}
	if list := a.BrowserProfileList(); len(list) != 0 {
		t.Fatal("deleted record reloaded")
	}
}

func TestClearCachePreservesPrivateDataAndRejectsRunningWindow(t *testing.T) {
	a, p, files := deletableWindow(t)
	p.Tags = []string{"One Browser"}
	root := a.browserMgr.ResolveUserDataDir(p)
	cache := filepath.Join(root, "Default", "Cache", "Cache_Data")
	if err := os.MkdirAll(cache, 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(cache, "data"), []byte("disposable"), 0600); err != nil {
		t.Fatal(err)
	}
	p.Running = true
	if _, err := a.OneBrowserClearCache(); err == nil {
		t.Fatal("cleared live cache")
	}
	if _, err := os.Stat(cache); err != nil {
		t.Fatal("live cache changed")
	}
	p.Running = false
	count, err := a.OneBrowserClearCache()
	if err != nil || count != 1 {
		t.Fatalf("%d %v", count, err)
	}
	if _, err := os.Stat(cache); !os.IsNotExist(err) {
		t.Fatal("cache not removed")
	}
	checkDirectoryTestData(t, root, files)
}

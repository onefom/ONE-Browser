package backend

import (
	"ant-chrome/backend/internal/browser"
	"ant-chrome/backend/internal/database"
	"context"
	"path/filepath"
	"slices"
	"strings"
	"testing"
)

func pluginTestApp(t *testing.T) (*App, *BrowserProfile, *BrowserProfile) {
	a, first := directoryTestApp(t)
	db, err := database.NewDB(filepath.Join(t.TempDir(), "plugins.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })
	if err := db.Migrate(); err != nil {
		t.Fatal(err)
	}
	a.browserMgr.ExtensionDAO = browser.NewSQLiteExtensionDAO(db.GetConn())
	a.ctx = context.Background()
	for _, id := range oneBrowserBuiltinExtensions {
		if err := a.browserMgr.ExtensionDAO.Upsert(browser.Extension{ExtensionID: id, Name: id, Version: "1.0", Enabled: true, DefaultInstall: true}); err != nil {
			t.Fatal(err)
		}
	}
	second, err := a.browserMgr.Create(BrowserProfileInput{ProfileName: "second", Keywords: oneBrowserWindowKeywords(nil, testWindowB)})
	if err != nil {
		t.Fatal(err)
	}
	return a, first, second
}

func TestOneBrowserPluginDefaultsAndWindowBindingsAreIndependent(t *testing.T) {
	a, first, second := pluginTestApp(t)
	files := putDirectoryTestData(t, a.browserMgr.ResolveUserDataDir(first))
	if err := a.configureOneBrowserPlugins(first.ProfileId, nil, nil); err != nil {
		t.Fatal(err)
	}
	settings, _ := a.browserMgr.ExtensionDAO.GetProfileSettings(first.ProfileId)
	if !settings.Configured || len(settings.ExtensionIDs) != 0 {
		t.Fatal("global defaults leaked into window", settings)
	}
	first.Running = true
	if _, err := a.OneBrowserSetWindowPlugins(testWindowA, first.ProfileId, []string{"adguard"}, nil); err != nil {
		t.Fatal(err)
	}
	if _, err := a.OneBrowserSetWindowPlugins(testWindowB, second.ProfileId, []string{"google-translate"}, nil); err != nil {
		t.Fatal(err)
	}
	one, _ := a.browserMgr.ExtensionDAO.GetProfileSettings(first.ProfileId)
	two, _ := a.browserMgr.ExtensionDAO.GetProfileSettings(second.ProfileId)
	if !slices.Equal(one.ExtensionIDs, []string{oneBrowserBuiltinExtensions["adguard"]}) || !slices.Equal(two.ExtensionIDs, []string{oneBrowserBuiltinExtensions["google-translate"]}) {
		t.Fatal(one, two)
	}
	for _, id := range oneBrowserBuiltinExtensions {
		ext, _ := a.browserMgr.ExtensionDAO.Get(id)
		if ext.DefaultInstall {
			t.Fatal("builtin became a global default")
		}
	}
	checkDirectoryTestData(t, a.browserMgr.ResolveUserDataDir(first), files)
	if _, err := a.OneBrowserSetWindowPlugins(testWindowB, first.ProfileId, nil, nil); err == nil {
		t.Fatal("accepted another window's profile")
	}
	if _, err := a.OneBrowserSetWindowPlugins(testWindowA, first.ProfileId, []string{"unknown"}, nil); err == nil {
		t.Fatal("accepted unknown plugin")
	}
	if _, err := a.OneBrowserSetWindowPlugins(testWindowA, first.ProfileId, nil, nil); err != nil {
		t.Fatal(err)
	}
	one, _ = a.browserMgr.ExtensionDAO.GetProfileSettings(first.ProfileId)
	two, _ = a.browserMgr.ExtensionDAO.GetProfileSettings(second.ProfileId)
	if len(one.ExtensionIDs) != 0 || len(two.ExtensionIDs) != 1 {
		t.Fatal("unlink crossed windows")
	}
	checkDirectoryTestData(t, a.browserMgr.ResolveUserDataDir(first), files)
}

func TestPluginAssociationsKeepImportedStateAndDoNotCreateBrowserProfiles(t *testing.T) {
	a, first, _ := pluginTestApp(t)
	custom := "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
	if err := a.browserMgr.ExtensionDAO.Upsert(browser.Extension{ExtensionID: custom, Name: "Imported", Version: "1", Enabled: true}); err != nil {
		t.Fatal(err)
	}
	if err := a.configureOneBrowserPlugins(first.ProfileId, []string{"adguard"}, []string{custom}); err != nil {
		t.Fatal(err)
	}
	if err := a.configureOneBrowserPlugins(first.ProfileId, nil, nil); err != nil {
		t.Fatal(err)
	}
	settings, _ := a.browserMgr.ExtensionDAO.GetProfileSettings(first.ProfileId)
	if !slices.Equal(settings.ExtensionIDs, []string{custom}) {
		t.Fatal("lost imported association", settings)
	}
	if err := a.configureOneBrowserPlugins(first.ProfileId, nil, []string{"bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb"}); err == nil {
		t.Fatal("accepted missing package")
	}
	settings, _ = a.browserMgr.ExtensionDAO.GetProfileSettings(first.ProfileId)
	if !slices.Equal(settings.ExtensionIDs, []string{custom}) {
		t.Fatal("failed binding modified saved settings")
	}
	count := len(a.browserMgr.Profiles)
	if _, err := a.OneBrowserSetWindowPlugins("33333333-3333-4333-8333-333333333333", "", []string{"adguard"}, nil); err != nil {
		t.Fatal(err)
	}
	if len(a.browserMgr.Profiles) != count {
		t.Fatal("association started/created a browser")
	}
}

func TestDefaultOffDoesNotInitializeDownloadsAndRestartRequiresGracefulClose(t *testing.T) {
	var nilApp *App
	if warnings := nilApp.OneBrowserSyncBuiltinExtensions(nil); len(warnings) != 0 {
		t.Fatal(warnings)
	}
	a, first, _ := pluginTestApp(t)
	first.Running = true
	first.DebugPort = 0
	files := putDirectoryTestData(t, a.browserMgr.ResolveUserDataDir(first))
	if _, err := a.OneBrowserRestartWindowForPlugins(testWindowA, first.ProfileId); err == nil {
		t.Fatal("unresponsive browser was restarted")
	}
	if !first.Running {
		t.Fatal("unresponsive browser was stopped")
	}
	checkDirectoryTestData(t, a.browserMgr.ResolveUserDataDir(first), files)
}

func TestLegacyImportedPluginRemainsOnlyInItsOriginalWindow(t *testing.T) {
	a, first, second := pluginTestApp(t)
	id := "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
	if err := a.browserMgr.ExtensionDAO.Upsert(browser.Extension{ExtensionID: id, Name: "Previously installed import", Version: "1", Enabled: true, DefaultInstall: true}); err != nil {
		t.Fatal(err)
	}
	if err := a.browserMgr.ExtensionDAO.UpsertProfileExtensionRuntime(browser.ProfileExtensionRuntime{ProfileID: first.ProfileId, ExtensionID: id, RuntimeExtensionID: id, InstalledVersion: "1", Status: browser.ExtensionRuntimeStatusInstalled}); err != nil {
		t.Fatal(err)
	}
	for _, profile := range []*BrowserProfile{first, second} {
		if err := a.configureOneBrowserPlugins(profile.ProfileId, nil, nil); err != nil {
			t.Fatal(err)
		}
	}
	one, _ := a.browserMgr.ExtensionDAO.GetProfileSettings(first.ProfileId)
	two, _ := a.browserMgr.ExtensionDAO.GetProfileSettings(second.ProfileId)
	if !slices.Equal(one.ExtensionIDs, []string{id}) || len(two.ExtensionIDs) != 0 {
		t.Fatal("legacy import leaked/lost", one, two)
	}
}

func TestPluginRestartRetainsDataIsolationGuard(t *testing.T) {
	a, first, second := pluginTestApp(t)
	first.Running = true
	second.UserDataDir = first.UserDataDir
	_, err := a.OneBrowserRestartWindowForPlugins(testWindowA, first.ProfileId)
	if err == nil || !strings.Contains(err.Error(), "重复") {
		t.Fatal("restart bypassed data isolation", err)
	}
	if !first.Running {
		t.Fatal("closed browser before isolation validation")
	}
}

func TestOneBrowserImportedPluginExplicitRemoval(t *testing.T) {
	a, first, second := pluginTestApp(t)
	id := "cccccccccccccccccccccccccccccccc"
	if err := a.browserMgr.ExtensionDAO.Upsert(browser.Extension{ExtensionID: id, Name: "Import", Version: "1", Enabled: true}); err != nil {
		t.Fatal(err)
	}
	for _, p := range []*BrowserProfile{first, second} {
		if err := a.configureOneBrowserPlugins(p.ProfileId, nil, []string{id}); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := a.OneBrowserSetWindowPluginsWithRemoval(testWindowA, first.ProfileId, nil, nil, []string{id}); err != nil {
		t.Fatal(err)
	}
	one, _ := a.browserMgr.ExtensionDAO.GetProfileSettings(first.ProfileId)
	two, _ := a.browserMgr.ExtensionDAO.GetProfileSettings(second.ProfileId)
	if slices.Contains(one.ExtensionIDs, id) || !slices.Contains(two.ExtensionIDs, id) {
		t.Fatal("explicit removal must only affect selected window", one, two)
	}
	if err := a.configureOneBrowserPlugins(first.ProfileId, nil, nil); err != nil {
		t.Fatal(err)
	}
	one, _ = a.browserMgr.ExtensionDAO.GetProfileSettings(first.ProfileId)
	if slices.Contains(one.ExtensionIDs, id) {
		t.Fatal("removed import returned on next launch")
	}
	if _, err := a.OneBrowserSetWindowPluginsWithRemoval(testWindowB, second.ProfileId, nil, []string{id}, []string{id}); err == nil {
		t.Fatal("conflicting operation accepted")
	}
}

package browser

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

var sharedExtensionCodeMu sync.Mutex

func isOneBrowserOwnedProfile(profile *Profile) bool {
	for _, key := range profile.Keywords {
		if strings.HasPrefix(key, "one-browser-window:") {
			return true
		}
	}
	return false
}

// Only app-managed extension program files are shared. Each Chromium profile
// still owns its settings, cookies, service workers and extension storage.
// Published hash paths are immutable: new packages always get a different path.
func (m *Manager) ensureSharedExtensionInstalled(profile *Profile, userDir, binary string, extension Extension) (string, error) {
	sharedExtensionCodeMu.Lock()
	defer sharedExtensionCodeMu.Unlock()
	packagePath, hash, runtimeID, err := m.resolveSharedExtensionPackage(extension, binary)
	if err != nil {
		return "", err
	}
	root := strings.TrimSpace(m.Config.Browser.UserDataRoot)
	if root == "" {
		root = "data"
	}
	codePath := filepath.Join(m.ResolveRelativePath(root), "shared", "extension-code", runtimeID, hash)
	codePath, err = filepath.Abs(codePath)
	if err != nil {
		return "", err
	}
	if extensionManifestVersion(codePath) != extension.Version {
		if _, err := os.Lstat(codePath); !os.IsNotExist(err) {
			return "", fmt.Errorf("共享插件代码异常，未覆盖已发布目录：%s", codePath)
		}
		packageData, err := os.ReadFile(packagePath)
		if err != nil {
			return "", err
		}
		archive, err := normalizeExtensionArchiveData(packageData)
		if err != nil {
			return "", err
		}
		if err := os.MkdirAll(filepath.Dir(codePath), 0700); err != nil {
			return "", err
		}
		temp, err := os.MkdirTemp(filepath.Dir(codePath), ".prepare-")
		if err != nil {
			return "", err
		}
		defer os.RemoveAll(temp)
		candidate := filepath.Join(temp, "code")
		if err := replaceExtensionDirFromZip(archive, candidate); err != nil {
			return "", err
		}
		if err := writeProfileScopedExtensionManifest(filepath.Join(candidate, "manifest.json"), packageData, runtimeID); err != nil {
			return "", err
		}
		if extensionManifestVersion(candidate) != extension.Version {
			return "", fmt.Errorf("共享插件版本校验失败")
		}
		if err := os.Rename(candidate, codePath); err != nil {
			return "", err
		}
	}
	state, stateErr := m.ExtensionDAO.GetProfileExtensionRuntime(profile.ProfileId, extension.ExtensionID)
	if stateErr != nil && stateErr != sql.ErrNoRows {
		return "", stateErr
	}
	if stateErr == nil && state.Status == ExtensionRuntimeStatusInstalled && state.InstalledVersion == extension.Version && state.PackageHash == hash && state.RuntimeExtensionID == runtimeID && sharedExtensionRegistrationMatches(userDir, codePath, runtimeID, extension.Version) {
		return codePath, nil
	}
	legacy, err := findLegacyRuntimeExtensionIDs(userDir, extension.InstallDir)
	if err != nil {
		return "", err
	}
	if state.RuntimeExtensionID != "" {
		legacy = append(legacy, state.RuntimeExtensionID)
	}
	legacy = uniqueExtensionIDs(legacy)
	for _, oldID := range legacy {
		if oldID == runtimeID {
			continue
		}
		backup, err := m.backupProfileExtensionState(profile.ProfileId, extension.ExtensionID, userDir, []string{oldID})
		if err != nil {
			return "", err
		}
		if err := migrateExtensionStorage(userDir, oldID, runtimeID); err != nil {
			_ = restoreProfileExtensionState(userDir, backup, []string{oldID}, runtimeID)
			return "", err
		}
	}
	// Registration only changes this profile. Do not rewrite the shared manifest.
	if err := ensureProfileExtensionRegistrationAtPath(userDir, codePath, runtimeID, "", true); err != nil {
		return "", err
	}
	now := time.Now().Format(time.RFC3339)
	if state.CreatedAt == "" {
		state.CreatedAt = now
	}
	state.ProfileID = profile.ProfileId
	state.ExtensionID = extension.ExtensionID
	state.RuntimeExtensionID = runtimeID
	state.InstallMode = ExtensionInstallModePersistent
	state.InstalledVersion = extension.Version
	state.PackageHash = hash
	state.Status = ExtensionRuntimeStatusInstalled
	state.LastVerifiedAt = now
	state.LastError = ""
	if err := m.ExtensionDAO.UpsertProfileExtensionRuntime(state); err != nil {
		return "", err
	}
	// Registration is durable before removing obsolete copies of managed code.
	// Never remove Local Extension Settings, IndexedDB or any other user storage.
	for _, oldID := range uniqueExtensionIDs(append(legacy, runtimeID)) {
		versionPath := persistentExtensionCodePath(userDir, oldID, extension.Version)
		if versionPath == "" {
			return "", fmt.Errorf("旧插件目录无效，未清理文件")
		}
		oldCode := filepath.Dir(versionPath)
		if err := os.RemoveAll(oldCode); err != nil {
			return "", fmt.Errorf("清理旧插件代码失败：%w", err)
		}
		if oldID != runtimeID {
			if err := removeProfileScopedExtensionRegistration(userDir, oldID); err != nil {
				return "", err
			}
		}
	}
	return codePath, nil
}

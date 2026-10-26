package browser

import (
	"crypto/sha256"
	"fmt"
	"path/filepath"
	"sort"
	"strings"
)

// Commit the marker only after every cleanup succeeds; a failed migration is
// retried next time. Normal bookmarks and browser user storage are unaffected.
func MigrateOneBrowserLegacyBookmarks(userDir, targetURL string) error {
	path := filepath.Join(userDir, ".onebrowser-startup-migrations.json")
	marker, err := readProfileJSON(path, true)
	if err != nil {
		return err
	}
	if marker["legacy_bookmarks"] == float64(1) {
		return nil
	}
	if _, err := RemoveBookmarkURL(userDir, targetURL); err != nil {
		return err
	}
	for _, name := range []string{"指纹检测", "Ant 指纹检测"} {
		if _, err := RemoveBookmarkName(userDir, name); err != nil {
			return err
		}
	}
	if marker == nil {
		marker = profileExtensionJSON{}
	}
	marker["legacy_bookmarks"] = 1
	return writeProfileJSON(path, marker)
}

func legacyExtensionCatalogSignature(extensions []Extension) string {
	items := make([]string, 0, len(extensions))
	for _, extension := range extensions {
		items = append(items, extension.ExtensionID+"\x00"+extension.InstallDir)
	}
	sort.Strings(items)
	return fmt.Sprintf("%x", sha256.Sum256([]byte(strings.Join(items, "\n"))))
}

func legacyExtensionMigrationDone(userDir string, extensions []Extension) bool {
	marker, err := readProfileJSON(filepath.Join(userDir, ".onebrowser-extension-migration.json"), false)
	return err == nil && marker["version"] == float64(1) && marker["catalog"] == legacyExtensionCatalogSignature(extensions)
}

func finishLegacyExtensionMigration(userDir string, extensions []Extension) error {
	return writeProfileJSON(filepath.Join(userDir, ".onebrowser-extension-migration.json"), profileExtensionJSON{"version": 1, "catalog": legacyExtensionCatalogSignature(extensions)})
}

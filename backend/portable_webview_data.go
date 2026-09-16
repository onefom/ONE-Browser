package backend

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
)

const portableWebviewMigrationMarker = ".webview2-portable-migration-v1"

type PortableWebviewMigrationResult struct {
	Target   string
	Source   string
	Migrated bool
}

// PortableWebviewDataPath keeps Wails/WebView2 localStorage and UI state next
// to the portable executable instead of under the Windows user profile.
func PortableWebviewDataPath(appRoot string) string {
	return ResolveRuntimePath(appRoot, filepath.Join("data", "webview2"))
}

// PreparePortableWebviewData creates the portable WebView2 directory and, on
// Windows, imports the legacy Wails default directory once. The legacy data is
// deliberately left untouched as a recovery copy.
func PreparePortableWebviewData(appRoot string) (PortableWebviewMigrationResult, error) {
	target := PortableWebviewDataPath(appRoot)
	result := PortableWebviewMigrationResult{Target: target}
	if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
		return result, fmt.Errorf("create portable data root: %w", err)
	}
	if runtime.GOOS != "windows" {
		if err := os.MkdirAll(target, 0o755); err != nil {
			return result, fmt.Errorf("create portable WebView2 directory: %w", err)
		}
		return result, nil
	}

	configDir, err := os.UserConfigDir()
	if err != nil {
		return result, fmt.Errorf("resolve Windows AppData directory: %w", err)
	}
	executableName := "OneBrowser.exe"
	if executablePath, executableErr := os.Executable(); executableErr == nil {
		if base := strings.TrimSpace(filepath.Base(executablePath)); base != "" {
			executableName = base
		}
	}
	return preparePortableWebviewDataFromCandidates(target, legacyWebviewCandidates(configDir, executableName))
}

func legacyWebviewCandidates(configDir, executableName string) []string {
	names := []string{executableName, "OneBrowser.exe", "one-browser.exe", "One Browser.exe"}
	seen := map[string]struct{}{}
	result := make([]string, 0, len(names))
	for _, name := range names {
		name = strings.TrimSpace(name)
		if name == "" {
			continue
		}
		key := strings.ToLower(name)
		if _, exists := seen[key]; exists {
			continue
		}
		seen[key] = struct{}{}
		result = append(result, filepath.Join(configDir, name))
	}
	return result
}

func preparePortableWebviewDataFromCandidates(target string, candidates []string) (PortableWebviewMigrationResult, error) {
	result := PortableWebviewMigrationResult{Target: target}
	dataRoot := filepath.Dir(target)
	marker := filepath.Join(dataRoot, portableWebviewMigrationMarker)
	if _, err := os.Stat(marker); err == nil {
		return result, os.MkdirAll(target, 0o755)
	}

	nonEmpty, err := directoryHasEntries(target)
	if err != nil {
		return result, err
	}
	if nonEmpty {
		return result, writePortableWebviewMigrationMarker(marker, "portable directory already contains data")
	}
	// Release packages contain data/README.md. Treat that file as an explicit
	// fresh-portable marker so an empty package never resurrects accounts and
	// windows from a legacy AppData directory without the user's consent.
	if _, readmeErr := os.Stat(filepath.Join(dataRoot, "README.md")); readmeErr == nil {
		if err := os.MkdirAll(target, 0o755); err != nil {
			return result, err
		}
		return result, writePortableWebviewMigrationMarker(marker, "fresh portable package; legacy import skipped")
	}

	for _, source := range candidates {
		if samePath(source, target) {
			continue
		}
		hasEntries, sourceErr := directoryHasEntries(source)
		if sourceErr != nil || !hasEntries {
			continue
		}
		staging := target + ".migrating"
		_ = os.RemoveAll(staging)
		if err := copyPortableDirectory(source, staging); err != nil {
			_ = os.RemoveAll(staging)
			return result, fmt.Errorf("migrate legacy WebView2 data from %s: %w", source, err)
		}
		// An empty directory may already have been created by an earlier startup.
		// os.Remove only succeeds for an empty directory, so this cannot discard
		// portable user data that appeared after the non-empty check above.
		if err := os.Remove(target); err != nil && !os.IsNotExist(err) {
			_ = os.RemoveAll(staging)
			return result, fmt.Errorf("replace empty portable WebView2 directory: %w", err)
		}
		if err := os.Rename(staging, target); err != nil {
			_ = os.RemoveAll(staging)
			return result, fmt.Errorf("activate portable WebView2 data: %w", err)
		}
		result.Source = source
		result.Migrated = true
		if err := writePortableWebviewMigrationMarker(marker, source); err != nil {
			return result, err
		}
		return result, nil
	}

	if err := os.MkdirAll(target, 0o755); err != nil {
		return result, err
	}
	return result, writePortableWebviewMigrationMarker(marker, "no legacy data found")
}

func directoryHasEntries(path string) (bool, error) {
	entries, err := os.ReadDir(path)
	if os.IsNotExist(err) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return len(entries) > 0, nil
}

func samePath(first, second string) bool {
	firstAbs, firstErr := filepath.Abs(first)
	secondAbs, secondErr := filepath.Abs(second)
	return firstErr == nil && secondErr == nil && strings.EqualFold(filepath.Clean(firstAbs), filepath.Clean(secondAbs))
}

func copyPortableDirectory(source, target string) error {
	if err := os.MkdirAll(target, 0o755); err != nil {
		return err
	}
	entries, err := os.ReadDir(source)
	if err != nil {
		return err
	}
	sort.Slice(entries, func(i, j int) bool { return entries[i].Name() < entries[j].Name() })
	for _, entry := range entries {
		if entry.Type()&os.ModeSymlink != 0 {
			continue
		}
		sourcePath := filepath.Join(source, entry.Name())
		targetPath := filepath.Join(target, entry.Name())
		if entry.IsDir() {
			if err := copyPortableDirectory(sourcePath, targetPath); err != nil {
				return err
			}
			continue
		}
		info, err := entry.Info()
		if err != nil {
			return err
		}
		if !info.Mode().IsRegular() {
			continue
		}
		if err := copyPortableFile(sourcePath, targetPath, info.Mode().Perm()); err != nil {
			return err
		}
	}
	return nil
}

func copyPortableFile(source, target string, mode os.FileMode) error {
	input, err := os.Open(source)
	if err != nil {
		return err
	}
	defer input.Close()
	output, err := os.OpenFile(target, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, mode)
	if err != nil {
		return err
	}
	if _, err := io.Copy(output, input); err != nil {
		_ = output.Close()
		return err
	}
	return output.Close()
}

func writePortableWebviewMigrationMarker(path, source string) error {
	return os.WriteFile(path, []byte(strings.TrimSpace(source)+"\n"), 0o600)
}

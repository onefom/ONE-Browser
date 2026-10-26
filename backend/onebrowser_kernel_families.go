package backend

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"ant-chrome/backend/internal/browser"
)

const firefoxKernelReleasesURL = "https://github.com/daijro/camoufox/releases"

// Check essential runtime files without launching the browser or hashing gigabytes.
// This is a structural check, not a publisher signature or content checksum check.
func oneBrowserCoreIntegrity(root string) string {
	exe, _, ok := browser.FindCoreExecutable(root)
	if !ok {
		return "incomplete"
	}
	return oneBrowserCoreFilesIntegrity(filepath.Dir(exe), strings.EqualFold(filepath.Base(exe), "camoufox.exe"))
}
func oneBrowserCoreFilesIntegrity(dir string, firefox bool) string {
	required := []string{"chrome.exe", "chrome.dll", "icudtl.dat", "resources.pak"}
	if firefox {
		required = []string{"camoufox.exe", "xul.dll", "mozglue.dll", "omni.ja", "application.ini"}
	}
	for _, name := range required {
		info, err := os.Stat(filepath.Join(dir, name))
		if err != nil || !info.Mode().IsRegular() || info.Size() == 0 {
			return "incomplete"
		}
	}
	return "complete"
}
func (a *App) OneBrowserFamilyKernelStatus(family string) OneBrowserKernelStatus {
	best := OneBrowserKernelStatus{}
	for _, item := range a.OneBrowserKernels() {
		if (item.Family == "camoufox") != (family == "camoufox") {
			continue
		}
		if !best.Installed || oneBrowserKernelNeedsUpdate(best.Version, item.Version) {
			best = item
		}
	}
	return best
}
func (a *App) OneBrowserCheckFamilyKernelUpdate(family string) (OneBrowserKernelRelease, error) {
	if family == "chrome" {
		return a.OneBrowserCheckKernelUpdate()
	}
	if family != "camoufox" {
		return OneBrowserKernelRelease{}, fmt.Errorf("不支持的内核类型")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	var lastErr error
	for _, client := range oneBrowserReleaseClients(12 * time.Second) {
		release, err := fetchFirefoxKernelRelease(ctx, client, "https://api.github.com/repos/daijro/camoufox/releases?per_page=30")
		if err == nil {
			return release, nil
		}
		lastErr = err
	}
	return OneBrowserKernelRelease{}, lastErr
}

// The list endpoint includes the upstream beta channel; /latest omits prereleases.
func fetchFirefoxKernelRelease(ctx context.Context, client *http.Client, endpoint string) (OneBrowserKernelRelease, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return OneBrowserKernelRelease{}, err
	}
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("User-Agent", "OneBrowser/2.0.7")
	resp, err := client.Do(req)
	if err != nil {
		return OneBrowserKernelRelease{}, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return OneBrowserKernelRelease{}, fmt.Errorf("Firefox 发布源返回 HTTP %d", resp.StatusCode)
	}
	var releases []struct {
		TagName string               `json:"tag_name"`
		Draft   bool                 `json:"draft"`
		Assets  []githubReleaseAsset `json:"assets"`
	}
	if err = json.NewDecoder(io.LimitReader(resp.Body, 8<<20)).Decode(&releases); err != nil {
		return OneBrowserKernelRelease{}, err
	}
	var best OneBrowserKernelRelease
	for _, release := range releases {
		if release.Draft {
			continue
		}
		version := strings.TrimPrefix(strings.TrimSpace(release.TagName), "v")
		if !regexp.MustCompile(`^\d+(?:\.\d+)*(?:-beta\.\d+)?$`).MatchString(version) {
			continue
		}
		for _, asset := range release.Assets {
			name := strings.ToLower(asset.Name)
			if name != "camoufox-"+strings.ToLower(version)+"-win.x86_64.zip" || !strings.HasPrefix(asset.BrowserDownloadURL, firefoxKernelReleasesURL+"/download/") {
				continue
			}
			if best.Version == "" || oneBrowserKernelNeedsUpdate(best.Version, version) {
				best = OneBrowserKernelRelease{Version: version, DownloadURL: asset.BrowserDownloadURL, ReleasesURL: firefoxKernelReleasesURL, AssetName: asset.Name}
			}
		}
	}
	if best.Version == "" {
		return best, fmt.Errorf("Firefox 发布源未提供 Windows x64 内核包")
	}
	return best, nil
}
func (a *App) OneBrowserDownloadFamilyKernel(family string) error {
	release, err := a.OneBrowserCheckFamilyKernelUpdate(family)
	if err != nil {
		return err
	}
	status := a.OneBrowserFamilyKernelStatus(family)
	if status.Installed && !oneBrowserKernelNeedsUpdate(status.Version, release.Version) {
		a.emitRuntimeEvent("download:progress", map[string]interface{}{"phase": "done", "progress": 100, "message": "当前内核已是最新版本，无需重新下载"})
		return nil
	}
	prefix := "fingerprint-chromium-"
	if family == "camoufox" {
		prefix = "camoufox-"
	}
	return a.BrowserCoreDownload(prefix+release.Version, release.DownloadURL, "__system__")
}

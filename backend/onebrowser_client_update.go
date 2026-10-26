package backend

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strings"
	"time"

	wailsruntime "github.com/wailsapp/wails/v2/pkg/runtime"
)

const oneBrowserClientReleasesURL = "https://github.com/onefom/ONE-Browser/releases"
const oneBrowserClientReleaseAPI = "https://api.github.com/repos/onefom/ONE-Browser/releases/latest"

type OneBrowserClientRelease struct {
	CurrentVersion  string `json:"currentVersion"`
	Version         string `json:"version"`
	DownloadURL     string `json:"downloadUrl"`
	ReleasesURL     string `json:"releasesUrl"`
	UpdateAvailable bool   `json:"updateAvailable"`
	Message         string `json:"message"`
}

func (a *App) OneBrowserClientVersion() string {
	if v := strings.TrimSpace(a.version); v != "" && v != "unknown" {
		return v
	}
	return "2.0.7"
}
func (a *App) OneBrowserCheckClientUpdate() (OneBrowserClientRelease, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	var lastErr error
	for _, client := range oneBrowserReleaseClients(12 * time.Second) {
		release, err := fetchOneBrowserClientRelease(ctx, client, oneBrowserClientReleaseAPI, a.OneBrowserClientVersion())
		if err == nil {
			return release, nil
		}
		lastErr = err
	}
	return OneBrowserClientRelease{}, lastErr
}
func fetchOneBrowserClientRelease(ctx context.Context, client *http.Client, endpoint, current string) (OneBrowserClientRelease, error) {
	result := OneBrowserClientRelease{CurrentVersion: current, ReleasesURL: oneBrowserClientReleasesURL}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return result, err
	}
	req.Header.Set("User-Agent", "OneBrowser/"+current)
	req.Header.Set("Accept", "application/vnd.github+json")
	resp, err := client.Do(req)
	if err != nil {
		return result, err
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusNotFound {
		result.Message = "尚未发布版本"
		return result, nil
	}
	if resp.StatusCode != http.StatusOK {
		return result, fmt.Errorf("客户端发布源返回 HTTP %d", resp.StatusCode)
	}
	var release struct {
		TagName    string               `json:"tag_name"`
		Draft      bool                 `json:"draft"`
		Prerelease bool                 `json:"prerelease"`
		Assets     []githubReleaseAsset `json:"assets"`
	}
	if err = json.NewDecoder(io.LimitReader(resp.Body, 2<<20)).Decode(&release); err != nil {
		return result, err
	}
	version := strings.TrimPrefix(strings.TrimSpace(release.TagName), "v")
	valid := regexp.MustCompile(`^\d+\.\d+\.\d+$`)
	if release.Draft || release.Prerelease || !valid.MatchString(version) {
		return result, fmt.Errorf("客户端发布版本无效")
	}
	result.Version = version
	for _, asset := range release.Assets {
		if strings.EqualFold(asset.Name, "OneBrowser-"+version+"-windows-x64-portable.zip") && strings.HasPrefix(asset.BrowserDownloadURL, oneBrowserClientReleasesURL+"/download/") {
			result.DownloadURL = asset.BrowserDownloadURL
			break
		}
	}
	newer := oneBrowserKernelNeedsUpdate(current, version)
	result.UpdateAvailable = newer && result.DownloadURL != ""
	switch {
	case result.UpdateAvailable:
		result.Message = "发现客户端新版"
	case newer:
		result.Message = "新版完整版尚未发布"
	case version == strings.TrimPrefix(current, "v"):
		result.Message = "当前已是最新版本"
	default:
		result.Message = "本地版本较新"
	}
	return result, nil
}

// The portable package is downloaded through the system browser. Running files
// and independent browser data are never overwritten by an unattended update.
func (a *App) OneBrowserDownloadClientUpdate() error {
	release, err := a.OneBrowserCheckClientUpdate()
	if err != nil {
		return err
	}
	if !release.UpdateAvailable {
		return fmt.Errorf("没有可下载的客户端新版")
	}
	wailsruntime.BrowserOpenURL(a.ctx, release.DownloadURL)
	return nil
}

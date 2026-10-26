package backend

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
)

func TestClientReleaseVersionsAndPortableAsset(t *testing.T) {
	for _, tc := range []struct {
		version, asset, url, message string
		update                       bool
	}{
		{"1.8.25", "OneBrowser-1.8.25-windows-x64-portable.zip", "https://github.com/onefom/ONE-Browser/releases/download/v1.8.25/full.zip", "本地版本较新", false},
		{"2.0.5", "OneBrowser-2.0.5-windows-x64-portable.zip", "https://github.com/onefom/ONE-Browser/releases/download/v2.0.5/full.zip", "当前已是最新版本", false},
		{"2.0.6", "OneBrowser-2.0.6-windows-x64-portable.zip", "https://github.com/onefom/ONE-Browser/releases/download/v2.0.6/full.zip", "发现客户端新版", true},
		{"2.0.6", "OneBrowser-2.0.6-source.zip", "https://github.com/onefom/ONE-Browser/releases/download/v2.0.6/source.zip", "新版完整版尚未发布", false},
		{"2.0.6", "OneBrowser-2.0.6-windows-x64-portable.zip", "https://example.com/foreign.zip", "新版完整版尚未发布", false},
	} {
		t.Run(tc.version+tc.asset+tc.url, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				fmt.Fprintf(w, `{"tag_name":"v%s","assets":[{"name":"%s","browser_download_url":"%s"}]}`, tc.version, tc.asset, tc.url)
			}))
			defer server.Close()
			release, err := fetchOneBrowserClientRelease(context.Background(), server.Client(), server.URL, "2.0.5")
			if err != nil || release.UpdateAvailable != tc.update || release.Message != tc.message {
				t.Fatalf("%+v %v", release, err)
			}
		})
	}
}
func TestClientReleaseMissingAndNetworkFailure(t *testing.T) {
	for _, status := range []int{404, 403, 500} {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(status) }))
		release, err := fetchOneBrowserClientRelease(context.Background(), server.Client(), server.URL, "2.0.5")
		server.Close()
		if status == 404 {
			if err != nil || release.Message != "尚未发布版本" || release.UpdateAvailable {
				t.Fatal(release, err)
			}
		} else if err == nil {
			t.Fatal("network failure reported as latest", status)
		}
	}
}

func TestOneBrowserOnlineReleaseSources(t *testing.T) {
	if os.Getenv("ONEBROWSER_ONLINE_RELEASE_TEST") != "1" {
		t.Skip("explicit online check only")
	}
	a := &App{version: "2.0.5"}
	for _, family := range []string{"chrome", "camoufox"} {
		r, err := a.OneBrowserCheckFamilyKernelUpdate(family)
		if err != nil || r.Version == "" || r.DownloadURL == "" {
			t.Fatalf("%s: %+v %v", family, r, err)
		}
		t.Log(family, r.Version)
	}
	r, err := a.OneBrowserCheckClientUpdate()
	if err != nil || r.CurrentVersion != "2.0.5" {
		t.Fatal(r, err)
	}
	t.Log("client", r.Version, r.Message)
}

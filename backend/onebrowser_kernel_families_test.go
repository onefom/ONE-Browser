package backend

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
)

func TestFirefoxKernelReleaseSelection(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, `[
 {"tag_name":"v158.0-beta.39","assets":[{"name":"camoufox-158.0-beta.39-win.x86_64.zip","browser_download_url":"https://github.com/daijro/camoufox/releases/download/v158.0-beta.39/test.zip"}]},
 {"tag_name":"v158.0-beta.41","assets":[{"name":"camoufox-158.0-beta.41-win.i686.zip","browser_download_url":"https://github.com/daijro/camoufox/releases/download/v158.0-beta.41/32.zip"}]},
 {"tag_name":"v159.0-beta.1","draft":true,"assets":[{"name":"camoufox-159.0-beta.1-win.x86_64.zip","browser_download_url":"https://github.com/daijro/camoufox/releases/download/v159.0-beta.1/test.zip"}]},
 {"tag_name":"font-bundle-v1","assets":[{"name":"camoufox-font-bundle-v1-win.x86_64.zip","browser_download_url":"https://github.com/daijro/camoufox/releases/download/font-bundle-v1/fonts.zip"}]},
 {"tag_name":"v158.0-beta.40","assets":[{"name":"camoufox-158.0-beta.40-win.x86_64.zip","browser_download_url":"https://github.com/daijro/camoufox/releases/download/v158.0-beta.40/test.zip"}]}
 , {"tag_name":"font-bundle-v2","assets":[{"name":"camoufox-font-bundle-v2-win.x86_64.zip","browser_download_url":"https://github.com/daijro/camoufox/releases/download/font-bundle-v2/fonts.zip"}]}
 ]`)
	}))
	defer server.Close()
	release, err := fetchFirefoxKernelRelease(context.Background(), server.Client(), server.URL)
	if err != nil || release.Version != "158.0-beta.40" {
		t.Fatalf("%+v %v", release, err)
	}
	for _, tc := range []struct {
		old, new string
		want     bool
	}{{"158.0-beta.40", "158.0-beta.39", false}, {"158.0-beta.40", "158.0-beta.41", true}, {"158.0-beta.40", "158.0", true}, {"158.0", "158.0-beta.41", false}, {"159.0-beta.1", "158.0-beta.41", false}} {
		if got := oneBrowserKernelNeedsUpdate(tc.old, tc.new); got != tc.want {
			t.Fatalf("%s -> %s = %v", tc.old, tc.new, got)
		}
	}
}
func TestKernelEssentialFilesIntegrity(t *testing.T) {
	for _, firefox := range []bool{false, true} {
		dir := t.TempDir()
		names := []string{"chrome.exe", "chrome.dll", "icudtl.dat", "resources.pak"}
		if firefox {
			names = []string{"camoufox.exe", "xul.dll", "mozglue.dll", "omni.ja", "application.ini"}
		}
		for _, name := range names {
			if err := os.WriteFile(filepath.Join(dir, name), []byte("fixture"), 0600); err != nil {
				t.Fatal(err)
			}
		}
		if oneBrowserCoreFilesIntegrity(dir, firefox) != "complete" {
			t.Fatal("complete runtime rejected")
		}
		os.WriteFile(filepath.Join(dir, names[1]), nil, 0600)
		if oneBrowserCoreFilesIntegrity(dir, firefox) != "incomplete" {
			t.Fatal("empty runtime accepted")
		}
		os.Remove(filepath.Join(dir, names[1]))
		if oneBrowserCoreFilesIntegrity(dir, firefox) != "incomplete" {
			t.Fatal("missing runtime accepted")
		}
	}
}

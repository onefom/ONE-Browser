package backend

import "testing"

func TestFingerprintChromiumWindowsX64AssetSelectsPortableZip(t *testing.T) {
	release := githubRelease{TagName: "150.0.7871.186", Assets: []githubReleaseAsset{
		{Name: "ungoogled-chromium_150.0.7871.186-1.1_installer_x64.exe", BrowserDownloadURL: "https://example.invalid/installer.exe"},
		{Name: "ungoogled-chromium-150.0.7871.186-1-x86_64_linux.tar.xz", BrowserDownloadURL: "https://example.invalid/linux.tar.xz"},
		{Name: "ungoogled-chromium_150.0.7871.186-1.1_windows_x64.zip", BrowserDownloadURL: "https://example.invalid/windows.zip"},
	}}

	asset, ok := fingerprintChromiumWindowsX64Asset(release)
	if !ok {
		t.Fatal("expected Windows x64 ZIP asset")
	}
	if asset.BrowserDownloadURL != "https://example.invalid/windows.zip" {
		t.Fatalf("selected unexpected asset: %s", asset.BrowserDownloadURL)
	}
}

func TestFallbackFingerprintChromiumReleaseTracksVersion150(t *testing.T) {
	release := fallbackFingerprintChromiumRelease()
	if release.Version != "150.0.7871.186" {
		t.Fatalf("unexpected fallback version: %s", release.Version)
	}
	if release.AssetName != "ungoogled-chromium_150.0.7871.186-1.1_windows_x64.zip" {
		t.Fatalf("unexpected fallback asset: %s", release.AssetName)
	}
}

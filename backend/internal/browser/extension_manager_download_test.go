package browser

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestDownloadChromeExtensionCRXOnceRejectsNoContent(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}))
	defer server.Close()

	_, err := downloadChromeExtensionCRXOnce(context.Background(), server.Client(), server.URL)
	if err == nil || !strings.Contains(err.Error(), "空插件包") {
		t.Fatalf("error = %v, want empty package error", err)
	}
}

func TestDownloadChromeExtensionCRXOnceRejectsEmptyOK(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	_, err := downloadChromeExtensionCRXOnce(context.Background(), server.Client(), server.URL)
	if err == nil || !strings.Contains(err.Error(), "空插件包") {
		t.Fatalf("error = %v, want empty package error", err)
	}
}

func TestBuildChromeExtensionDownloadURLsUseCurrentChromium(t *testing.T) {
	urls := buildChromeExtensionDownloadURLs("bgnkhhnnamicmpeenaelnjfhikgbkllg")
	if len(urls) < 3 {
		t.Fatalf("got %d fallback URLs, want at least 3", len(urls))
	}
	for _, item := range urls {
		if !strings.Contains(item, "prodversion=148.0.7778.215") {
			t.Fatalf("download URL does not use current Chromium version: %s", item)
		}
	}
}

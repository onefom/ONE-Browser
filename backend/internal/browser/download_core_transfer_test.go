package browser

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"strconv"
	"strings"
	"sync"
	"testing"
)

func TestConcurrentDownloadResumesAfterUnexpectedEOF(t *testing.T) {
	payload := []byte(strings.Repeat("OneBrowser-resume-test-", 600000))
	var mu sync.Mutex
	failed := map[string]bool{}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		rangeHeader := r.Header.Get("Range")
		if rangeHeader == "" {
			w.Header().Set("Content-Length", strconv.Itoa(len(payload)))
			_, _ = w.Write(payload)
			return
		}
		var start, end int
		if _, err := fmt.Sscanf(rangeHeader, "bytes=%d-%d", &start, &end); err != nil {
			if _, err = fmt.Sscanf(rangeHeader, "bytes=%d-", &start); err != nil {
				http.Error(w, "bad range", http.StatusBadRequest)
				return
			}
			end = len(payload) - 1
		}
		if end >= len(payload) {
			end = len(payload) - 1
		}
		w.Header().Set("Accept-Ranges", "bytes")
		w.Header().Set("Content-Range", fmt.Sprintf("bytes %d-%d/%d", start, end, len(payload)))
		w.Header().Set("Content-Length", strconv.Itoa(end-start+1))
		w.WriteHeader(http.StatusPartialContent)
		if start == 0 && end == 0 {
			_, _ = w.Write(payload[:1])
			return
		}
		key := rangeHeader
		mu.Lock()
		shouldFail := !failed[key]
		failed[key] = true
		mu.Unlock()
		if shouldFail {
			cut := start + (end-start+1)/2
			_, _ = w.Write(payload[start:cut])
			return
		}
		_, _ = w.Write(payload[start : end+1])
	}))
	defer server.Close()

	file, err := os.CreateTemp(t.TempDir(), "core-*.zip")
	if err != nil {
		t.Fatal(err)
	}
	defer file.Close()
	if err = doConcurrentDownload(context.Background(), server.Client(), server.URL, file, func(string, int, string) {}); err != nil {
		t.Fatalf("download failed: %v", err)
	}
	got, err := os.ReadFile(file.Name())
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != string(payload) {
		t.Fatalf("download mismatch: got %d bytes, want %d", len(got), len(payload))
	}
}

package backend

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func TestOneBrowserKernelNeedsUpdate(t *testing.T) {
	for _, tc := range []struct {
		current, latest string
		want            bool
	}{
		{"150.0.7871.186", "150.0.7871.186", false},
		{"151.0.1.1", "150.0.7871.186", false},
		{"v150.0.7871.186", "150.0.7871.186", false},
		{"149.0.9000.9", "150.0.1.1", true},
		{"150.0.1.2", "150.0.1.10", true},
		{"本地版本", "150.0.1.1", true},
	} {
		if got := oneBrowserKernelNeedsUpdate(tc.current, tc.latest); got != tc.want {
			t.Errorf("%s -> %s: %v, want %v", tc.current, tc.latest, got, tc.want)
		}
	}
}

func TestOneBrowserSavedSessionAndPlacement(t *testing.T) {
	root := t.TempDir()
	if oneBrowserHasSavedSession(root) {
		t.Fatal("new profile has saved session")
	}
	dir := filepath.Join(root, "Default", "Sessions")
	if err := os.MkdirAll(dir, 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "Session_123"), nil, 0600); err != nil {
		t.Fatal(err)
	}
	if oneBrowserHasSavedSession(root) {
		t.Fatal("empty session must not suppress initial pages")
	}
	if err := os.WriteFile(filepath.Join(dir, "Session_123"), []byte("SNSS-session"), 0600); err != nil {
		t.Fatal(err)
	}
	if !oneBrowserHasSavedSession(root) {
		t.Fatal("existing session not detected")
	}
	args := []string{"--window-size=900,680", "--window-position=12,12", "--user-agent=test", "--fingerprint=123", "--no-first-run"}
	want := []string{"--user-agent=test", "--fingerprint=123", "--no-first-run"}
	if got := oneBrowserSessionLaunchArgs(args); !reflect.DeepEqual(got, want) {
		t.Fatalf("args = %v", got)
	}
}

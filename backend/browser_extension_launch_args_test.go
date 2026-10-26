package backend

import (
	"slices"
	"strings"
	"testing"
)

func TestBaseBrowserLaunchArgsHasNoUnverifiedExtensionPaths(t *testing.T) {
	args := buildBrowserLaunchArgs("profile-dir", 9222, "direct://", nil, nil, nil, nil, false)
	if slices.Contains(args, "--load-extension") || slices.Contains(args, "--disable-extensions-except") {
		t.Fatalf("args = %#v, base arguments must not contain unverified extension paths", args)
	}
	if !slices.Contains(args, "--new-window") {
		t.Fatalf("args = %#v, every profile must open an independent browser window", args)
	}
}

func TestManagedExtensionLoadPreservesUserExtensions(t *testing.T) {
	args := appendBrowserExtensionLaunchArgs([]string{"--new-window"}, []string{t.TempDir(), t.TempDir()})
	loads := 0
	for _, arg := range args {
		if strings.HasPrefix(arg, "--load-extension=") {
			loads++
			if !strings.Contains(arg, ",") {
				t.Fatal("extensions not combined")
			}
		}
		if strings.HasPrefix(arg, "--disable-extensions-except") {
			t.Fatal("must preserve manually installed extensions")
		}
	}
	if loads != 1 {
		t.Fatalf("load flags=%d", loads)
	}
}

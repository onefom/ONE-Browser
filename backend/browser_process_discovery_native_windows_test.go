//go:build windows

package backend

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestNativeBrowserProcessDiscoveryHelper(t *testing.T) {
	if os.Getenv("ONEBROWSER_NATIVE_PROCESS_TEST") != "1" {
		return
	}
	fmt.Println("ready")
	_ = bufio.NewScanner(os.Stdin).Scan()
	os.Exit(0)
}

func TestNativeBrowserProcessDiscoveryMatchesExactQuotedDirectory(t *testing.T) {
	root := filepath.Join(t.TempDir(), "窗口 data")
	cmd := exec.Command(os.Args[0], "-test.run=^TestNativeBrowserProcessDiscoveryHelper$", "--", "--user-data-dir="+root, "--remote-debugging-port=9237")
	cmd.Env = append(os.Environ(), "ONEBROWSER_NATIVE_PROCESS_TEST=1")
	stdin, err := cmd.StdinPipe()
	if err != nil {
		t.Fatal(err)
	}
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		t.Fatal(err)
	}
	if err := cmd.Start(); err != nil {
		t.Fatal(err)
	}
	defer func() { stdin.Close(); _ = cmd.Process.Kill(); _ = cmd.Wait() }()
	scanner := bufio.NewScanner(stdout)
	if !scanner.Scan() || scanner.Text() != "ready" {
		t.Fatal("helper not ready")
	}
	command, err := nativeProcessCommandLine(uint32(cmd.Process.Pid))
	if err != nil || !strings.Contains(command, "--remote-debugging-port=9237") {
		t.Fatalf("native command line: %v", err)
	}
	matches, err := findBrowserUserDataProcessesNative(root)
	if err != nil {
		t.Fatal(err)
	}
	found := false
	for _, match := range matches {
		if match.PID == cmd.Process.Pid && match.DebugPort == 9237 {
			found = true
		}
	}
	if !found {
		t.Fatal("native scan did not find exact directory")
	}
	matches, err = findBrowserUserDataProcessesNative(root + "-other")
	if err != nil {
		t.Fatal(err)
	}
	for _, match := range matches {
		if match.PID == cmd.Process.Pid {
			t.Fatal("prefix directory matched")
		}
	}
}

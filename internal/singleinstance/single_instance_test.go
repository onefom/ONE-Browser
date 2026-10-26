package singleinstance

import (
	"os"
	"os/exec"
	"testing"
	"time"
)

// Run the second launch in another process: Windows byte-range locks reject
// overlapping reads in a way an in-process mock does not reproduce.
func TestSecondProcessActivatesPrimary(t *testing.T) {
	if root := os.Getenv("ONE_BROWSER_TEST_SECOND_LAUNCH"); root != "" {
		guard, primary, err := Acquire(root)
		if guard != nil {
			guard.Close()
		}
		if err != nil || primary {
			t.Fatalf("second launch primary=%v error=%v", primary, err)
		}
		return
	}
	// Isolate the Windows per-user state directory from installed applications.
	t.Setenv("LOCALAPPDATA", t.TempDir())
	root := t.TempDir()
	guard, primary, err := Acquire(root)
	if err != nil || !primary {
		t.Fatalf("acquire primary: %v", err)
	}
	defer guard.Close()
	exe, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	child := exec.Command(exe, "-test.run=^TestSecondProcessActivatesPrimary$")
	child.Env = append(os.Environ(), "ONE_BROWSER_TEST_SECOND_LAUNCH="+root)
	done := make(chan error, 1)
	go func() {
		data, err := child.CombinedOutput()
		if err != nil {
			t.Logf("child: %s", data)
		}
		done <- err
	}()
	select {
	case request := <-guard.Activations():
		request.Complete()
	case err := <-done:
		t.Fatalf("child exited without activation: %v", err)
	case <-time.After(8 * time.Second):
		_ = child.Process.Kill()
		t.Fatal("activation timed out")
	}
	select {
	case err := <-done:
		if err != nil {
			t.Fatal(err)
		}
	case <-time.After(4 * time.Second):
		_ = child.Process.Kill()
		t.Fatal("second launch did not exit")
	}
}

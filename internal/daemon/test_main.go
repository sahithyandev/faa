package daemon

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
)

func TestMain(m *testing.M) {
	tmpDir, err := os.MkdirTemp("", "faa-hosts-")
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to create temp hosts dir: %v\n", err)
		os.Exit(1)
	}
	hostsPath := filepath.Join(tmpDir, "hosts")
	if err := os.WriteFile(hostsPath, []byte("127.0.0.1 localhost\n"), 0644); err != nil {
		fmt.Fprintf(os.Stderr, "failed to write temp hosts file: %v\n", err)
		os.Exit(1)
	}
	if err := os.Setenv("FAA_HOSTS_PATH", hostsPath); err != nil {
		fmt.Fprintf(os.Stderr, "failed to set FAA_HOSTS_PATH: %v\n", err)
		os.Exit(1)
	}

	code := 1
	func() {
		defer func() {
			_ = os.RemoveAll(tmpDir)
		}()
		code = m.Run()
	}()
	os.Exit(code)
}

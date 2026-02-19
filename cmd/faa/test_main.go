package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestMain(m *testing.M) {
	tmpDir, err := os.MkdirTemp("", "faa-hosts-")
	if err != nil {
		os.Exit(1)
	}
	hostsPath := filepath.Join(tmpDir, "hosts")
	_ = os.WriteFile(hostsPath, []byte("127.0.0.1 localhost\n"), 0644)
	_ = os.Setenv("FAA_HOSTS_PATH", hostsPath)

	code := m.Run()
	_ = os.RemoveAll(tmpDir)
	os.Exit(code)
}

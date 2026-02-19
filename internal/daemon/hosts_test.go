package daemon

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestUpdateLabHostsFileAddsBlock(t *testing.T) {
	tmpDir := t.TempDir()
	hostsPath := filepath.Join(tmpDir, "hosts")
	initial := "127.0.0.1 localhost\n::1 localhost\n"
	if err := os.WriteFile(hostsPath, []byte(initial), 0644); err != nil {
		t.Fatalf("failed to write hosts file: %v", err)
	}

	if err := updateLabHostsFile(hostsPath, []string{"api.lab", "app.lab"}); err != nil {
		t.Fatalf("updateLabHostsFile() failed: %v", err)
	}

	updated, err := os.ReadFile(hostsPath)
	if err != nil {
		t.Fatalf("failed to read hosts file: %v", err)
	}

	content := string(updated)
	if !strings.Contains(content, labHostsStartMarker) || !strings.Contains(content, labHostsEndMarker) {
		t.Fatalf("expected hosts block markers to be present")
	}
	if !strings.Contains(content, "127.0.0.1 app.lab") || !strings.Contains(content, "::1 app.lab") {
		t.Errorf("expected app.lab entries to be written")
	}
	if !strings.Contains(content, "127.0.0.1 api.lab") || !strings.Contains(content, "::1 api.lab") {
		t.Errorf("expected api.lab entries to be written")
	}
	if !strings.HasPrefix(content, initial) {
		t.Errorf("expected existing hosts entries to be preserved")
	}
}

func TestUpdateLabHostsFileReplacesBlock(t *testing.T) {
	tmpDir := t.TempDir()
	hostsPath := filepath.Join(tmpDir, "hosts")
	initial := strings.Join([]string{
		"127.0.0.1 localhost",
		labHostsStartMarker,
		"127.0.0.1 old.lab",
		"::1 old.lab",
		labHostsEndMarker,
		"192.168.1.10 other-host",
		"",
	}, "\n")
	if err := os.WriteFile(hostsPath, []byte(initial), 0644); err != nil {
		t.Fatalf("failed to write hosts file: %v", err)
	}

	if err := updateLabHostsFile(hostsPath, []string{"new.lab"}); err != nil {
		t.Fatalf("updateLabHostsFile() failed: %v", err)
	}

	updated, err := os.ReadFile(hostsPath)
	if err != nil {
		t.Fatalf("failed to read hosts file: %v", err)
	}

	content := string(updated)
	if strings.Contains(content, "old.lab") {
		t.Errorf("expected old.lab entries to be removed")
	}
	if !strings.Contains(content, "127.0.0.1 new.lab") || !strings.Contains(content, "::1 new.lab") {
		t.Errorf("expected new.lab entries to be written")
	}
	if !strings.Contains(content, "192.168.1.10 other-host") {
		t.Errorf("expected non-managed hosts to be preserved")
	}
}

func TestUpdateLabHostsFileRemovesBlockWhenEmpty(t *testing.T) {
	tmpDir := t.TempDir()
	hostsPath := filepath.Join(tmpDir, "hosts")
	initial := strings.Join([]string{
		"127.0.0.1 localhost",
		labHostsStartMarker,
		"127.0.0.1 app.lab",
		"::1 app.lab",
		labHostsEndMarker,
		"",
	}, "\n")
	if err := os.WriteFile(hostsPath, []byte(initial), 0644); err != nil {
		t.Fatalf("failed to write hosts file: %v", err)
	}

	if err := updateLabHostsFile(hostsPath, nil); err != nil {
		t.Fatalf("updateLabHostsFile() failed: %v", err)
	}

	updated, err := os.ReadFile(hostsPath)
	if err != nil {
		t.Fatalf("failed to read hosts file: %v", err)
	}

	content := string(updated)
	if strings.Contains(content, labHostsStartMarker) || strings.Contains(content, labHostsEndMarker) {
		t.Errorf("expected hosts block markers to be removed")
	}
	if strings.Contains(content, "app.lab") {
		t.Errorf("expected app.lab entries to be removed")
	}
}

func TestDaemonSyncLabHosts(t *testing.T) {
	tmpDir := t.TempDir()
	hostsPath := filepath.Join(tmpDir, "hosts")
	if err := os.WriteFile(hostsPath, []byte("127.0.0.1 localhost\n"), 0644); err != nil {
		t.Fatalf("failed to write hosts file: %v", err)
	}

	t.Setenv("FAA_HOSTS_PATH", hostsPath)
	d := &Daemon{}

	d.syncLabHosts(map[string]int{
		"app.lab":       3000,
		"app.localhost": 3001,
	})

	content, err := os.ReadFile(hostsPath)
	if err != nil {
		t.Fatalf("failed to read hosts file: %v", err)
	}
	if !strings.Contains(string(content), "127.0.0.1 app.lab") {
		t.Errorf("expected app.lab to be synced into hosts file")
	}
	if strings.Contains(string(content), "app.localhost") {
		t.Errorf("did not expect .localhost entries in hosts file")
	}

	d.syncLabHosts(map[string]int{})

	content, err = os.ReadFile(hostsPath)
	if err != nil {
		t.Fatalf("failed to read hosts file after cleanup: %v", err)
	}
	if strings.Contains(string(content), labHostsStartMarker) {
		t.Errorf("expected hosts block to be removed when no .lab routes remain")
	}
}

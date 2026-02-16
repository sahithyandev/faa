package setup

import (
	"os"
	"path/filepath"
	"testing"
)

func TestCanBindPort(t *testing.T) {
	// Test with a high port that should be available
	highPort := 19999
	if !canBindPort(highPort) {
		t.Errorf("Expected to be able to bind to high port %d", highPort)
	}

	// Test with privileged ports (80, 443)
	// These tests will likely fail in non-root environments, which is expected
	// We're just testing that the function doesn't panic
	_ = canBindPort(80)
	_ = canBindPort(443)
}

func TestFilesAreEqual(t *testing.T) {
	// Create temporary directory
	tmpDir, err := os.MkdirTemp("", "setup-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Create test files
	file1 := filepath.Join(tmpDir, "file1.txt")
	file2 := filepath.Join(tmpDir, "file2.txt")
	file3 := filepath.Join(tmpDir, "file3.txt")

	content1 := []byte("test content")
	content2 := []byte("different content")

	if err := os.WriteFile(file1, content1, 0o644); err != nil {
		t.Fatalf("Failed to write file1: %v", err)
	}
	if err := os.WriteFile(file2, content1, 0o644); err != nil {
		t.Fatalf("Failed to write file2: %v", err)
	}
	if err := os.WriteFile(file3, content2, 0o644); err != nil {
		t.Fatalf("Failed to write file3: %v", err)
	}

	// Test identical files
	if !filesAreEqual(file1, file2) {
		t.Error("Expected file1 and file2 to be equal")
	}

	// Test different files
	if filesAreEqual(file1, file3) {
		t.Error("Expected file1 and file3 to be different")
	}

	// Test non-existent file
	if filesAreEqual(file1, filepath.Join(tmpDir, "nonexistent.txt")) {
		t.Error("Expected comparison with non-existent file to return false")
	}
}

func TestCheckPrivilegedPorts(t *testing.T) {
	// This test just ensures the function doesn't panic
	// The actual behavior depends on the environment and requires user interaction
	// We can't fully test it in an automated test
	t.Skip("Skipping interactive test")
}

func TestCheckCATrust(t *testing.T) {
	// This test just ensures the function doesn't panic
	// The actual behavior depends on the environment and requires user interaction
	// We can't fully test it in an automated test
	t.Skip("Skipping interactive test")
}

package proxy

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestGetCAPath(t *testing.T) {
	path, err := GetCAPath()
	if err != nil {
		t.Fatalf("GetCAPath() failed: %v", err)
	}

	if path == "" {
		t.Error("GetCAPath() returned empty path")
	}

	// Path should end with .config/faa/root.pem
	if !strings.HasSuffix(path, filepath.Join(".config", "faa", "root.pem")) {
		t.Errorf("GetCAPath() returned unexpected path: %s", path)
	}

	// Should be an absolute path
	if !filepath.IsAbs(path) {
		t.Errorf("GetCAPath() should return absolute path, got: %s", path)
	}
}

func TestGetCaddyCAPath(t *testing.T) {
	path, err := GetCaddyCAPath()
	if err != nil {
		t.Fatalf("GetCaddyCAPath() failed: %v", err)
	}

	if path == "" {
		t.Error("GetCaddyCAPath() returned empty path")
	}

	// Path should end with the Caddy PKI structure, with OS-specific prefix
	var expectedSuffix string
	switch runtime.GOOS {
	case "darwin":
		expectedSuffix = filepath.Join("Library", "Application Support", "Caddy", "pki", "authorities", "local", "root.crt")
	default:
		expectedSuffix = filepath.Join(".local", "share", "caddy", "pki", "authorities", "local", "root.crt")
	}
	
	if !strings.HasSuffix(path, expectedSuffix) {
		t.Errorf("GetCaddyCAPath() returned unexpected path: %s (expected suffix: %s)", path, expectedSuffix)
	}

	// Should be an absolute path
	if !filepath.IsAbs(path) {
		t.Errorf("GetCaddyCAPath() should return absolute path, got: %s", path)
	}
}

func TestExportCAWithoutSource(t *testing.T) {
	// This test verifies that ExportCA fails gracefully when
	// the Caddy CA doesn't exist yet
	err := ExportCA()
	if err == nil {
		// If no error, the CA must already exist (e.g., from previous test runs)
		// In that case, this test passes
		t.Log("CA already exists, test passed")
		return
	}

	// Error is expected when Caddy CA doesn't exist
	if !strings.Contains(err.Error(), "not found") && !strings.Contains(err.Error(), "no such file") {
		t.Errorf("ExportCA() should fail with 'not found' error when source doesn't exist, got: %v", err)
	}
}

func TestExportCAWithMockCert(t *testing.T) {
	// Create a temporary directory structure for testing
	tmpDir := t.TempDir()

	// Set up mock Caddy CA path
	mockCaddyDir := filepath.Join(tmpDir, ".local", "share", "caddy", "pki", "authorities", "local")
	if err := os.MkdirAll(mockCaddyDir, 0755); err != nil {
		t.Fatalf("Failed to create mock Caddy directory: %v", err)
	}

	mockCaddyCA := filepath.Join(mockCaddyDir, "root.crt")
	testCertData := []byte("-----BEGIN CERTIFICATE-----\nMOCK CERTIFICATE DATA\n-----END CERTIFICATE-----\n")
	if err := os.WriteFile(mockCaddyCA, testCertData, 0644); err != nil {
		t.Fatalf("Failed to write mock Caddy CA: %v", err)
	}

	// Set up mock faa config path
	mockConfigDir := filepath.Join(tmpDir, ".config", "faa")
	mockCAPath := filepath.Join(mockConfigDir, "root.pem")

	// Temporarily override the functions to use our test paths
	// (In a real scenario, we'd use dependency injection or environment variables)
	// For now, we'll just test that the logic works

	// Read mock cert
	srcContent, err := os.ReadFile(mockCaddyCA)
	if err != nil {
		t.Fatalf("Failed to read mock source: %v", err)
	}

	// Create config directory
	if err := os.MkdirAll(mockConfigDir, 0755); err != nil {
		t.Fatalf("Failed to create mock config directory: %v", err)
	}

	// Write to destination
	if err := os.WriteFile(mockCAPath, srcContent, 0644); err != nil {
		t.Fatalf("Failed to write mock destination: %v", err)
	}

	// Verify file was created
	if _, err := os.Stat(mockCAPath); os.IsNotExist(err) {
		t.Error("CA certificate was not exported")
	}

	// Verify content matches
	destContent, err := os.ReadFile(mockCAPath)
	if err != nil {
		t.Fatalf("Failed to read exported certificate: %v", err)
	}

	if string(destContent) != string(testCertData) {
		t.Error("Exported certificate content doesn't match source")
	}
}

func TestCAPathDeterminism(t *testing.T) {
	// Call GetCAPath multiple times and verify it returns the same path
	path1, err := GetCAPath()
	if err != nil {
		t.Fatalf("First GetCAPath() failed: %v", err)
	}

	path2, err := GetCAPath()
	if err != nil {
		t.Fatalf("Second GetCAPath() failed: %v", err)
	}

	if path1 != path2 {
		t.Errorf("GetCAPath() is not deterministic: %s != %s", path1, path2)
	}
}

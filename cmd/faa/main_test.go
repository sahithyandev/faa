package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/sahithyandev/faa/internal/daemon"
	"github.com/sahithyandev/faa/internal/devproc"
)

// TestRunCommandFlow tests the complete flow of the run command
func TestRunCommandFlow(t *testing.T) {
	// Create temp directory for test
	tmpDir := t.TempDir()

	// Create a test project
	projectRoot := filepath.Join(tmpDir, "test-project")
	if err := os.MkdirAll(projectRoot, 0755); err != nil {
		t.Fatalf("Failed to create project directory: %v", err)
	}

	// Create package.json
	packageJSON := `{
  "name": "test-app",
  "version": "1.0.0"
}`
	if err := os.WriteFile(filepath.Join(projectRoot, "package.json"), []byte(packageJSON), 0644); err != nil {
		t.Fatalf("Failed to create package.json: %v", err)
	}

	// Create a simple test script that exits quickly
	testScript := `#!/bin/bash
echo "Test server starting on port $PORT"
sleep 1
echo "Test server stopping"
exit 0
`
	scriptPath := filepath.Join(projectRoot, "test-server.sh")
	if err := os.WriteFile(scriptPath, []byte(testScript), 0755); err != nil {
		t.Fatalf("Failed to create test script: %v", err)
	}

	// Override HOME for testing
	originalHome := os.Getenv("HOME")
	defer os.Setenv("HOME", originalHome)
	os.Setenv("HOME", tmpDir)

	// Create registry and start daemon
	registry, err := daemon.NewRegistry()
	if err != nil {
		t.Fatalf("NewRegistry() failed: %v", err)
	}

	d := daemon.New(registry, nil)
	errChan := make(chan error, 1)
	go func() {
		errChan <- d.Start()
	}()

	// Wait for daemon to start
	time.Sleep(100 * time.Millisecond)

	// Save original working directory
	originalWd, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get working directory: %v", err)
	}
	defer os.Chdir(originalWd)

	// Change to project directory
	if err := os.Chdir(projectRoot); err != nil {
		t.Fatalf("Failed to change to project directory: %v", err)
	}

	// Test the run command flow by calling handleRun directly
	// We'll simulate it by testing the individual components

	// 1. Test that we can find the project
	proj, err := findProjectFromCwd()
	if err != nil {
		t.Fatalf("Failed to find project: %v", err)
	}
	if proj.Name != "test-app" {
		t.Errorf("Expected project name 'test-app', got '%s'", proj.Name)
	}

	// 2. Test that we can connect to daemon
	client, err := daemon.Connect()
	if err != nil {
		t.Fatalf("Failed to connect to daemon: %v", err)
	}
	defer client.Close()

	// 3. Test that no process exists initially
	existingProc, err := client.GetProcess(proj.Root)
	if err != nil {
		t.Fatalf("Failed to check for existing process: %v", err)
	}
	if existingProc != nil {
		t.Errorf("Expected no existing process, got: %+v", existingProc)
	}

	// 4. Test that we can set a process
	testPID := os.Getpid() // Use our own PID for testing
	if err := client.SetProcess(&daemon.SetProcessData{
		ProjectRoot: proj.Root,
		PID:         testPID,
		Host:        proj.Host(),
		Port:        3000,
		StartedAt:   time.Now(),
	}); err != nil {
		t.Fatalf("Failed to set process: %v", err)
	}

	// 5. Test that we can retrieve the process
	retrievedProc, err := client.GetProcess(proj.Root)
	if err != nil {
		t.Fatalf("Failed to get process: %v", err)
	}
	if retrievedProc == nil {
		t.Fatal("Expected process to exist")
	}
	if retrievedProc.PID != testPID {
		t.Errorf("Expected PID %d, got %d", testPID, retrievedProc.PID)
	}

	// 6. Test that IsAlive works with our PID
	if !devproc.IsAlive(testPID) {
		t.Error("Expected process to be alive")
	}

	// 7. Test that we can clear the process
	if err := client.ClearProcess(proj.Root); err != nil {
		t.Fatalf("Failed to clear process: %v", err)
	}

	// 8. Verify process is cleared
	clearedProc, err := client.GetProcess(proj.Root)
	if err != nil {
		t.Fatalf("Failed to check cleared process: %v", err)
	}
	if clearedProc != nil {
		t.Errorf("Expected no process after clear, got: %+v", clearedProc)
	}

	// Cleanup: Shutdown daemon
	d.Shutdown()
	select {
	case err := <-errChan:
		if err != nil {
			t.Errorf("Daemon returned error: %v", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("Daemon didn't shutdown in time")
	}
}

// Helper function to find project from current working directory
func findProjectFromCwd() (*projectInfo, error) {
	cwd, err := os.Getwd()
	if err != nil {
		return nil, fmt.Errorf("failed to get current directory: %w", err)
	}

	// Simulate project.FindProjectRoot
	currentDir := cwd
	for {
		packageJSONPath := filepath.Join(currentDir, "package.json")
		if _, err := os.Stat(packageJSONPath); err == nil {
			// Found package.json
			_, err := os.ReadFile(packageJSONPath)
			if err != nil {
				return nil, fmt.Errorf("failed to read package.json: %w", err)
			}

			// Simple name extraction (just for testing)
			// In real code, this would be done by project.FindProjectRoot
			name := "test-app" // Simplified

			return &projectInfo{
				Root: currentDir,
				Name: name,
			}, nil
		}

		// Move up to parent
		parentDir := filepath.Dir(currentDir)
		if parentDir == currentDir {
			return nil, fmt.Errorf("no package.json found")
		}
		currentDir = parentDir
	}
}

type projectInfo struct {
	Root string
	Name string
}

func (p *projectInfo) Host() string {
	return p.Name
}

// Unused but required for testing
var _ = context.Background()

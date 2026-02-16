package daemon

import (
	"os"
	"testing"
	"time"
)

func TestClientPing(t *testing.T) {
	tmpDir := t.TempDir()

	// Override HOME for testing
	originalHome := os.Getenv("HOME")
	defer os.Setenv("HOME", originalHome)
	os.Setenv("HOME", tmpDir)

	// Create registry
	registry, err := NewRegistry()
	if err != nil {
		t.Fatalf("NewRegistry() failed: %v", err)
	}

	// Start daemon in a goroutine
	d := New(registry, nil)
	errChan := make(chan error, 1)
	go func() {
		errChan <- d.Start()
	}()

	// Wait for daemon to start
	time.Sleep(100 * time.Millisecond)

	// Connect to daemon
	client, err := Connect()
	if err != nil {
		t.Fatalf("Connect() failed: %v", err)
	}
	defer client.Close()

	// Test ping
	if err := client.Ping(); err != nil {
		t.Errorf("Ping() failed: %v", err)
	}

	// Shutdown daemon
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

func TestClientGetProcess(t *testing.T) {
	tmpDir := t.TempDir()

	// Override HOME for testing
	originalHome := os.Getenv("HOME")
	defer os.Setenv("HOME", originalHome)
	os.Setenv("HOME", tmpDir)

	// Create registry
	registry, err := NewRegistry()
	if err != nil {
		t.Fatalf("NewRegistry() failed: %v", err)
	}

	// Start daemon in a goroutine
	d := New(registry, nil)
	errChan := make(chan error, 1)
	go func() {
		errChan <- d.Start()
	}()

	// Wait for daemon to start
	time.Sleep(100 * time.Millisecond)

	// Connect to daemon
	client, err := Connect()
	if err != nil {
		t.Fatalf("Connect() failed: %v", err)
	}
	defer client.Close()

	// Test GetProcess when no process exists
	proc, err := client.GetProcess("/tmp/test-project")
	if err != nil {
		t.Fatalf("GetProcess() failed: %v", err)
	}
	if proc != nil {
		t.Errorf("Expected nil process, got: %+v", proc)
	}

	// Set a process
	startedAt := time.Now()
	if err := client.SetProcess(&SetProcessData{
		ProjectRoot: "/tmp/test-project",
		PID:         12345,
		Host:        "test.local",
		Port:        3000,
		StartedAt:   startedAt,
	}); err != nil {
		t.Fatalf("SetProcess() failed: %v", err)
	}

	// Get the process
	proc, err = client.GetProcess("/tmp/test-project")
	if err != nil {
		t.Fatalf("GetProcess() failed: %v", err)
	}
	if proc == nil {
		t.Fatal("Expected process, got nil")
	}
	if proc.ProjectRoot != "/tmp/test-project" {
		t.Errorf("ProjectRoot = %s, want /tmp/test-project", proc.ProjectRoot)
	}
	if proc.PID != 12345 {
		t.Errorf("PID = %d, want 12345", proc.PID)
	}
	if proc.Host != "test.local" {
		t.Errorf("Host = %s, want test.local", proc.Host)
	}
	if proc.Port != 3000 {
		t.Errorf("Port = %d, want 3000", proc.Port)
	}

	// Clear the process
	if err := client.ClearProcess("/tmp/test-project"); err != nil {
		t.Fatalf("ClearProcess() failed: %v", err)
	}

	// Verify it's cleared
	proc, err = client.GetProcess("/tmp/test-project")
	if err != nil {
		t.Fatalf("GetProcess() failed: %v", err)
	}
	if proc != nil {
		t.Errorf("Expected nil process after clear, got: %+v", proc)
	}

	// Shutdown daemon
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

func TestClientUpsertRoute(t *testing.T) {
	tmpDir := t.TempDir()

	// Override HOME for testing
	originalHome := os.Getenv("HOME")
	defer os.Setenv("HOME", originalHome)
	os.Setenv("HOME", tmpDir)

	// Create registry
	registry, err := NewRegistry()
	if err != nil {
		t.Fatalf("NewRegistry() failed: %v", err)
	}

	// Start daemon in a goroutine
	d := New(registry, nil)
	errChan := make(chan error, 1)
	go func() {
		errChan <- d.Start()
	}()

	// Wait for daemon to start
	time.Sleep(100 * time.Millisecond)

	// Connect to daemon
	client, err := Connect()
	if err != nil {
		t.Fatalf("Connect() failed: %v", err)
	}
	defer client.Close()

	// Test UpsertRoute
	if err := client.UpsertRoute("test.local", 3000); err != nil {
		t.Fatalf("UpsertRoute() failed: %v", err)
	}

	// Verify the route was added
	routes, err := registry.ListRoutes()
	if err != nil {
		t.Fatalf("ListRoutes() failed: %v", err)
	}

	found := false
	for _, route := range routes {
		if route.Host == "test.local" && route.Port == 3000 {
			found = true
			break
		}
	}
	if !found {
		t.Error("Route not found in registry after UpsertRoute")
	}

	// Shutdown daemon
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

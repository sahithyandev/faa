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

	// Set a process using current PID (so it won't be cleaned up as stale)
	startedAt := time.Now()
	currentPID := os.Getpid()
	if err := client.SetProcess(&SetProcessData{
		ProjectRoot: "/tmp/test-project",
		PID:         currentPID,
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
	if proc.PID != currentPID {
		t.Errorf("PID = %d, want %d", proc.PID, currentPID)
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

func TestClientGetRoute(t *testing.T) {
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

	// Test GetRoute when route doesn't exist
	port, err := client.GetRoute("nonexistent.local")
	if err != nil {
		t.Fatalf("GetRoute() failed: %v", err)
	}
	if port != 0 {
		t.Errorf("GetRoute() for nonexistent route = %d, want 0", port)
	}

	// Add a route
	if err := client.UpsertRoute("test.local", 3000); err != nil {
		t.Fatalf("UpsertRoute() failed: %v", err)
	}

	// Test GetRoute when route exists
	port, err = client.GetRoute("test.local")
	if err != nil {
		t.Fatalf("GetRoute() failed: %v", err)
	}
	if port != 3000 {
		t.Errorf("GetRoute() = %d, want 3000", port)
	}

	// Update the route
	if err := client.UpsertRoute("test.local", 3001); err != nil {
		t.Fatalf("UpsertRoute() update failed: %v", err)
	}

	// Test GetRoute after update
	port, err = client.GetRoute("test.local")
	if err != nil {
		t.Fatalf("GetRoute() after update failed: %v", err)
	}
	if port != 3001 {
		t.Errorf("GetRoute() after update = %d, want 3001", port)
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


func TestClientStatus(t *testing.T) {
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

	// Add a route and process for testing
	if err := client.UpsertRoute("test.local", 3000); err != nil {
		t.Fatalf("UpsertRoute() failed: %v", err)
	}

	startedAt := time.Now()
	currentPID := os.Getpid()
	if err := client.SetProcess(&SetProcessData{
		ProjectRoot: "/tmp/test-project",
		PID:         currentPID,
		Host:        "test.local",
		Port:        3000,
		StartedAt:   startedAt,
	}); err != nil {
		t.Fatalf("SetProcess() failed: %v", err)
	}

	// Test Status
	status, err := client.Status()
	if err != nil {
		t.Fatalf("Status() failed: %v", err)
	}

	// Verify routes
	if len(status.Routes) == 0 {
		t.Error("Expected at least one route in status")
	}
	foundRoute := false
	for _, route := range status.Routes {
		if route.Host == "test.local" && route.Port == 3000 {
			foundRoute = true
			break
		}
	}
	if !foundRoute {
		t.Error("Expected route not found in status")
	}

	// Verify processes
	if len(status.Processes) == 0 {
		t.Error("Expected at least one process in status")
	}
	foundProcess := false
	for _, proc := range status.Processes {
		if proc.ProjectRoot == "/tmp/test-project" && proc.PID == currentPID {
			foundProcess = true
			break
		}
	}
	if !foundProcess {
		t.Error("Expected process not found in status")
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

func TestClientListRoutes(t *testing.T) {
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

	// Add routes
	if err := client.UpsertRoute("test1.local", 3001); err != nil {
		t.Fatalf("UpsertRoute() failed: %v", err)
	}
	if err := client.UpsertRoute("test2.local", 3002); err != nil {
		t.Fatalf("UpsertRoute() failed: %v", err)
	}

	// Test ListRoutes
	routes, err := client.ListRoutes()
	if err != nil {
		t.Fatalf("ListRoutes() failed: %v", err)
	}

	if len(routes) != 2 {
		t.Errorf("Expected 2 routes, got %d", len(routes))
	}

	// Verify both routes exist
	foundTest1 := false
	foundTest2 := false
	for _, route := range routes {
		if route.Host == "test1.local" && route.Port == 3001 {
			foundTest1 = true
		}
		if route.Host == "test2.local" && route.Port == 3002 {
			foundTest2 = true
		}
	}
	if !foundTest1 {
		t.Error("Route test1.local not found")
	}
	if !foundTest2 {
		t.Error("Route test2.local not found")
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

func TestClientStop(t *testing.T) {
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

	// Test Stop
	if err := client.Stop(false); err != nil {
		t.Fatalf("Stop() failed: %v", err)
	}

	// Wait for daemon to shutdown
	select {
	case err := <-errChan:
		if err != nil {
			t.Errorf("Daemon returned error: %v", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("Daemon didn't shutdown in time after Stop()")
	}
}

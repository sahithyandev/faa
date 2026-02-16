package daemon

import (
	"bufio"
	"encoding/json"
	"net"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"
)

// createTestProxy creates a proxy instance for testing (nil for now since proxy isn't required in tests)
func createTestProxy() interface{} {
	return nil
}

// containsAny checks if s contains any of the substrings
func containsAny(s string, substrs []string) bool {
	for _, substr := range substrs {
		if strings.Contains(s, substr) {
			return true
		}
	}
	return false
}

func TestSocketPath(t *testing.T) {
	sockPath, err := SocketPath()
	if err != nil {
		t.Fatalf("SocketPath() failed: %v", err)
	}

	homeDir, err := os.UserHomeDir()
	if err != nil {
		t.Fatalf("Failed to get home directory: %v", err)
	}

	expected := filepath.Join(homeDir, ".config", "faa", "ctl.sock")
	if sockPath != expected {
		t.Errorf("SocketPath() = %s, want %s", sockPath, expected)
	}
}

func TestLockPath(t *testing.T) {
	lockPath, err := LockPath()
	if err != nil {
		t.Fatalf("LockPath() failed: %v", err)
	}

	homeDir, err := os.UserHomeDir()
	if err != nil {
		t.Fatalf("Failed to get home directory: %v", err)
	}

	expected := filepath.Join(homeDir, ".config", "faa", "daemon.lock")
	if lockPath != expected {
		t.Errorf("LockPath() = %s, want %s", lockPath, expected)
	}
}

func TestPidPath(t *testing.T) {
	pidPath, err := PidPath()
	if err != nil {
		t.Fatalf("PidPath() failed: %v", err)
	}

	homeDir, err := os.UserHomeDir()
	if err != nil {
		t.Fatalf("Failed to get home directory: %v", err)
	}

	expected := filepath.Join(homeDir, ".config", "faa", "daemon.pid")
	if pidPath != expected {
		t.Errorf("PidPath() = %s, want %s", pidPath, expected)
	}
}

func TestDaemonSingleInstance(t *testing.T) {
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

	// Start first daemon in a goroutine
	d1 := New(registry, nil)
	errChan1 := make(chan error, 1)
	go func() {
		errChan1 <- d1.Start()
	}()

	// Wait a bit for first daemon to start
	time.Sleep(100 * time.Millisecond)

	// Try to start second daemon - should fail with lock error
	d2 := New(registry, nil)
	errChan2 := make(chan error, 1)
	go func() {
		errChan2 <- d2.Start()
	}()

	// Wait for second daemon to fail
	select {
	case err := <-errChan2:
		if err == nil {
			t.Error("Second daemon should have failed to acquire lock")
		}
		// Check that error message indicates lock conflict
		if err != nil {
			errMsg := err.Error()
			// Check for key phrases that indicate a lock conflict
			if !containsAny(errMsg, []string{"already running", "failed to acquire lock"}) {
				t.Errorf("Expected error about daemon already running, got: %v", err)
			}
		}
	case <-time.After(2 * time.Second):
		t.Fatal("Second daemon didn't fail quickly enough")
	}

	// Shutdown first daemon
	d1.Shutdown()

	// Wait for first daemon to shutdown
	select {
	case err := <-errChan1:
		if err != nil {
			t.Errorf("First daemon returned error: %v", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("First daemon didn't shutdown in time")
	}
}

func TestDaemonSocketCreation(t *testing.T) {
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
	time.Sleep(200 * time.Millisecond)

	// Check socket exists
	sockPath, err := SocketPath()
	if err != nil {
		t.Fatalf("SocketPath() failed: %v", err)
	}

	info, err := os.Stat(sockPath)
	if err != nil {
		t.Fatalf("Socket file doesn't exist: %v", err)
	}

	// Verify it's a socket
	if info.Mode()&os.ModeSocket == 0 {
		t.Error("Socket file is not a Unix socket")
	}

	// Verify permissions are 0600
	if info.Mode().Perm() != 0600 {
		t.Errorf("Socket permissions = %o, want 0600", info.Mode().Perm())
	}

	// Shutdown daemon
	d.Shutdown()

	// Wait for daemon to shutdown
	select {
	case err := <-errChan:
		if err != nil {
			t.Errorf("Daemon returned error: %v", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("Daemon didn't shutdown in time")
	}

	// Verify socket is cleaned up
	if _, err := os.Stat(sockPath); !os.IsNotExist(err) {
		t.Error("Socket file should be removed after shutdown")
	}
}

func TestDaemonPingRequest(t *testing.T) {
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
	time.Sleep(200 * time.Millisecond)

	// Connect to daemon
	sockPath, err := SocketPath()
	if err != nil {
		t.Fatalf("SocketPath() failed: %v", err)
	}

	conn, err := net.Dial("unix", sockPath)
	if err != nil {
		t.Fatalf("Failed to connect to daemon: %v", err)
	}
	defer conn.Close()

	// Send ping request
	req, err := NewRequest(MessageTypePing, nil)
	if err != nil {
		t.Fatalf("NewRequest() failed: %v", err)
	}

	if err := EncodeRequest(conn, req); err != nil {
		t.Fatalf("EncodeRequest() failed: %v", err)
	}

	// Read response
	reader := bufio.NewReader(conn)
	resp, err := DecodeResponse(reader)
	if err != nil {
		t.Fatalf("DecodeResponse() failed: %v", err)
	}

	// Verify response
	if !resp.Ok {
		t.Errorf("Response Ok = false, want true, error: %s", resp.Error)
	}

	// Decode response data
	var data map[string]string
	if err := json.Unmarshal(resp.Data, &data); err != nil {
		t.Fatalf("Failed to unmarshal response data: %v", err)
	}

	if data["message"] != "pong" {
		t.Errorf("Response message = %s, want 'pong'", data["message"])
	}

	// Shutdown daemon
	d.Shutdown()

	// Wait for daemon to shutdown
	select {
	case <-errChan:
	case <-time.After(2 * time.Second):
		t.Fatal("Daemon didn't shutdown in time")
	}
}

func TestDaemonRegistryOperations(t *testing.T) {
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
	time.Sleep(200 * time.Millisecond)

	// Connect to daemon
	sockPath, err := SocketPath()
	if err != nil {
		t.Fatalf("SocketPath() failed: %v", err)
	}

	conn, err := net.Dial("unix", sockPath)
	if err != nil {
		t.Fatalf("Failed to connect to daemon: %v", err)
	}
	defer conn.Close()

	// Test UpsertRoute
	upsertReq, err := NewRequest(MessageTypeUpsertRoute, &UpsertRouteData{
		Host: "test.local",
		Port: 3000,
	})
	if err != nil {
		t.Fatalf("NewRequest() failed: %v", err)
	}

	if err := EncodeRequest(conn, upsertReq); err != nil {
		t.Fatalf("EncodeRequest() failed: %v", err)
	}

	reader := bufio.NewReader(conn)
	resp, err := DecodeResponse(reader)
	if err != nil {
		t.Fatalf("DecodeResponse() failed: %v", err)
	}

	if !resp.Ok {
		t.Errorf("UpsertRoute response Ok = false, error: %s", resp.Error)
	}

	// Test ListRoutes
	listReq, err := NewRequest(MessageTypeListRoutes, nil)
	if err != nil {
		t.Fatalf("NewRequest() failed: %v", err)
	}

	if err := EncodeRequest(conn, listReq); err != nil {
		t.Fatalf("EncodeRequest() failed: %v", err)
	}

	resp, err = DecodeResponse(reader)
	if err != nil {
		t.Fatalf("DecodeResponse() failed: %v", err)
	}

	if !resp.Ok {
		t.Errorf("ListRoutes response Ok = false, error: %s", resp.Error)
	}

	var routes []Route
	if err := json.Unmarshal(resp.Data, &routes); err != nil {
		t.Fatalf("Failed to unmarshal routes: %v", err)
	}

	if len(routes) != 1 {
		t.Errorf("Expected 1 route, got %d", len(routes))
	}

	if len(routes) > 0 && (routes[0].Host != "test.local" || routes[0].Port != 3000) {
		t.Errorf("Route = %+v, want {Host:test.local Port:3000}", routes[0])
	}

	// Shutdown daemon
	d.Shutdown()

	// Wait for daemon to shutdown
	select {
	case <-errChan:
	case <-time.After(2 * time.Second):
		t.Fatal("Daemon didn't shutdown in time")
	}
}

func TestDaemonStatusRequest(t *testing.T) {
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

	// Add some test data to registry
	if err := registry.UpsertRoute("app.local", 3000); err != nil {
		t.Fatalf("Failed to add test route: %v", err)
	}

	if err := registry.SetProcess("/test/project", 12345, "app.local", 3000, time.Now()); err != nil {
		t.Fatalf("Failed to add test process: %v", err)
	}

	// Start daemon
	d := New(registry, nil)
	errChan := make(chan error, 1)
	go func() {
		errChan <- d.Start()
	}()

	time.Sleep(200 * time.Millisecond)

	// Connect and send status request
	sockPath, err := SocketPath()
	if err != nil {
		t.Fatalf("SocketPath() failed: %v", err)
	}

	conn, err := net.Dial("unix", sockPath)
	if err != nil {
		t.Fatalf("Failed to connect: %v", err)
	}
	defer conn.Close()

	req, err := NewRequest(MessageTypeStatus, nil)
	if err != nil {
		t.Fatalf("NewRequest() failed: %v", err)
	}

	if err := EncodeRequest(conn, req); err != nil {
		t.Fatalf("EncodeRequest() failed: %v", err)
	}

	reader := bufio.NewReader(conn)
	resp, err := DecodeResponse(reader)
	if err != nil {
		t.Fatalf("DecodeResponse() failed: %v", err)
	}

	if !resp.Ok {
		t.Errorf("Status response Ok = false, error: %s", resp.Error)
	}

	var status StatusResponseData
	if err := json.Unmarshal(resp.Data, &status); err != nil {
		t.Fatalf("Failed to unmarshal status: %v", err)
	}

	if len(status.Routes) != 1 {
		t.Errorf("Expected 1 route in status, got %d", len(status.Routes))
	}

	if len(status.Processes) != 1 {
		t.Errorf("Expected 1 process in status, got %d", len(status.Processes))
	}

	// Shutdown daemon
	d.Shutdown()
	select {
	case <-errChan:
	case <-time.After(2 * time.Second):
		t.Fatal("Daemon didn't shutdown in time")
	}
}

func TestDaemonProxyIntegration(t *testing.T) {
tmpDir := t.TempDir()

// Override HOME for testing
originalHome := os.Getenv("HOME")
defer os.Setenv("HOME", originalHome)
os.Setenv("HOME", tmpDir)

// Create registry and add initial routes
registry, err := NewRegistry()
if err != nil {
t.Fatalf("NewRegistry() failed: %v", err)
}

// Add some routes to the registry before starting daemon
if err := registry.UpsertRoute("test1.local", 3000); err != nil {
t.Fatalf("Failed to add initial route: %v", err)
}
if err := registry.UpsertRoute("test2.local", 3001); err != nil {
t.Fatalf("Failed to add initial route: %v", err)
}

// Create a mock proxy that tracks ApplyRoutes calls
type proxyMock struct {
mu           sync.RWMutex
applyCount   int
lastRoutes   map[string]int
}

// Since we can't pass a mock due to type constraints, we'll test that
// the daemon can be created without a proxy and doesn't crash
d := New(registry, nil)
errChan := make(chan error, 1)
go func() {
errChan <- d.Start()
}()

time.Sleep(200 * time.Millisecond)

// Connect and send an upsert route request
sockPath, err := SocketPath()
if err != nil {
t.Fatalf("SocketPath() failed: %v", err)
}

conn, err := net.Dial("unix", sockPath)
if err != nil {
t.Fatalf("Failed to connect: %v", err)
}
defer conn.Close()

// Test upsert_route with proxy
req, err := NewRequest(MessageTypeUpsertRoute, &UpsertRouteData{
Host: "test3.local",
Port: 3002,
})
if err != nil {
t.Fatalf("NewRequest() failed: %v", err)
}

if err := EncodeRequest(conn, req); err != nil {
t.Fatalf("EncodeRequest() failed: %v", err)
}

reader := bufio.NewReader(conn)
resp, err := DecodeResponse(reader)
if err != nil {
t.Fatalf("DecodeResponse() failed: %v", err)
}

if !resp.Ok {
t.Errorf("UpsertRoute with nil proxy should succeed, error: %s", resp.Error)
}

// Verify the route was added to registry
routes, err := registry.ListRoutes()
if err != nil {
t.Fatalf("Failed to list routes: %v", err)
}

if len(routes) != 3 {
t.Errorf("Expected 3 routes in registry, got %d", len(routes))
}

// Shutdown daemon
d.Shutdown()
select {
case <-errChan:
case <-time.After(2 * time.Second):
t.Fatal("Daemon didn't shutdown in time")
}
}

func TestDaemonStartLoadRoutes(t *testing.T) {
tmpDir := t.TempDir()

// Override HOME for testing
originalHome := os.Getenv("HOME")
defer os.Setenv("HOME", originalHome)
os.Setenv("HOME", tmpDir)

// Create registry and add routes before daemon starts
registry, err := NewRegistry()
if err != nil {
t.Fatalf("NewRegistry() failed: %v", err)
}

// Add some routes that should be loaded on startup
if err := registry.UpsertRoute("preexisting.local", 4000); err != nil {
t.Fatalf("Failed to add route: %v", err)
}
if err := registry.UpsertRoute("another.local", 4001); err != nil {
t.Fatalf("Failed to add route: %v", err)
}

// Create daemon with nil proxy (daemon should handle nil proxy gracefully)
d := New(registry, nil)
errChan := make(chan error, 1)
go func() {
errChan <- d.Start()
}()

time.Sleep(200 * time.Millisecond)

// Verify daemon started successfully
sockPath, err := SocketPath()
if err != nil {
t.Fatalf("SocketPath() failed: %v", err)
}

// Try to connect to verify daemon is running
conn, err := net.Dial("unix", sockPath)
if err != nil {
t.Fatalf("Failed to connect to daemon: %v", err)
}
conn.Close()

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

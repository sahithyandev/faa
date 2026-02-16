package daemon

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestConfigDir(t *testing.T) {
	dir, err := ConfigDir()
	if err != nil {
		t.Fatalf("ConfigDir() failed: %v", err)
	}

	homeDir, err := os.UserHomeDir()
	if err != nil {
		t.Fatalf("Failed to get home directory: %v", err)
	}

	expected := filepath.Join(homeDir, ".config", "faa")
	if dir != expected {
		t.Errorf("ConfigDir() = %s, want %s", dir, expected)
	}
}

func TestNewRegistry(t *testing.T) {
	// Use a temporary directory for testing
	tmpDir := t.TempDir()

	// Override home directory for testing
	originalHome := os.Getenv("HOME")
	defer os.Setenv("HOME", originalHome)
	os.Setenv("HOME", tmpDir)

	reg, err := NewRegistry()
	if err != nil {
		t.Fatalf("NewRegistry() failed: %v", err)
	}

	expectedDir := filepath.Join(tmpDir, ".config", "faa")
	if reg.configDir != expectedDir {
		t.Errorf("Registry.configDir = %s, want %s", reg.configDir, expectedDir)
	}

	// Verify directory was created
	if _, err := os.Stat(expectedDir); os.IsNotExist(err) {
		t.Errorf("Config directory was not created at %s", expectedDir)
	}
}

func TestUpsertRoute(t *testing.T) {
	tmpDir := t.TempDir()
	reg := &Registry{configDir: tmpDir}

	// Test inserting a new route
	err := reg.UpsertRoute("example.local", 3000)
	if err != nil {
		t.Fatalf("UpsertRoute() failed: %v", err)
	}

	// Verify the route was saved
	routes, err := reg.loadRoutes()
	if err != nil {
		t.Fatalf("loadRoutes() failed: %v", err)
	}

	if port, ok := routes["example.local"]; !ok || port != 3000 {
		t.Errorf("Route not saved correctly: got port %d, want 3000", port)
	}

	// Test updating an existing route
	err = reg.UpsertRoute("example.local", 3001)
	if err != nil {
		t.Fatalf("UpsertRoute() update failed: %v", err)
	}

	routes, err = reg.loadRoutes()
	if err != nil {
		t.Fatalf("loadRoutes() failed: %v", err)
	}

	if port, ok := routes["example.local"]; !ok || port != 3001 {
		t.Errorf("Route not updated correctly: got port %d, want 3001", port)
	}
}

func TestListRoutes(t *testing.T) {
	tmpDir := t.TempDir()
	reg := &Registry{configDir: tmpDir}

	// Initially should be empty
	routes, err := reg.ListRoutes()
	if err != nil {
		t.Fatalf("ListRoutes() failed: %v", err)
	}
	if len(routes) != 0 {
		t.Errorf("Expected 0 routes, got %d", len(routes))
	}

	// Add some routes
	reg.UpsertRoute("app1.local", 3000)
	reg.UpsertRoute("app2.local", 3001)
	reg.UpsertRoute("app3.local", 3002)

	routes, err = reg.ListRoutes()
	if err != nil {
		t.Fatalf("ListRoutes() failed: %v", err)
	}

	if len(routes) != 3 {
		t.Errorf("Expected 3 routes, got %d", len(routes))
	}

	// Verify routes contain expected data
	routeMap := make(map[string]int)
	for _, r := range routes {
		routeMap[r.Host] = r.Port
	}

	if routeMap["app1.local"] != 3000 {
		t.Errorf("app1.local: got port %d, want 3000", routeMap["app1.local"])
	}
	if routeMap["app2.local"] != 3001 {
		t.Errorf("app2.local: got port %d, want 3001", routeMap["app2.local"])
	}
	if routeMap["app3.local"] != 3002 {
		t.Errorf("app3.local: got port %d, want 3002", routeMap["app3.local"])
	}
}

func TestSetProcess(t *testing.T) {
	tmpDir := t.TempDir()
	reg := &Registry{configDir: tmpDir}

	startTime := time.Now()
	projectRoot := "/home/user/project"

	err := reg.SetProcess(projectRoot, 12345, "myapp.local", 3000, startTime)
	if err != nil {
		t.Fatalf("SetProcess() failed: %v", err)
	}

	// Verify the process was saved
	processes, err := reg.loadProcesses()
	if err != nil {
		t.Fatalf("loadProcesses() failed: %v", err)
	}

	proc, ok := processes[projectRoot]
	if !ok {
		t.Fatalf("Process not found for projectRoot %s", projectRoot)
	}

	if proc.PID != 12345 {
		t.Errorf("PID = %d, want 12345", proc.PID)
	}
	if proc.Host != "myapp.local" {
		t.Errorf("Host = %s, want myapp.local", proc.Host)
	}
	if proc.Port != 3000 {
		t.Errorf("Port = %d, want 3000", proc.Port)
	}
	if !proc.StartedAt.Equal(startTime) {
		t.Errorf("StartedAt = %v, want %v", proc.StartedAt, startTime)
	}
}

func TestClearProcess(t *testing.T) {
	tmpDir := t.TempDir()
	reg := &Registry{configDir: tmpDir}

	projectRoot := "/home/user/project"
	startTime := time.Now()

	// Add a process
	err := reg.SetProcess(projectRoot, 12345, "myapp.local", 3000, startTime)
	if err != nil {
		t.Fatalf("SetProcess() failed: %v", err)
	}

	// Clear the process
	err = reg.ClearProcess(projectRoot)
	if err != nil {
		t.Fatalf("ClearProcess() failed: %v", err)
	}

	// Verify it was removed
	processes, err := reg.loadProcesses()
	if err != nil {
		t.Fatalf("loadProcesses() failed: %v", err)
	}

	if _, ok := processes[projectRoot]; ok {
		t.Errorf("Process should have been cleared but still exists")
	}
}

func TestListProcesses(t *testing.T) {
	tmpDir := t.TempDir()
	reg := &Registry{configDir: tmpDir}

	// Initially should be empty
	processes, err := reg.ListProcesses()
	if err != nil {
		t.Fatalf("ListProcesses() failed: %v", err)
	}
	if len(processes) != 0 {
		t.Errorf("Expected 0 processes, got %d", len(processes))
	}

	// Add some processes
	startTime := time.Now()
	reg.SetProcess("/home/user/project1", 12345, "app1.local", 3000, startTime)
	reg.SetProcess("/home/user/project2", 12346, "app2.local", 3001, startTime)
	reg.SetProcess("/home/user/project3", 12347, "app3.local", 3002, startTime)

	processes, err = reg.ListProcesses()
	if err != nil {
		t.Fatalf("ListProcesses() failed: %v", err)
	}

	if len(processes) != 3 {
		t.Errorf("Expected 3 processes, got %d", len(processes))
	}

	// Verify processes contain expected data
	procMap := make(map[string]*Process)
	for _, p := range processes {
		procMap[p.ProjectRoot] = p
	}

	if proc, ok := procMap["/home/user/project1"]; !ok || proc.PID != 12345 || proc.Port != 3000 {
		t.Errorf("Process 1 not found or incorrect")
	}
	if proc, ok := procMap["/home/user/project2"]; !ok || proc.PID != 12346 || proc.Port != 3001 {
		t.Errorf("Process 2 not found or incorrect")
	}
	if proc, ok := procMap["/home/user/project3"]; !ok || proc.PID != 12347 || proc.Port != 3002 {
		t.Errorf("Process 3 not found or incorrect")
	}
}

func TestAtomicWrite_Routes(t *testing.T) {
	tmpDir := t.TempDir()
	reg := &Registry{configDir: tmpDir}

	// Add a route
	err := reg.UpsertRoute("test.local", 3000)
	if err != nil {
		t.Fatalf("UpsertRoute() failed: %v", err)
	}

	// Verify no .tmp file is left behind
	tmpFile := reg.routesPath() + ".tmp"
	if _, err := os.Stat(tmpFile); !os.IsNotExist(err) {
		t.Errorf("Temporary file still exists: %s", tmpFile)
	}

	// Verify the actual file exists
	if _, err := os.Stat(reg.routesPath()); err != nil {
		t.Errorf("Routes file doesn't exist: %v", err)
	}
}

func TestAtomicWrite_Processes(t *testing.T) {
	tmpDir := t.TempDir()
	reg := &Registry{configDir: tmpDir}

	// Add a process
	err := reg.SetProcess("/home/user/project", 12345, "test.local", 3000, time.Now())
	if err != nil {
		t.Fatalf("SetProcess() failed: %v", err)
	}

	// Verify no .tmp file is left behind
	tmpFile := reg.processesPath() + ".tmp"
	if _, err := os.Stat(tmpFile); !os.IsNotExist(err) {
		t.Errorf("Temporary file still exists: %s", tmpFile)
	}

	// Verify the actual file exists
	if _, err := os.Stat(reg.processesPath()); err != nil {
		t.Errorf("Processes file doesn't exist: %v", err)
	}
}

func TestFilePermissions(t *testing.T) {
	tmpDir := t.TempDir()
	reg := &Registry{configDir: tmpDir}

	// Create routes file
	err := reg.UpsertRoute("test.local", 3000)
	if err != nil {
		t.Fatalf("UpsertRoute() failed: %v", err)
	}

	// Check file permissions for routes.json
	info, err := os.Stat(reg.routesPath())
	if err != nil {
		t.Fatalf("Failed to stat routes file: %v", err)
	}

	// File should be readable and writable by owner, readable by group and others (0644)
	mode := info.Mode()
	if mode.Perm() != 0644 {
		t.Errorf("Routes file permissions = %o, want 0644", mode.Perm())
	}

	// Create processes file
	err = reg.SetProcess("/home/user/project", 12345, "test.local", 3000, time.Now())
	if err != nil {
		t.Fatalf("SetProcess() failed: %v", err)
	}

	// Check file permissions for processes.json
	info, err = os.Stat(reg.processesPath())
	if err != nil {
		t.Fatalf("Failed to stat processes file: %v", err)
	}

	mode = info.Mode()
	if mode.Perm() != 0644 {
		t.Errorf("Processes file permissions = %o, want 0644", mode.Perm())
	}
}

func TestLoadEmptyFiles(t *testing.T) {
	tmpDir := t.TempDir()
	reg := &Registry{configDir: tmpDir}

	// Create empty routes.json
	emptyRoutesPath := reg.routesPath()
	if err := os.WriteFile(emptyRoutesPath, []byte(""), 0644); err != nil {
		t.Fatalf("Failed to create empty routes file: %v", err)
	}

	routes, err := reg.loadRoutes()
	if err != nil {
		t.Fatalf("loadRoutes() failed on empty file: %v", err)
	}
	if len(routes) != 0 {
		t.Errorf("Expected 0 routes from empty file, got %d", len(routes))
	}

	// Create empty processes.json
	emptyProcessesPath := reg.processesPath()
	if err := os.WriteFile(emptyProcessesPath, []byte(""), 0644); err != nil {
		t.Fatalf("Failed to create empty processes file: %v", err)
	}

	processes, err := reg.loadProcesses()
	if err != nil {
		t.Fatalf("loadProcesses() failed on empty file: %v", err)
	}
	if len(processes) != 0 {
		t.Errorf("Expected 0 processes from empty file, got %d", len(processes))
	}
}

func TestUpdateExistingProcess(t *testing.T) {
	tmpDir := t.TempDir()
	reg := &Registry{configDir: tmpDir}

	projectRoot := "/home/user/project"
	startTime1 := time.Now()

	// Add initial process
	err := reg.SetProcess(projectRoot, 12345, "app.local", 3000, startTime1)
	if err != nil {
		t.Fatalf("SetProcess() failed: %v", err)
	}

	// Update the same process with new values
	startTime2 := time.Now().Add(time.Hour)
	err = reg.SetProcess(projectRoot, 67890, "app2.local", 3001, startTime2)
	if err != nil {
		t.Fatalf("SetProcess() update failed: %v", err)
	}

	// Verify only one process exists with updated values
	processes, err := reg.ListProcesses()
	if err != nil {
		t.Fatalf("ListProcesses() failed: %v", err)
	}

	if len(processes) != 1 {
		t.Errorf("Expected 1 process after update, got %d", len(processes))
	}

	proc := processes[0]
	if proc.PID != 67890 {
		t.Errorf("PID = %d, want 67890", proc.PID)
	}
	if proc.Host != "app2.local" {
		t.Errorf("Host = %s, want app2.local", proc.Host)
	}
	if proc.Port != 3001 {
		t.Errorf("Port = %d, want 3001", proc.Port)
	}
}

func TestConfigDirPermissions(t *testing.T) {
	tmpDir := t.TempDir()

	// Override home directory for testing
	originalHome := os.Getenv("HOME")
	defer os.Setenv("HOME", originalHome)
	os.Setenv("HOME", tmpDir)

	_, err := NewRegistry()
	if err != nil {
		t.Fatalf("NewRegistry() failed: %v", err)
	}

	configDir := filepath.Join(tmpDir, ".config", "faa")
	info, err := os.Stat(configDir)
	if err != nil {
		t.Fatalf("Failed to stat config directory: %v", err)
	}

	// Directory should have 0755 permissions
	if info.Mode().Perm() != 0755 {
		t.Errorf("Config directory permissions = %o, want 0755", info.Mode().Perm())
	}
}

func TestCleanupStaleProcesses(t *testing.T) {
	tmpDir := t.TempDir()
	reg := &Registry{configDir: tmpDir}

	startTime := time.Now()

	// Add some processes: one with current PID (alive), one with fake PID (dead)
	currentPID := os.Getpid()
	fakePID := 999999 // This PID should not exist

	// Add alive process
	err := reg.SetProcess("/home/user/project1", currentPID, "app1.local", 3000, startTime)
	if err != nil {
		t.Fatalf("SetProcess() failed: %v", err)
	}

	// Add dead process
	err = reg.SetProcess("/home/user/project2", fakePID, "app2.local", 3001, startTime)
	if err != nil {
		t.Fatalf("SetProcess() failed: %v", err)
	}

	// Verify both processes exist
	processes, err := reg.ListProcesses()
	if err != nil {
		t.Fatalf("ListProcesses() failed: %v", err)
	}
	if len(processes) != 2 {
		t.Errorf("Expected 2 processes before cleanup, got %d", len(processes))
	}

	// Clean up stale processes
	staleCount, err := reg.CleanupStaleProcesses()
	if err != nil {
		t.Fatalf("CleanupStaleProcesses() failed: %v", err)
	}

	// Should have cleaned up 1 stale process
	if staleCount != 1 {
		t.Errorf("Expected 1 stale process, got %d", staleCount)
	}

	// Verify only alive process remains
	processes, err = reg.ListProcesses()
	if err != nil {
		t.Fatalf("ListProcesses() failed: %v", err)
	}
	if len(processes) != 1 {
		t.Errorf("Expected 1 process after cleanup, got %d", len(processes))
	}

	// Verify the remaining process is the alive one
	if processes[0].PID != currentPID {
		t.Errorf("Expected alive process with PID %d, got PID %d", currentPID, processes[0].PID)
	}
}

func TestCleanupStaleProcesses_NoStaleProcesses(t *testing.T) {
	tmpDir := t.TempDir()
	reg := &Registry{configDir: tmpDir}

	startTime := time.Now()
	currentPID := os.Getpid()

	// Add only alive process
	err := reg.SetProcess("/home/user/project1", currentPID, "app1.local", 3000, startTime)
	if err != nil {
		t.Fatalf("SetProcess() failed: %v", err)
	}

	// Clean up stale processes
	staleCount, err := reg.CleanupStaleProcesses()
	if err != nil {
		t.Fatalf("CleanupStaleProcesses() failed: %v", err)
	}

	// Should have cleaned up 0 stale processes
	if staleCount != 0 {
		t.Errorf("Expected 0 stale processes, got %d", staleCount)
	}

	// Verify process still exists
	processes, err := reg.ListProcesses()
	if err != nil {
		t.Fatalf("ListProcesses() failed: %v", err)
	}
	if len(processes) != 1 {
		t.Errorf("Expected 1 process, got %d", len(processes))
	}
}

func TestCleanupStaleProcesses_EmptyRegistry(t *testing.T) {
	tmpDir := t.TempDir()
	reg := &Registry{configDir: tmpDir}

	// Clean up with no processes
	staleCount, err := reg.CleanupStaleProcesses()
	if err != nil {
		t.Fatalf("CleanupStaleProcesses() failed: %v", err)
	}

	// Should have cleaned up 0 stale processes
	if staleCount != 0 {
		t.Errorf("Expected 0 stale processes, got %d", staleCount)
	}
}

func TestIsProcessAlive(t *testing.T) {
	// Test with current process (should be alive)
	currentPID := os.Getpid()
	if !isProcessAlive(currentPID) {
		t.Error("Current process should be alive")
	}

	// Test with invalid PID
	if isProcessAlive(0) {
		t.Error("PID 0 should not be alive")
	}

	if isProcessAlive(-1) {
		t.Error("Negative PID should not be alive")
	}

	// Test with fake PID (should not be alive)
	fakePID := 999999
	if isProcessAlive(fakePID) {
		t.Errorf("Fake PID %d should not be alive", fakePID)
	}
}

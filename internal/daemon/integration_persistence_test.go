package daemon

import (
"encoding/json"
"os"
"path/filepath"
"testing"
"time"
)

// TestPortPersistenceIntegration tests the full flow of port persistence:
// 1. Pre-populate routes.json with a specific port
// 2. Start daemon and verify it loads the route
// 3. Client queries the route and gets the predefined port
func TestPortPersistenceIntegration(t *testing.T) {
tmpDir := t.TempDir()

// Override HOME for testing
originalHome := os.Getenv("HOME")
defer os.Setenv("HOME", originalHome)
os.Setenv("HOME", tmpDir)

// Step 1: Create registry and pre-populate with a route
registry, err := NewRegistry()
if err != nil {
t.Fatalf("NewRegistry() failed: %v", err)
}

testHost := "my-project.local"
predefinedPort := 12345

if err := registry.UpsertRoute(testHost, predefinedPort); err != nil {
t.Fatalf("Failed to pre-populate route: %v", err)
}

// Verify it was written to disk
configDir := filepath.Join(tmpDir, ".config", "faa")
routesFile := filepath.Join(configDir, "routes.json")

data, err := os.ReadFile(routesFile)
if err != nil {
t.Fatalf("Failed to read routes.json: %v", err)
}

var routes map[string]int
if err := json.Unmarshal(data, &routes); err != nil {
t.Fatalf("Failed to parse routes.json: %v", err)
}

if routes[testHost] != predefinedPort {
t.Fatalf("Route not saved to disk. Expected %d, got %d", predefinedPort, routes[testHost])
}

// Step 2: Start daemon (it should load the existing routes)
d := New(registry, nil)
errChan := make(chan error, 1)
go func() {
errChan <- d.Start()
}()

time.Sleep(200 * time.Millisecond)

// Step 3: Connect as client and verify GetRoute returns the predefined port
client, err := Connect()
if err != nil {
t.Fatalf("Failed to connect to daemon: %v", err)
}
defer client.Close()

port, err := client.GetRoute(testHost)
if err != nil {
t.Fatalf("GetRoute() failed: %v", err)
}

if port != predefinedPort {
t.Errorf("GetRoute() = %d, want %d (from pre-populated routes.json)", port, predefinedPort)
}

// Step 4: Verify non-existent routes return 0
port2, err := client.GetRoute("nonexistent.local")
if err != nil {
t.Fatalf("GetRoute() for nonexistent host failed: %v", err)
}

if port2 != 0 {
t.Errorf("GetRoute() for nonexistent host = %d, want 0", port2)
}

// Cleanup
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

package proxy

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"
)

func TestNew(t *testing.T) {
	p := New()
	if p == nil {
		t.Fatal("New() returned nil")
	}

	if p.routes == nil {
		t.Error("routes map should be initialized")
	}

	if p.running {
		t.Error("proxy should not be running initially")
	}

	if p.httpPort != 80 {
		t.Errorf("httpPort should be 80 by default, got %d", p.httpPort)
	}

	if p.httpsPort != 443 {
		t.Errorf("httpsPort should be 443 by default, got %d", p.httpsPort)
	}
}

func TestNewWithPorts(t *testing.T) {
	p := NewWithPorts(8080, 8443)
	if p == nil {
		t.Fatal("NewWithPorts() returned nil")
	}

	if p.httpPort != 8080 {
		t.Errorf("httpPort should be 8080, got %d", p.httpPort)
	}

	if p.httpsPort != 8443 {
		t.Errorf("httpsPort should be 8443, got %d", p.httpsPort)
	}
}

func TestStartStop(t *testing.T) {
	// Use unprivileged ports for testing
	p := NewWithPorts(18080, 18443)
	ctx := context.Background()

	// Start the proxy
	err := p.Start(ctx)
	if err != nil {
		t.Fatalf("Start() failed: %v", err)
	}

	// Give it a moment to initialize
	time.Sleep(500 * time.Millisecond)

	// Verify it's running
	p.mu.RLock()
	running := p.running
	p.mu.RUnlock()

	if !running {
		t.Error("proxy should be running after Start()")
	}

	// Stop the proxy
	err = p.Stop()
	if err != nil {
		t.Fatalf("Stop() failed: %v", err)
	}

	// Verify it's not running
	p.mu.RLock()
	running = p.running
	p.mu.RUnlock()

	if running {
		t.Error("proxy should not be running after Stop()")
	}
}

func TestStartAlreadyRunning(t *testing.T) {
	// Use unprivileged ports for testing
	p := NewWithPorts(18081, 18444)
	ctx := context.Background()

	// Start the proxy
	err := p.Start(ctx)
	if err != nil {
		t.Fatalf("First Start() failed: %v", err)
	}
	defer p.Stop()

	// Give it a moment to initialize
	time.Sleep(500 * time.Millisecond)

	// Try to start again - should fail
	err = p.Start(ctx)
	if err == nil {
		t.Error("Second Start() should have failed")
	}
}

func TestStopWhenNotRunning(t *testing.T) {
	p := NewWithPorts(18082, 18445)

	// Stop when not running - should not error
	err := p.Stop()
	if err != nil {
		t.Errorf("Stop() on non-running proxy should not error: %v", err)
	}
}

func TestApplyRoutesWhenNotRunning(t *testing.T) {
	p := NewWithPorts(18083, 18446)

	routes := map[string]int{
		"test1.local": 3000,
		"test2.local": 3001,
	}

	err := p.ApplyRoutes(routes)
	if err != nil {
		t.Fatalf("ApplyRoutes() failed: %v", err)
	}

	// Verify routes were updated
	p.mu.RLock()
	defer p.mu.RUnlock()

	if len(p.routes) != 2 {
		t.Errorf("Expected 2 routes, got %d", len(p.routes))
	}

	if p.routes["test1.local"] != 3000 {
		t.Errorf("Expected test1.local -> 3000, got %d", p.routes["test1.local"])
	}

	if p.routes["test2.local"] != 3001 {
		t.Errorf("Expected test2.local -> 3001, got %d", p.routes["test2.local"])
	}
}

func TestApplyRoutesWhenRunning(t *testing.T) {
	// Use unprivileged ports for testing
	p := NewWithPorts(18084, 18447)
	ctx := context.Background()

	// Start the proxy
	err := p.Start(ctx)
	if err != nil {
		t.Fatalf("Start() failed: %v", err)
	}
	defer p.Stop()

	// Give it a moment to initialize
	time.Sleep(500 * time.Millisecond)

	// Apply new routes
	routes := map[string]int{
		"newhost.local": 4000,
	}

	err = p.ApplyRoutes(routes)
	if err != nil {
		t.Fatalf("ApplyRoutes() failed: %v", err)
	}

	// Verify routes were updated
	p.mu.RLock()
	defer p.mu.RUnlock()

	if len(p.routes) != 1 {
		t.Errorf("Expected 1 route, got %d", len(p.routes))
	}

	if p.routes["newhost.local"] != 4000 {
		t.Errorf("Expected newhost.local -> 4000, got %d", p.routes["newhost.local"])
	}
}

func TestDefaultRoute(t *testing.T) {
	// Use unprivileged ports for testing
	p := NewWithPorts(18085, 18448)
	ctx := context.Background()

	// Set a test route before starting
	err := p.ApplyRoutes(map[string]int{
		"example.local": 12345,
	})
	if err != nil {
		t.Fatalf("ApplyRoutes() failed: %v", err)
	}

	// Start the proxy
	err = p.Start(ctx)
	if err != nil {
		t.Fatalf("Start() failed: %v", err)
	}
	defer p.Stop()

	// Give it a moment to initialize
	time.Sleep(500 * time.Millisecond)

	// Check that route is present
	p.mu.RLock()
	defer p.mu.RUnlock()

	if len(p.routes) == 0 {
		t.Error("Expected at least one route")
	}

	port, exists := p.routes["example.local"]
	if !exists {
		t.Error("Expected route 'example.local' to exist")
	}

	if port != 12345 {
		t.Errorf("Expected port 12345, got %d", port)
	}
}

func TestBuildConfigJSON(t *testing.T) {
	p := NewWithPorts(18086, 18449)
	p.routes["test.local"] = 8080

	configJSON, err := p.buildConfigJSON()
	if err != nil {
		t.Fatalf("buildConfigJSON() failed: %v", err)
	}

	if len(configJSON) == 0 {
		t.Error("buildConfigJSON() returned empty config")
	}

	// Basic check that it's valid JSON
	if configJSON[0] != '{' {
		t.Error("buildConfigJSON() should return JSON starting with '{'")
	}
}

func TestMultipleRoutes(t *testing.T) {
	p := NewWithPorts(18087, 18450)

	routes := map[string]int{
		"app1.local": 3000,
		"app2.local": 3001,
		"app3.local": 3002,
	}

	err := p.ApplyRoutes(routes)
	if err != nil {
		t.Fatalf("ApplyRoutes() failed: %v", err)
	}

	p.mu.RLock()
	defer p.mu.RUnlock()

	if len(p.routes) != 3 {
		t.Errorf("Expected 3 routes, got %d", len(p.routes))
	}

	for host, expectedPort := range routes {
		if actualPort, exists := p.routes[host]; !exists {
			t.Errorf("Route for %s not found", host)
		} else if actualPort != expectedPort {
			t.Errorf("Route for %s: expected port %d, got %d", host, expectedPort, actualPort)
		}
	}
}

func TestEmptyRoutes(t *testing.T) {
	p := NewWithPorts(18088, 18451)

	routes := map[string]int{}

	err := p.ApplyRoutes(routes)
	if err != nil {
		t.Fatalf("ApplyRoutes() with empty routes failed: %v", err)
	}

	p.mu.RLock()
	defer p.mu.RUnlock()

	if len(p.routes) != 0 {
		t.Errorf("Expected 0 routes, got %d", len(p.routes))
	}
}

func TestApplyRoutesConcurrency(t *testing.T) {
	p := NewWithPorts(18089, 18452)

	// Number of concurrent goroutines
	numGoroutines := 10
	var wg sync.WaitGroup
	wg.Add(numGoroutines)

	// Concurrently apply routes
	for i := 0; i < numGoroutines; i++ {
		go func(id int) {
			defer wg.Done()
			routes := map[string]int{
				fmt.Sprintf("app%d.local", id): 3000 + id,
			}
			err := p.ApplyRoutes(routes)
			if err != nil {
				t.Errorf("ApplyRoutes failed in goroutine %d: %v", id, err)
			}
		}(i)
	}

	wg.Wait()

	// Verify that proxy didn't crash and final state is consistent
	p.mu.RLock()
	defer p.mu.RUnlock()

	if len(p.routes) != 1 {
		t.Logf("Final routes count: %d (expected 1 from last write)", len(p.routes))
	}
}

func TestApplyRoutesRepeated(t *testing.T) {
	p := NewWithPorts(18090, 18453)

	// Apply routes multiple times to verify no crashes
	for i := 0; i < 10; i++ {
		routes := map[string]int{
			"test.local": 3000 + i,
		}
		err := p.ApplyRoutes(routes)
		if err != nil {
			t.Fatalf("ApplyRoutes failed on iteration %d: %v", i, err)
		}

		// Verify routes were updated
		p.mu.RLock()
		if p.routes["test.local"] != 3000+i {
			t.Errorf("Route not updated correctly on iteration %d: got %d, want %d",
				i, p.routes["test.local"], 3000+i)
		}
		p.mu.RUnlock()
	}
}

func TestApplyRoutesEmptyMap(t *testing.T) {
	p := NewWithPorts(18091, 18454)

	// Apply empty routes multiple times
	for i := 0; i < 5; i++ {
		routes := map[string]int{}
		err := p.ApplyRoutes(routes)
		if err != nil {
			t.Fatalf("ApplyRoutes with empty map failed on iteration %d: %v", i, err)
		}

		p.mu.RLock()
		if len(p.routes) != 0 {
			t.Errorf("Expected 0 routes on iteration %d, got %d", i, len(p.routes))
		}
		p.mu.RUnlock()
	}
}

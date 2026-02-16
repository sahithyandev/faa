package port

import (
	"fmt"
	"net"
	"testing"
)

func TestStablePort_Stability(t *testing.T) {
	// Test that the same name always produces the same initial port
	tests := []struct {
		name        string
		projectName string
	}{
		{
			name:        "simple project",
			projectName: "myproject",
		},
		{
			name:        "with dashes",
			projectName: "my-project",
		},
		{
			name:        "complex name",
			projectName: "complex-project-name-123",
		},
		{
			name:        "short name",
			projectName: "a",
		},
		{
			name:        "scoped package",
			projectName: "@myorg/myproject",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Call multiple times and ensure same port
			port1, err1 := StablePort(tt.projectName)
			port2, err2 := StablePort(tt.projectName)
			port3, err3 := StablePort(tt.projectName)

			if err1 != nil || err2 != nil || err3 != nil {
				t.Fatalf("unexpected errors: %v, %v, %v", err1, err2, err3)
			}

			if port1 != port2 || port2 != port3 {
				t.Errorf("StablePort(%q) not stable: got %d, %d, %d", tt.projectName, port1, port2, port3)
			}

			// Verify port is in valid range
			if port1 < minPort || port1 > maxPort {
				t.Errorf("StablePort(%q) = %d, want port in range [%d, %d]", tt.projectName, port1, minPort, maxPort)
			}
		})
	}
}

func TestStablePort_AvoidList(t *testing.T) {
	// Test that avoided ports are never returned
	// We'll test multiple project names to increase coverage
	projectNames := []string{
		"project1", "project2", "project3", "project4", "project5",
		"test-app", "my-service", "web-app", "api-server", "frontend",
		"backend", "microservice", "dashboard", "admin", "client",
	}

	for _, name := range projectNames {
		port, err := StablePort(name)
		if err != nil {
			t.Fatalf("StablePort(%q) unexpected error: %v", name, err)
		}

		if avoidPorts[port] {
			t.Errorf("StablePort(%q) = %d, which is in avoid list", name, port)
		}
	}
}

func TestStablePort_DifferentNames(t *testing.T) {
	// Test that different names produce different initial attempts
	// (though they might collide and probe to same final port)
	name1 := "project-alpha"
	name2 := "project-beta"

	port1, err1 := StablePort(name1)
	port2, err2 := StablePort(name2)

	if err1 != nil || err2 != nil {
		t.Fatalf("unexpected errors: %v, %v", err1, err2)
	}

	// Both should be in valid range
	if port1 < minPort || port1 > maxPort {
		t.Errorf("StablePort(%q) = %d, want port in range [%d, %d]", name1, port1, minPort, maxPort)
	}
	if port2 < minPort || port2 > maxPort {
		t.Errorf("StablePort(%q) = %d, want port in range [%d, %d]", name2, port2, minPort, maxPort)
	}

	// Neither should be in avoid list
	if avoidPorts[port1] {
		t.Errorf("StablePort(%q) = %d, which is in avoid list", name1, port1)
	}
	if avoidPorts[port2] {
		t.Errorf("StablePort(%q) = %d, which is in avoid list", name2, port2)
	}
}

func TestStablePort_CollisionHandling(t *testing.T) {
	// Test that when a port is occupied, it finds the next available one
	projectName := "collision-test"

	// Get the stable port for this project
	port1, err := StablePort(projectName)
	if err != nil {
		t.Fatalf("StablePort(%q) unexpected error: %v", projectName, err)
	}

	// Occupy this port
	listener, err := net.Listen("tcp", fmt.Sprintf(":%d", port1))
	if err != nil {
		t.Fatalf("failed to occupy port %d: %v", port1, err)
	}
	defer listener.Close()

	// Call StablePort again - it should find a different port
	port2, err := StablePort(projectName)
	if err != nil {
		t.Fatalf("StablePort(%q) with collision unexpected error: %v", projectName, err)
	}

	if port2 == port1 {
		t.Errorf("StablePort(%q) returned occupied port %d", projectName, port1)
	}

	// Port2 should be different and available
	if port2 < minPort || port2 > maxPort {
		t.Errorf("StablePort(%q) collision resolution = %d, want port in range [%d, %d]", projectName, port2, minPort, maxPort)
	}

	// Verify port2 is actually free by trying to bind to it
	listener2, err := net.Listen("tcp", fmt.Sprintf(":%d", port2))
	if err != nil {
		t.Errorf("StablePort(%q) returned unavailable port %d: %v", projectName, port2, err)
	} else {
		listener2.Close()
	}
}

func TestStablePort_Wrapping(t *testing.T) {
	// This test verifies that probing wraps around correctly
	// We can't easily test the actual wrapping without occupying many ports,
	// but we can verify the logic by checking multiple project names

	// Test a variety of names to ensure wrapping logic doesn't break
	projectNames := make([]string, 100)
	for i := 0; i < 100; i++ {
		projectNames[i] = fmt.Sprintf("wrap-test-%d", i)
	}

	ports := make(map[int]bool)
	for _, name := range projectNames {
		port, err := StablePort(name)
		if err != nil {
			t.Fatalf("StablePort(%q) unexpected error: %v", name, err)
		}

		// Verify port is in range
		if port < minPort || port > maxPort {
			t.Errorf("StablePort(%q) = %d, want port in range [%d, %d]", name, port, minPort, maxPort)
		}

		// Verify not in avoid list
		if avoidPorts[port] {
			t.Errorf("StablePort(%q) = %d, which is in avoid list", name, port)
		}

		ports[port] = true
	}

	// We should get some variety in ports (not all the same)
	if len(ports) < 10 {
		t.Errorf("Expected more port variety, got only %d unique ports from 100 names", len(ports))
	}
}

func TestIsPortFree(t *testing.T) {
	// Test with a port that should be free
	// Use a high port that's unlikely to be in use
	testPort := 45678

	// Check if it's free
	if !IsPortFree(testPort) {
		// Port might actually be in use, skip this assertion
		t.Logf("Port %d is not free (may be in use by system)", testPort)
	}

	// Occupy a port and verify IsPortFree returns false
	listener, err := net.Listen("tcp", fmt.Sprintf(":%d", testPort))
	if err != nil {
		t.Skipf("Cannot bind to port %d: %v", testPort, err)
	}
	defer listener.Close()

	// Now the port should not be free
	if IsPortFree(testPort) {
		t.Errorf("IsPortFree(%d) = true, want false (port is occupied)", testPort)
	}

	// Close the listener
	listener.Close()

	// After closing, the port should be free again
	if !IsPortFree(testPort) {
		t.Errorf("IsPortFree(%d) = false, want true (port was released)", testPort)
	}
}

func TestIsPortFree_InvalidPorts(t *testing.T) {
	// Test with invalid port numbers
	tests := []struct {
		name string
		port int
		want bool
	}{
		{
			name: "negative port",
			port: -1,
			want: false,
		},
		{
			name: "port too high",
			port: 70000,
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsPortFree(tt.port)
			if result != tt.want {
				t.Errorf("IsPortFree(%d) = %v, want %v", tt.port, result, tt.want)
			}
		})
	}
}

func TestAvoidPorts_Coverage(t *testing.T) {
	// Verify all the specified ports are in the avoid list
	expectedAvoidPorts := []int{3000, 4321, 5173, 8080, 8000, 5000, 4000, 9229, 8787}

	for _, port := range expectedAvoidPorts {
		if !avoidPorts[port] {
			t.Errorf("Port %d should be in avoid list but is not", port)
		}
	}

	// Verify the count matches
	if len(avoidPorts) != len(expectedAvoidPorts) {
		t.Errorf("avoidPorts has %d entries, expected %d", len(avoidPorts), len(expectedAvoidPorts))
	}
}

func TestPortRange(t *testing.T) {
	// Verify constants are set correctly
	if minPort != 10240 {
		t.Errorf("minPort = %d, want 10240", minPort)
	}
	if maxPort != 49151 {
		t.Errorf("maxPort = %d, want 49151", maxPort)
	}
}

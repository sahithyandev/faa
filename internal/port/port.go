package port

import (
	"crypto/sha256"
	"encoding/binary"
	"fmt"
	"net"
)

// Package port handles port allocation and management

const (
	minPort = 10240
	maxPort = 49151
)

// avoidPorts is the list of commonly used development ports that should be avoided
var avoidPorts = map[int]bool{
	3000: true,
	4000: true,
	4321: true,
	5000: true,
	5173: true,
	8000: true,
	8080: true,
	8787: true,
	9229: true,
}

// StablePort generates a deterministic port number for a given name using SHA256.
// The port is in the range 10240..49151 and avoids commonly used development ports.
// If the initial port is unavailable or in the avoid list, it probes deterministically
// by incrementing with wrapping until a free port is found.
func StablePort(name string) (int, error) {
	// Generate initial port from hash
	hash := sha256.Sum256([]byte(name))
	// Use first 4 bytes of hash to get a number
	hashNum := binary.BigEndian.Uint32(hash[:4])
	
	// Map to port range
	portRange := maxPort - minPort + 1
	initialPort := minPort + int(hashNum%uint32(portRange))
	
	// Probe for available port
	port := initialPort
	attempts := 0
	maxAttempts := portRange // Try all ports in range
	
	for attempts < maxAttempts {
		// Skip avoided ports
		if !avoidPorts[port] {
			// Check if port is free
			if IsPortFree(port) {
				return port, nil
			}
		}
		
		// Increment and wrap
		port++
		if port > maxPort {
			port = minPort
		}
		
		attempts++
		
		// If we've wrapped around to initial port, we've tried all
		if port == initialPort && attempts > 0 {
			break
		}
	}
	
	return 0, fmt.Errorf("no free port found in range %d-%d", minPort, maxPort)
}

// IsPortFree checks if a TCP port is available for binding.
func IsPortFree(port int) bool {
	// Try to listen on the port
	addr := fmt.Sprintf(":%d", port)
	listener, err := net.Listen("tcp", addr)
	if err != nil {
		return false
	}
	listener.Close()
	return true
}

// Package hosts provides functionality to manage /etc/hosts entries for .local domains
// This is required on Linux to ensure .local domains resolve to 127.0.0.1
package hosts

import (
	"bufio"
	"bytes"
	"fmt"
	"os"
	"runtime"
	"strings"
)

const (
	// hostsFile is the path to the system hosts file
	hostsFile = "/etc/hosts"
	
	// faaMarker is used to identify entries managed by faa
	faaMarkerStart = "# faa-managed-start"
	faaMarkerEnd   = "# faa-managed-end"
)

// IsSupported returns true if hosts file management is supported on this platform
func IsSupported() bool {
	return runtime.GOOS == "linux"
}

// AddEntry adds a hosts entry mapping the given hostname to 127.0.0.1
// If the entry already exists, it is not duplicated
// This operation requires elevated privileges (sudo)
func AddEntry(hostname string) error {
	if !IsSupported() {
		return nil // Silently skip on unsupported platforms
	}
	
	// Read current hosts file
	content, err := readHostsFile()
	if err != nil {
		return fmt.Errorf("failed to read hosts file: %w", err)
	}
	
	// Check if entry already exists
	if hasEntry(content, hostname) {
		return nil // Entry already exists, nothing to do
	}
	
	// Add entry
	newContent := addEntryToContent(content, hostname)
	
	// Write updated hosts file
	if err := writeHostsFile(newContent); err != nil {
		return fmt.Errorf("failed to write hosts file: %w", err)
	}
	
	return nil
}

// RemoveEntry removes a hosts entry for the given hostname
// This operation requires elevated privileges (sudo)
func RemoveEntry(hostname string) error {
	if !IsSupported() {
		return nil // Silently skip on unsupported platforms
	}
	
	// Read current hosts file
	content, err := readHostsFile()
	if err != nil {
		return fmt.Errorf("failed to read hosts file: %w", err)
	}
	
	// Check if entry exists
	if !hasEntry(content, hostname) {
		return nil // Entry doesn't exist, nothing to do
	}
	
	// Remove entry
	newContent := removeEntryFromContent(content, hostname)
	
	// Write updated hosts file
	if err := writeHostsFile(newContent); err != nil {
		return fmt.Errorf("failed to write hosts file: %w", err)
	}
	
	return nil
}

// SyncEntries ensures /etc/hosts contains entries for all given hostnames
// and removes any faa-managed entries not in the list
// This operation requires elevated privileges (sudo)
func SyncEntries(hostnames []string) error {
	if !IsSupported() {
		return nil // Silently skip on unsupported platforms
	}
	
	// Read current hosts file
	content, err := readHostsFile()
	if err != nil {
		return fmt.Errorf("failed to read hosts file: %w", err)
	}
	
	// Sync entries
	newContent := syncEntriesInContent(content, hostnames)
	
	// Write updated hosts file
	if err := writeHostsFile(newContent); err != nil {
		return fmt.Errorf("failed to write hosts file: %w", err)
	}
	
	return nil
}

// readHostsFile reads and returns the content of /etc/hosts
func readHostsFile() ([]byte, error) {
	return os.ReadFile(hostsFile)
}

// writeHostsFile writes content to /etc/hosts
// This requires elevated privileges
func writeHostsFile(content []byte) error {
	// Write to a temporary file first
	tmpFile := hostsFile + ".faa.tmp"
	if err := os.WriteFile(tmpFile, content, 0644); err != nil {
		return err
	}
	
	// Rename to actual hosts file (atomic operation)
	return os.Rename(tmpFile, hostsFile)
}

// hasEntry checks if a hostname entry exists in the faa-managed section
func hasEntry(content []byte, hostname string) bool {
	scanner := bufio.NewScanner(bytes.NewReader(content))
	inFaaSection := false
	
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		
		if line == faaMarkerStart {
			inFaaSection = true
			continue
		}
		if line == faaMarkerEnd {
			inFaaSection = false
			continue
		}
		
		if inFaaSection {
			// Check if it's a valid entry (not a comment)
			if !strings.HasPrefix(line, "#") {
				// Split line into IP and hostname(s)
				fields := strings.Fields(line)
				// Check for exact hostname match in the fields (skip first field which is IP)
				for i := 1; i < len(fields); i++ {
					if fields[i] == hostname {
						return true
					}
				}
			}
		}
	}
	
	return false
}

// addEntryToContent adds a hostname entry to the content
func addEntryToContent(content []byte, hostname string) []byte {
	scanner := bufio.NewScanner(bytes.NewReader(content))
	var result bytes.Buffer
	inFaaSection := false
	faaSectionFound := false
	entryAdded := false
	
	for scanner.Scan() {
		line := scanner.Text()
		
		if strings.TrimSpace(line) == faaMarkerStart {
			inFaaSection = true
			faaSectionFound = true
			result.WriteString(line + "\n")
			continue
		}
		
		if strings.TrimSpace(line) == faaMarkerEnd {
			// Add entry before the end marker if we're in the section and haven't added it yet
			if inFaaSection && !entryAdded {
				result.WriteString(fmt.Sprintf("127.0.0.1 %s\n", hostname))
				entryAdded = true
			}
			inFaaSection = false
			result.WriteString(line + "\n")
			continue
		}
		
		result.WriteString(line + "\n")
	}
	
	// If no faa section exists, create it at the end
	if !faaSectionFound {
		// Ensure content ends with newline
		if len(content) > 0 && content[len(content)-1] != '\n' {
			result.WriteString("\n")
		}
		result.WriteString("\n")
		result.WriteString(faaMarkerStart + "\n")
		result.WriteString(fmt.Sprintf("127.0.0.1 %s\n", hostname))
		result.WriteString(faaMarkerEnd + "\n")
	}
	
	return result.Bytes()
}

// removeEntryFromContent removes a hostname entry from the content
func removeEntryFromContent(content []byte, hostname string) []byte {
	scanner := bufio.NewScanner(bytes.NewReader(content))
	var result bytes.Buffer
	inFaaSection := false
	
	for scanner.Scan() {
		line := scanner.Text()
		
		if strings.TrimSpace(line) == faaMarkerStart {
			inFaaSection = true
			result.WriteString(line + "\n")
			continue
		}
		
		if strings.TrimSpace(line) == faaMarkerEnd {
			inFaaSection = false
			result.WriteString(line + "\n")
			continue
		}
		
		// Skip lines in faa section that contain the exact hostname
		if inFaaSection {
			trimmedLine := strings.TrimSpace(line)
			if !strings.HasPrefix(trimmedLine, "#") {
				// Split line into IP and hostname(s)
				fields := strings.Fields(trimmedLine)
				// Check for exact hostname match in the fields (skip first field which is IP)
				shouldRemove := false
				for i := 1; i < len(fields); i++ {
					if fields[i] == hostname {
						shouldRemove = true
						break
					}
				}
				if shouldRemove {
					continue // Skip this line
				}
			}
		}
		
		result.WriteString(line + "\n")
	}
	
	return result.Bytes()
}

// syncEntriesInContent syncs the faa-managed section with the given hostnames
func syncEntriesInContent(content []byte, hostnames []string) []byte {
	scanner := bufio.NewScanner(bytes.NewReader(content))
	var result bytes.Buffer
	inFaaSection := false
	faaSectionFound := false
	
	// Read all lines except faa-managed section
	for scanner.Scan() {
		line := scanner.Text()
		
		if strings.TrimSpace(line) == faaMarkerStart {
			inFaaSection = true
			faaSectionFound = true
			continue
		}
		
		if strings.TrimSpace(line) == faaMarkerEnd {
			inFaaSection = false
			continue
		}
		
		// Skip lines in faa section
		if inFaaSection {
			continue
		}
		
		result.WriteString(line + "\n")
	}
	
	// Add faa section with current hostnames
	if len(hostnames) > 0 {
		// Ensure content ends with newline
		resultBytes := result.Bytes()
		if len(resultBytes) > 0 && resultBytes[len(resultBytes)-1] != '\n' {
			result.WriteString("\n")
		}
		if !faaSectionFound {
			result.WriteString("\n")
		}
		result.WriteString(faaMarkerStart + "\n")
		for _, hostname := range hostnames {
			result.WriteString(fmt.Sprintf("127.0.0.1 %s\n", hostname))
		}
		result.WriteString(faaMarkerEnd + "\n")
	}
	
	return result.Bytes()
}

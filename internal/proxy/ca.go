// Package proxy provides CA certificate management for Caddy.
package proxy

import (
	"fmt"
	"os"
	"path/filepath"
)

// GetCAPath returns the deterministic path where the Caddy CA certificate
// should be stored in the faa config directory.
// The path is: ~/.config/faa/root.pem
func GetCAPath() (string, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("failed to get home directory: %w", err)
	}

	configDir := filepath.Join(homeDir, ".config", "faa")
	caPath := filepath.Join(configDir, "root.pem")

	return caPath, nil
}

// GetCaddyCAPath returns the path to the Caddy-generated CA certificate.
// This is the internal Caddy PKI CA root certificate.
func GetCaddyCAPath() (string, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("failed to get home directory: %w", err)
	}

	// Caddy stores its CA certificate at this standard location
	caddyCAPath := filepath.Join(homeDir, ".local", "share", "caddy", "pki", "authorities", "local", "root.crt")

	return caddyCAPath, nil
}

// ExportCA exports the Caddy internal CA certificate to the faa config directory.
// If the certificate already exists at the destination and matches the source,
// it does nothing. If the source certificate doesn't exist yet (e.g., before
// first proxy start), it returns an error.
func ExportCA() error {
	srcPath, err := GetCaddyCAPath()
	if err != nil {
		return fmt.Errorf("failed to get Caddy CA path: %w", err)
	}

	destPath, err := GetCAPath()
	if err != nil {
		return fmt.Errorf("failed to get faa CA path: %w", err)
	}

	// Check if source exists
	if _, err := os.Stat(srcPath); os.IsNotExist(err) {
		return fmt.Errorf("Caddy CA certificate not found at %s (start proxy first to generate it)", srcPath)
	} else if err != nil {
		return fmt.Errorf("failed to check Caddy CA certificate: %w", err)
	}

	// Create config directory if it doesn't exist
	configDir := filepath.Dir(destPath)
	if err := os.MkdirAll(configDir, 0755); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}

	// Check if destination exists and is up-to-date
	if _, err := os.Stat(destPath); err == nil {
		// Compare file contents to see if we need to update
		srcContent, err := os.ReadFile(srcPath)
		if err != nil {
			return fmt.Errorf("failed to read source certificate: %w", err)
		}

		destContent, err := os.ReadFile(destPath)
		if err != nil {
			return fmt.Errorf("failed to read destination certificate: %w", err)
		}

		// If they match, we're done
		if string(srcContent) == string(destContent) {
			return nil
		}

		// Files differ, need to update
	}

	// Read source certificate
	certData, err := os.ReadFile(srcPath)
	if err != nil {
		return fmt.Errorf("failed to read Caddy CA certificate: %w", err)
	}

	// Write to destination
	if err := os.WriteFile(destPath, certData, 0644); err != nil {
		return fmt.Errorf("failed to write CA certificate to %s: %w", destPath, err)
	}

	return nil
}

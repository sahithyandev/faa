package project

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

// Package project handles project management operations

// Project represents a Node.js project with its root directory and name
type Project struct {
	Root string
	Name string
}

// PackageJSON represents the structure of a package.json file
type PackageJSON struct {
	Name string `json:"name"`
}

// FindProjectRoot walks up the directory tree from startDir to find the nearest package.json
// and returns a Project with the root directory and normalized project name
func FindProjectRoot(startDir string) (*Project, error) {
	// Resolve to absolute path
	absPath, err := filepath.Abs(startDir)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve absolute path: %w", err)
	}

	currentDir := absPath
	for {
		packageJSONPath := filepath.Join(currentDir, "package.json")

		// Check if package.json exists
		if _, err := os.Stat(packageJSONPath); err == nil {
			// Found package.json, read and parse it
			data, err := os.ReadFile(packageJSONPath)
			if err != nil {
				return nil, fmt.Errorf("failed to read package.json: %w", err)
			}

			var pkgJSON PackageJSON
			if err := json.Unmarshal(data, &pkgJSON); err != nil {
				return nil, fmt.Errorf("failed to parse package.json: %w", err)
			}

			// Normalize the project name
			normalizedName := normalizeName(pkgJSON.Name)

			return &Project{
				Root: currentDir,
				Name: normalizedName,
			}, nil
		}

		// Move up to parent directory
		parentDir := filepath.Dir(currentDir)

		// Check if we've reached the root
		if parentDir == currentDir {
			return nil, fmt.Errorf("no package.json found")
		}

		currentDir = parentDir
	}
}

// normalizeName converts a project name to a hostname-safe label
// Rules:
// - Convert to lowercase
// - Keep only [a-z0-9-] characters
// - Collapse consecutive dashes into single dash
// - Trim dashes from start and end
func normalizeName(name string) string {
	// Convert to lowercase
	name = strings.ToLower(name)

	// Replace any character that is not a-z, 0-9, or dash with a dash
	re := regexp.MustCompile(`[^a-z0-9-]+`)
	name = re.ReplaceAllString(name, "-")

	// Collapse consecutive dashes
	re = regexp.MustCompile(`-+`)
	name = re.ReplaceAllString(name, "-")

	// Trim dashes from start and end
	name = strings.Trim(name, "-")

	return name
}

// Host returns the hostname-safe label for the project
// This is the normalized project name suitable for use as a hostname
func (p *Project) Host() string {
	return p.Name
}

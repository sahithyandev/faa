package daemon

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// Registry manages routes and processes configuration
type Registry struct {
	configDir string
}

// Route represents a mapping from host to port
type Route struct {
	Host string `json:"host"`
	Port int    `json:"port"`
}

// Process represents a running development process
type Process struct {
	ProjectRoot string    `json:"projectRoot"`
	PID         int       `json:"pid"`
	Host        string    `json:"host"`
	Port        int       `json:"port"`
	StartedAt   time.Time `json:"startedAt"`
}

// NewRegistry creates a new Registry instance
func NewRegistry() (*Registry, error) {
	configDir, err := ConfigDir()
	if err != nil {
		return nil, err
	}

	// Ensure config directory exists
	if err := ensureConfigDir(configDir); err != nil {
		return nil, err
	}

	return &Registry{
		configDir: configDir,
	}, nil
}

// ConfigDir returns the configuration directory path (~/.config/faa)
func ConfigDir() (string, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("failed to get home directory: %w", err)
	}
	return filepath.Join(homeDir, ".config", "faa"), nil
}

// ensureConfigDir creates the config directory if it doesn't exist
func ensureConfigDir(dir string) error {
	// mkdir -p equivalent
	return os.MkdirAll(dir, 0755)
}

// routesPath returns the full path to routes.json
func (r *Registry) routesPath() string {
	return filepath.Join(r.configDir, "routes.json")
}

// processesPath returns the full path to processes.json
func (r *Registry) processesPath() string {
	return filepath.Join(r.configDir, "processes.json")
}

// loadRoutes loads routes from routes.json
func (r *Registry) loadRoutes() (map[string]int, error) {
	routes := make(map[string]int)
	
	data, err := os.ReadFile(r.routesPath())
	if err != nil {
		if os.IsNotExist(err) {
			// File doesn't exist yet, return empty map
			return routes, nil
		}
		return nil, fmt.Errorf("failed to read routes.json: %w", err)
	}

	if len(data) == 0 {
		// Empty file, return empty map
		return routes, nil
	}

	if err := json.Unmarshal(data, &routes); err != nil {
		return nil, fmt.Errorf("failed to parse routes.json: %w", err)
	}

	return routes, nil
}

// saveRoutes saves routes to routes.json with atomic write
func (r *Registry) saveRoutes(routes map[string]int) error {
	data, err := json.MarshalIndent(routes, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal routes: %w", err)
	}

	// Atomic write: write to temp file, then rename
	tempPath := r.routesPath() + ".tmp"
	if err := os.WriteFile(tempPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write temp routes file: %w", err)
	}

	if err := os.Rename(tempPath, r.routesPath()); err != nil {
		os.Remove(tempPath) // Clean up temp file on error
		return fmt.Errorf("failed to rename routes file: %w", err)
	}

	return nil
}

// loadProcesses loads processes from processes.json
func (r *Registry) loadProcesses() (map[string]*Process, error) {
	processes := make(map[string]*Process)
	
	data, err := os.ReadFile(r.processesPath())
	if err != nil {
		if os.IsNotExist(err) {
			// File doesn't exist yet, return empty map
			return processes, nil
		}
		return nil, fmt.Errorf("failed to read processes.json: %w", err)
	}

	if len(data) == 0 {
		// Empty file, return empty map
		return processes, nil
	}

	if err := json.Unmarshal(data, &processes); err != nil {
		return nil, fmt.Errorf("failed to parse processes.json: %w", err)
	}

	return processes, nil
}

// saveProcesses saves processes to processes.json with atomic write
func (r *Registry) saveProcesses(processes map[string]*Process) error {
	data, err := json.MarshalIndent(processes, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal processes: %w", err)
	}

	// Atomic write: write to temp file, then rename
	tempPath := r.processesPath() + ".tmp"
	if err := os.WriteFile(tempPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write temp processes file: %w", err)
	}

	if err := os.Rename(tempPath, r.processesPath()); err != nil {
		os.Remove(tempPath) // Clean up temp file on error
		return fmt.Errorf("failed to rename processes file: %w", err)
	}

	return nil
}

// UpsertRoute adds or updates a route mapping
func (r *Registry) UpsertRoute(host string, port int) error {
	routes, err := r.loadRoutes()
	if err != nil {
		return err
	}

	routes[host] = port
	return r.saveRoutes(routes)
}

// ListRoutes returns all routes as a slice
func (r *Registry) ListRoutes() ([]Route, error) {
	routes, err := r.loadRoutes()
	if err != nil {
		return nil, err
	}

	result := make([]Route, 0, len(routes))
	for host, port := range routes {
		result = append(result, Route{
			Host: host,
			Port: port,
		})
	}

	return result, nil
}

// SetProcess adds or updates a process entry
func (r *Registry) SetProcess(projectRoot string, pid int, host string, port int, startedAt time.Time) error {
	processes, err := r.loadProcesses()
	if err != nil {
		return err
	}

	processes[projectRoot] = &Process{
		ProjectRoot: projectRoot,
		PID:         pid,
		Host:        host,
		Port:        port,
		StartedAt:   startedAt,
	}

	return r.saveProcesses(processes)
}

// ClearProcess removes a process entry
func (r *Registry) ClearProcess(projectRoot string) error {
	processes, err := r.loadProcesses()
	if err != nil {
		return err
	}

	delete(processes, projectRoot)
	return r.saveProcesses(processes)
}

// ListProcesses returns all processes as a slice
func (r *Registry) ListProcesses() ([]*Process, error) {
	processes, err := r.loadProcesses()
	if err != nil {
		return nil, err
	}

	result := make([]*Process, 0, len(processes))
	for _, proc := range processes {
		result = append(result, proc)
	}

	return result, nil
}

package daemon

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"net"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"github.com/sahithyandev/faa/internal/lock"
	"github.com/sahithyandev/faa/internal/proxy"
)

// Daemon represents the daemon process that manages routes and processes
type Daemon struct {
	registry *Registry
	proxy    *proxy.Proxy
	lock     *lock.Lock
	listener net.Listener
	shutdown chan struct{}
}

// New creates a new Daemon instance with the given registry and proxy
func New(registry *Registry, proxy *proxy.Proxy) *Daemon {
	return &Daemon{
		registry: registry,
		proxy:    proxy,
		shutdown: make(chan struct{}),
	}
}

// SocketPath returns the path to the Unix socket
func SocketPath() (string, error) {
	// On macOS with LaunchDaemon, use /var/run/faa if FAA_SOCKET_DIR is set
	if socketDir := os.Getenv("FAA_SOCKET_DIR"); socketDir != "" {
		return filepath.Join(socketDir, "ctl.sock"), nil
	}
	
	// Default to user's config directory
	configDir, err := ConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(configDir, "ctl.sock"), nil
}

// LockPath returns the path to the daemon lock file
func LockPath() (string, error) {
	configDir, err := ConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(configDir, "daemon.lock"), nil
}

// PidPath returns the path to the daemon PID file
func PidPath() (string, error) {
	configDir, err := ConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(configDir, "daemon.pid"), nil
}

// acquireLock acquires the daemon lock to ensure single instance
func (d *Daemon) acquireLock() error {
	lockPath, err := LockPath()
	if err != nil {
		return fmt.Errorf("failed to get lock path: %w", err)
	}

	daemonLock, err := lock.Acquire(lockPath)
	if err != nil {
		return fmt.Errorf("another daemon instance is already running: %w", err)
	}

	d.lock = daemonLock
	return nil
}

// writePidFile writes the current process PID to the PID file
func (d *Daemon) writePidFile() error {
	pidPath, err := PidPath()
	if err != nil {
		return err
	}

	pid := os.Getpid()
	pidStr := fmt.Sprintf("%d\n", pid)

	if err := os.WriteFile(pidPath, []byte(pidStr), 0644); err != nil {
		return fmt.Errorf("failed to write PID file: %w", err)
	}

	return nil
}

// removePidFile removes the PID file
func (d *Daemon) removePidFile() {
	pidPath, err := PidPath()
	if err != nil {
		return
	}
	_ = os.Remove(pidPath)
}

// loadAndApplyRoutes loads routes from registry and applies them to proxy
func (d *Daemon) loadAndApplyRoutes() error {
	// Load routes from registry
	routes, err := d.registry.loadRoutes()
	if err != nil {
		return fmt.Errorf("failed to load routes: %w", err)
	}

	// Apply routes to proxy
	if d.proxy != nil {
		if err := d.proxy.ApplyRoutes(routes); err != nil {
			return fmt.Errorf("failed to apply routes to proxy: %w", err)
		}
	}

	return nil
}

// removeSocket removes the socket file if it exists
func (d *Daemon) removeSocket() {
	sockPath, err := SocketPath()
	if err != nil {
		return
	}
	_ = os.Remove(sockPath)
}

// Start starts the daemon server
func (d *Daemon) Start() error {
	// Acquire lock to ensure single instance
	if err := d.acquireLock(); err != nil {
		return err
	}
	defer d.lock.Release()

	// Write PID file
	if err := d.writePidFile(); err != nil {
		return err
	}
	defer d.removePidFile()

	// Load existing routes from routes.json and apply to proxy
	if err := d.loadAndApplyRoutes(); err != nil {
		return fmt.Errorf("failed to load and apply routes: %w", err)
	}

	// Get socket path
	sockPath, err := SocketPath()
	if err != nil {
		return err
	}

	// Remove existing socket file if it exists (from previous unclean shutdown)
	d.removeSocket()

	// Ensure socket directory exists with proper permissions
	sockDir := filepath.Dir(sockPath)
	if err := os.MkdirAll(sockDir, 0755); err != nil {
		return fmt.Errorf("failed to create socket directory: %w", err)
	}

	// Create Unix socket listener
	listener, err := net.Listen("unix", sockPath)
	if err != nil {
		return fmt.Errorf("failed to create Unix socket: %w", err)
	}
	d.listener = listener
	defer listener.Close()
	defer d.removeSocket()

	// Set socket permissions to 0666 to allow all users to connect
	// This is needed when daemon runs as root but users need to connect
	socketPerms := os.FileMode(0600)
	if os.Getenv("FAA_SOCKET_DIR") != "" {
		// When using shared socket directory (macOS LaunchDaemon), allow all users
		socketPerms = 0666
	}
	if err := os.Chmod(sockPath, socketPerms); err != nil {
		return fmt.Errorf("failed to set socket permissions: %w", err)
	}

	// Setup signal handling for graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGTERM, syscall.SIGINT)

	// Start accepting connections in a goroutine
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	errChan := make(chan error, 1)
	go func() {
		errChan <- d.acceptLoop(ctx)
	}()

	// Wait for shutdown signal or error
	select {
	case sig := <-sigChan:
		fmt.Printf("Received signal %v, shutting down gracefully...\n", sig)
	case err := <-errChan:
		if err != nil {
			return err
		}
	case <-d.shutdown:
		fmt.Println("Shutdown requested, stopping daemon...")
	}

	// Cancel context to stop accepting new connections
	cancel()

	// Close listener to stop accepting connections
	if err := d.listener.Close(); err != nil {
		fmt.Fprintf(os.Stderr, "Error closing listener: %v\n", err)
	}

	return nil
}

// Shutdown triggers a graceful shutdown of the daemon
func (d *Daemon) Shutdown() {
	close(d.shutdown)
}

// acceptLoop accepts and handles connections
func (d *Daemon) acceptLoop(ctx context.Context) error {
	for {
		select {
		case <-ctx.Done():
			return nil
		default:
		}

		conn, err := d.listener.Accept()
		if err != nil {
			// Check if context was cancelled (shutdown)
			select {
			case <-ctx.Done():
				return nil
			default:
				return fmt.Errorf("failed to accept connection: %w", err)
			}
		}

		// Handle connection in a goroutine
		go d.handleConnection(conn)
	}
}

// handleConnection handles a single client connection
func (d *Daemon) handleConnection(conn net.Conn) {
	defer conn.Close()

	reader := bufio.NewReader(conn)

	for {
		// Decode request
		req, err := DecodeRequest(reader)
		if err != nil {
			// Connection closed or error reading
			return
		}

		// Handle request and generate response
		resp := d.handleRequest(req)

		// Encode and send response
		if err := EncodeResponse(conn, resp); err != nil {
			fmt.Fprintf(os.Stderr, "Error encoding response: %v\n", err)
			return
		}
	}
}

// handleRequest processes a request and returns a response
func (d *Daemon) handleRequest(req *Request) *Response {
	switch req.Type {
	case MessageTypePing:
		return d.handlePing(req)
	case MessageTypeUpsertRoute:
		return d.handleUpsertRoute(req)
	case MessageTypeListRoutes:
		return d.handleListRoutes(req)
	case MessageTypeSetProcess:
		return d.handleSetProcess(req)
	case MessageTypeGetProcess:
		return d.handleGetProcess(req)
	case MessageTypeClearProcess:
		return d.handleClearProcess(req)
	case MessageTypeStatus:
		return d.handleStatus(req)
	case MessageTypeStop:
		return d.handleStop(req)
	default:
		return NewErrorResponse(fmt.Errorf("unknown message type: %s", req.Type))
	}
}

// handlePing handles ping requests
func (d *Daemon) handlePing(req *Request) *Response {
	resp, _ := NewSuccessResponse(map[string]string{"message": "pong"})
	return resp
}

// handleUpsertRoute handles upsert_route requests
func (d *Daemon) handleUpsertRoute(req *Request) *Response {
	var data UpsertRouteData
	if err := json.Unmarshal(req.Data, &data); err != nil {
		return NewErrorResponse(fmt.Errorf("invalid request data: %w", err))
	}

	if err := d.registry.UpsertRoute(data.Host, data.Port); err != nil {
		return NewErrorResponse(err)
	}

	// Load all routes and apply to proxy
	routes, err := d.registry.loadRoutes()
	if err != nil {
		return NewErrorResponse(fmt.Errorf("failed to load routes: %w", err))
	}

	// Apply routes to proxy
	if d.proxy != nil {
		if err := d.proxy.ApplyRoutes(routes); err != nil {
			return NewErrorResponse(fmt.Errorf("failed to apply routes to proxy: %w", err))
		}
	}

	resp, _ := NewSuccessResponse(nil)
	return resp
}

// handleListRoutes handles list_routes requests
func (d *Daemon) handleListRoutes(req *Request) *Response {
	routes, err := d.registry.ListRoutes()
	if err != nil {
		return NewErrorResponse(err)
	}

	resp, _ := NewSuccessResponse(routes)
	return resp
}

// handleSetProcess handles set_process requests
func (d *Daemon) handleSetProcess(req *Request) *Response {
	var data SetProcessData
	if err := json.Unmarshal(req.Data, &data); err != nil {
		return NewErrorResponse(fmt.Errorf("invalid request data: %w", err))
	}

	startedAt := data.StartedAt
	if startedAt.IsZero() {
		startedAt = time.Now()
	}

	if err := d.registry.SetProcess(data.ProjectRoot, data.PID, data.Host, data.Port, startedAt); err != nil {
		return NewErrorResponse(err)
	}

	resp, _ := NewSuccessResponse(nil)
	return resp
}

// handleGetProcess handles get_process requests
func (d *Daemon) handleGetProcess(req *Request) *Response {
	var data GetProcessData
	if err := json.Unmarshal(req.Data, &data); err != nil {
		return NewErrorResponse(fmt.Errorf("invalid request data: %w", err))
	}

	// Clean up stale processes before checking
	if _, err := d.registry.CleanupStaleProcesses(); err != nil {
		// Log the error but continue - this shouldn't fail the request
		fmt.Fprintf(os.Stderr, "Warning: Failed to cleanup stale processes: %v\n", err)
	}

	proc, err := d.registry.GetProcess(data.ProjectRoot)
	if err != nil {
		return NewErrorResponse(err)
	}

	resp, _ := NewSuccessResponse(proc)
	return resp
}

// handleClearProcess handles clear_process requests
func (d *Daemon) handleClearProcess(req *Request) *Response {
	var data ClearProcessData
	if err := json.Unmarshal(req.Data, &data); err != nil {
		return NewErrorResponse(fmt.Errorf("invalid request data: %w", err))
	}

	if err := d.registry.ClearProcess(data.ProjectRoot); err != nil {
		return NewErrorResponse(err)
	}

	resp, _ := NewSuccessResponse(nil)
	return resp
}

// handleStatus handles status requests
func (d *Daemon) handleStatus(req *Request) *Response {
	// Clean up stale processes before returning status
	if _, err := d.registry.CleanupStaleProcesses(); err != nil {
		// Log the error but continue - this shouldn't fail the request
		fmt.Fprintf(os.Stderr, "Warning: Failed to cleanup stale processes: %v\n", err)
	}

	routes, err := d.registry.ListRoutes()
	if err != nil {
		return NewErrorResponse(err)
	}

	processes, err := d.registry.ListProcesses()
	if err != nil {
		return NewErrorResponse(err)
	}

	statusData := StatusResponseData{
		Routes:    routes,
		Processes: processes,
	}

	resp, _ := NewSuccessResponse(statusData)
	return resp
}

// handleStop handles stop requests
func (d *Daemon) handleStop(req *Request) *Response {
	var data StopData
	if req.Data != nil {
		if err := json.Unmarshal(req.Data, &data); err != nil {
			return NewErrorResponse(fmt.Errorf("invalid request data: %w", err))
		}
	}

	// Note: clearRoutes option is available but not implemented yet as it requires
	// coordination with Caddy, which is not part of this initial implementation.
	// For now, routes are only cleared when explicitly requested via clear_process
	// or when the user manually deletes the routes.json file.

	// Trigger graceful shutdown after sending response
	go func() {
		time.Sleep(100 * time.Millisecond)
		d.Shutdown()
	}()

	resp, _ := NewSuccessResponse(nil)
	return resp
}

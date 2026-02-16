package main

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"time"

	"github.com/sahithyandev/faa/internal/daemon"
	"github.com/sahithyandev/faa/internal/devproc"
	"github.com/sahithyandev/faa/internal/lock"
	"github.com/sahithyandev/faa/internal/port"
	"github.com/sahithyandev/faa/internal/project"
	"github.com/sahithyandev/faa/internal/proxy"
	"github.com/sahithyandev/faa/internal/setup"
)

const (
	ExitSuccess = 0
	ExitError   = 1

	// caddyInitTimeout is the time to wait for Caddy to initialize and generate CA
	caddyInitTimeout = 2 * time.Second

	// daemonStartupTimeout is the maximum time to wait for daemon to start
	daemonStartupTimeout = 5 * time.Second

	// daemonStartupRetryDelay is the delay between connection attempts
	daemonStartupRetryDelay = 100 * time.Millisecond
)

func main() {
	os.Exit(run(os.Args[1:]))
}

func run(args []string) int {
	// Check for help flags first
	if len(args) == 0 {
		printUsage()
		return ExitSuccess
	}

	// Check for global help flags
	if args[0] == "-h" || args[0] == "--help" {
		printUsage()
		return ExitSuccess
	}

	subcommand := args[0]
	subArgs := args[1:]

	// Check for help flag in subcommand args
	if len(subArgs) > 0 && (subArgs[0] == "-h" || subArgs[0] == "--help") {
		printSubcommandHelp(subcommand)
		return ExitSuccess
	}

	// Handle known subcommands
	switch subcommand {
	case "setup":
		return handleSetup(subArgs)
	case "daemon":
		return handleDaemon(subArgs)
	case "run":
		return handleRun(subArgs)
	case "status":
		return handleStatus(subArgs)
	case "stop":
		return handleStop(subArgs)
	case "routes":
		return handleRoutes(subArgs)
	case "ca-path":
		return handleCAPath(subArgs)
	default:
		// Implicit run: faa <cmd> [args...] becomes run -- <cmd> [args...]
		return handleRun(args)
	}
}

// printError prints a formatted error message to stderr
// This provides consistent error formatting across all commands
func printError(format string, args ...interface{}) {
	fmt.Fprintf(os.Stderr, "Error: "+format+"\n", args...)
}

func printUsage() {
	fmt.Println("Usage: faa [options] <command> [args...]")
	fmt.Println()
	fmt.Println("Options:")
	fmt.Println("  -h, --help    Show this help message")
	fmt.Println()
	fmt.Println("Commands:")
	fmt.Println("  setup         Setup the development environment")
	fmt.Println("  daemon        Start the daemon process")
	fmt.Println("  run           Run a command or project (default)")
	fmt.Println("  status        Show daemon status, routes, and running processes")
	fmt.Println("  stop          Stop the daemon")
	fmt.Println("  routes        Display configured routes")
	fmt.Println("  ca-path       Show the path to the CA certificate")
	fmt.Println()
	fmt.Println("If <command> is not a recognized subcommand, it is treated as:")
	fmt.Println("  faa run -- <command> [args...]")
	fmt.Println()
	fmt.Println("Use 'faa <command> -h' for more information about a command.")
}

func printSubcommandHelp(subcommand string) {
	switch subcommand {
	case "setup":
		fmt.Println("Usage: faa setup [options]")
		fmt.Println()
		fmt.Println("Setup the development environment.")
		fmt.Println()
		fmt.Println("Options:")
		fmt.Println("  -h, --help    Show this help message")
	case "daemon":
		fmt.Println("Usage: faa daemon [options]")
		fmt.Println()
		fmt.Println("Start the daemon process.")
		fmt.Println()
		fmt.Println("Options:")
		fmt.Println("  -h, --help    Show this help message")
	case "run":
		fmt.Println("Usage: faa run [options] [-- <command> [args...]]")
		fmt.Println()
		fmt.Println("Run a command or project.")
		fmt.Println()
		fmt.Println("Options:")
		fmt.Println("  -h, --help    Show this help message")
	case "status":
		fmt.Println("Usage: faa status [options]")
		fmt.Println()
		fmt.Println("Show status of running projects.")
		fmt.Println()
		fmt.Println("Options:")
		fmt.Println("  -h, --help    Show this help message")
	case "stop":
		fmt.Println("Usage: faa stop [options]")
		fmt.Println()
		fmt.Println("Stop the daemon.")
		fmt.Println()
		fmt.Println("Options:")
		fmt.Println("  -h, --help         Show this help message")
		fmt.Println("  --clear-routes     Clear all routes when stopping")
	case "routes":
		fmt.Println("Usage: faa routes [options]")
		fmt.Println()
		fmt.Println("Display route information.")
		fmt.Println()
		fmt.Println("Options:")
		fmt.Println("  -h, --help    Show this help message")
	case "ca-path":
		fmt.Println("Usage: faa ca-path [options]")
		fmt.Println()
		fmt.Println("Show the path to the CA certificate.")
		fmt.Println()
		fmt.Println("This command displays the path where the Caddy CA root certificate")
		fmt.Println("is stored in the faa configuration directory. This certificate can")
		fmt.Println("be used to trust HTTPS connections to *.local domains.")
		fmt.Println()
		fmt.Println("Options:")
		fmt.Println("  -h, --help    Show this help message")
	default:
		// For implicit run commands, show run help
		printSubcommandHelp("run")
	}
}

func handleSetup(args []string) int {
	if err := setup.Run(); err != nil {
		printError("Setup failed: %v", err)
		return ExitError
	}
	return ExitSuccess
}

func handleDaemon(args []string) int {
	// Create registry
	registry, err := daemon.NewRegistry()
	if err != nil {
		printError("Failed to create registry: %v", err)
		return ExitError
	}

	// Create proxy
	p := proxy.New()

	// Start proxy
	ctx := context.Background()
	if err := p.Start(ctx); err != nil {
		printError("Failed to start proxy: %v", err)
		return ExitError
	}
	defer p.Stop()

	// Wait a moment for Caddy to initialize and generate CA if needed
	time.Sleep(caddyInitTimeout)

	// Export CA certificate to config directory
	if err := proxy.ExportCA(); err != nil {
		// Log warning but don't fail - CA might not be generated yet
		fmt.Fprintf(os.Stderr, "Warning: Failed to export CA certificate: %v\n", err)
		fmt.Fprintf(os.Stderr, "The CA certificate will be available after Caddy generates it.\n")
	} else {
		if caPath, err := proxy.GetCAPath(); err == nil {
			fmt.Printf("CA certificate exported to: %s\n", caPath)
		}
	}

	// Create and start daemon
	d := daemon.New(registry, p)
	if err := d.Start(); err != nil {
		printError("Failed to start daemon: %v", err)
		return ExitError
	}

	return ExitSuccess
}

// isDaemonRunning checks if the daemon is currently running
func isDaemonRunning() bool {
	client, err := daemon.Connect()
	if err != nil {
		return false
	}
	defer client.Close()

	// Try to ping the daemon
	if err := client.Ping(); err != nil {
		return false
	}

	return true
}

// startDaemonInBackground starts the daemon process in the background
func startDaemonInBackground() error {
	// Get the path to the current executable
	execPath, err := os.Executable()
	if err != nil {
		return fmt.Errorf("failed to get executable path: %w", err)
	}

	// Start daemon as a detached background process
	cmd := exec.Command(execPath, "daemon")

	// Redirect output to /dev/null to suppress daemon output
	// Users can check daemon status with 'faa status' or manually run 'faa daemon' to see output
	devNull, err := os.OpenFile(os.DevNull, os.O_WRONLY, 0)
	if err != nil {
		return fmt.Errorf("failed to open /dev/null: %w", err)
	}
	cmd.Stdout = devNull
	cmd.Stderr = devNull
	cmd.Stdin = nil

	if err := cmd.Start(); err != nil {
		devNull.Close()
		return fmt.Errorf("failed to start daemon: %w", err)
	}

	// Wait for the daemon process in a goroutine and log any errors
	go func() {
		defer devNull.Close()
		if err := cmd.Wait(); err != nil {
			// Log to stderr so users can see if daemon exits unexpectedly
			fmt.Fprintf(os.Stderr, "Warning: daemon process exited: %v\n", err)
		}
	}()

	return nil
}

// ensureDaemonRunning checks if the daemon is running, and if not, starts it
func ensureDaemonRunning() error {
	// Check if daemon is already running
	if isDaemonRunning() {
		return nil
	}

	// Start daemon in background
	if err := startDaemonInBackground(); err != nil {
		return fmt.Errorf("failed to start daemon: %w", err)
	}

	// Wait for daemon to be ready with retry logic
	deadline := time.Now().Add(daemonStartupTimeout)
	for time.Now().Before(deadline) {
		if isDaemonRunning() {
			return nil
		}
		time.Sleep(daemonStartupRetryDelay)
	}

	return fmt.Errorf("daemon failed to start within %v. The daemon may require elevated permissions. Try running 'faa setup' to configure permissions, or start the daemon manually with 'faa daemon'", daemonStartupTimeout)
}

func handleRun(args []string) int {
	// Parse arguments to find the command
	var command []string

	// Check if there's a "--" separator
	foundSeparator := false
	for i, arg := range args {
		if arg == "--" {
			// Command starts after "--"
			if i+1 < len(args) {
				command = args[i+1:]
			}
			foundSeparator = true
			break
		}
	}

	// If no "--" found, treat all args as the command
	if !foundSeparator {
		command = args
	}

	// Validate command
	if len(command) == 0 {
		printError("No command specified. Usage: faa run -- <command> [args...]")
		return ExitError
	}

	// Find project root and name
	cwd, err := os.Getwd()
	if err != nil {
		printError("Failed to get current directory: %v", err)
		return ExitError
	}

	proj, err := project.FindProjectRoot(cwd)
	if err != nil {
		printError("Failed to find project root: %v", err)
		return ExitError
	}

	// Compute host
	host := proj.Host()

	// Get lock path for this project
	lockPath := filepath.Join(proj.Root, ".faa.lock")

	// Acquire project lock
	projectLock, err := lock.Acquire(lockPath)
	if err != nil {
		printError("Failed to acquire project lock (is another instance running?): %v", err)
		return ExitError
	}
	defer projectLock.Release()

	// Ensure daemon is running (start it if needed)
	if err := ensureDaemonRunning(); err != nil {
		printError("Failed to ensure daemon is running: %v", err)
		return ExitError
	}

	// Connect to daemon
	client, err := daemon.Connect()
	if err != nil {
		printError("Failed to connect to daemon: %v", err)
		return ExitError
	}
	defer client.Close()

	// Check if process already running
	existingProc, err := client.GetProcess(proj.Root)
	if err != nil {
		printError("Failed to check for existing process: %v", err)
		return ExitError
	}

	// If process exists, check if it's still alive
	if existingProc != nil {
		if devproc.IsAlive(existingProc.PID) {
			// Process is still running
			fmt.Printf("Already running: https://%s (PID %d, port %d)\n",
				existingProc.Host, existingProc.PID, existingProc.Port)
			return ExitSuccess
		}
		// Process is dead, clean it up
		if err := client.ClearProcess(proj.Root); err != nil {
			printError("Failed to clear dead process: %v", err)
			return ExitError
		}
	}

	// Check if route already exists for this host
	existingPort, err := client.GetRoute(host)
	if err != nil {
		printError("Failed to check for existing route: %v", err)
		return ExitError
	}

	// Determine final port: use existing if available, else compute stable port
	var finalPort int
	if existingPort > 0 {
		// Reuse existing port from routes.json
		finalPort = existingPort
	} else {
		// Compute and probe for stable port
		finalPort, err = port.StablePort(proj.Name)
		if err != nil {
			printError("Failed to compute stable port: %v", err)
			return ExitError
		}
	}

	// Inject port into command
	cmdWithPort, env := devproc.InjectPort(command, finalPort)

	// Call daemon upsert_route
	if err := client.UpsertRoute(host, finalPort); err != nil {
		printError("Failed to upsert route: %v", err)
		return ExitError
	}

	// Start dev server with signal handler
	proc, err := devproc.StartWithSignalHandler(cmdWithPort, proj.Root, env)
	if err != nil {
		printError("Failed to start dev server: %v", err)
		return ExitError
	}

	// Record process in daemon registry
	startedAt := time.Now()
	if err := client.SetProcess(&daemon.SetProcessData{
		ProjectRoot: proj.Root,
		PID:         proc.PID,
		Host:        host,
		Port:        finalPort,
		StartedAt:   startedAt,
	}); err != nil {
		printError("Failed to register process: %v", err)
		// Try to stop the process we just started
		if stopErr := proc.Stop(); stopErr != nil {
			fmt.Fprintf(os.Stderr, "Warning: Failed to stop process after registration failure: %v\n", stopErr)
		}
		return ExitError
	}

	// Print URL and PID
	fmt.Printf("Started: https://%s (PID %d, port %d)\n", host, proc.PID, finalPort)

	// Wait for process to exit
	err = <-proc.Wait

	// Clear process from registry
	if clearErr := client.ClearProcess(proj.Root); clearErr != nil {
		fmt.Fprintf(os.Stderr, "Warning: Failed to clear process from registry during cleanup: %v\n", clearErr)
	}

	// Return appropriate exit code
	if err != nil {
		fmt.Fprintf(os.Stderr, "Process exited with error: %v\n", err)
		return ExitError
	}

	fmt.Println("Process exited successfully")
	return ExitSuccess
}

func handleStatus(args []string) int {
	// Connect to daemon
	client, err := daemon.Connect()
	if err != nil {
		printError("Daemon is not running. Start it with: faa daemon")
		return ExitError
	}
	defer client.Close()

	// Get status from daemon
	status, err := client.Status()
	if err != nil {
		printError("Failed to get status: %v", err)
		return ExitError
	}

	// Print daemon status
	fmt.Println("Daemon Status: Running")
	fmt.Println()

	// Print routes
	fmt.Println("Routes:")
	if len(status.Routes) == 0 {
		fmt.Println("  No routes configured")
	} else {
		for _, route := range status.Routes {
			fmt.Printf("  %s -> localhost:%d\n", route.Host, route.Port)
		}
	}
	fmt.Println()

	// Print processes
	fmt.Println("Running Processes:")
	if len(status.Processes) == 0 {
		fmt.Println("  No processes running")
	} else {
		for _, proc := range status.Processes {
			fmt.Printf("  PID %d: %s (https://%s, port %d)\n",
				proc.PID, proc.ProjectRoot, proc.Host, proc.Port)
		}
	}

	return ExitSuccess
}

func handleStop(args []string) int {
	// Parse flags
	clearRoutes := false
	for _, arg := range args {
		if arg == "--clear-routes" {
			clearRoutes = true
		}
	}

	// Connect to daemon
	client, err := daemon.Connect()
	if err != nil {
		printError("Daemon is not running")
		return ExitError
	}
	defer client.Close()

	// Send stop request
	if err := client.Stop(clearRoutes); err != nil {
		printError("Failed to stop daemon: %v", err)
		return ExitError
	}

	fmt.Println("Daemon shutdown requested")
	if clearRoutes {
		fmt.Println("Routes will be cleared")
	}

	return ExitSuccess
}

func handleRoutes(args []string) int {
	// Connect to daemon
	client, err := daemon.Connect()
	if err != nil {
		printError("Daemon is not running. Start it with: faa daemon")
		return ExitError
	}
	defer client.Close()

	// Get routes from daemon
	routes, err := client.ListRoutes()
	if err != nil {
		printError("Failed to get routes: %v", err)
		return ExitError
	}

	// Print routes
	if len(routes) == 0 {
		fmt.Println("No routes configured")
		return ExitSuccess
	}

	fmt.Println("Configured Routes:")
	for _, route := range routes {
		fmt.Printf("  %s -> localhost:%d\n", route.Host, route.Port)
	}

	return ExitSuccess
}

func handleCAPath(args []string) int {
	// Get the CA certificate path
	caPath, err := proxy.GetCAPath()
	if err != nil {
		printError("Failed to get CA certificate path: %v", err)
		return ExitError
	}

	// Check if the certificate exists
	if _, err := os.Stat(caPath); os.IsNotExist(err) {
		fmt.Println("CA certificate not yet exported.")
		fmt.Printf("Path: %s\n", caPath)
		fmt.Println()
		fmt.Println("The certificate will be created when you start the daemon.")
		fmt.Println("After starting the daemon, the certificate will be available at this path.")
		return ExitSuccess
	} else if err != nil {
		printError("Failed to check CA certificate: %v", err)
		return ExitError
	}

	// Certificate exists
	fmt.Printf("%s\n", caPath)
	return ExitSuccess
}

package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/sahithyandev/faa/internal/daemon"
	"github.com/sahithyandev/faa/internal/devproc"
	"github.com/sahithyandev/faa/internal/lock"
	"github.com/sahithyandev/faa/internal/port"
	"github.com/sahithyandev/faa/internal/project"
	"github.com/sahithyandev/faa/internal/proxy"
)

const (
	ExitSuccess = 0
	ExitError   = 1
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
	fmt.Println("  status        Show status of running projects")
	fmt.Println("  stop          Stop a running project")
	fmt.Println("  routes        Display route information")
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
		fmt.Println("Usage: faa stop [options] [project]")
		fmt.Println()
		fmt.Println("Stop a running project.")
		fmt.Println()
		fmt.Println("Options:")
		fmt.Println("  -h, --help    Show this help message")
	case "routes":
		fmt.Println("Usage: faa routes [options]")
		fmt.Println()
		fmt.Println("Display route information.")
		fmt.Println()
		fmt.Println("Options:")
		fmt.Println("  -h, --help    Show this help message")
	default:
		// For implicit run commands, show run help
		printSubcommandHelp("run")
	}
}

func handleSetup(args []string) int {
	fmt.Println("Setup command with args:", args)
	// TODO: Dispatch to internal/setup package
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

	// Create and start daemon
	d := daemon.New(registry, p)
	if err := d.Start(); err != nil {
		printError("Failed to start daemon: %v", err)
		return ExitError
	}

	return ExitSuccess
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
	
	// Compute host and stable port
	host := proj.Host()
	stablePort, err := port.StablePort(proj.Name)
	if err != nil {
		printError("Failed to compute stable port: %v", err)
		return ExitError
	}
	
	// Get lock path for this project
	lockPath := filepath.Join(proj.Root, ".faa.lock")
	
	// Acquire project lock
	projectLock, err := lock.Acquire(lockPath)
	if err != nil {
		printError("Failed to acquire project lock (is another instance running?): %v", err)
		return ExitError
	}
	defer projectLock.Release()
	
	// Connect to daemon
	client, err := daemon.Connect()
	if err != nil {
		printError("Failed to connect to daemon (is daemon running?): %v", err)
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
	
	// Inject port into command
	cmdWithPort, env := devproc.InjectPort(command, stablePort)
	
	// Call daemon upsert_route
	if err := client.UpsertRoute(host, stablePort); err != nil {
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
		Port:        stablePort,
		StartedAt:   startedAt,
	}); err != nil {
		printError("Failed to register process: %v", err)
		// Try to stop the process we just started
		proc.Stop()
		return ExitError
	}
	
	// Print URL and PID
	fmt.Printf("Started: https://%s (PID %d, port %d)\n", host, proc.PID, stablePort)
	
	// Wait for process to exit
	err = <-proc.Wait
	
	// Clear process from registry
	if clearErr := client.ClearProcess(proj.Root); clearErr != nil {
		fmt.Fprintf(os.Stderr, "Warning: Failed to clear process from registry: %v\n", clearErr)
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
	fmt.Println("Status command with args:", args)
	// TODO: Dispatch to internal package
	return ExitSuccess
}

func handleStop(args []string) int {
	fmt.Println("Stop command with args:", args)
	// TODO: Dispatch to internal package
	return ExitSuccess
}

func handleRoutes(args []string) int {
	fmt.Println("Routes command with args:", args)
	// TODO: Dispatch to internal package
	return ExitSuccess
}

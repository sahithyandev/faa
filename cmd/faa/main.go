package main

import (
	"fmt"
	"os"

	"github.com/sahithyandev/faa/internal/daemon"
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

	// Create and start daemon
	d := daemon.New(registry)
	if err := d.Start(); err != nil {
		printError("Failed to start daemon: %v", err)
		return ExitError
	}

	return ExitSuccess
}

func handleRun(args []string) int {
	fmt.Println("Run command with args:", args)
	// TODO: Dispatch to internal package
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

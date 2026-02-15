package main

import (
	"fmt"
	"os"
)

func main() {
	if len(os.Args) < 2 {
		printUsage()
		os.Exit(1)
	}

	subcommand := os.Args[1]

	switch subcommand {
	case "setup":
		handleSetup()
	case "daemon":
		handleDaemon()
	case "run":
		handleRun()
	case "status":
		handleStatus()
	case "stop":
		handleStop()
	case "routes":
		handleRoutes()
	default:
		fmt.Fprintf(os.Stderr, "Unknown subcommand: %s\n", subcommand)
		printUsage()
		os.Exit(1)
	}
}

func printUsage() {
	fmt.Println("Usage: faa <subcommand> [options]")
	fmt.Println()
	fmt.Println("Subcommands:")
	fmt.Println("  setup   - Setup the development environment")
	fmt.Println("  daemon  - Start the daemon process")
	fmt.Println("  run     - Run a project")
	fmt.Println("  status  - Show status of running projects")
	fmt.Println("  stop    - Stop a running project")
	fmt.Println("  routes  - Display route information")
}

func handleSetup() {
	fmt.Println("Setup subcommand executed")
}

func handleDaemon() {
	fmt.Println("Daemon subcommand executed")
}

func handleRun() {
	fmt.Println("Run subcommand executed")
}

func handleStatus() {
	fmt.Println("Status subcommand executed")
}

func handleStop() {
	fmt.Println("Stop subcommand executed")
}

func handleRoutes() {
	fmt.Println("Routes subcommand executed")
}

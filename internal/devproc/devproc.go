package devproc

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"sync"
	"syscall"

	"golang.org/x/sys/unix"
)

// Package devproc handles development process management

// Process represents a managed development process
type Process struct {
	PID  int
	Wait chan error
	cmd  *exec.Cmd
	ctx  context.Context
	mu   sync.Mutex
}

// Start starts a new process with the given command, working directory, and environment.
// It returns the PID and a channel that will receive an error when the process exits.
// The stdio of the child process is forwarded to the parent process.
// The process is started in a new process group for proper signal handling.
func Start(command []string, cwd string, env map[string]string) (*Process, error) {
	if len(command) == 0 {
		return nil, fmt.Errorf("command cannot be empty")
	}

	// Create command
	ctx := context.Background()
	cmd := exec.CommandContext(ctx, command[0], command[1:]...)

	// Set working directory
	if cwd != "" {
		cmd.Dir = cwd
	}

	// Set environment variables
	if env != nil {
		// Start with parent environment
		cmd.Env = os.Environ()
		// Add/override with provided env
		for k, v := range env {
			cmd.Env = append(cmd.Env, fmt.Sprintf("%s=%s", k, v))
		}
	}

	// Forward stdio to parent
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin

	// Set process group attributes for Unix systems
	// This allows us to kill the entire process tree later
	cmd.SysProcAttr = &syscall.SysProcAttr{
		Setpgid: true,
	}

	// Start the process
	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("failed to start process: %w", err)
	}

	// Create process object
	proc := &Process{
		PID:  cmd.Process.Pid,
		Wait: make(chan error, 1),
		cmd:  cmd,
		ctx:  ctx,
	}

	// Start goroutine to wait for process to finish
	go func() {
		err := cmd.Wait()
		proc.Wait <- err
		close(proc.Wait)
	}()

	return proc, nil
}

// Stop terminates the process and its entire process group.
// It sends SIGTERM to gracefully request termination.
func (p *Process) Stop() error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.cmd == nil || p.cmd.Process == nil {
		return fmt.Errorf("process not started or already stopped")
	}

	// Get process group ID (which is the same as PID since we used Setpgid)
	pgid := p.PID

	// Try to kill the entire process group with SIGTERM
	if err := unix.Kill(-pgid, syscall.SIGTERM); err != nil {
		// If the process group doesn't exist, it might have already exited
		if err == syscall.ESRCH {
			return nil
		}
		return fmt.Errorf("failed to send SIGTERM to process group: %w", err)
	}

	return nil
}

// IsAlive checks if a process with the given PID is still running.
func IsAlive(pid int) bool {
	if pid <= 0 {
		return false
	}

	// Send signal 0 to check if process exists
	// This doesn't actually send a signal, just checks permissions and existence
	err := unix.Kill(pid, syscall.Signal(0))
	return err == nil
}

// StartWithSignalHandler starts a process and sets up signal handlers for SIGINT and SIGTERM.
// When a signal is received, it terminates the child process group and performs cleanup.
// This is a convenience function that combines Start with signal handling.
func StartWithSignalHandler(command []string, cwd string, env map[string]string) (*Process, error) {
	proc, err := Start(command, cwd, env)
	if err != nil {
		return nil, err
	}

	// Set up signal handling
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	// Handle signals in a goroutine
	go func() {
		sig := <-sigChan
		fmt.Fprintf(os.Stderr, "\nReceived signal %v, terminating process...\n", sig)

		// Stop the process
		if err := proc.Stop(); err != nil {
			fmt.Fprintf(os.Stderr, "Error stopping process: %v\n", err)
		}

		// Stop listening for signals
		signal.Stop(sigChan)
	}()

	return proc, nil
}

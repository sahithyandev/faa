package devproc

import (
	"fmt"
	"os"
	"path/filepath"
	"syscall"
	"testing"
	"time"
)

func TestStart_Basic(t *testing.T) {
	// Test starting a simple command
	proc, err := Start([]string{"sleep", "0.1"}, "", nil)
	if err != nil {
		t.Fatalf("Start failed: %v", err)
	}

	if proc.PID <= 0 {
		t.Errorf("Invalid PID: %d", proc.PID)
	}

	// Wait for process to complete
	select {
	case err := <-proc.Wait:
		if err != nil {
			t.Errorf("Process exited with error: %v", err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("Process did not complete in time")
	}
}

func TestStart_EmptyCommand(t *testing.T) {
	_, err := Start([]string{}, "", nil)
	if err == nil {
		t.Error("Expected error for empty command, got nil")
	}
}

func TestStart_InvalidCommand(t *testing.T) {
	_, err := Start([]string{"nonexistent-command-xyz"}, "", nil)
	if err == nil {
		t.Error("Expected error for invalid command, got nil")
	}
}

func TestStart_WithWorkingDirectory(t *testing.T) {
	// Create a temporary directory
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test.txt")

	// Create a test file
	if err := os.WriteFile(testFile, []byte("test"), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	// Start a command that lists files in the directory
	proc, err := Start([]string{"ls", "test.txt"}, tmpDir, nil)
	if err != nil {
		t.Fatalf("Start failed: %v", err)
	}

	// Wait for process to complete
	select {
	case err := <-proc.Wait:
		if err != nil {
			t.Errorf("Process exited with error: %v", err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("Process did not complete in time")
	}
}

func TestStart_WithEnvironment(t *testing.T) {
	env := map[string]string{
		"TEST_VAR": "test_value",
	}

	// Start a command that prints the environment variable
	proc, err := Start([]string{"sh", "-c", "echo $TEST_VAR"}, "", env)
	if err != nil {
		t.Fatalf("Start failed: %v", err)
	}

	// Wait for process to complete
	select {
	case err := <-proc.Wait:
		if err != nil {
			t.Errorf("Process exited with error: %v", err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("Process did not complete in time")
	}
}

func TestStart_WithArguments(t *testing.T) {
	// Start echo command with arguments
	proc, err := Start([]string{"echo", "hello", "world"}, "", nil)
	if err != nil {
		t.Fatalf("Start failed: %v", err)
	}

	// Wait for process to complete
	select {
	case err := <-proc.Wait:
		if err != nil {
			t.Errorf("Process exited with error: %v", err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("Process did not complete in time")
	}
}

func TestStop(t *testing.T) {
	// Start a long-running process
	proc, err := Start([]string{"sleep", "60"}, "", nil)
	if err != nil {
		t.Fatalf("Start failed: %v", err)
	}

	// Give it a moment to start
	time.Sleep(100 * time.Millisecond)

	// Verify process is alive
	if !IsAlive(proc.PID) {
		t.Fatal("Process should be alive")
	}

	// Stop the process
	if err := proc.Stop(); err != nil {
		t.Fatalf("Stop failed: %v", err)
	}

	// Wait for process to exit
	select {
	case <-proc.Wait:
		// Process exited, which is expected
	case <-time.After(5 * time.Second):
		t.Fatal("Process did not exit in time")
	}

	// Give the OS time to clean up
	time.Sleep(100 * time.Millisecond)

	// Verify process is no longer alive
	if IsAlive(proc.PID) {
		t.Error("Process should not be alive after Stop")
	}
}

func TestStop_ProcessGroup(t *testing.T) {
	// Start a shell that spawns child processes
	// This tests that we properly kill the entire process group
	proc, err := Start([]string{"sh", "-c", "sleep 60 & sleep 60 & wait"}, "", nil)
	if err != nil {
		t.Fatalf("Start failed: %v", err)
	}

	// Give it a moment to start child processes
	time.Sleep(200 * time.Millisecond)

	// Stop the process
	if err := proc.Stop(); err != nil {
		t.Fatalf("Stop failed: %v", err)
	}

	// Wait for process to exit
	select {
	case <-proc.Wait:
		// Process exited, which is expected
	case <-time.After(5 * time.Second):
		t.Fatal("Process did not exit in time")
	}

	// Give the OS time to clean up
	time.Sleep(100 * time.Millisecond)

	// Verify process is no longer alive
	if IsAlive(proc.PID) {
		t.Error("Process should not be alive after Stop")
	}
}

func TestIsAlive(t *testing.T) {
	tests := []struct {
		name string
		pid  int
		want bool
	}{
		{
			name: "current process",
			pid:  os.Getpid(),
			want: true,
		},
		{
			name: "invalid negative pid",
			pid:  -1,
			want: false,
		},
		{
			name: "invalid zero pid",
			pid:  0,
			want: false,
		},
		{
			name: "non-existent high pid",
			pid:  999999,
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := IsAlive(tt.pid)
			if got != tt.want {
				t.Errorf("IsAlive(%d) = %v, want %v", tt.pid, got, tt.want)
			}
		})
	}

	// Test parent process separately since we can be sure it exists
	t.Run("parent process", func(t *testing.T) {
		ppid := os.Getppid()
		if ppid > 0 && !IsAlive(ppid) {
			t.Errorf("IsAlive(%d) = false, want true for parent process", ppid)
		}
	})
}

func TestIsAlive_ExitedProcess(t *testing.T) {
	// Start a short-lived process
	proc, err := Start([]string{"echo", "test"}, "", nil)
	if err != nil {
		t.Fatalf("Start failed: %v", err)
	}

	pid := proc.PID

	// Process should be alive initially
	if !IsAlive(pid) {
		t.Error("Process should be alive initially")
	}

	// Wait for process to exit
	select {
	case <-proc.Wait:
		// Process exited
	case <-time.After(5 * time.Second):
		t.Fatal("Process did not exit in time")
	}

	// Give the OS time to clean up
	time.Sleep(100 * time.Millisecond)

	// Process should no longer be alive
	if IsAlive(pid) {
		t.Error("Process should not be alive after exit")
	}
}

func TestStartWithSignalHandler(t *testing.T) {
	// Start a long-running process with signal handler
	proc, err := StartWithSignalHandler([]string{"sleep", "60"}, "", nil)
	if err != nil {
		t.Fatalf("StartWithSignalHandler failed: %v", err)
	}

	// Give it a moment to start
	time.Sleep(100 * time.Millisecond)

	// Verify process is alive
	if !IsAlive(proc.PID) {
		t.Fatal("Process should be alive")
	}

	// Send SIGTERM to ourselves to trigger the signal handler
	if err := syscall.Kill(os.Getpid(), syscall.SIGTERM); err != nil {
		t.Fatalf("Failed to send SIGTERM: %v", err)
	}

	// Wait for process to exit
	select {
	case <-proc.Wait:
		// Process exited, which is expected
	case <-time.After(5 * time.Second):
		// Clean up the process if test fails
		proc.Stop()
		t.Fatal("Process did not exit in time after signal")
	}
}

func TestStart_ProcessExitCode(t *testing.T) {
	tests := []struct {
		name       string
		command    []string
		wantError  bool
	}{
		{
			name:      "successful exit",
			command:   []string{"true"},
			wantError: false,
		},
		{
			name:      "failed exit",
			command:   []string{"false"},
			wantError: true,
		},
		{
			name:      "exit with code",
			command:   []string{"sh", "-c", "exit 42"},
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			proc, err := Start(tt.command, "", nil)
			if err != nil {
				t.Fatalf("Start failed: %v", err)
			}

			// Wait for process to complete
			select {
			case err := <-proc.Wait:
				if tt.wantError && err == nil {
					t.Error("Expected process to exit with error, but it succeeded")
				}
				if !tt.wantError && err != nil {
					t.Errorf("Expected process to succeed, but got error: %v", err)
				}
			case <-time.After(5 * time.Second):
				t.Fatal("Process did not complete in time")
			}
		})
	}
}

func TestStart_Concurrent(t *testing.T) {
	// Test starting multiple processes concurrently
	const numProcesses = 5
	procs := make([]*Process, numProcesses)
	
	for i := 0; i < numProcesses; i++ {
		var err error
		procs[i], err = Start([]string{"echo", fmt.Sprintf("process-%d", i)}, "", nil)
		if err != nil {
			t.Fatalf("Failed to start process %d: %v", i, err)
		}
	}

	// Wait for all processes to complete
	for i, proc := range procs {
		select {
		case err := <-proc.Wait:
			if err != nil {
				t.Errorf("Process %d exited with error: %v", i, err)
			}
		case <-time.After(5 * time.Second):
			t.Fatalf("Process %d did not complete in time", i)
		}
	}
}

func TestStop_AlreadyStopped(t *testing.T) {
	// Start and immediately stop a process
	proc, err := Start([]string{"sleep", "60"}, "", nil)
	if err != nil {
		t.Fatalf("Start failed: %v", err)
	}

	// Stop the process
	if err := proc.Stop(); err != nil {
		t.Fatalf("First Stop failed: %v", err)
	}

	// Wait for process to exit
	<-proc.Wait

	// Try to stop again - this should handle gracefully
	err = proc.Stop()
	// We expect an error or nil depending on timing
	// The important thing is it doesn't panic
	if err != nil {
		t.Logf("Second Stop returned error (expected): %v", err)
	}
}

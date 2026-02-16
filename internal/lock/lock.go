package lock

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"syscall"

	"golang.org/x/sys/unix"
)

// Package lock handles file locking mechanisms

// Lock represents a file lock handle
type Lock struct {
	file *os.File
	path string
}

// Acquire acquires an advisory lock on the specified file path.
// It creates the file if it doesn't exist and obtains an exclusive lock.
// The returned Lock must be released by calling Release() when done.
// If the lock is held by a dead process, it will be cleaned up automatically.
func Acquire(path string) (*Lock, error) {
	// Open or create the lock file
	file, err := os.OpenFile(path, os.O_CREATE|os.O_RDWR, 0666)
	if err != nil {
		return nil, fmt.Errorf("failed to open lock file: %w", err)
	}

	// Try to acquire an exclusive lock (non-blocking)
	// Use LOCK_EX for exclusive lock and LOCK_NB for non-blocking
	err = unix.Flock(int(file.Fd()), unix.LOCK_EX|unix.LOCK_NB)
	if err != nil {
		// Check if lock is held by a dead process
		if err == unix.EWOULDBLOCK || err == unix.EAGAIN {
			// Try to read PID from lock file to check if it's stale
			file.Close()
			if isLockStale(path) {
				// Remove stale lock file and try again
				_ = os.Remove(path)
				file, err = os.OpenFile(path, os.O_CREATE|os.O_RDWR, 0666)
				if err != nil {
					return nil, fmt.Errorf("failed to open lock file after cleanup: %w", err)
				}
				err = unix.Flock(int(file.Fd()), unix.LOCK_EX|unix.LOCK_NB)
				if err != nil {
					file.Close()
					return nil, fmt.Errorf("failed to acquire lock after cleanup: %w", err)
				}
			} else {
				return nil, fmt.Errorf("failed to acquire lock: %w", err)
			}
		} else {
			file.Close()
			return nil, fmt.Errorf("failed to acquire lock: %w", err)
		}
	}

	return &Lock{
		file: file,
		path: path,
	}, nil
}

// isLockStale checks if a lock file is held by a dead process
// This is a best-effort check and may not be accurate in all cases
func isLockStale(path string) bool {
	// Try to read the lock file
	data, err := os.ReadFile(path)
	if err != nil {
		// Can't read the file, assume it's not stale
		return false
	}

	// Try to parse PID from the file content
	pidStr := strings.TrimSpace(string(data))
	pid, err := strconv.Atoi(pidStr)
	if err != nil {
		// No valid PID in file, can't determine if stale
		return false
	}

	// Check if the process is alive
	if pid <= 0 {
		return true
	}

	err = unix.Kill(pid, syscall.Signal(0))
	return err != nil // If kill returns error, process is dead
}

// Release releases the lock and closes the associated file.
func (l *Lock) Release() error {
	if l.file == nil {
		return fmt.Errorf("lock already released")
	}

	// Unlock the file
	err := unix.Flock(int(l.file.Fd()), unix.LOCK_UN)
	if err != nil {
		l.file.Close()
		l.file = nil
		return fmt.Errorf("failed to release lock: %w", err)
	}

	// Close the file
	err = l.file.Close()
	l.file = nil
	if err != nil {
		return fmt.Errorf("failed to close lock file: %w", err)
	}

	return nil
}

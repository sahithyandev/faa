package lock

import (
	"fmt"
	"os"

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
		file.Close()
		return nil, fmt.Errorf("failed to acquire lock: %w", err)
	}

	return &Lock{
		file: file,
		path: path,
	}, nil
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

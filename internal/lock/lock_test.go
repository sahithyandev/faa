package lock

import (
	"os"
	"path/filepath"
	"testing"
)

func TestAcquire_Success(t *testing.T) {
	// Create a temporary directory for test files
	tmpDir := t.TempDir()
	lockPath := filepath.Join(tmpDir, "test.lock")

	// Acquire the lock
	lock, err := Acquire(lockPath)
	if err != nil {
		t.Fatalf("Acquire() failed: %v", err)
	}
	defer lock.Release()

	// Verify the lock file was created
	if _, err := os.Stat(lockPath); os.IsNotExist(err) {
		t.Errorf("Lock file was not created at %s", lockPath)
	}

	// Verify the lock object is valid
	if lock.file == nil {
		t.Error("Lock file handle is nil")
	}
	if lock.path != lockPath {
		t.Errorf("Lock path = %s, want %s", lock.path, lockPath)
	}
}

func TestAcquire_Conflict(t *testing.T) {
	// Create a temporary directory for test files
	tmpDir := t.TempDir()
	lockPath := filepath.Join(tmpDir, "test.lock")

	// Acquire the first lock
	lock1, err := Acquire(lockPath)
	if err != nil {
		t.Fatalf("First Acquire() failed: %v", err)
	}
	defer lock1.Release()

	// Try to acquire a second lock on the same file
	lock2, err := Acquire(lockPath)
	if err == nil {
		lock2.Release()
		t.Error("Second Acquire() should have failed but succeeded")
	}
}

func TestRelease_Success(t *testing.T) {
	tmpDir := t.TempDir()
	lockPath := filepath.Join(tmpDir, "test.lock")

	// Acquire the lock
	lock, err := Acquire(lockPath)
	if err != nil {
		t.Fatalf("Acquire() failed: %v", err)
	}

	// Release the lock
	err = lock.Release()
	if err != nil {
		t.Errorf("Release() failed: %v", err)
	}

	// After release, we should be able to acquire again
	lock2, err := Acquire(lockPath)
	if err != nil {
		t.Errorf("Acquire() after Release() failed: %v", err)
	}
	defer lock2.Release()
}

func TestRelease_DoubleRelease(t *testing.T) {
	tmpDir := t.TempDir()
	lockPath := filepath.Join(tmpDir, "test.lock")

	// Acquire the lock
	lock, err := Acquire(lockPath)
	if err != nil {
		t.Fatalf("Acquire() failed: %v", err)
	}

	// Release the lock
	err = lock.Release()
	if err != nil {
		t.Errorf("First Release() failed: %v", err)
	}

	// Try to release again
	err = lock.Release()
	if err == nil {
		t.Error("Second Release() should have failed but succeeded")
	}
}

func TestAcquire_CreateParentDirectory(t *testing.T) {
	// Test that Acquire works when parent directory exists
	tmpDir := t.TempDir()
	subDir := filepath.Join(tmpDir, "subdir")
	err := os.MkdirAll(subDir, 0755)
	if err != nil {
		t.Fatalf("Failed to create subdirectory: %v", err)
	}

	lockPath := filepath.Join(subDir, "test.lock")

	lock, err := Acquire(lockPath)
	if err != nil {
		t.Fatalf("Acquire() failed: %v", err)
	}
	defer lock.Release()

	// Verify the lock file was created
	if _, err := os.Stat(lockPath); os.IsNotExist(err) {
		t.Errorf("Lock file was not created at %s", lockPath)
	}
}

func TestAcquire_InvalidPath(t *testing.T) {
	// Try to acquire a lock in a non-existent directory
	lockPath := "/nonexistent/directory/test.lock"

	lock, err := Acquire(lockPath)
	if err == nil {
		lock.Release()
		t.Error("Acquire() should have failed for invalid path but succeeded")
	}
}

func TestMultipleLocks_DifferentFiles(t *testing.T) {
	// Test that multiple locks on different files work simultaneously
	tmpDir := t.TempDir()
	lockPath1 := filepath.Join(tmpDir, "test1.lock")
	lockPath2 := filepath.Join(tmpDir, "test2.lock")

	// Acquire both locks
	lock1, err1 := Acquire(lockPath1)
	if err1 != nil {
		t.Fatalf("First Acquire() failed: %v", err1)
	}
	defer lock1.Release()

	lock2, err2 := Acquire(lockPath2)
	if err2 != nil {
		t.Fatalf("Second Acquire() failed: %v", err2)
	}
	defer lock2.Release()

	// Both should be held successfully
	if lock1.file == nil || lock2.file == nil {
		t.Error("One or both locks have nil file handles")
	}
}

func TestLock_ConcurrentAccess(t *testing.T) {
	// Test that the lock prevents concurrent access
	tmpDir := t.TempDir()
	lockPath := filepath.Join(tmpDir, "test.lock")

	// Acquire the lock in main goroutine
	lock1, err := Acquire(lockPath)
	if err != nil {
		t.Fatalf("Acquire() failed: %v", err)
	}

	// Try to acquire in the same process (should fail)
	lock2, err := Acquire(lockPath)
	if err == nil {
		lock2.Release()
		t.Error("Second Acquire() should have failed but succeeded")
	}

	// Release the first lock
	err = lock1.Release()
	if err != nil {
		t.Errorf("Release() failed: %v", err)
	}

	// Now the second acquire should succeed
	lock3, err := Acquire(lockPath)
	if err != nil {
		t.Errorf("Acquire() after release failed: %v", err)
	}
	defer lock3.Release()
}

func TestLock_FilePermissions(t *testing.T) {
	tmpDir := t.TempDir()
	lockPath := filepath.Join(tmpDir, "test.lock")

	// Acquire the lock
	lock, err := Acquire(lockPath)
	if err != nil {
		t.Fatalf("Acquire() failed: %v", err)
	}
	defer lock.Release()

	// Check file permissions
	info, err := os.Stat(lockPath)
	if err != nil {
		t.Fatalf("Failed to stat lock file: %v", err)
	}

	// The file should be readable and writable (0666 or similar depending on umask)
	mode := info.Mode()
	if mode&0400 == 0 {
		t.Error("Lock file is not readable")
	}
	if mode&0200 == 0 {
		t.Error("Lock file is not writable")
	}
}

func TestLock_ReleaseAfterAcquire(t *testing.T) {
	// Test the full lifecycle: acquire, use, release
	tmpDir := t.TempDir()
	lockPath := filepath.Join(tmpDir, "project.lock")

	// Acquire
	lock, err := Acquire(lockPath)
	if err != nil {
		t.Fatalf("Acquire() failed: %v", err)
	}

	// Verify it's locked
	_, err = Acquire(lockPath)
	if err == nil {
		t.Error("Lock should be held, but second Acquire() succeeded")
	}

	// Release
	err = lock.Release()
	if err != nil {
		t.Fatalf("Release() failed: %v", err)
	}

	// Verify it's released
	lock2, err := Acquire(lockPath)
	if err != nil {
		t.Errorf("Acquire() after Release() failed: %v", err)
	} else {
		lock2.Release()
	}
}

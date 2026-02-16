package lock

import (
	"fmt"
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

func TestAcquire_StaleLockCleanup(t *testing.T) {
	tmpDir := t.TempDir()
	lockPath := filepath.Join(tmpDir, "stale.lock")

	// Write a fake PID to the lock file to simulate a stale lock
	fakePID := 999999
	err := os.WriteFile(lockPath, []byte(fmt.Sprintf("%d", fakePID)), 0666)
	if err != nil {
		t.Fatalf("Failed to write fake lock file: %v", err)
	}

	// Try to acquire the lock - should succeed after cleaning up stale lock
	lock, err := Acquire(lockPath)
	if err != nil {
		t.Fatalf("Acquire() should succeed with stale lock, got error: %v", err)
	}
	defer lock.Release()

	// Verify the lock was acquired
	if lock.file == nil {
		t.Error("Lock file handle is nil")
	}
}

func TestAcquire_StaleLockWithCurrentPID(t *testing.T) {
	tmpDir := t.TempDir()
	lockPath := filepath.Join(tmpDir, "current.lock")

	// Write current PID to the lock file
	currentPID := os.Getpid()
	err := os.WriteFile(lockPath, []byte(fmt.Sprintf("%d", currentPID)), 0666)
	if err != nil {
		t.Fatalf("Failed to write lock file: %v", err)
	}

	// First acquire should work
	lock1, err := Acquire(lockPath)
	if err != nil {
		t.Fatalf("First Acquire() failed: %v", err)
	}
	defer lock1.Release()

	// Second acquire should fail because process is still alive
	lock2, err := Acquire(lockPath)
	if err == nil {
		lock2.Release()
		t.Error("Second Acquire() should fail with live process")
	}
}

func TestIsLockStale(t *testing.T) {
	tmpDir := t.TempDir()

	// Test with fake PID (should be stale)
	fakeLockPath := filepath.Join(tmpDir, "fake.lock")
	fakePID := 999999
	err := os.WriteFile(fakeLockPath, []byte(fmt.Sprintf("%d", fakePID)), 0666)
	if err != nil {
		t.Fatalf("Failed to write fake lock file: %v", err)
	}

	if !isLockStale(fakeLockPath) {
		t.Error("Lock with fake PID should be stale")
	}

	// Test with current PID (should not be stale)
	liveLockPath := filepath.Join(tmpDir, "live.lock")
	currentPID := os.Getpid()
	err = os.WriteFile(liveLockPath, []byte(fmt.Sprintf("%d", currentPID)), 0666)
	if err != nil {
		t.Fatalf("Failed to write live lock file: %v", err)
	}

	if isLockStale(liveLockPath) {
		t.Error("Lock with current PID should not be stale")
	}

	// Test with invalid content (should not be stale - can't determine)
	invalidLockPath := filepath.Join(tmpDir, "invalid.lock")
	err = os.WriteFile(invalidLockPath, []byte("not-a-pid"), 0666)
	if err != nil {
		t.Fatalf("Failed to write invalid lock file: %v", err)
	}

	if isLockStale(invalidLockPath) {
		t.Error("Lock with invalid content should not be detected as stale")
	}

	// Test with non-existent file (should not be stale)
	nonExistentPath := filepath.Join(tmpDir, "nonexistent.lock")
	if isLockStale(nonExistentPath) {
		t.Error("Non-existent lock file should not be stale")
	}

	// Test with zero PID (should be stale)
	zeroLockPath := filepath.Join(tmpDir, "zero.lock")
	err = os.WriteFile(zeroLockPath, []byte("0"), 0666)
	if err != nil {
		t.Fatalf("Failed to write zero lock file: %v", err)
	}

	if !isLockStale(zeroLockPath) {
		t.Error("Lock with zero PID should be stale")
	}

	// Test with negative PID (should be stale)
	negativeLockPath := filepath.Join(tmpDir, "negative.lock")
	err = os.WriteFile(negativeLockPath, []byte("-1"), 0666)
	if err != nil {
		t.Fatalf("Failed to write negative lock file: %v", err)
	}

	if !isLockStale(negativeLockPath) {
		t.Error("Lock with negative PID should be stale")
	}
}

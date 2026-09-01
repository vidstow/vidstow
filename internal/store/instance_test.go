package store

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func TestAcquireInstanceLockRejectsASecondProcess(t *testing.T) {
	dir := t.TempDir()
	if err := os.Chmod(dir, 0o700); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(dir, "state.json")
	first, err := AcquireInstanceLock(path)
	if err != nil {
		t.Fatalf("first AcquireInstanceLock: %v", err)
	}
	defer first.Close()

	second, err := AcquireInstanceLock(path)
	if !errors.Is(err, ErrAlreadyRunning) || second != nil {
		t.Fatalf("second AcquireInstanceLock = (%v, %v); want ErrAlreadyRunning", second, err)
	}

	if err := first.Close(); err != nil {
		t.Fatalf("release first lock: %v", err)
	}
	third, err := AcquireInstanceLock(path)
	if err != nil {
		t.Fatalf("AcquireInstanceLock after release: %v", err)
	}
	if err := third.Close(); err != nil {
		t.Fatalf("release third lock: %v", err)
	}
}

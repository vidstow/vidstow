package store

import (
	"fmt"
	"path/filepath"
)

// ErrAlreadyRunning means another VidStow process holds the instance lock.
var ErrAlreadyRunning = fmt.Errorf("store: vidstow is already running")

// InstanceGuard is held for the life of one VidStow process so a second
// launch cannot open another queue.
type InstanceGuard struct {
	lock *stateLock
}

// AcquireInstanceLock takes a non-blocking exclusive lock beside the state file.
func AcquireInstanceLock(statePath string) (*InstanceGuard, error) {
	if statePath == "" {
		return nil, fmt.Errorf("store: empty instance lock path")
	}
	absPath, err := filepath.Abs(statePath)
	if err != nil {
		return nil, err
	}
	if !validPath(absPath) {
		return nil, fmt.Errorf("store: invalid instance lock path")
	}
	if err := ensureStateDirectory(filepath.Dir(absPath)); err != nil {
		return nil, err
	}
	canonicalPath, err := canonicalizeStatePath(absPath)
	if err != nil {
		return nil, err
	}
	lockPath := canonicalPath + ".instance"
	if !validPath(lockPath) {
		return nil, fmt.Errorf("store: instance lock path exceeds limit")
	}
	lock, err := acquireExclusiveNonBlocking(lockPath)
	if err != nil {
		return nil, err
	}
	return &InstanceGuard{lock: lock}, nil
}

// Close releases the process instance lock.
func (g *InstanceGuard) Close() error {
	if g == nil {
		return nil
	}
	if g.lock == nil {
		return nil
	}
	err := releaseStateLock(g.lock)
	g.lock = nil
	return err
}

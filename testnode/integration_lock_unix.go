//go:build !windows
// +build !windows

package testnode

import (
	"os"
	"path/filepath"
	"syscall"
)

func AcquireIntegrationTestLock() (func(), error) {
	lockPath := filepath.Join(os.TempDir(), "quorum-integration-test.lock")
	lockFile, err := os.OpenFile(lockPath, os.O_CREATE|os.O_RDWR, 0o600)
	if err != nil {
		return nil, err
	}
	if err := syscall.Flock(int(lockFile.Fd()), syscall.LOCK_EX); err != nil {
		_ = lockFile.Close()
		return nil, err
	}

	return func() {
		_ = syscall.Flock(int(lockFile.Fd()), syscall.LOCK_UN)
		_ = lockFile.Close()
	}, nil
}

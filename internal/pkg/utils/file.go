package utils

import (
	"fmt"
	"os"
	"path/filepath"
)

func CheckAndCreateDir(path string) error {
	path, err := filepath.Abs(path)
	if err != nil {
		return fmt.Errorf("filepath.Abs(%s) failed: %s", path, err)
	}

	// if path is exists, return
	if FileExist(path) {
		return fmt.Errorf("file %s is exists", path)
	}

	// if path is dir, but not empty, return
	if DirExist(path) {
		empty, err := IsDirEmpty(path)
		if err != nil {
			return err
		}
		if !empty {
			return fmt.Errorf("dir %s is not empty", path)
		}
	} else {
		// create path
		if err := os.MkdirAll(path, 0700); err != nil {
			return fmt.Errorf("os.MkdirAll(%s, 0700) failed: %s", path, err)
		}
	}

	return nil
}

// RemoveAll wrap os.RemoveAll and output log
func RemoveAll(path string) error {
	err := os.RemoveAll(path)
	if err != nil {
		logger.Errorf("remove %s failed: %s", path, err)
	}

	return err
}

package utils

import (
	"errors"
	"fmt"
	"os"

	logging "github.com/ipfs/go-log/v2"
	maddr "github.com/multiformats/go-multiaddr"
)

var logger = logging.Logger("utils")

func StringsToAddrs(addrStrings []string) (maddrs []maddr.Multiaddr, err error) {
	for _, addrString := range addrStrings {
		addr, err := maddr.NewMultiaddr(addrString)
		if err != nil {
			return maddrs, err
		}
		maddrs = append(maddrs, addr)
	}
	return
}

// EnsureDir make sure `dir` exist, or create it
func EnsureDir(dir string) error {
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		logger.Infof("try to create directory: %s", dir)
		if err := os.MkdirAll(dir, os.ModePerm); err != nil {
			logger.Errorf("make directory %s failed: %s", dir, err)
			return err
		}
	}

	if fileInfo, err := os.Stat(dir); err != nil {
		logger.Errorf("os.Stat on %s failed: %s", dir, err)
		return err
	} else if !fileInfo.IsDir() {
		msg := fmt.Sprintf("config path %s is not a directory",
			dir)
		logger.Error(msg)
		return errors.New(msg)
	}

	return nil
}

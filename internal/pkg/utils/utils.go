package utils

import (
	"crypto/rand"
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

// FileExist check if file is exist
func FileExist(path string) bool {
	info, err := os.Stat(path)
	if os.IsNotExist(err) {
		return false
	}

	return !info.IsDir()
}

// DirExist check if file is exist
func DirExist(path string) bool {
	info, err := os.Stat(path)
	if os.IsNotExist(err) {
		return false
	}

	return info.IsDir()
}

// EnsureDir make sure `dir` exist, or create it
func EnsureDir(dir string) error {
	if !DirExist(dir) {
		logger.Infof("try to create directory: %s", dir)
		if err := os.MkdirAll(dir, os.ModePerm); err != nil {
			logger.Errorf("make directory %s failed: %s", dir, err)
			return err
		}
	}

	return nil
}

func GetRandomStr(n int) string {
	const letters = "0123456789abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"
	const lettersLength = int64(len(letters))

	b := make([]byte, n)
	for i := range b {
		b[i] = letters[rand.Read(b)%lettersLength]
	}
	return string(b)

}

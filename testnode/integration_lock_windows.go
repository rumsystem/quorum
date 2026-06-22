//go:build windows
// +build windows

package testnode

func AcquireIntegrationTestLock() (func(), error) {
	return func() {}, nil
}

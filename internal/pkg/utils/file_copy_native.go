//go:build !js
// +build !js

package utils

import (
	cp "github.com/otiai10/copy"
)

// Copy copies src to dest, doesn't matter if src is a directory or a file.
func Copy(src string, dst string, opt ...cp.Options) error {
	return cp.Copy(src, dst, opt...)
}

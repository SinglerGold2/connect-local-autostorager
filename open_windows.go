//go:build windows

package main

import "os"

// openRegular opens path. Windows has no O_NOFOLLOW here, so the lstat before
// the open and the descriptor's Stat after it are the checks.
func openRegular(path string) (*os.File, error) {
	return os.Open(path)
}

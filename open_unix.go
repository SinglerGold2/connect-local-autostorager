//go:build unix

package main

import (
	"os"
	"syscall"
)

// openRegular opens path without following a symbolic link. regularFile and
// the descriptor's Stat are the checks; O_NOFOLLOW closes the moment between
// them, when a file could be replaced by a link.
func openRegular(path string) (*os.File, error) {
	return os.OpenFile(path, os.O_RDONLY|syscall.O_NOFOLLOW, 0)
}

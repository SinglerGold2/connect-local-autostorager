package main

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

// makeSymlink creates a link or skips a platform that will not let the test
// make one. Windows needs a privilege for this, so the security check is
// tested where the fixture can be built.
func makeSymlink(t *testing.T, target, link string) {
	t.Helper()

	if err := os.Symlink(target, link); err != nil {
		t.Skipf("symbolic links are not available here: %v", err)
	}
}

func TestRegularFileRefusesSymbolicLinks(t *testing.T) {
	dir := t.TempDir()
	secret := filepath.Join(dir, "secret.txt")
	if err := os.WriteFile(secret, []byte("server configuration"), 0o600); err != nil {
		t.Fatal(err)
	}

	link := filepath.Join(dir, "x.gcode")
	makeSymlink(t, secret, link)

	if _, err := regularFile(link); !errors.Is(err, errNotRegular) {
		t.Fatalf("regularFile(symlink) = %v, want errNotRegular", err)
	}

	info, err := regularFile(secret)
	if err != nil {
		t.Fatalf("regularFile(regular) = %v", err)
	}

	if !info.Mode().IsRegular() {
		t.Fatalf("regularFile(regular) mode = %v, want regular", info.Mode())
	}
}

// TestScanSkipsSymbolicLinks is the regression for the review finding: a link
// with a print-file name must not become ready, and must not reach the store.
func TestScanSkipsSymbolicLinks(t *testing.T) {
	dir := t.TempDir()
	secret := filepath.Join(dir, "secret.txt")
	if err := os.WriteFile(secret, []byte("server configuration"), 0o600); err != nil {
		t.Fatal(err)
	}

	link := filepath.Join(dir, "x.gcode")
	makeSymlink(t, secret, link)

	sizes := map[string]int64{}
	stable := map[string]int{}
	for range sizeSettle {
		scan(context.Background(), Settings{}, dir, sizes, stable)
	}

	if len(sizes) != 0 || len(stable) != 0 {
		t.Fatalf("a symbolic link was counted as a print file: sizes=%v stable=%v", sizes, stable)
	}
}

func TestOpenRegularRefusesSymbolicLinks(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Windows has no O_NOFOLLOW here; the lstat and the descriptor check are the guard")
	}

	dir := t.TempDir()
	secret := filepath.Join(dir, "secret.txt")
	if err := os.WriteFile(secret, []byte("server configuration"), 0o600); err != nil {
		t.Fatal(err)
	}

	link := filepath.Join(dir, "x.gcode")
	makeSymlink(t, secret, link)

	body, err := openRegular(link)
	if err == nil {
		_ = body.Close()
		t.Fatal("openRegular followed a symbolic link")
	}
}

func TestMoveToRefusesSymbolicLinks(t *testing.T) {
	dir := t.TempDir()
	secret := filepath.Join(dir, "secret.txt")
	if err := os.WriteFile(secret, []byte("server configuration"), 0o600); err != nil {
		t.Fatal(err)
	}

	link := filepath.Join(dir, "x.gcode")
	makeSymlink(t, secret, link)

	destination := filepath.Join(dir, "successful", "x.gcode")
	if err := moveTo(link, destination); !errors.Is(err, errNotRegular) {
		t.Fatalf("moveTo(symlink) = %v, want errNotRegular", err)
	}

	if _, err := os.Lstat(destination); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("moveTo created %s from a symbolic link", destination)
	}
}

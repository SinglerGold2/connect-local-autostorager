package main

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"
)

// makeSymlink creates a link or skips a platform that will not let the test
// make one. Windows needs a privilege for this, so the check is tested where
// the fixture can be built.
func makeSymlink(t *testing.T, target, link string) {
	t.Helper()

	if err := os.Symlink(target, link); err != nil {
		t.Skipf("symbolic links are not available here: %v", err)
	}
}

func openTestRoot(t *testing.T, dir string) *os.Root {
	t.Helper()

	root, err := os.OpenRoot(dir)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = root.Close() })

	return root
}

func TestRegularFileRefusesSymbolicLinks(t *testing.T) {
	dir := t.TempDir()
	secret := filepath.Join(dir, "secret.txt")
	if err := os.WriteFile(secret, []byte("server configuration"), 0o600); err != nil {
		t.Fatal(err)
	}

	link := filepath.Join(dir, "x.gcode")
	makeSymlink(t, secret, link)

	root := openTestRoot(t, dir)

	if _, err := regularFile(root, "x.gcode"); !errors.Is(err, errNotRegular) {
		t.Fatalf("regularFile(symlink) = %v, want errNotRegular", err)
	}

	info, err := regularFile(root, "secret.txt")
	if err != nil {
		t.Fatalf("regularFile(regular) = %v", err)
	}

	if !info.Mode().IsRegular() {
		t.Fatalf("regularFile(regular) mode = %v, want regular", info.Mode())
	}
}

// TestScanSkipsSymbolicLinks is the source-side regression: a link with a
// print-file name must not become ready, and must not reach the store.
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

func TestMoveToRefusesSymbolicLinks(t *testing.T) {
	dir := t.TempDir()
	secret := filepath.Join(dir, "secret.txt")
	if err := os.WriteFile(secret, []byte("server configuration"), 0o600); err != nil {
		t.Fatal(err)
	}

	link := filepath.Join(dir, "x.gcode")
	makeSymlink(t, secret, link)

	root := openTestRoot(t, dir)

	if _, err := moveTo(root, "x.gcode", filepath.Join("successful", "x.gcode")); !errors.Is(err, errNotRegular) {
		t.Fatalf("moveTo(symlink) = %v, want errNotRegular", err)
	}

	if _, err := os.Lstat(filepath.Join(dir, "successful", "x.gcode")); !errors.Is(err, os.ErrNotExist) {
		t.Fatal("moveTo created a destination from a symbolic link")
	}
}

// TestFailureReportDoesNotFollowSymlink is the destination-side regression
// from the review: a failure report pre-created as a link must not make
// fail() truncate the file it points at.
func TestFailureReportDoesNotFollowSymlink(t *testing.T) {
	dir := t.TempDir()
	outside := t.TempDir()
	secret := filepath.Join(outside, "secret.txt")
	const content = "server database"

	if err := os.WriteFile(secret, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}

	if err := os.MkdirAll(filepath.Join(dir, "failed"), 0o755); err != nil {
		t.Fatal(err)
	}

	report := filepath.Join(dir, "failed", "x.gcode-failure.txt")
	makeSymlink(t, secret, report)

	if err := os.WriteFile(filepath.Join(dir, "x.gcode"), make([]byte, 2<<20), 0o600); err != nil {
		t.Fatal(err)
	}

	root := openTestRoot(t, dir)
	importOne(context.Background(), Settings{MaxMB: 1}, dir, root, "x.gcode")

	got, err := os.ReadFile(secret)
	if err != nil {
		t.Fatal(err)
	}

	if string(got) != content {
		t.Fatalf("the failure report followed the link and changed its target: %q", got)
	}

	info, err := os.Lstat(filepath.Join(dir, "failed", "x.gcode"))
	if err != nil || !info.Mode().IsRegular() {
		t.Fatalf("the source was not moved to failed/: %v", err)
	}
}

// TestFailedDirectorySymlinkDoesNotEscape checks that a failed/ link to a
// directory outside the watched folder cannot receive the moved file.
func TestFailedDirectorySymlinkDoesNotEscape(t *testing.T) {
	dir := t.TempDir()
	outside := t.TempDir()

	makeSymlink(t, outside, filepath.Join(dir, "failed"))

	if err := os.WriteFile(filepath.Join(dir, "x.gcode"), make([]byte, 2<<20), 0o600); err != nil {
		t.Fatal(err)
	}

	root := openTestRoot(t, dir)
	importOne(context.Background(), Settings{MaxMB: 1}, dir, root, "x.gcode")

	for _, name := range []string{"x.gcode", "x.gcode-failure.txt"} {
		if _, err := os.Lstat(filepath.Join(outside, name)); !errors.Is(err, os.ErrNotExist) {
			t.Fatalf("a failed/ symlink wrote %s outside the watched folder", name)
		}
	}

	if info, err := os.Lstat(filepath.Join(dir, "x.gcode")); err != nil || !info.Mode().IsRegular() {
		t.Fatalf("the source should stay in the watched folder when the move escapes: %v", err)
	}
}

// TestSuccessfulDirectorySymlinkDoesNotEscape is the same check for the move
// after a committed import.
func TestSuccessfulDirectorySymlinkDoesNotEscape(t *testing.T) {
	dir := t.TempDir()
	outside := t.TempDir()

	makeSymlink(t, outside, filepath.Join(dir, "successful"))

	if err := os.WriteFile(filepath.Join(dir, "x.gcode"), []byte("print"), 0o600); err != nil {
		t.Fatal(err)
	}

	root := openTestRoot(t, dir)
	if _, err := moveTo(root, "x.gcode", filepath.Join("successful", "x.gcode")); err == nil {
		t.Fatal("moveTo accepted a successful/ symlink that leaves the watched folder")
	}

	if _, err := os.Lstat(filepath.Join(outside, "x.gcode")); !errors.Is(err, os.ErrNotExist) {
		t.Fatal("a successful/ symlink moved the file outside the watched folder")
	}

	if info, err := os.Lstat(filepath.Join(dir, "x.gcode")); err != nil || !info.Mode().IsRegular() {
		t.Fatalf("the source should stay in the watched folder when the move escapes: %v", err)
	}
}

// TestCopyFallbackRefusesExistingDestination is the cross-device half of the
// move: an existing file or symlink at the destination must not be truncated
// or followed.
func TestCopyFallbackRefusesExistingDestination(t *testing.T) {
	dir := t.TempDir()
	outside := t.TempDir()

	source := filepath.Join(dir, "x.gcode")
	if err := os.WriteFile(source, []byte("new print"), 0o600); err != nil {
		t.Fatal(err)
	}

	secret := filepath.Join(outside, "secret.txt")
	if err := os.WriteFile(secret, []byte("server database"), 0o600); err != nil {
		t.Fatal(err)
	}

	link := filepath.Join(dir, "x.gcode-copy")
	makeSymlink(t, secret, link)

	root := openTestRoot(t, dir)
	if err := copyTo(root, "x.gcode", "x.gcode-copy"); err == nil {
		t.Fatal("copyTo accepted an existing symlink at the destination")
	}

	got, err := os.ReadFile(secret)
	if err != nil {
		t.Fatal(err)
	}

	if string(got) != "server database" {
		t.Fatalf("copyTo followed the destination link and changed its target: %q", got)
	}

	if info, err := os.Lstat(source); err != nil || !info.Mode().IsRegular() {
		t.Fatalf("copyTo removed the source after a refused destination: %v", err)
	}
}

// TestMoveToRenamesExistingDestination keeps both source files when the
// destination name is already taken.
func TestMoveToRenamesExistingDestination(t *testing.T) {
	dir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(dir, "successful"), 0o755); err != nil {
		t.Fatal(err)
	}

	if err := os.WriteFile(filepath.Join(dir, "successful", "test.gcode"), []byte("old"), 0o600); err != nil {
		t.Fatal(err)
	}

	if err := os.WriteFile(filepath.Join(dir, "test.gcode"), []byte("new"), 0o600); err != nil {
		t.Fatal(err)
	}

	root := openTestRoot(t, dir)
	moved, err := moveTo(root, "test.gcode", filepath.Join("successful", "test.gcode"))
	if err != nil {
		t.Fatal(err)
	}

	if moved != filepath.Join("successful", "test-1.gcode") {
		t.Fatalf("moved to %q, want successful/test-1.gcode", moved)
	}

	old, err := os.ReadFile(filepath.Join(dir, "successful", "test.gcode"))
	if err != nil || string(old) != "old" {
		t.Fatalf("existing destination was changed: %q, %v", old, err)
	}

	fresh, err := os.ReadFile(filepath.Join(dir, "successful", "test-1.gcode"))
	if err != nil || string(fresh) != "new" {
		t.Fatalf("renamed source is wrong: %q, %v", fresh, err)
	}
}

// TestScanMarksMissingSourceDown keeps an unreachable source out of the
// recent-import list and reports it as a status instead.
func TestScanMarksMissingSourceDown(t *testing.T) {
	view = &shared{}

	missing := filepath.Join(t.TempDir(), "missing")
	scan(context.Background(), Settings{}, missing, map[string]int64{}, map[string]int{})

	if got := recentImports(); len(got) != 0 {
		t.Fatalf("a missing source wrote %d recent imports", len(got))
	}

	if got := logEntries(); len(got) != 0 {
		t.Fatalf("a missing source wrote %d logs", len(got))
	}

	statuses := sourceStatuses()
	if len(statuses) != 1 {
		t.Fatalf("sourceStatuses() = %d entries, want 1", len(statuses))
	}

	if statuses[0].State != "down" {
		t.Fatalf("missing source state = %q, want down", statuses[0].State)
	}

	if statuses[0].Detail == "" {
		t.Fatal("missing source has no detail")
	}
}

func TestScanMarksSourceOnline(t *testing.T) {
	view = &shared{}

	dir := t.TempDir()
	scan(context.Background(), Settings{}, dir, map[string]int64{}, map[string]int{})

	statuses := sourceStatuses()
	if len(statuses) != 1 || statuses[0].State != "online" {
		t.Fatalf("sourceStatuses() = %+v, want one online source", statuses)
	}
}

func TestNoteSplitsRecentAndLogs(t *testing.T) {
	view = &shared{}

	note(Import{At: "2026-11-25T04:00:00Z", Source: "a.gcode", Result: "imported"})
	note(Import{At: "2026-11-25T04:01:00Z", Source: "b.gcode", Result: "failed"})
	noteLog(Import{At: "2026-11-25T04:02:00Z", Source: "c.gcode", Result: "imported", Detail: "could not move the source"})

	if got := recentImports(); len(got) != 1 || got[0].Source != "a.gcode" {
		t.Fatalf("recentImports() = %+v, want only the imported entry", got)
	}

	if got := logEntries(); len(got) != 2 {
		t.Fatalf("logEntries() = %d entries, want 2", len(got))
	}
}

func TestSourceUnavailable(t *testing.T) {
	if !sourceUnavailable(os.ErrNotExist) {
		t.Fatal("os.ErrNotExist should be a down source")
	}

	if !sourceUnavailable(errors.New("readdir /app/config/autostorager-g: host is down")) {
		t.Fatal("host is down should be a down source")
	}

	if sourceUnavailable(errors.New("readdir /app/config/autostorager-g: permission denied")) {
		t.Fatal("permission denied should not be a down source")
	}
}

package main

// The watchers: one loop per supervised folder. Every file is handed to the
// storage, and the source moves into the folder's own successful/ or failed/
// only after the storage has the bytes.

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/prusa3d/prusa-connect/sdk/storageapi"
)

// recentLimit bounds the list the page and the card read.
const recentLimit = 20

// sizeSettle is how many scans a file's size must hold before it is read.
const sizeSettle = 2

// errNoFile is what the storage answers for a name it does not hold.
var errNoFile = storageapi.ErrNoFile

// errNotRegular is a path that is not a regular file: a directory, a device,
// a pipe, a socket or a symbolic link.
var errNotRegular = errors.New("not a regular file")

// Import is one line of the page's list and of the card.
type Import struct {
	At     string `json:"at"`
	Folder string `json:"folder"`
	Source string `json:"source"`
	Target string `json:"target"`
	Result string `json:"result"`
	Detail string `json:"detail,omitempty"`
}

// shared is what every watcher and the page read together.
type shared struct {
	mu      sync.Mutex
	recent  []Import
	scans   int64
	imports int64
	failed  int64
	lastAt  string
}

var (
	// store is the storage client of this session, set once the session exists.
	store *storageapi.Client

	// watching cancels every loop, so a new configuration replaces the old set
	// instead of stacking another one.
	watching []context.CancelFunc

	// view is the one piece of state the module keeps for its page.
	view = &shared{}

	// watcherMu guards the set of loops against two configurations arriving at
	// once.
	watcherMu sync.Mutex
)

// allowed reports whether a name is a print file this module takes.
func allowed(name string) bool {
	lower := strings.ToLower(name)

	return strings.HasSuffix(lower, ".gcode") || strings.HasSuffix(lower, ".bgcode")
}

// regularFile returns the lstat of name when it is a regular file. The root
// makes every lookup relative to the watched folder; a symbolic link is
// reported as itself, never as the file it points at.
func regularFile(root *os.Root, name string) (os.FileInfo, error) {
	info, err := root.Lstat(name)
	if err != nil {
		return nil, err
	}

	if !info.Mode().IsRegular() {
		return nil, errNotRegular
	}

	return info, nil
}

// startWatcher stops every running loop and starts one per folder.
func startWatcher(s Settings) {
	watcherMu.Lock()
	defer watcherMu.Unlock()

	for _, cancel := range watching {
		cancel()
	}

	watching = nil

	for _, folder := range s.Folders {
		if strings.TrimSpace(folder) == "" {
			continue
		}

		ctx, cancel := context.WithCancel(context.Background())
		watching = append(watching, cancel)

		go loop(ctx, s, folder)
	}
}

// loop reads one folder until it is replaced or the module stops.
func loop(ctx context.Context, s Settings, folder string) {
	interval := time.Duration(s.IntervalSeconds) * time.Second
	timer := time.NewTimer(0)
	defer timer.Stop()

	sizes := map[string]int64{}
	stable := map[string]int{}

	for {
		select {
		case <-ctx.Done():
			return
		case <-timer.C:
		}

		scan(ctx, s, folder, sizes, stable)
		timer.Reset(interval)
	}
}

// scan reads one folder once and imports what is ready. Every path below the
// folder goes through one os.Root, so no operation can leave it.
func scan(ctx context.Context, s Settings, folder string, sizes map[string]int64, stable map[string]int) {
	root, err := os.OpenRoot(folder)
	if err != nil {
		note(Import{At: now(), Folder: folder, Source: folder, Result: "error", Detail: err.Error()})

		return
	}
	defer func() { _ = root.Close() }()

	dir, err := root.Open(".")
	if err != nil {
		note(Import{At: now(), Folder: folder, Source: folder, Result: "error", Detail: err.Error()})

		return
	}

	entries, err := dir.ReadDir(-1)
	_ = dir.Close()

	if err != nil {
		note(Import{At: now(), Folder: folder, Source: folder, Result: "error", Detail: err.Error()})

		return
	}

	view.mu.Lock()
	view.scans++
	view.lastAt = now()
	view.mu.Unlock()

	ready := make([]string, 0, len(entries))

	for _, entry := range entries {
		if entry.IsDir() || !allowed(entry.Name()) {
			continue
		}

		name := entry.Name()

		// Lstat through the root: a symbolic link is the link itself, so a
		// link with a print-file name is skipped and never read.
		info, infoErr := root.Lstat(name)
		if infoErr != nil || !info.Mode().IsRegular() {
			continue
		}

		if sizes[name] == info.Size() {
			stable[name]++
		} else {
			sizes[name] = info.Size()
			stable[name] = 1
		}

		if stable[name] >= sizeSettle {
			ready = append(ready, name)
		}
	}

	for _, name := range ready {
		importOne(ctx, s, folder, root, name)

		delete(sizes, name)
		delete(stable, name)
	}
}

// importOne copies one file into the storage and moves the source afterwards.
// The root keeps the read, the storage write and the move on the same tree.
func importOne(ctx context.Context, s Settings, folder string, root *os.Root, name string) {
	path := filepath.Join(folder, name)

	info, err := regularFile(root, name)
	if err != nil {
		if errors.Is(err, errNotRegular) {
			note(Import{At: now(), Folder: folder, Source: path, Result: "error", Detail: "skipped: not a regular file"})

			return
		}

		fail(root, s, folder, name, "reading the file", err)

		return
	}

	if info.Size() > int64(s.MaxMB)<<20 {
		fail(root, s, folder, name, "the file is larger than the limit",
			fmt.Errorf("%d MB is more than %d MB", info.Size()>>20, s.MaxMB))

		return
	}

	target, err := freeName(ctx, s.Storage, name)
	if err != nil {
		fail(root, s, folder, name, "looking for a free name", err)

		return
	}

	body, err := root.Open(name)
	if err != nil {
		if _, statErr := regularFile(root, name); statErr != nil {
			note(Import{At: now(), Folder: folder, Source: path, Result: "error", Detail: "skipped: not a regular file"})

			return
		}

		fail(root, s, folder, name, "opening the file", err)

		return
	}

	// The descriptor is what is read, so its mode and size decide, not the
	// directory entry that may have been swapped since the scan.
	opened, err := body.Stat()
	if err != nil {
		_ = body.Close()
		fail(root, s, folder, name, "reading the opened file", err)

		return
	}

	if !opened.Mode().IsRegular() {
		_ = body.Close()
		note(Import{At: now(), Folder: folder, Source: path, Result: "error", Detail: "skipped: not a regular file"})

		return
	}

	if opened.Size() > int64(s.MaxMB)<<20 {
		_ = body.Close()
		fail(root, s, folder, name, "the file is larger than the limit",
			fmt.Errorf("%d MB is more than %d MB", opened.Size()>>20, s.MaxMB))

		return
	}

	written, err := store.Write(ctx, storageapi.WriteParams{Storage: s.Storage, Path: target, Size: opened.Size()}, body)
	_ = body.Close()

	if err != nil {
		fail(root, s, folder, name, "writing into the storage", err)

		return
	}

	view.mu.Lock()
	view.imports++
	view.recent = append(view.recent, Import{At: now(), Folder: folder, Source: path, Target: written.Path, Result: "imported"})
	view.recent = trim(view.recent)
	view.mu.Unlock()

	if s.MoveOnSuccess {
		if moveErr := moveTo(root, name, filepath.Join("successful", name)); moveErr != nil {
			note(Import{At: now(), Folder: folder, Source: path, Target: written.Path, Result: "imported", Detail: "could not move the source: " + moveErr.Error()})
		}
	}
}

// freeName finds a name the storage does not hold yet.
func freeName(ctx context.Context, storage, name string) (string, error) {
	ext := filepath.Ext(name)
	base := strings.TrimSuffix(name, ext)

	for attempt := 0; attempt < 1000; attempt++ {
		candidate := name
		if attempt > 0 {
			candidate = fmt.Sprintf("%s-%d%s", base, attempt, ext)
		}

		_, err := store.Stat(ctx, storageapi.StatParams{Storage: storage, Path: candidate})
		if errors.Is(err, errNoFile) {
			return candidate, nil
		}

		if err != nil {
			return "", err
		}
	}

	return "", fmt.Errorf("no free name for %s", name)
}

// fail moves a source it could not import into failed/ and writes what went
// wrong beside it. The move and the report stay under the watched folder, and
// the report is created exclusively so an existing symlink is never followed.
func fail(root *os.Root, s Settings, folder, name, step string, cause error) {
	path := filepath.Join(folder, name)

	if info, statErr := root.Lstat(name); statErr == nil && !info.Mode().IsRegular() {
		note(Import{At: now(), Folder: folder, Source: path, Result: "error", Detail: step + ": not a regular file"})

		return
	}

	view.mu.Lock()
	view.failed++
	view.mu.Unlock()

	reason := fmt.Sprintf("Step: %s\nReason: %v\nTime: %s\nSource: %s\n", step, cause, now(), path)
	note(Import{At: now(), Folder: folder, Source: path, Result: "failed", Detail: step + ": " + cause.Error()})

	destination := filepath.Join("failed", name)
	if moveErr := moveTo(root, name, destination); moveErr != nil {
		note(Import{At: now(), Folder: folder, Source: path, Result: "failed", Detail: "could not move the source: " + moveErr.Error()})

		return
	}

	report := filepath.Join("failed", name+"-failure.txt")
	body, err := root.OpenFile(report, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600)
	if err != nil {
		note(Import{At: now(), Folder: folder, Source: path, Result: "failed", Detail: "could not write the failure report: " + err.Error()})

		return
	}

	if _, err = body.WriteString(reason); err != nil {
		_ = body.Close()
		note(Import{At: now(), Folder: folder, Source: path, Result: "failed", Detail: "could not write the failure report: " + err.Error()})

		return
	}

	if err = body.Close(); err != nil {
		note(Import{At: now(), Folder: folder, Source: path, Result: "failed", Detail: "could not write the failure report: " + err.Error()})
	}
}

// moveTo moves a file, copying when the two paths are on different devices.
// Both ends are relative to the watched folder and go through its root.
func moveTo(root *os.Root, from, to string) error {
	info, err := root.Lstat(from)
	if err != nil {
		return err
	}

	if !info.Mode().IsRegular() {
		return errNotRegular
	}

	if err := root.MkdirAll(filepath.Dir(to), 0o755); err != nil {
		return err
	}

	if err := root.Rename(from, to); err == nil {
		return nil
	}

	source, err := root.Open(from)
	if err != nil {
		return err
	}

	defer func() { _ = source.Close() }()

	target, err := root.OpenFile(to, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0o666)
	if err != nil {
		return err
	}

	if _, err = io.Copy(target, source); err != nil {
		_ = target.Close()
		_ = root.Remove(to)

		return err
	}

	if err = target.Close(); err != nil {
		return err
	}

	return root.Remove(from)
}

// note keeps one line for the page.
func note(entry Import) {
	view.mu.Lock()
	defer view.mu.Unlock()

	view.recent = append(view.recent, entry)
	view.recent = trim(view.recent)
}

// trim keeps the newest lines.
func trim(lines []Import) []Import {
	if len(lines) > recentLimit {
		return lines[len(lines)-recentLimit:]
	}

	return lines
}

// recentImports is the list the page reads.
func recentImports() []Import {
	view.mu.Lock()
	defer view.mu.Unlock()

	out := make([]Import, len(view.recent))
	copy(out, view.recent)

	return out
}

// now is the timestamp the page shows.
func now() string {
	return time.Now().Format(time.RFC3339)
}

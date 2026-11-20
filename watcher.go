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

// regularFile returns the lstat of path when it is a regular file. A symbolic
// link is reported as itself, never as the file it points at, so a link that
// carries a .gcode name cannot make the module read a database or a config.
func regularFile(path string) (os.FileInfo, error) {
	info, err := os.Lstat(path)
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

// scan reads one folder once and imports what is ready.
func scan(ctx context.Context, s Settings, folder string, sizes map[string]int64, stable map[string]int) {
	entries, err := os.ReadDir(folder)
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

		// Info is an lstat: for a symbolic link it reports the link, not the
		// file it points at, so a link with a print-file name is skipped.
		info, infoErr := entry.Info()
		if infoErr != nil || !info.Mode().IsRegular() {
			continue
		}

		name := entry.Name()
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
		importOne(ctx, s, folder, filepath.Join(folder, name))

		delete(sizes, name)
		delete(stable, name)
	}
}

// importOne copies one file into the storage and moves the source afterwards.
func importOne(ctx context.Context, s Settings, folder, path string) {
	name := filepath.Base(path)

	info, err := regularFile(path)
	if err != nil {
		if errors.Is(err, errNotRegular) {
			note(Import{At: now(), Folder: folder, Source: path, Result: "error", Detail: "skipped: not a regular file"})

			return
		}

		fail(s, folder, path, name, "reading the file", err)

		return
	}

	if info.Size() > int64(s.MaxMB)<<20 {
		fail(s, folder, path, name, "the file is larger than the limit",
			fmt.Errorf("%d MB is more than %d MB", info.Size()>>20, s.MaxMB))

		return
	}

	target, err := freeName(ctx, s.Storage, name)
	if err != nil {
		fail(s, folder, path, name, "looking for a free name", err)

		return
	}

	body, err := openRegular(path)
	if err != nil {
		// A file swapped for a symbolic link between the scan and the open is
		// not moved into failed/, which would copy its target across devices.
		if _, statErr := regularFile(path); statErr != nil {
			note(Import{At: now(), Folder: folder, Source: path, Result: "error", Detail: "skipped: not a regular file"})

			return
		}

		fail(s, folder, path, name, "opening the file", err)

		return
	}

	// The descriptor is what is read, so its mode and size decide, not the
	// directory entry that may have been swapped since the scan.
	opened, err := body.Stat()
	if err != nil {
		_ = body.Close()
		fail(s, folder, path, name, "reading the opened file", err)

		return
	}

	if !opened.Mode().IsRegular() {
		_ = body.Close()
		note(Import{At: now(), Folder: folder, Source: path, Result: "error", Detail: "skipped: not a regular file"})

		return
	}

	if opened.Size() > int64(s.MaxMB)<<20 {
		_ = body.Close()
		fail(s, folder, path, name, "the file is larger than the limit",
			fmt.Errorf("%d MB is more than %d MB", opened.Size()>>20, s.MaxMB))

		return
	}

	written, err := store.Write(ctx, storageapi.WriteParams{Storage: s.Storage, Path: target, Size: opened.Size()}, body)
	_ = body.Close()

	if err != nil {
		fail(s, folder, path, name, "writing into the storage", err)

		return
	}

	view.mu.Lock()
	view.imports++
	view.recent = append(view.recent, Import{At: now(), Folder: folder, Source: path, Target: written.Path, Result: "imported"})
	view.recent = trim(view.recent)
	view.mu.Unlock()

	if s.MoveOnSuccess {
		if moveErr := moveTo(path, filepath.Join(folder, "successful", name)); moveErr != nil {
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
// wrong beside it.
func fail(s Settings, folder, path, name, step string, cause error) {
	// fail runs after a read or an open; never move a non-regular path, because
	// the cross-device branch of moveTo would copy the link's target.
	if info, statErr := os.Lstat(path); statErr == nil && !info.Mode().IsRegular() {
		note(Import{At: now(), Folder: folder, Source: path, Result: "error", Detail: step + ": not a regular file"})

		return
	}

	view.mu.Lock()
	view.failed++
	view.mu.Unlock()

	reason := fmt.Sprintf("Step: %s\nReason: %v\nTime: %s\nSource: %s\n", step, cause, now(), path)
	note(Import{At: now(), Folder: folder, Source: path, Result: "failed", Detail: step + ": " + cause.Error()})

	destination := filepath.Join(folder, "failed", name)
	if moveErr := moveTo(path, destination); moveErr != nil {
		note(Import{At: now(), Folder: folder, Source: path, Result: "failed", Detail: "could not move the source: " + moveErr.Error()})

		return
	}

	_ = os.WriteFile(destination+"-failure.txt", []byte(reason), 0o600)
}

// moveTo moves a file, copying when the two paths are on different devices.
func moveTo(from, to string) error {
	info, err := os.Lstat(from)
	if err != nil {
		return err
	}

	if !info.Mode().IsRegular() {
		return errNotRegular
	}

	if err := os.MkdirAll(filepath.Dir(to), 0o755); err != nil {
		return err
	}

	if err := os.Rename(from, to); err == nil {
		return nil
	}

	source, err := openRegular(from)
	if err != nil {
		return err
	}

	defer func() { _ = source.Close() }()

	target, err := os.Create(to)
	if err != nil {
		return err
	}

	if _, err = io.Copy(target, source); err != nil {
		_ = target.Close()
		_ = os.Remove(to)

		return err
	}

	if err = target.Close(); err != nil {
		return err
	}

	return os.Remove(from)
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

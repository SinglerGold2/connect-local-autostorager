package main

// The watcher: read a folder the operator named, and hand every print file it
// holds to the storage. Nothing is moved or deleted before the storage has the
// bytes.

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

// errNoFile is what the storage answers for a name it does not hold.
var errNoFile = storageapi.ErrNoFile

// recentLimit bounds the list the page reads.
const recentLimit = 20

// sizeSettle is how many scans a file's size must hold before it is read. A
// file a slicer is still writing grows between two scans.
const sizeSettle = 2

// Import is one line of the page's list.
type Import struct {
	At     string `json:"at"`
	Source string `json:"source"`
	Target string `json:"target"`
	Result string `json:"result"`
	Detail string `json:"detail,omitempty"`
}

// watchState is what the module knows about its work right now.
type watchState struct {
	mu      sync.Mutex
	folder  string
	recent  []Import
	scans   int64
	imports int64
	failed  int64
	lastAt  string
	sizes   map[string]int64
	stable  map[string]int
}

var (
	// store is the storage client of this session, set once the session exists.
	store *storageapi.Client

	// watching is the cancel of the running loop, so a new configuration
	// replaces it instead of stacking a second one.
	watching context.CancelFunc

	// state is the one piece of mutable state the module keeps.
	state = &watchState{sizes: map[string]int64{}, stable: map[string]int{}}

	// watcherMu guards the loop's lifetime against two configurations arriving
	// at once.
	watcherMu sync.Mutex
)

// allowed reports whether a name is a print file this module takes.
func allowed(name string) bool {
	lower := strings.ToLower(name)

	return strings.HasSuffix(lower, ".gcode") || strings.HasSuffix(lower, ".bgcode")
}

// startWatcher stops the running loop and starts one for these settings. An
// empty folder stops watching, which is how an operator switches it off.
func startWatcher(s Settings) {
	watcherMu.Lock()
	defer watcherMu.Unlock()

	if watching != nil {
		watching()
		watching = nil
	}

	state.mu.Lock()
	state.folder = s.Folder
	state.sizes = map[string]int64{}
	state.stable = map[string]int{}
	state.mu.Unlock()

	if s.Folder == "" {
		return
	}

	ctx, cancel := context.WithCancel(context.Background())
	watching = cancel

	go loop(ctx, s)
}

// loop reads the folder until it is replaced or the module stops.
func loop(ctx context.Context, s Settings) {
	interval := time.Duration(s.IntervalSeconds) * time.Second
	timer := time.NewTimer(0)
	defer timer.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-timer.C:
		}

		scan(ctx, s)
		timer.Reset(interval)
	}
}

// scan reads the folder once and imports what is ready.
func scan(ctx context.Context, s Settings) {
	entries, err := os.ReadDir(s.Folder)
	if err != nil {
		note(Import{At: now(), Source: s.Folder, Result: "error", Detail: err.Error()})

		return
	}

	state.mu.Lock()
	state.scans++
	state.lastAt = now()

	for _, entry := range entries {
		if entry.IsDir() || !allowed(entry.Name()) {
			continue
		}

		info, infoErr := entry.Info()
		if infoErr != nil {
			continue
		}

		name := entry.Name()
		if state.sizes[name] == info.Size() {
			state.stable[name]++
		} else {
			state.sizes[name] = info.Size()
			state.stable[name] = 1
		}
	}
	state.mu.Unlock()

	state.mu.Lock()
	ready := make([]string, 0, len(state.stable))

	for name, count := range state.stable {
		if count >= sizeSettle {
			ready = append(ready, name)
		}
	}
	state.mu.Unlock()

	for _, name := range ready {
		importOne(ctx, s, filepath.Join(s.Folder, name))

		state.mu.Lock()
		delete(state.sizes, name)
		delete(state.stable, name)
		state.mu.Unlock()
	}
}

// importOne copies one file into the storage and moves the source afterwards.
func importOne(ctx context.Context, s Settings, path string) {
	name := filepath.Base(path)

	info, err := os.Stat(path)
	if err != nil {
		fail(s, path, name, "reading the file", err)

		return
	}

	if info.Size() > int64(s.MaxMB)<<20 {
		fail(s, path, name, "the file is larger than the limit", fmt.Errorf("%d MB is more than %d MB", info.Size()>>20, s.MaxMB))

		return
	}

	target, err := freeName(ctx, s.Storage, name, info.Size())
	if err != nil {
		fail(s, path, name, "looking for a free name", err)

		return
	}

	body, err := os.Open(path)
	if err != nil {
		fail(s, path, name, "opening the file", err)

		return
	}

	written, err := store.Write(ctx, storageapi.WriteParams{Storage: s.Storage, Path: target, Size: info.Size()}, body)
	_ = body.Close()

	if err != nil {
		fail(s, path, name, "writing into the storage", err)

		return
	}

	state.mu.Lock()
	state.imports++
	state.recent = append(state.recent, Import{At: now(), Source: path, Target: written.Path, Result: "imported"})
	state.recent = trim(state.recent)
	state.mu.Unlock()

	if s.MoveOnSuccess {
		if moveErr := moveTo(path, filepath.Join(s.Folder, "successful", name)); moveErr != nil {
			note(Import{At: now(), Source: path, Target: written.Path, Result: "imported", Detail: "could not move the source: " + moveErr.Error()})
		}
	}
}

// freeName finds a name the storage does not hold yet. A dollar sign is not
// part of it: the suffix is -1, -2 and so on.
func freeName(ctx context.Context, storage, name string, size int64) (string, error) {
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
func fail(s Settings, path, name, step string, cause error) {
	state.mu.Lock()
	state.failed++
	state.mu.Unlock()

	reason := fmt.Sprintf("Step: %s\nReason: %v\nTime: %s\nSource: %s\n", step, cause, now(), path)
	note(Import{At: now(), Source: path, Result: "failed", Detail: step + ": " + cause.Error()})

	destination := filepath.Join(s.Folder, "failed", name)
	if moveErr := moveTo(path, destination); moveErr != nil {
		note(Import{At: now(), Source: path, Result: "failed", Detail: "could not move the source: " + moveErr.Error()})

		return
	}

	_ = os.WriteFile(destination+"-failure.txt", []byte(reason), 0o600)
}

// moveTo moves a file, copying when the two paths are on different devices.
func moveTo(from, to string) error {
	if err := os.MkdirAll(filepath.Dir(to), 0o755); err != nil {
		return err
	}

	if err := os.Rename(from, to); err == nil {
		return nil
	}

	source, err := os.Open(from)
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
	state.mu.Lock()
	defer state.mu.Unlock()

	state.recent = append(state.recent, entry)
	state.recent = trim(state.recent)
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
	state.mu.Lock()
	defer state.mu.Unlock()

	out := make([]Import, len(state.recent))
	copy(out, state.recent)

	return out
}

// now is the timestamp the page shows, in the module's own words.
func now() string {
	return time.Now().Format(time.RFC3339)
}

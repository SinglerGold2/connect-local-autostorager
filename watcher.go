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
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/prusa3d/prusa-connect/sdk/storageapi"
)

// recentLimit bounds the list the page and the card read.
const recentLimit = 25

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

// SourceStatus is the current state of one watched folder. A client that is
// off is "down", not a failed import: the page keeps this separate from the
// recent list, so a sleeping PC does not fill the list with errors.
type SourceStatus struct {
	Folder string `json:"folder"`
	State  string `json:"state"`
	Detail string `json:"detail,omitempty"`
	At     string `json:"at"`
}

// shared is what every watcher and the page read together.
type shared struct {
	mu      sync.Mutex
	recent  []Import
	logs    []Import
	sources map[string]SourceStatus
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

	view.mu.Lock()
	view.sources = make(map[string]SourceStatus)

	for _, folder := range s.Folders {
		folder = strings.TrimSpace(folder)
		if folder == "" {
			continue
		}

		view.sources[folder] = SourceStatus{Folder: folder, State: "starting", At: now()}
	}
	view.mu.Unlock()

	for _, folder := range s.Folders {
		folder = strings.TrimSpace(folder)
		if folder == "" {
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
		sourceDown(folder, err)

		return
	}
	defer func() { _ = root.Close() }()

	dir, err := root.Open(".")
	if err != nil {
		sourceDown(folder, err)

		return
	}

	entries, err := dir.ReadDir(-1)
	_ = dir.Close()

	if err != nil {
		sourceDown(folder, err)

		return
	}

	sourceOnline(folder)

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

	entry := Import{At: now(), Folder: folder, Source: path, Target: written.Path, Result: "imported"}

	view.mu.Lock()
	view.imports++
	view.recent = append(view.recent, entry)
	view.recent = trim(view.recent)
	view.mu.Unlock()

	if err := recordImport(ctx, entry); err != nil {
		fmt.Fprintf(os.Stderr, "%s: %v\n", name, err)
	}

	if s.MoveOnSuccess {
		if _, moveErr := moveTo(root, name, filepath.Join("successful", name)); moveErr != nil {
			noteLog(Import{At: now(), Folder: folder, Source: path, Target: written.Path, Result: "imported", Detail: "could not move the source: " + moveErr.Error()})
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
	moved, moveErr := moveTo(root, name, destination)
	if moveErr != nil {
		note(Import{At: now(), Folder: folder, Source: path, Result: "failed", Detail: "could not move the source: " + moveErr.Error()})

		return
	}

	report := moved + "-failure.txt"
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
func moveTo(root *os.Root, from, to string) (string, error) {
	info, err := root.Lstat(from)
	if err != nil {
		return "", err
	}

	if !info.Mode().IsRegular() {
		return "", errNotRegular
	}

	if err := root.MkdirAll(filepath.Dir(to), 0o755); err != nil {
		return "", err
	}

	to, err = freeMoveName(root, to)
	if err != nil {
		return "", err
	}

	if err := root.Rename(from, to); err == nil {
		return to, nil
	}

	if err := copyTo(root, from, to); err != nil {
		return "", err
	}

	return to, nil
}

// freeMoveName keeps both files when the destination already exists. The
// Storage name has its own freeName; this one is for the source move into
// successful/ or failed/.
func freeMoveName(root *os.Root, to string) (string, error) {
	ext := filepath.Ext(to)
	base := strings.TrimSuffix(to, ext)

	for attempt := 0; attempt < 1000; attempt++ {
		candidate := to
		if attempt > 0 {
			candidate = fmt.Sprintf("%s-%d%s", base, attempt, ext)
		}

		_, err := root.Lstat(candidate)
		if errors.Is(err, os.ErrNotExist) {
			return candidate, nil
		}

		if err != nil {
			return "", err
		}
	}

	return "", fmt.Errorf("no free move name for %s", to)
}

// copyTo is the cross-device fallback of moveTo. It creates the destination
// exclusively: a pre-created file or symlink at that path makes it fail rather
// than truncate or follow it, just like the failure report.
func copyTo(root *os.Root, from, to string) error {
	source, err := root.Open(from)
	if err != nil {
		return err
	}

	defer func() { _ = source.Close() }()

	target, err := root.OpenFile(to, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o666)
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

// sourceDown records why a folder could not be read. A host that is down is a
// status, not an import error; a permission or other configuration failure is
// kept as an error but still stays out of the recent-import list.
func sourceDown(folder string, err error) {
	state := "error"
	if sourceUnavailable(err) {
		state = "down"
	}

	setSource(folder, state, err.Error())
}

// sourceUnavailable reports whether err means the source is unreachable: the
// client is off, the share is gone or the mount is stale. A permission error
// is deliberately not in this list.
func sourceUnavailable(err error) bool {
	if errors.Is(err, os.ErrNotExist) {
		return true
	}

	text := strings.ToLower(err.Error())
	for _, fragment := range []string{
		"host is down",
		"host is unreachable",
		"network is unreachable",
		"network is down",
		"connection refused",
		"connection timed out",
		"no route to host",
		"transport endpoint is not connected",
		"stale file handle",
		"input/output error",
		"not mounted",
		"operation timed out",
	} {
		if strings.Contains(text, fragment) {
			return true
		}
	}

	return false
}

func sourceOnline(folder string) {
	setSource(folder, "online", "")
}

func setSource(folder, state, detail string) {
	view.mu.Lock()
	defer view.mu.Unlock()

	if view.sources == nil {
		view.sources = make(map[string]SourceStatus)
	}

	view.sources[folder] = SourceStatus{Folder: folder, State: state, Detail: detail, At: now()}
}

// sourceStatuses is the current state of every watched folder, sorted so the
// page does not reshuffle the rows on every poll.
func sourceStatuses() []SourceStatus {
	view.mu.Lock()
	defer view.mu.Unlock()

	out := make([]SourceStatus, 0, len(view.sources))
	for _, source := range view.sources {
		out = append(out, source)
	}

	sort.Slice(out, func(i, j int) bool {
		return out[i].Folder < out[j].Folder
	})

	return out
}

// note keeps one line for the page: a successful import in "Recent imports",
// everything else in the logs.
func note(entry Import) {
	view.mu.Lock()
	defer view.mu.Unlock()

	if entry.Result == "imported" {
		view.recent = append(view.recent, entry)
		view.recent = trim(view.recent)
	} else {
		view.logs = append(view.logs, entry)
		view.logs = trim(view.logs)
	}
}

// noteLog keeps a warning that belongs to a committed import out of the
// imports list and in the logs.
func noteLog(entry Import) {
	view.mu.Lock()
	defer view.mu.Unlock()

	view.logs = append(view.logs, entry)
	view.logs = trim(view.logs)
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

// logEntries is the separate list for failed imports and warnings.
func logEntries() []Import {
	view.mu.Lock()
	defer view.mu.Unlock()

	out := make([]Import, len(view.logs))
	copy(out, view.logs)

	return out
}

// now is the timestamp the page shows.
func now() string {
	return time.Now().Format(time.RFC3339)
}

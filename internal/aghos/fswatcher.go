package aghos

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"sync"

	"github.com/AdguardTeam/golibs/container"
	"github.com/AdguardTeam/golibs/errors"
	"github.com/AdguardTeam/golibs/logutil/slogutil"
	"github.com/AdguardTeam/golibs/service"
	"github.com/fsnotify/fsnotify"
)

// Event is a convenient alias for an empty struct to signal that watched file
// event happened.
type Event = struct{}

// FSWatcher tracks all the file system events and notifies about those.
//
// TODO(e.burkov, a.garipov): Move into another package like aghfs.
type FSWatcher interface {
	service.Interface

	// Events returns the channel to notify about the file system events.
	Events() (e <-chan Event)

	// Add starts tracking the file.  It returns an error if the file can't be
	// tracked.  Adding the same file multiple times must not result in an
	// error.
	Add(name string) (err error)

	// Remove stops tracking the file.  Removing a non-tracked file must not
	// result in an error.
	Remove(name string) (err error)
}

// OSWatcherConfig is the configuration structure for [NewOSWatcher].
//
// TODO(e.burkov):  Consider using [os.Root].
type OSWatcherConfig struct {
	// Logger is used for logging the operations of watcher.  It must not be
	// nil.
	Logger *slog.Logger
}

// OSWatcher tracks the file system provided by the OS.
//
// TODO(e.burkov):  Add tests.
type OSWatcher struct {
	// logger is used for logging the operations of watcher.
	logger *slog.Logger

	// filesMu protects files.
	filesMu *sync.RWMutex

	// watcher is the actual notifier.
	watcher *fsnotify.Watcher

	// events is the channel to notify.
	events chan Event

	// files maps directories to the files tracked in them.  If the tracked file
	// is a directory, it is mapped to itself.
	files map[string]*container.MapSet[string]
}

// osWatcherPref is a prefix for logging and wrapping errors in osWathcer's
// methods.
const osWatcherPref = "os_watcher"

// NewOSWatcher creates an [FSWatcher] that tracks the file system of the OS.  c
// must not be nil.
func NewOSWatcher(c *OSWatcherConfig) (w *OSWatcher, err error) {
	defer func() { err = errors.Annotate(err, "%s: %w", osWatcherPref) }()

	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, fmt.Errorf("creating watcher: %w", err)
	}

	return &OSWatcher{
		logger:  c.Logger,
		filesMu: &sync.RWMutex{},
		watcher: watcher,
		events:  make(chan Event, 1),
		files:   map[string]*container.MapSet[string]{},
	}, nil
}

// type check
var _ FSWatcher = (*OSWatcher)(nil)

// Start implements the [service.Interface] interface for *OSWatcher.
func (w *OSWatcher) Start(ctx context.Context) (err error) {
	go w.handleErrors(ctx)
	go w.handleEvents(ctx)

	return nil
}

// Shutdown implements the [service.Interface] interface for *OSWatcher.
func (w *OSWatcher) Shutdown(_ context.Context) (err error) {
	return errors.Annotate(w.watcher.Close(), "%s: %w", osWatcherPref)
}

// Events implements the [FSWatcher] interface for *OSWatcher.
func (w *OSWatcher) Events() (e <-chan Event) {
	return w.events
}

// Add implements the [FSWatcher] interface for *OSWatcher.  It's safe for
// concurrent use.
//
// TODO(e.burkov):  Make it accept non-existing files to detect it's creating.
func (w *OSWatcher) Add(name string) (err error) {
	defer func() { err = errors.Annotate(err, "%s: %w", osWatcherPref) }()

	fi, err := os.Stat(name)
	if err != nil {
		return fmt.Errorf("checking file %q: %w", name, err)
	}

	// Watch the directory and filter the events by the file name, since the
	// common recomendation to the fsnotify package is to watch the directory
	// instead of the file itself.
	//
	// See https://pkg.go.dev/github.com/fsnotify/fsnotify@v1.7.0#readme-watching-a-file-doesn-t-work-well.
	dirName := name
	if !fi.IsDir() {
		dirName = filepath.Dir(name)
	}

	w.filesMu.Lock()
	defer w.filesMu.Unlock()

	names := w.files[dirName]
	if names == nil {
		names = container.NewMapSet[string]()
		w.files[dirName] = names
	}
	names.Add(name)

	err = w.watcher.Add(dirName)
	if err != nil {
		return fmt.Errorf("adding %q: %w", dirName, err)
	}

	return nil
}

// Remove implements the [FSWatcher] interface for *OSWatcher.  It's safe for
// concurrent use.
func (w *OSWatcher) Remove(name string) (err error) {
	defer func() { err = errors.Annotate(err, "%s: %w", osWatcherPref) }()

	dirName := filepath.Dir(name)

	w.filesMu.Lock()
	defer w.filesMu.Unlock()

	names, ok := w.files[name]
	if ok {
		dirName = name
	} else {
		names = w.files[dirName]
	}

	if !names.Has(name) {
		// Name is not tracked.
		return nil
	}

	names.Delete(name)
	if names.Len() > 0 {
		// Some files are still tracked in the directory.
		return nil
	}

	// No more files tracked in the directory, unwatch it.
	delete(w.files, dirName)

	err = w.watcher.Remove(dirName)
	if err != nil {
		return fmt.Errorf("removing %q: %w", dirName, err)
	}

	return nil
}

// handleEvents notifies about the received file system's event if needed.  It
// is intended to be used as a goroutine.
func (w *OSWatcher) handleEvents(ctx context.Context) {
	defer slogutil.RecoverAndLog(ctx, w.logger)

	defer close(w.events)

	ch := w.watcher.Events
	for e := range ch {
		if !w.isTrackedEvent(e) {
			continue
		}

		skipDuplicates(ch)

		select {
		case w.events <- Event{}:
			// Go on.
		default:
			w.logger.DebugContext(ctx, "events buffer is full")
		}
	}
}

// isTrackedEvent returns true if the event is about change of a file that is
// tracked.
func (w *OSWatcher) isTrackedEvent(e fsnotify.Event) (isDir bool) {
	// changeEvent is a combination of events that indicate a file change.
	const changeEvent = fsnotify.Write | fsnotify.Create | fsnotify.Rename | fsnotify.Remove

	if !e.Has(changeEvent) {
		return false
	}

	dirName := filepath.Dir(e.Name)

	w.filesMu.RLock()
	defer w.filesMu.RUnlock()

	names, isDir := w.files[e.Name]
	if !isDir {
		names = w.files[dirName]
	}

	return names.Has(e.Name)
}

// skipDuplicates drains the given channel of events, assuming that some events
// might occur multiple times.
//
// TODO(e.burkov):  Check if this is still needed.
func skipDuplicates(ch <-chan fsnotify.Event) {
	for {
		select {
		case <-ch:
			// Go on.
		default:
			return
		}
	}
}

// handleErrors handles accompanying errors.  It used to be called in a separate
// goroutine.
func (w *OSWatcher) handleErrors(ctx context.Context) {
	defer slogutil.RecoverAndLog(ctx, w.logger)

	for err := range w.watcher.Errors {
		w.logger.ErrorContext(ctx, "handling error", slogutil.KeyError, err)
	}
}

// EmptyFSWatcher is a no-op implementation of the [FSWatcher] interface.  It
// may be used on systems not supporting filesystem events.
type EmptyFSWatcher struct{}

// type check
var _ FSWatcher = EmptyFSWatcher{}

// Start implements the [FSWatcher] interface for EmptyFSWatcher.  It always
// returns nil error.
func (EmptyFSWatcher) Start(_ context.Context) (err error) {
	return nil
}

// Shutdown implements the [FSWatcher] interface for EmptyFSWatcher.  It always
// returns nil error.
func (EmptyFSWatcher) Shutdown(_ context.Context) (err error) {
	return nil
}

// Events implements the [FSWatcher] interface for EmptyFSWatcher.  It always
// returns nil channel.
func (EmptyFSWatcher) Events() (e <-chan Event) {
	return nil
}

// Add implements the [FSWatcher] interface for EmptyFSWatcher.  It always
// returns nil error.
func (EmptyFSWatcher) Add(_ string) (err error) {
	return nil
}

// Remove implements the [FSWatcher] interface for EmptyFSWatcher.  It always
// returns nil error.
func (EmptyFSWatcher) Remove(_ string) (err error) {
	return nil
}

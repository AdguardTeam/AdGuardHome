package aghos

import (
	"context"
	"fmt"
	"io/fs"
	"log/slog"
	"path/filepath"

	"github.com/AdguardTeam/golibs/container"
	"github.com/AdguardTeam/golibs/errors"
	"github.com/AdguardTeam/golibs/logutil/slogutil"
	"github.com/AdguardTeam/golibs/osutil"
	"github.com/AdguardTeam/golibs/service"
	"github.com/fsnotify/fsnotify"
)

// event is a convenient alias for an empty struct to signal that watching
// event happened.
type event = struct{}

// FSWatcher tracks all the fyle system events and notifies about those.
//
// TODO(e.burkov, a.garipov): Move into another package like aghfs.
//
// TODO(e.burkov):  Add tests.
type FSWatcher interface {
	service.Interface

	// Events returns the channel to notify about the file system events.
	Events() (e <-chan event)

	// Add starts tracking the file.  It returns an error if the file can't be
	// tracked.  It must not be called after Start.
	Add(name string) (err error)
}

// osWatcher tracks the file system provided by the OS.
type osWatcher struct {
	// logger is used for logging the operations of the osWatcher.
	logger *slog.Logger

	// watcher is the actual notifier that is handled by osWatcher.
	watcher *fsnotify.Watcher

	// events is the channel to notify.
	events chan event

	// files is the set of tracked files.
	files *container.MapSet[string]
}

// osWatcherPref is a prefix for logging and wrapping errors in osWathcer's
// methods.
const osWatcherPref = "os watcher"

// NewOSWritesWatcher creates FSWatcher that tracks the real file system of the
// OS and notifies only about writing events.  l must not be nil.
func NewOSWritesWatcher(l *slog.Logger) (w FSWatcher, err error) {
	defer func() { err = errors.Annotate(err, "%s: %w", osWatcherPref) }()

	var watcher *fsnotify.Watcher
	watcher, err = fsnotify.NewWatcher()
	if err != nil {
		return nil, fmt.Errorf("creating watcher: %w", err)
	}

	return &osWatcher{
		logger:  l,
		watcher: watcher,
		events:  make(chan event, 1),
		files:   container.NewMapSet[string](),
	}, nil
}

// type check
var _ FSWatcher = (*osWatcher)(nil)

// Start implements the [FSWatcher] interface for *osWatcher.
func (w *osWatcher) Start(ctx context.Context) (err error) {
	go w.handleErrors(ctx)
	go w.handleEvents(ctx)

	return nil
}

// Shutdown implements the [FSWatcher] interface for *osWatcher.
func (w *osWatcher) Shutdown(_ context.Context) (err error) {
	return w.watcher.Close()
}

// Events implements the FSWatcher interface for *osWatcher.
func (w *osWatcher) Events() (e <-chan event) {
	return w.events
}

// Add implements the [FSWatcher] interface for *osWatcher.
//
// TODO(e.burkov):  Make it accept non-existing files to detect it's creating.
func (w *osWatcher) Add(name string) (err error) {
	defer func() { err = errors.Annotate(err, "%s: %w", osWatcherPref) }()

	fi, err := fs.Stat(osutil.RootDirFS(), name)
	if err != nil {
		return fmt.Errorf("checking file %q: %w", name, err)
	}

	name = filepath.Join("/", name)
	w.files.Add(name)

	// Watch the directory and filter the events by the file name, since the
	// common recomendation to the fsnotify package is to watch the directory
	// instead of the file itself.
	//
	// See https://pkg.go.dev/github.com/fsnotify/fsnotify@v1.7.0#readme-watching-a-file-doesn-t-work-well.
	if !fi.IsDir() {
		name = filepath.Dir(name)
	}

	return w.watcher.Add(name)
}

// handleEvents notifies about the received file system's event if needed.  It
// is intended to be used as a goroutine.
func (w *osWatcher) handleEvents(ctx context.Context) {
	defer slogutil.RecoverAndLog(ctx, w.logger)

	defer close(w.events)

	ch := w.watcher.Events
	for e := range ch {
		if e.Op&fsnotify.Write == 0 || !w.files.Has(e.Name) {
			continue
		}

		skipDuplicates(ch)

		select {
		case w.events <- event{}:
			// Go on.
		default:
			w.logger.DebugContext(ctx, "events buffer is full")
		}
	}
}

// skipDuplicates drains the given channel of events, assuming that some events
// might occur multiple times.
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
func (w *osWatcher) handleErrors(ctx context.Context) {
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
func (EmptyFSWatcher) Events() (e <-chan event) {
	return nil
}

// Add implements the [FSWatcher] interface for EmptyFSWatcher.  It always
// returns nil error.
func (EmptyFSWatcher) Add(_ string) (err error) {
	return nil
}

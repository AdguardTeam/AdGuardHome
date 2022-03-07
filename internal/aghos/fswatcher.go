package aghos

import (
	"fmt"
	"io"
	"io/fs"
	"path/filepath"

	"github.com/AdguardTeam/golibs/errors"
	"github.com/AdguardTeam/golibs/log"
	"github.com/fsnotify/fsnotify"
)

// event is a convenient alias for an empty struct to signal that watching
// event happened.
type event = struct{}

// FSWatcher tracks all the fyle system events and notifies about those.
//
// TODO(e.burkov, a.garipov): Move into another package like aghfs.
type FSWatcher interface {
	io.Closer

	// Events should return a read-only channel which notifies about events.
	Events() (e <-chan event)

	// Add should check if the file named name is accessible and starts tracking
	// it.
	Add(name string) (err error)
}

// osWatcher tracks the file system provided by the OS.
type osWatcher struct {
	// w is the actual notifier that is handled by osWatcher.
	w *fsnotify.Watcher

	// events is the channel to notify.
	events chan event
}

const (
	// osWatcherPref is a prefix for logging and wrapping errors in osWathcer's
	// methods.
	osWatcherPref = "os watcher"
)

// NewOSWritesWatcher creates FSWatcher that tracks the real file system of the
// OS and notifies only about writing events.
func NewOSWritesWatcher() (w FSWatcher, err error) {
	defer func() { err = errors.Annotate(err, "%s: %w", osWatcherPref) }()

	var watcher *fsnotify.Watcher
	watcher, err = fsnotify.NewWatcher()
	if err != nil {
		return nil, fmt.Errorf("creating watcher: %w", err)
	}

	fsw := &osWatcher{
		w:      watcher,
		events: make(chan event, 1),
	}

	go fsw.handleErrors()
	go fsw.handleEvents()

	return fsw, nil
}

// handleErrors handles accompanying errors.  It used to be called in a separate
// goroutine.
func (w *osWatcher) handleErrors() {
	defer log.OnPanic(fmt.Sprintf("%s: handling errors", osWatcherPref))

	for err := range w.w.Errors {
		log.Error("%s: %s", osWatcherPref, err)
	}
}

// Events implements the FSWatcher interface for *osWatcher.
func (w *osWatcher) Events() (e <-chan event) {
	return w.events
}

// Add implements the FSWatcher interface for *osWatcher.
//
// TODO(e.burkov):  Make it accept non-existing files to detect it's creating.
func (w *osWatcher) Add(name string) (err error) {
	defer func() { err = errors.Annotate(err, "%s: %w", osWatcherPref) }()

	if _, err = fs.Stat(RootDirFS(), name); err != nil {
		return fmt.Errorf("checking file %q: %w", name, err)
	}

	return w.w.Add(filepath.Join("/", name))
}

// Close implements the FSWatcher interface for *osWatcher.
func (w *osWatcher) Close() (err error) {
	return w.w.Close()
}

// handleEvents notifies about the received file system's event if needed.  It
// used to be called in a separate goroutine.
func (w *osWatcher) handleEvents() {
	defer log.OnPanic(fmt.Sprintf("%s: handling events", osWatcherPref))

	defer close(w.events)

	ch := w.w.Events
	for e := range ch {
		if e.Op&fsnotify.Write == 0 {
			continue
		}

		// Skip the following events assuming that sometimes the same event
		// occurrs several times.
		for ok := true; ok; {
			select {
			case _, ok = <-ch:
				// Go on.
			default:
				ok = false
			}
		}

		select {
		case w.events <- event{}:
			// Go on.
		default:
			log.Debug("%s: events buffer is full", osWatcherPref)
		}
	}
}

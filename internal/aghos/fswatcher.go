package aghos

import (
	"fmt"
	"io"
	"io/fs"
	"path/filepath"

	"github.com/AdguardTeam/golibs/container"
	"github.com/AdguardTeam/golibs/errors"
	"github.com/AdguardTeam/golibs/log"
	"github.com/AdguardTeam/golibs/osutil"
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
	// Start starts watching the added files.
	Start() (err error)

	// Close stops watching the files and closes an update channel.
	io.Closer

	// Events returns the channel to notify about the file system events.
	Events() (e <-chan event)

	// Add starts tracking the file.  It returns an error if the file can't be
	// tracked.  It must not be called after Start.
	Add(name string) (err error)
}

// osWatcher tracks the file system provided by the OS.
type osWatcher struct {
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
// OS and notifies only about writing events.
func NewOSWritesWatcher() (w FSWatcher, err error) {
	defer func() { err = errors.Annotate(err, "%s: %w", osWatcherPref) }()

	var watcher *fsnotify.Watcher
	watcher, err = fsnotify.NewWatcher()
	if err != nil {
		return nil, fmt.Errorf("creating watcher: %w", err)
	}

	return &osWatcher{
		watcher: watcher,
		events:  make(chan event, 1),
		files:   container.NewMapSet[string](),
	}, nil
}

// type check
var _ FSWatcher = (*osWatcher)(nil)

// Start implements the FSWatcher interface for *osWatcher.
func (w *osWatcher) Start() (err error) {
	go w.handleErrors()
	go w.handleEvents()

	return nil
}

// Close implements the FSWatcher interface for *osWatcher.
func (w *osWatcher) Close() (err error) {
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
func (w *osWatcher) handleEvents() {
	defer log.OnPanic(fmt.Sprintf("%s: handling events", osWatcherPref))

	defer close(w.events)

	ch := w.watcher.Events
	for e := range ch {
		if e.Op&fsnotify.Write == 0 || !w.files.Has(e.Name) {
			continue
		}

		// Skip the following events assuming that sometimes the same event
		// occurs several times.
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

// handleErrors handles accompanying errors.  It used to be called in a separate
// goroutine.
func (w *osWatcher) handleErrors() {
	defer log.OnPanic(fmt.Sprintf("%s: handling errors", osWatcherPref))

	for err := range w.watcher.Errors {
		log.Error("%s: %s", osWatcherPref, err)
	}
}

// EmptyFSWatcher is a no-op implementation of the [FSWatcher] interface.  It
// may be used on systems not supporting filesystem events.
type EmptyFSWatcher struct{}

// type check
var _ FSWatcher = EmptyFSWatcher{}

// Start implements the [FSWatcher] interface for EmptyFSWatcher.  It always
// returns nil error.
func (EmptyFSWatcher) Start() (err error) {
	return nil
}

// Close implements the [FSWatcher] interface for EmptyFSWatcher.  It always
// returns nil error.
func (EmptyFSWatcher) Close() (err error) {
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

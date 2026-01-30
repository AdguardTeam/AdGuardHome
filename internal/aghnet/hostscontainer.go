package aghnet

import (
	"context"
	"fmt"
	"io"
	"io/fs"
	"log/slog"
	"net/netip"
	"path"
	"path/filepath"
	"sync/atomic"

	"github.com/AdguardTeam/AdGuardHome/internal/aghos"
	"github.com/AdguardTeam/golibs/errors"
	"github.com/AdguardTeam/golibs/hostsfile"
	"github.com/AdguardTeam/golibs/logutil/slogutil"
)

// hostsContainerPrefix is a prefix for wrapping errors in HostsContainer's
// methods.
const hostsContainerPrefix = "hosts container"

// HostsContainer stores the relevant hosts database provided by the OS and
// processes both A/AAAA and PTR DNS requests for those.
type HostsContainer struct {
	// logger is used for logging the operation of the hosts container.  It must
	// not be nil.
	logger *slog.Logger

	// done is the channel to signal closing the container.
	done chan struct{}

	// updates is the channel for receiving updated hosts.
	updates chan *hostsfile.DefaultStorage

	// current is the last set of hosts parsed.
	current atomic.Pointer[hostsfile.DefaultStorage]

	// fsys is the working file system to read hosts files from.  It must not be
	// nil.
	fsys fs.FS

	// watcher tracks the changes in specified files and directories.  It must
	// not be nil.
	watcher aghos.FSWatcher

	// patterns stores specified paths in the fs.Glob-compatible form.
	patterns []string
}

// ErrNoHostsPaths is returned when there are no valid paths to watch passed to
// the HostsContainer.
const ErrNoHostsPaths errors.Error = "no valid paths to hosts files provided"

// NewHostsContainer creates a container of hosts that watches the paths with w.
// paths shouldn't be empty, and each path should be relative and refer either a
// file or a directory in fsys.  fsys should be a filesystem mounted at the
// operating system root.  l and w must not be nil.
//
// TODO(e.burkov):  Add configuration.
//
// TODO(e.burkov):  Reconsider using fs.
func NewHostsContainer(
	ctx context.Context,
	l *slog.Logger,
	fsys fs.FS,
	w aghos.FSWatcher,
	paths ...string,
) (hc *HostsContainer, err error) {
	defer func() { err = errors.Annotate(err, "%s: %w", hostsContainerPrefix) }()

	if len(paths) == 0 {
		return nil, ErrNoHostsPaths
	}

	var patterns []string
	patterns, err = pathsToPatterns(fsys, paths)
	if err != nil {
		return nil, err
	} else if len(patterns) == 0 {
		return nil, ErrNoHostsPaths
	}

	hc = &HostsContainer{
		logger:   l,
		done:     make(chan struct{}, 1),
		updates:  make(chan *hostsfile.DefaultStorage, 1),
		fsys:     fsys,
		watcher:  w,
		patterns: patterns,
	}

	l.DebugContext(ctx, "starting")

	// Load initially.
	if err = hc.refresh(ctx); err != nil {
		return nil, err
	}

	// Don't use absolute paths for watching, since those are required to be
	// relative to the operating system root, not the current directory.  See
	// TODO at NewHostsContainer.
	rootDir := aghos.RootDir()

	for _, p := range paths {
		p = filepath.Join(rootDir, p)

		err = w.Add(p)
		if err != nil {
			if !errors.Is(err, fs.ErrNotExist) {
				return nil, fmt.Errorf("adding path: %w", err)
			}

			l.DebugContext(ctx, "expected path does not exist", "path", p, slogutil.KeyError, err)
		}
	}

	go hc.handleEvents(ctx)

	return hc, nil
}

// Close implements the [io.Closer] interface for *HostsContainer.  It closes
// both itself and its [aghos.FSWatcher].  Close must only be called once.
//
// TODO(s.chzhen):  Implement [service.Interface].
func (hc *HostsContainer) Close() (err error) {
	// TODO(s.chzhen):  Pass context.
	ctx := context.TODO()
	hc.logger.DebugContext(ctx, "closing")

	err = errors.Annotate(hc.watcher.Shutdown(ctx), "closing fs watcher: %w")

	// Go on and close the container either way.
	close(hc.done)

	return err
}

// Upd returns the channel into which the updates are sent.  The updates
// themselves must not be modified.
func (hc *HostsContainer) Upd() (updates <-chan *hostsfile.DefaultStorage) {
	return hc.updates
}

// type check
var _ hostsfile.Storage = (*HostsContainer)(nil)

// ByAddr implements the [hostsfile.Storage] interface for *HostsContainer.
func (hc *HostsContainer) ByAddr(addr netip.Addr) (names []string) {
	return hc.current.Load().ByAddr(addr)
}

// ByName implements the [hostsfile.Storage] interface for *HostsContainer.
func (hc *HostsContainer) ByName(name string) (addrs []netip.Addr) {
	return hc.current.Load().ByName(name)
}

// pathsToPatterns converts paths into patterns compatible with fs.Glob.
func pathsToPatterns(fsys fs.FS, paths []string) (patterns []string, err error) {
	for i, p := range paths {
		var fi fs.FileInfo
		fi, err = fs.Stat(fsys, p)
		if err != nil {
			if errors.Is(err, fs.ErrNotExist) {
				continue
			}

			// Don't put a filename here since it's already added by [fs.Stat].
			return nil, fmt.Errorf("path at index %d: %w", i, err)
		}

		if fi.IsDir() {
			p = path.Join(p, "*")
		}

		patterns = append(patterns, p)
	}

	return patterns, nil
}

// handleEvents concurrently handles the file system events.  It closes the
// update channel of HostsContainer when finishes.  It is intended to be used as
// a goroutine.
func (hc *HostsContainer) handleEvents(ctx context.Context) {
	defer slogutil.RecoverAndLog(ctx, hc.logger)

	defer close(hc.updates)

	eventsCh := hc.watcher.Events()
	ok := eventsCh != nil
	for ok {
		select {
		case _, ok = <-eventsCh:
			if !ok {
				hc.logger.DebugContext(ctx, "watcher closed the events channel")

				continue
			}

			if err := hc.refresh(ctx); err != nil {
				hc.logger.ErrorContext(ctx, "refreshing", slogutil.KeyError, err)
			}
		case _, ok = <-hc.done:
			// Go on.
		}
	}
}

// sendUpd tries to send the parsed data to the ch.
func (hc *HostsContainer) sendUpd(ctx context.Context, recs *hostsfile.DefaultStorage) {
	hc.logger.DebugContext(ctx, "sending update")

	ch := hc.updates
	select {
	case ch <- recs:
		// Updates are delivered.  Go on.
	case <-ch:
		ch <- recs
		hc.logger.DebugContext(ctx, "replaced the last update")
	case ch <- recs:
		// The previous update was just read and the next one pushed.  Go on.
	default:
		hc.logger.ErrorContext(ctx, "updates channel is broken")
	}
}

// refresh gets the data from specified files and propagates the updates if
// needed.
//
// TODO(e.burkov):  Accept a parameter to specify the files to refresh.
func (hc *HostsContainer) refresh(ctx context.Context) (err error) {
	hc.logger.DebugContext(ctx, "refreshing")

	// The error is always nil here since no readers passed.
	strg, _ := hostsfile.NewDefaultStorage(ctx, &hostsfile.DefaultStorageConfig{
		Logger: hc.logger,
	})
	_, err = aghos.FileWalker(func(r io.Reader) (patterns []string, cont bool, err error) {
		// Don't wrap the error since it's already informative enough as is.
		return nil, true, hostsfile.Parse(ctx, strg, r, nil)
	}).Walk(hc.fsys, hc.patterns...)
	if err != nil {
		// Don't wrap the error since it's informative enough as is.
		return err
	}

	// TODO(e.burkov):  Serialize updates using [time.Time].
	if !hc.current.Load().Equal(strg) {
		hc.current.Store(strg)
		hc.sendUpd(ctx, strg)
	}

	return nil
}

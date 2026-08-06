package aghtls

import (
	"context"
	"fmt"
	"log/slog"
	"sync"

	"github.com/AdguardTeam/AdGuardHome/internal/aghos"
	"github.com/AdguardTeam/golibs/errors"
	"github.com/AdguardTeam/golibs/logutil/slogutil"
)

// DefaultManagerConfig is the configuration structure for [NewDefaultManager].
type DefaultManagerConfig struct {
	// Logger is used for logging the operation of the manager.  It must not be
	// nil.
	Logger *slog.Logger

	// Watcher is used to watch the TLS certificate and key files.  It must not
	// be nil.
	Watcher aghos.FSWatcher
}

// DefaultManager is the default implementation of the [Manager] interface.
//
// TODO(e.burkov):  Add tests.
type DefaultManager struct {
	logger  *slog.Logger
	pairMu  *sync.Mutex
	updates chan UpdateSignal
	watcher aghos.FSWatcher
	pair    TLSPair
}

// NewDefaultManager returns a new properly initialized default manager.
func NewDefaultManager(c *DefaultManagerConfig) (mgr *DefaultManager) {
	return &DefaultManager{
		logger: c.Logger,
		pairMu: &sync.Mutex{},
		pair:   TLSPair{},
		// Buffer the channel to avoid missing updates.
		updates: make(chan UpdateSignal, 1),
		watcher: c.Watcher,
	}
}

// type check
var _ Manager = (*DefaultManager)(nil)

// Set implements the [Manager] interface for *DefaultManager.
func (mgr *DefaultManager) Set(ctx context.Context, certKey TLSPair) (err error) {
	mgr.logger.DebugContext(ctx, "setting", "cert", certKey.CertPath, "key", certKey.KeyPath)

	var errs []error

	mgr.pairMu.Lock()
	defer mgr.pairMu.Unlock()

	old := mgr.pair

	errs = mgr.appendUnwatchErr(errs, "old cert", old.CertPath)
	errs = mgr.appendUnwatchErr(errs, "old key", old.KeyPath)
	errs = mgr.appendWatchErr(errs, "new cert", certKey.CertPath)
	errs = mgr.appendWatchErr(errs, "new key", certKey.KeyPath)

	mgr.pair = certKey

	return errors.Join(errs...)
}

// appendUnwatchErr stops watching a file at path p described by what and
// appends an error to the errs slice, if any.  Empty p is ignored.
func (mgr *DefaultManager) appendUnwatchErr(errs []error, what, p string) (result []error) {
	if p == "" {
		return errs
	}

	err := mgr.watcher.Remove(p)
	if err != nil {
		errs = append(errs, fmt.Errorf("unwatching %s %s: %w", what, p, err))
	}

	return errs
}

// appendWatchErr starts watching a file at path p described by what and
// appends an error to the errs slice, if any.  Empty p is ignored.
func (mgr *DefaultManager) appendWatchErr(errs []error, what, p string) (result []error) {
	if p == "" {
		return errs
	}

	err := mgr.watcher.Add(p)
	if err != nil {
		errs = append(errs, fmt.Errorf("watching %s %s: %w", what, p, err))
	}

	return errs
}

// Refresh implements the [service.Refresher] interface for *DefaultManager.
func (mgr *DefaultManager) Refresh(ctx context.Context) (err error) {
	mgr.logger.DebugContext(ctx, "refreshing")

	select {
	case mgr.updates <- UpdateSignal{}:
		return nil
	case <-ctx.Done():
		return fmt.Errorf("refreshing: %w", ctx.Err())
	default:
		return nil
	}
}

// Start implements the [service.Interface] interface for *DefaultManager.
func (mgr *DefaultManager) Start(ctx context.Context) (err error) {
	err = mgr.watcher.Start(ctx)
	if err != nil {
		return fmt.Errorf("starting watcher: %w", err)
	}

	go mgr.handleEvents(ctx)

	return nil
}

// Shutdown implements the [service.Interface] interface for *DefaultManager.
func (mgr *DefaultManager) Shutdown(ctx context.Context) (err error) {
	defer close(mgr.updates)

	err = mgr.watcher.Shutdown(ctx)
	if err != nil {
		return fmt.Errorf("shutting down watcher: %w", err)
	}

	return nil
}

// Updates implements the [Manager] interface for *DefaultManager.
func (mgr *DefaultManager) Updates(ctx context.Context) (updates <-chan UpdateSignal) {
	return mgr.updates
}

// handleEvents handles changes of the tracked files.  It is intended to be run
// in a separate goroutine.
func (mgr *DefaultManager) handleEvents(ctx context.Context) {
	defer slogutil.RecoverAndLog(ctx, mgr.logger)

	eventsCh := mgr.watcher.Events()
	if eventsCh == nil {
		mgr.logger.DebugContext(ctx, "watcher does not emit events")

		return
	}

	for range eventsCh {
		err := mgr.Refresh(ctx)
		if err != nil {
			mgr.logger.ErrorContext(ctx, "refreshing", slogutil.KeyError, err)
		}
	}
}

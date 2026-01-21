package aghtls

import (
	"context"
	"fmt"
	"log/slog"
	"sync/atomic"

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
// TODO(e.burkov):  Use.
type DefaultManager struct {
	logger  *slog.Logger
	pair    *atomic.Pointer[TLSPair]
	updates chan UpdateSignal
	watcher aghos.FSWatcher
}

// NewDefaultManager returns a new properly initialized default manager.
func NewDefaultManager(c *DefaultManagerConfig) (mgr *DefaultManager) {
	return &DefaultManager{
		logger: c.Logger,
		pair:   &atomic.Pointer[TLSPair]{},
		// Buffer the channel to avoid missing updates.
		updates: make(chan UpdateSignal, 1),
		watcher: c.Watcher,
	}
}

// type check
var _ Manager = (*DefaultManager)(nil)

// Set implements the [Manager] interface for *DefaultManager.
func (mgr *DefaultManager) Set(ctx context.Context, certKey *TLSPair) (err error) {
	old := mgr.pair.Swap(certKey)

	if old != nil {
		err = errors.Join(
			mgr.watcher.Remove(old.CertPath),
			mgr.watcher.Remove(old.KeyPath),
		)
		if err != nil {
			return fmt.Errorf("removing old certificate and key: %w", err)
		}
	}

	if certKey != nil {
		err = errors.Join(
			mgr.watcher.Add(certKey.CertPath),
			mgr.watcher.Add(certKey.KeyPath),
		)
		if err != nil {
			return fmt.Errorf("adding new certificate and key: %w", err)
		}
	}

	return nil
}

// Refresh implements the [service.Refresher] interface for *DefaultManager.
func (mgr *DefaultManager) Refresh(ctx context.Context) (err error) {
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

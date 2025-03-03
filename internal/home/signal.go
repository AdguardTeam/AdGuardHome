package home

import (
	"context"
	"log/slog"
	"os"
	"sync"
	"sync/atomic"
	"syscall"

	"github.com/AdguardTeam/AdGuardHome/internal/client"
	"github.com/AdguardTeam/golibs/logutil/slogutil"
	"github.com/AdguardTeam/golibs/osutil"
)

// signalHandler processes incoming signals.  It reloads configurations of
// stored entities on SIGHUP and performs cleanup on all other signals.
type signalHandler struct {
	// logger is used to log the operation of the signal handler.  Initially,
	// [slog.Default] is used, but it should be swapped later using
	// [signalHandler.swapLogger].
	logger *atomic.Pointer[slog.Logger]

	// mu protects clientStorage and tlsManager.
	mu *sync.Mutex

	// clientStorage is used to reload information about runtime clients with an
	// ARP source.
	clientStorage *client.Storage

	// tlsManager is used to reload the TLS configuration.
	tlsManager *tlsManager

	// signals receives incoming signals.
	signals <-chan os.Signal

	// cleanup is called to perform cleanup on all incoming signals, except
	// SIGHUP.
	cleanup func(ctx context.Context)
}

// newSignalHandler returns a new properly initialized *signalHandler.
func newSignalHandler(
	signals <-chan os.Signal,
	cleanup func(ctx context.Context),
) (h *signalHandler) {
	h = &signalHandler{
		logger:  &atomic.Pointer[slog.Logger]{},
		mu:      &sync.Mutex{},
		signals: signals,
		cleanup: cleanup,
	}

	h.logger.Store(slog.Default())

	return h
}

// swapLogger replaces the stored logger with the given logger.
func (h *signalHandler) swapLogger(logger *slog.Logger) {
	h.logger.Swap(logger)
}

// addClientStorage stores the client storage.
func (h *signalHandler) addClientStorage(s *client.Storage) {
	h.mu.Lock()
	defer h.mu.Unlock()

	h.clientStorage = s
}

// addTLSManager stores the TLS manager.
func (h *signalHandler) addTLSManager(m *tlsManager) {
	h.mu.Lock()
	defer h.mu.Unlock()

	h.tlsManager = m
}

// handle processes incoming signals.  It blocks until a signal is received.  It
// reloads configurations of stored entities on SIGHUP, or performs cleanup on
// all other signals.  It is intended to be used as a goroutine.
func (h *signalHandler) handle(ctx context.Context) {
	// NOTE:  Avoid using [slogutil.RecoverAndExit] to prevent immediate
	// evaluation of the logger.
	defer func() {
		v := recover()
		if v == nil {
			return
		}

		slogutil.PrintRecovered(ctx, h.logger.Load(), v)

		os.Exit(osutil.ExitCodeFailure)
	}()

	for {
		sig := <-h.signals
		h.logger.Load().InfoContext(ctx, "received signal", "signal", sig)
		switch sig {
		case syscall.SIGHUP:
			h.reloadConfig(ctx)
		default:
			h.cleanup(ctx)
		}
	}
}

// reloadConfig refreshes configurations of stored entities.
func (h *signalHandler) reloadConfig(ctx context.Context) {
	h.mu.Lock()
	defer h.mu.Unlock()

	if h.clientStorage != nil {
		h.clientStorage.ReloadARP(ctx)
	}

	if h.tlsManager != nil {
		h.tlsManager.reload()
	}
}

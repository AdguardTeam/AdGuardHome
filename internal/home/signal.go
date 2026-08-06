package home

import (
	"context"
	"log/slog"
	"os"
	"sync"
	"syscall"

	"github.com/AdguardTeam/AdGuardHome/internal/aghnet"
	"github.com/AdguardTeam/AdGuardHome/internal/aghtls"
	"github.com/AdguardTeam/AdGuardHome/internal/client"
	"github.com/AdguardTeam/golibs/logutil/slogutil"
	"github.com/AdguardTeam/golibs/osutil"
	"github.com/AdguardTeam/golibs/service"
)

// signalHandler processes incoming signals.  It reloads configurations of
// stored entities on SIGHUP and performs cleanup on all other signals.
//
// TODO(e.burkov):  Use [service.SignalHandler] instead.
type signalHandler struct {
	// logger is used to log the operation of the signal handler.
	logger *slog.Logger

	// mu protects clientStorage, tlsManager, and web.
	mu *sync.Mutex

	// clientStorage is used to reload information about runtime clients with an
	// ARP source.
	clientStorage *client.Storage

	// tlsManager is used to reload the TLS configuration.
	tlsManager aghtls.Manager

	// web is the web API server.
	web service.Shutdowner

	// signals receives incoming signals.
	signals <-chan os.Signal

	// cleanup is called to perform cleanup on all incoming signals, except
	// SIGHUP.
	cleanup func(ctx context.Context)
}

// newSignalHandler returns a new properly initialized *signalHandler.
func newSignalHandler(
	l *slog.Logger,
	signals <-chan os.Signal,
	cleanup func(ctx context.Context),
) (h *signalHandler) {
	return &signalHandler{
		logger:  l,
		mu:      &sync.Mutex{},
		signals: signals,
		cleanup: cleanup,
	}
}

// addClientStorage stores the client storage.
func (h *signalHandler) addClientStorage(s *client.Storage) {
	h.mu.Lock()
	defer h.mu.Unlock()

	h.clientStorage = s
}

// addTLSManager stores the TLS manager.
func (h *signalHandler) addTLSManager(m aghtls.Manager) {
	h.mu.Lock()
	defer h.mu.Unlock()

	h.tlsManager = m
}

// addWeb stores the web API server.
func (h *signalHandler) addWeb(w *webAPI) {
	h.mu.Lock()
	defer h.mu.Unlock()

	h.web = w
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

		slogutil.PrintRecovered(ctx, h.logger, v)

		os.Exit(osutil.ExitCodeFailure)
	}()

	for {
		sig := <-h.signals
		h.logger.InfoContext(ctx, "received signal", "signal", sig)
		switch sig {
		case syscall.SIGHUP:
			h.reloadConfig(ctx)
		default:
			h.shutdown(ctx)
		}
	}
}

// shutdown shuts the system down.
func (h *signalHandler) shutdown(ctx context.Context) {
	h.mu.Lock()
	defer h.mu.Unlock()

	if h.tlsManager != nil {
		err := h.tlsManager.Shutdown(ctx)
		if err != nil {
			h.logger.ErrorContext(ctx, "shutting down tls manager", slogutil.KeyError, err)
		}
	}

	if h.web != nil {
		err := h.web.Shutdown(ctx)
		if err != nil {
			h.logger.ErrorContext(ctx, "shutting down web", slogutil.KeyError, err)
		}
	}

	h.cleanup(ctx)
}

// reloadConfig refreshes configurations of stored entities.
func (h *signalHandler) reloadConfig(ctx context.Context) {
	h.mu.Lock()
	defer h.mu.Unlock()

	if h.clientStorage != nil {
		h.clientStorage.ReloadARP(ctx)
	}

	if h.tlsManager != nil {
		err := h.tlsManager.Refresh(ctx)
		if err != nil {
			h.logger.ErrorContext(ctx, "refreshing tls manager", slogutil.KeyError, err)
		}
	}
}

// signalHandlerCleanup performs application resources cleanup for
// [signalHandler].
type signalHandlerCleanup struct {
	logger          *slog.Logger
	hostsContainer  *aghnet.HostsContainer
	glTokenFileRoot *os.Root
	hcWatcher       service.Interface
	done            chan struct{}
	pidFilePath     string
	glinetMode      bool
}

// cleanup performs application cleanup.
func (c *signalHandlerCleanup) cleanup(ctx context.Context) {
	defer close(c.done)

	cleanup(ctx, c.logger, c.hostsContainer)
	cleanupAlways(ctx, c.logger, c.pidFilePath)

	if c.glinetMode {
		err := c.glTokenFileRoot.Close()
		checkCleanupErr(ctx, c.logger, err, "closing glinet token root")
	}

	if c.hcWatcher != nil {
		err := c.hcWatcher.Shutdown(ctx)
		checkCleanupErr(ctx, c.logger, err, "shutting down hosts file watcher")
	}
}

// checkCleanupErr logs err and exits with [osutil.ExitCodeFailure] if err is
// not nil.  l must not be nil.
func checkCleanupErr(ctx context.Context, l *slog.Logger, err error, msg string) {
	if err == nil {
		return
	}

	l.ErrorContext(ctx, msg, slogutil.KeyError, err)
	os.Exit(osutil.ExitCodeFailure)
}

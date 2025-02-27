package home

import (
	"context"
	"os"
	"sync"
	"syscall"

	"github.com/AdguardTeam/AdGuardHome/internal/client"
	"github.com/AdguardTeam/golibs/log"
)

// signalHandler processes incoming signals.  It reloads configurations of
// stored entities on SIGHUP and performs cleanup on all other signals.
type signalHandler struct {
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
	return &signalHandler{
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
func (h *signalHandler) addTLSManager(m *tlsManager) {
	h.mu.Lock()
	defer h.mu.Unlock()

	h.tlsManager = m
}

// handle processes incoming signals.  It blocks until a signal is received.  It
// reloads configurations of stored entities on SIGHUP, or performs cleanup on
// all other signals.  It is intended to be used as a goroutine.
func (h *signalHandler) handle(ctx context.Context) {
	defer log.OnPanic("handling signal")

	for {
		sig := <-h.signals
		log.Info("received signal %q", sig)
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

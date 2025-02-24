package home

import (
	"context"
	"os"
	"sync"
	"syscall"

	"github.com/AdguardTeam/AdGuardHome/internal/client"
	"github.com/AdguardTeam/golibs/log"
)

// TODO(s.chzhen): !! Improve documentation, naming.
type signalHandler struct {
	mu            *sync.Mutex
	clientStorage *client.Storage
	tlsManager    *tlsManager
	signals       <-chan os.Signal
	cleanup       func(ctx context.Context)
}

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

func (h *signalHandler) addClientStorage(s *client.Storage) {
	h.mu.Lock()
	defer h.mu.Unlock()

	h.clientStorage = s
}

func (h *signalHandler) addTLSManager(m *tlsManager) {
	h.mu.Lock()
	defer h.mu.Unlock()

	h.tlsManager = m
}

func (h *signalHandler) handle(ctx context.Context) {
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

package cmd

import (
	"os"

	"github.com/AdguardTeam/AdGuardHome/internal/aghos"
	"github.com/AdguardTeam/AdGuardHome/internal/v1/agh"
	"github.com/AdguardTeam/golibs/log"
)

// signalHandler processes incoming signals and shuts services down.
type signalHandler struct {
	signal chan os.Signal

	// services are the services that are shut down before application
	// exiting.
	services []agh.Service
}

// handle processes OS signals.
func (h *signalHandler) handle() {
	defer log.OnPanic("signalProcessor.handle")

	for sig := range h.signal {
		log.Info("sigproc: received signal %q", sig)

		if aghos.IsShutdownSignal(sig) {
			h.shutdown()
		}
	}
}

// Exit status constants.
const (
	statusSuccess = 0
	statusError   = 1
)

// shutdown gracefully shuts down all services.
func (h *signalHandler) shutdown() {
	ctx, cancel := ctxWithDefaultTimeout()
	defer cancel()

	status := statusSuccess

	log.Info("sigproc: shutting down services")
	for i, service := range h.services {
		err := service.Shutdown(ctx)
		if err != nil {
			log.Error("sigproc: shutting down service at index %d: %s", i, err)
			status = statusError
		}
	}

	log.Info("sigproc: shutting down adguard home")

	os.Exit(status)
}

// newSignalHandler returns a new signalHandler that shuts down svcs.
func newSignalHandler(svcs ...agh.Service) (h *signalHandler) {
	h = &signalHandler{
		signal:   make(chan os.Signal, 1),
		services: svcs,
	}

	aghos.NotifyShutdownSignal(h.signal)

	return h
}

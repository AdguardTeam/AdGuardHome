package cmd

import (
	"io/fs"
	"os"
	"time"

	"github.com/AdguardTeam/AdGuardHome/internal/aghos"
	"github.com/AdguardTeam/AdGuardHome/internal/next/agh"
	"github.com/AdguardTeam/golibs/log"
)

// signalHandler processes incoming signals and shuts services down.
type signalHandler struct {
	// signal is the channel to which OS signals are sent.
	signal chan os.Signal

	// confFile is the path to the configuration file.
	confFile string

	// frontend is the filesystem with the frontend and other statically
	// compiled files.
	frontend fs.FS

	// start is the time at which AdGuard Home has been started.
	start time.Time

	// services are the services that are shut down before application exiting.
	services []agh.Service
}

// handle processes OS signals.
func (h *signalHandler) handle() {
	defer log.OnPanic("signalHandler.handle")

	for sig := range h.signal {
		log.Info("sighdlr: received signal %q", sig)

		if aghos.IsReconfigureSignal(sig) {
			h.reconfigure()
		} else if aghos.IsShutdownSignal(sig) {
			status := h.shutdown()
			log.Info("sighdlr: exiting with status %d", status)

			os.Exit(status)
		}
	}
}

// reconfigure rereads the configuration file and updates and restarts services.
func (h *signalHandler) reconfigure() {
	log.Info("sighdlr: reconfiguring adguard home")

	status := h.shutdown()
	if status != statusSuccess {
		log.Info("sighdlr: reconfiguring: exiting with status %d", status)

		os.Exit(status)
	}

	// TODO(a.garipov): This is a very rough way to do it.  Some services can be
	// reconfigured without the full shutdown, and the error handling is
	// currently not the best.

	confMgr, err := newConfigMgr(h.confFile, h.frontend, h.start)
	check(err)

	web := confMgr.Web()
	err = web.Start()
	check(err)

	dns := confMgr.DNS()
	err = dns.Start()
	check(err)

	h.services = []agh.Service{
		dns,
		web,
	}

	log.Info("sighdlr: successfully reconfigured adguard home")
}

// Exit status constants.
const (
	statusSuccess = 0
	statusError   = 1
)

// shutdown gracefully shuts down all services.
func (h *signalHandler) shutdown() (status int) {
	ctx, cancel := ctxWithDefaultTimeout()
	defer cancel()

	status = statusSuccess

	log.Info("sighdlr: shutting down services")
	for i, service := range h.services {
		err := service.Shutdown(ctx)
		if err != nil {
			log.Error("sighdlr: shutting down service at index %d: %s", i, err)
			status = statusError
		}
	}

	return status
}

// newSignalHandler returns a new signalHandler that shuts down svcs.
func newSignalHandler(
	confFile string,
	frontend fs.FS,
	start time.Time,
	svcs ...agh.Service,
) (h *signalHandler) {
	h = &signalHandler{
		signal:   make(chan os.Signal, 1),
		confFile: confFile,
		frontend: frontend,
		start:    start,
		services: svcs,
	}

	aghos.NotifyShutdownSignal(h.signal)
	aghos.NotifyReconfigureSignal(h.signal)

	return h
}

package cmd

import (
	"os"
	"strconv"

	"github.com/AdguardTeam/AdGuardHome/internal/aghos"
	"github.com/AdguardTeam/AdGuardHome/internal/next/agh"
	"github.com/AdguardTeam/AdGuardHome/internal/next/configmgr"
	"github.com/AdguardTeam/golibs/log"
	"github.com/google/renameio/v2/maybe"
)

// signalHandler processes incoming signals and shuts services down.
type signalHandler struct {
	// confMgrConf contains the configuration parameters for the configuration
	// manager.
	confMgrConf *configmgr.Config

	// signal is the channel to which OS signals are sent.
	signal chan os.Signal

	// pidFile is the path to the file where to store the PID, if any.
	pidFile string

	// services are the services that are shut down before application exiting.
	services []agh.Service
}

// handle processes OS signals.
func (h *signalHandler) handle() {
	defer log.OnPanic("signalHandler.handle")

	h.writePID()

	for sig := range h.signal {
		log.Info("sighdlr: received signal %q", sig)

		if aghos.IsReconfigureSignal(sig) {
			h.reconfigure()
		} else if aghos.IsShutdownSignal(sig) {
			status := h.shutdown()
			h.removePID()

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

	confMgr, err := newConfigMgr(h.confMgrConf)
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
	statusSuccess       = 0
	statusError         = 1
	statusArgumentError = 2
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
	confMgrConf *configmgr.Config,
	pidFile string,
	svcs ...agh.Service,
) (h *signalHandler) {
	h = &signalHandler{
		confMgrConf: confMgrConf,
		signal:      make(chan os.Signal, 1),
		pidFile:     pidFile,
		services:    svcs,
	}

	aghos.NotifyShutdownSignal(h.signal)
	aghos.NotifyReconfigureSignal(h.signal)

	return h
}

// writePID writes the PID to the file, if needed.  Any errors are reported to
// log.
func (h *signalHandler) writePID() {
	if h.pidFile == "" {
		return
	}

	// Use 8, since most PIDs will fit.
	data := make([]byte, 0, 8)
	data = strconv.AppendInt(data, int64(os.Getpid()), 10)
	data = append(data, '\n')

	err := maybe.WriteFile(h.pidFile, data, 0o644)
	if err != nil {
		log.Error("sighdlr: writing pidfile: %s", err)

		return
	}

	log.Debug("sighdlr: wrote pid to %q", h.pidFile)
}

// removePID removes the PID file, if any.
func (h *signalHandler) removePID() {
	if h.pidFile == "" {
		return
	}

	err := os.Remove(h.pidFile)
	if err != nil {
		log.Error("sighdlr: removing pidfile: %s", err)

		return
	}

	log.Debug("sighdlr: removed pid at %q", h.pidFile)
}

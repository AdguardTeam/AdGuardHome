package cmd

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"strconv"
	"time"

	"github.com/AdguardTeam/AdGuardHome/internal/next/configmgr"
	"github.com/AdguardTeam/golibs/errors"
	"github.com/AdguardTeam/golibs/logutil/slogutil"
	"github.com/AdguardTeam/golibs/osutil"
	"github.com/AdguardTeam/golibs/service"
	"github.com/google/renameio/v2/maybe"
)

// signalHandler processes incoming signals and shuts services down.
type signalHandler struct {
	// logger is used for logging the operation of the signal handler.
	logger *slog.Logger

	// confMgrConf contains the configuration parameters for the configuration
	// manager.
	confMgrConf *configmgr.Config

	// signal is the channel to which OS signals are sent.
	signal chan os.Signal

	// pidFile is the path to the file where to store the PID, if any.
	pidFile string

	// services are the services that are shut down before application exiting.
	services []service.Interface

	// shutdownTimeout is the timeout for the shutdown operation.
	shutdownTimeout time.Duration
}

// handle processes OS signals.  It blocks until a termination or a
// reconfiguration signal is received, after which it either shuts down all
// services or reconfigures them.  ctx is used for logging and serves as the
// base for the shutdown timeout.  status is [osutil.ExitCodeSuccess] on success
// and [osutil.ExitCodeFailure] on error.
//
// TODO(a.garipov):  Add reconfiguration logic to golibs.
func (h *signalHandler) handle(ctx context.Context) (status osutil.ExitCode) {
	defer slogutil.RecoverAndLog(ctx, h.logger)

	h.writePID(ctx)

	for sig := range h.signal {
		h.logger.InfoContext(ctx, "received", "signal", sig)

		if osutil.IsReconfigureSignal(sig) {
			err := h.reconfigure(ctx)
			if err != nil {
				h.logger.ErrorContext(ctx, "reconfiguration error", slogutil.KeyError, err)

				return osutil.ExitCodeFailure
			}
		} else if osutil.IsShutdownSignal(sig) {
			status = h.shutdown(ctx)

			h.removePID(ctx)

			return status
		}
	}

	// Shouldn't happen, since h.signal is currently never closed.
	panic("unexpected close of h.signal")
}

// writePID writes the PID to the file, if needed.  Any errors are reported to
// log.
func (h *signalHandler) writePID(ctx context.Context) {
	if h.pidFile == "" {
		return
	}

	pid := os.Getpid()
	data := strconv.AppendInt(nil, int64(pid), 10)
	data = append(data, '\n')

	err := maybe.WriteFile(h.pidFile, data, 0o644)
	if err != nil {
		h.logger.ErrorContext(ctx, "writing pidfile", slogutil.KeyError, err)

		return
	}

	h.logger.DebugContext(ctx, "wrote pid", "file", h.pidFile, "pid", pid)
}

// reconfigure rereads the configuration file and updates and restarts services.
func (h *signalHandler) reconfigure(ctx context.Context) (err error) {
	h.logger.InfoContext(ctx, "reconfiguring started")

	status := h.shutdown(ctx)
	if status != osutil.ExitCodeSuccess {
		return errors.Error("shutdown failed")
	}

	// TODO(a.garipov):  This is a very rough way to do it.  Some services can
	// be reconfigured without the full shutdown, and the error handling is
	// currently not the best.

	var errs []error

	ctx, cancel := context.WithTimeout(ctx, defaultTimeoutStart)
	defer cancel()

	confMgr, err := newConfigMgr(ctx, h.confMgrConf)
	if err != nil {
		errs = append(errs, fmt.Errorf("configuration manager: %w", err))
	}

	web := confMgr.Web()
	err = web.Start(ctx)
	if err != nil {
		errs = append(errs, fmt.Errorf("starting web: %w", err))
	}

	dns := confMgr.DNS()
	err = dns.Start(ctx)
	if err != nil {
		errs = append(errs, fmt.Errorf("starting dns: %w", err))
	}

	if len(errs) > 0 {
		return errors.Join(errs...)
	}

	h.services = []service.Interface{
		dns,
		web,
	}

	h.logger.InfoContext(ctx, "reconfiguring finished")

	return nil
}

// shutdown gracefully shuts down all services.
func (h *signalHandler) shutdown(ctx context.Context) (status int) {
	ctx, cancel := context.WithTimeout(ctx, h.shutdownTimeout)
	defer cancel()

	status = osutil.ExitCodeSuccess

	h.logger.InfoContext(ctx, "shutting down")
	for i, svc := range h.services {
		err := svc.Shutdown(ctx)
		if err != nil {
			h.logger.ErrorContext(ctx, "shutting down service", "idx", i, slogutil.KeyError, err)
			status = osutil.ExitCodeFailure
		}
	}

	return status
}

// newSignalHandler returns a new signalHandler that shuts down svcs.  logger
// and confMgrConf must not be nil.
func newSignalHandler(
	logger *slog.Logger,
	confMgrConf *configmgr.Config,
	pidFile string,
	svcs ...service.Interface,
) (h *signalHandler) {
	h = &signalHandler{
		logger:          logger,
		confMgrConf:     confMgrConf,
		signal:          make(chan os.Signal, 1),
		pidFile:         pidFile,
		services:        svcs,
		shutdownTimeout: defaultTimeoutShutdown,
	}

	notifier := osutil.DefaultSignalNotifier{}
	osutil.NotifyShutdownSignal(notifier, h.signal)
	osutil.NotifyReconfigureSignal(notifier, h.signal)

	return h
}

// removePID removes the PID file, if any.
func (h *signalHandler) removePID(ctx context.Context) {
	if h.pidFile == "" {
		return
	}

	err := os.Remove(h.pidFile)
	if err != nil {
		h.logger.ErrorContext(ctx, "removing pidfile", slogutil.KeyError, err)

		return
	}

	h.logger.DebugContext(ctx, "removed pidfile", "file", h.pidFile)
}

package cmd

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"strconv"
	"sync"

	"github.com/AdguardTeam/AdGuardHome/internal/next/configmgr"
	"github.com/AdguardTeam/golibs/errors"
	"github.com/AdguardTeam/golibs/logutil/slogutil"
	"github.com/AdguardTeam/golibs/service"
	"github.com/google/renameio/v2/maybe"
)

// serviceMgr manages AdGuard Home services.
type serviceMgr struct {
	// confMgrMu protects confMgr.
	confMgrMu *sync.RWMutex

	confMgr     *configmgr.Manager
	confMgrConf *configmgr.Config
	logger      *slog.Logger
	pidFilePath string
}

// serviceMgrConfig contains service manager configuration parameters.
type serviceMgrConfig struct {
	// confMgrConf is the configuration manager config, it must not be nil.
	confMgrConf *configmgr.Config

	// logger is the logger used to log services activity, it must not be nil.
	logger *slog.Logger

	// pidFilePath is the path to the file where to store the PID, if any.
	pidFilePath string
}

// newServiceMgr creates a new *serviceMgr.
func newServiceMgr(ctx context.Context, conf *serviceMgrConfig) (s *serviceMgr, err error) {
	confMgr, err := configmgr.New(ctx, conf.confMgrConf)
	if err != nil {
		return nil, fmt.Errorf("creating config manager: %w", err)
	}

	return &serviceMgr{
		confMgr:     confMgr,
		confMgrMu:   &sync.RWMutex{},
		confMgrConf: conf.confMgrConf,
		logger:      conf.logger,
		pidFilePath: conf.pidFilePath,
	}, nil
}

// type check
var _ service.Interface = (*serviceMgr)(nil)

// Start implements the [service.Interface] interface for *serviceMgr.
func (s *serviceMgr) Start(ctx context.Context) (err error) {
	s.writePID(ctx)

	s.confMgrMu.RLock()
	defer s.confMgrMu.RUnlock()

	var errs []error

	err = s.confMgr.Web().Start(ctx)
	if err != nil {
		errs = append(errs, fmt.Errorf("starting web: %w", err))
	}

	err = s.confMgr.DNS().Start(ctx)
	if err != nil {
		errs = append(errs, fmt.Errorf("starting dnssvc: %w", err))
	}

	return errors.Join(errs...)
}

// writePID writes the PID to the file.  Any errors are reported to log.
func (s *serviceMgr) writePID(ctx context.Context) {
	if s.pidFilePath == "" {
		return
	}

	pid := os.Getpid()
	data := strconv.AppendInt(nil, int64(pid), 10)
	data = append(data, '\n')

	err := maybe.WriteFile(s.pidFilePath, data, 0o644)
	if err != nil {
		s.logger.ErrorContext(ctx, "writing pidfile", slogutil.KeyError, err)

		return
	}

	s.logger.DebugContext(ctx, "wrote pid", "file", s.pidFilePath, "pid", pid)
}

// Shutdown implements the [service.Interface] interface for *serviceMgr.
func (s *serviceMgr) Shutdown(ctx context.Context) (err error) {
	s.confMgrMu.RLock()
	defer s.confMgrMu.RUnlock()

	var errs []error

	err = s.confMgr.Web().Shutdown(ctx)
	if err != nil {
		errs = append(errs, fmt.Errorf("shutting down web: %w", err))
	}

	err = s.confMgr.DNS().Shutdown(ctx)
	if err != nil {
		errs = append(errs, fmt.Errorf("shutting down dnssvc: %w", err))
	}

	s.removePID(ctx)

	return errors.Join(errs...)
}

// removePID removes the PID file.  Any errors are reported to log.
func (s *serviceMgr) removePID(ctx context.Context) {
	if s.pidFilePath == "" {
		return
	}

	err := os.Remove(s.pidFilePath)
	if err != nil {
		s.logger.ErrorContext(ctx, "removing pidfile", slogutil.KeyError, err)

		return
	}

	s.logger.DebugContext(ctx, "removed pidfile", "file", s.pidFilePath)
}

// type check
var _ service.Refresher = (*serviceMgr)(nil)

// Refresh implements the [service.Refresher] interface for *serviceMgr.
func (s *serviceMgr) Refresh(ctx context.Context) (err error) {
	s.logger.InfoContext(ctx, "reconfiguring started")

	err = s.Shutdown(ctx)
	if err != nil {
		return fmt.Errorf("shutdown failed: %w", err)
	}

	// TODO(a.garipov):  This is a very rough way to do it.  Some services can
	// be reconfigured without the full shutdown, and the error handling is
	// currently not the best.

	ctx, cancel := context.WithTimeout(ctx, defaultTimeoutStart)
	defer cancel()

	err = s.updConfMgr(ctx)
	if err != nil {
		return fmt.Errorf("updating configuration manager: %w", err)
	}

	err = s.Start(ctx)
	if err != nil {
		return fmt.Errorf("restarting services: %w", err)
	}

	s.logger.InfoContext(ctx, "reconfiguring finished")

	return nil
}

// updConfMgr updates the configuration manager.
func (s *serviceMgr) updConfMgr(ctx context.Context) (err error) {
	confMgr, err := configmgr.New(ctx, s.confMgrConf)
	if err != nil {
		return fmt.Errorf("creating config manager: %w", err)
	}

	s.confMgrMu.Lock()
	defer s.confMgrMu.Unlock()

	s.confMgr = confMgr

	return nil
}

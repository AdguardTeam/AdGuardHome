package ossvc

import (
	"context"
	_ "embed"
	"fmt"
	"log/slog"

	"github.com/AdguardTeam/golibs/errors"
	"github.com/kardianos/service"
)

// TODO(e.burkov):  Declare managers for each OS.

// manager is the implementation of [Manager] that wraps [service.Service].
type manager struct {
	logger *slog.Logger
}

// newManager creates a new [Manager] that wraps [service.Service].
//
// TODO(e.burkov):  Return error.
func newManager(_ context.Context, conf *ManagerConfig) (mgr *manager) {
	return &manager{
		logger: conf.Logger,
	}
}

// type check
var _ Manager = (*manager)(nil)

// Perform implements the [Manager] interface for *manager.
func (m *manager) Perform(ctx context.Context, action Action) (err error) {
	switch action := action.(type) {
	case *ActionInstall:
		return m.install(ctx, action)
	case *ActionReload:
		return m.reload(ctx, action)
	case *ActionStart:
		return m.start(ctx, action)
	case *ActionStop:
		return m.stop(ctx, action)
	case *ActionUninstall:
		return m.uninstall(ctx, action)
	default:
		return fmt.Errorf("action: %w: %T(%[2]v)", errors.ErrBadEnumValue, action)
	}
}

// install installs the service in the service manager.
func (m *manager) install(ctx context.Context, action *ActionInstall) (err error) {
	m.logger.InfoContext(ctx, "installing service", "name", action.ServiceConf.Name)

	s, err := service.New(nil, action.ServiceConf)
	if err != nil {
		return fmt.Errorf("creating service: %w", err)
	}

	return s.Install()
}

// reload stops, if not yet, and starts the configured service in the service
// manager.
func (m *manager) reload(ctx context.Context, action *ActionReload) (err error) {
	m.logger.InfoContext(ctx, "reloading service", "name", action.ServiceConf.Name)

	s, err := service.New(nil, action.ServiceConf)
	if err != nil {
		return fmt.Errorf("creating service: %w", err)
	}

	return s.Restart()
}

// start starts the configured service in the service manager.
func (m *manager) start(ctx context.Context, action *ActionStart) (err error) {
	m.logger.InfoContext(ctx, "starting service", "name", action.ServiceConf.Name)

	s, err := service.New(nil, action.ServiceConf)
	if err != nil {
		return fmt.Errorf("creating service: %w", err)
	}

	return s.Start()
}

// Status implements the [Manager] interface for *manager.
func (m *manager) Status(ctx context.Context, name ServiceName) (status Status, err error) {
	m.logger.InfoContext(ctx, "getting service status", "name", name)

	s, err := service.New(nil, &service.Config{
		Name: string(name),
	})
	if err != nil {
		return "", fmt.Errorf("creating service: %w", err)
	}

	svcStatus, err := s.Status()
	if err != nil {
		if errors.Is(err, service.ErrNotInstalled) {
			return StatusNotInstalled, nil
		}

		return "", fmt.Errorf("getting service status: %w", err)
	}

	switch svcStatus {
	case service.StatusRunning:
		return StatusRunning, nil
	case service.StatusStopped:
		return StatusStopped, nil
	default:
		return "", fmt.Errorf("service status: %w: %v", errors.ErrBadEnumValue, svcStatus)
	}
}

// stop stops the service in the service manager.
func (m *manager) stop(ctx context.Context, action *ActionStop) (err error) {
	m.logger.InfoContext(ctx, "stopping service", "name", action.ServiceConf.Name)

	s, err := service.New(nil, action.ServiceConf)
	if err != nil {
		return fmt.Errorf("creating service: %w", err)
	}

	return s.Stop()
}

// uninstall uninstalls the service from the service manager.
func (m *manager) uninstall(ctx context.Context, action *ActionUninstall) (err error) {
	m.logger.InfoContext(ctx, "uninstalling service", "name", action.ServiceConf.Name)

	s, err := service.New(nil, action.ServiceConf)
	if err != nil {
		return fmt.Errorf("creating service: %w", err)
	}

	return s.Uninstall()
}

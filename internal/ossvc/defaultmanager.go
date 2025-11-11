package ossvc

import (
	"context"
	_ "embed"
	"fmt"
	"log/slog"
	"os"
	"runtime"

	"github.com/AdguardTeam/AdGuardHome/internal/aghos"
	"github.com/AdguardTeam/golibs/errors"
	"github.com/AdguardTeam/golibs/osutil/executil"
	"github.com/kardianos/service"
)

// TODO(e.burkov):  Declare managers for each OS.

// manager is the implementation of [Manager] that wraps [service.Service].
type manager struct {
	logger  *slog.Logger
	cmdCons executil.CommandConstructor
}

// newManager creates a new [Manager] that wraps [service.Service].
//
// TODO(e.burkov):  Return error.
func newManager(_ context.Context, conf *ManagerConfig) (mgr *manager) {
	// Call chooseSystem explicitly to introduce platform-specific support for
	// service package.  It's a noop for other GOOS values.
	chooseSystem()

	return &manager{
		logger:  conf.Logger,
		cmdCons: conf.CommandConstructor,
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
	if err != nil && service.Platform() == "unix-systemv" {
		var code int
		code, err = m.runInitdCommand(ctx, string(name), "status")
		if err != nil || code != 0 {
			return StatusStopped, nil
		}

		return StatusRunning, nil
	}

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

// install installs the service in the service manager.
func (m *manager) install(ctx context.Context, action *ActionInstall) (err error) {
	m.logger.InfoContext(ctx, "installing service", "name", action.ServiceConf.Name)

	s, err := service.New(nil, action.ServiceConf)
	if err != nil {
		return fmt.Errorf("creating service: %w", err)
	}

	if err = s.Install(); err != nil {
		return fmt.Errorf("installing service: %w", err)
	}

	if aghos.IsOpenWrt() {
		// On OpenWrt it is important to run enable after the service
		// installation.  Otherwise, the service won't start on the system
		// startup.
		_, err = m.runInitdCommand(ctx, action.ServiceConf.Name, "enable")
		if err != nil {
			return fmt.Errorf("enabling service on openwrt: %w", err)
		}
	}

	return nil
}

// reload stops, if not yet, and starts the configured service in the service
// manager.
func (m *manager) reload(ctx context.Context, action *ActionReload) (err error) {
	m.logger.InfoContext(ctx, "reloading service", "name", action.ServiceConf.Name)

	s, err := service.New(nil, action.ServiceConf)
	if err != nil {
		return fmt.Errorf("creating service: %w", err)
	}

	err = s.Restart()
	if err != nil && service.Platform() == "unix-systemv" {
		_, err = m.runInitdCommand(ctx, action.ServiceConf.Name, "restart")
	}

	return err
}

// start starts the configured service in the service manager.
func (m *manager) start(ctx context.Context, action *ActionStart) (err error) {
	m.logger.InfoContext(ctx, "starting service", "name", action.ServiceConf.Name)

	// Perform pre-check before starting service.
	if err = aghos.PreCheckActionStart(); err != nil {
		m.logger.ErrorContext(ctx, "pre-check failed", "err", err)
	}

	s, err := service.New(nil, action.ServiceConf)
	if err != nil {
		return fmt.Errorf("creating service: %w", err)
	}

	err = s.Start()
	if err != nil && service.Platform() == "unix-systemv" {
		_, err = m.runInitdCommand(ctx, action.ServiceConf.Name, "start")
	}

	return err
}

// stop stops the service in the service manager.
func (m *manager) stop(ctx context.Context, action *ActionStop) (err error) {
	m.logger.InfoContext(ctx, "stopping service", "name", action.ServiceConf.Name)

	s, err := service.New(nil, action.ServiceConf)
	if err != nil {
		return fmt.Errorf("creating service: %w", err)
	}

	err = s.Stop()
	if err != nil && service.Platform() == "unix-systemv" {
		_, err = m.runInitdCommand(ctx, action.ServiceConf.Name, "stop")
	}

	return err
}

// uninstall uninstalls the service from the service manager.
func (m *manager) uninstall(ctx context.Context, action *ActionUninstall) (err error) {
	m.logger.InfoContext(ctx, "uninstalling service", "name", action.ServiceConf.Name)

	if aghos.IsOpenWrt() {
		// On OpenWrt it is important to run disable command first as it will
		// remove the symlink.
		_, err = m.runInitdCommand(ctx, action.ServiceConf.Name, "disable")
		if err != nil {
			return fmt.Errorf("disabling service on openwrt: %w", err)
		}
	}

	s, err := service.New(nil, action.ServiceConf)
	if err != nil {
		return fmt.Errorf("creating service: %w", err)
	}

	if err = s.Stop(); err != nil {
		m.logger.DebugContext(ctx, "stopping service", "err", err)
	}

	if err = s.Uninstall(); err != nil {
		return fmt.Errorf("uninstalling service: %w", err)
	}

	removeLaunchdStdLogs(ctx, m.logger)

	return nil
}

// Paths to stdout and stderr logs for Darwin service manager.
//
// TODO(e.burkov):  Move to config_darwin.go.
const (
	launchdStdoutPath = "/var/log/AdGuardHome.stdout.log"
	launchdStderrPath = "/var/log/AdGuardHome.stderr.log"
)

// removeLaunchdStdLogs removes launchd stdout and stderr log files, if needed,
// and logs errors at warning level.
//
// TODO(e.burkov):  Move to manager_darwin.go.
func removeLaunchdStdLogs(ctx context.Context, logger *slog.Logger) {
	if runtime.GOOS != "darwin" {
		return
	}

	err := os.Remove(launchdStdoutPath)
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		logger.WarnContext(ctx, "removing stdout file", "err", err)
	}

	err = os.Remove(launchdStderrPath)
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		logger.WarnContext(ctx, "removing stderr file", "err", err)
	}
}

// runInitdCommand runs init.d service command.  It returns command code or
// error if any.
//
// TODO(e.burkov):  Move to manager_linux.go.
func (m *manager) runInitdCommand(
	ctx context.Context,
	serviceName string,
	action string,
) (code int, err error) {
	confPath := "/etc/init.d/" + serviceName
	// Pass the script and action as a single string argument.
	code, _, err = aghos.RunCommand(ctx, m.cmdCons, "sh", "-c", confPath, action)

	return code, err
}

package ossvc

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"runtime"

	"github.com/AdguardTeam/AdGuardHome/internal/aghos"
	"github.com/AdguardTeam/golibs/errors"
	"github.com/AdguardTeam/golibs/osutil/executil"
	"github.com/kardianos/service"
)

// TODO(e.burkov):  Declare managers for each OS.

// manager is the implementation of [Manager] that uses [service.Service].
type manager struct {
	logger        *slog.Logger
	cmdCons       executil.CommandConstructor
	isOpenWrt     bool
	isUnixSystemV bool
}

// newManager creates a new [Manager] that uses [service.Service].
//
// TODO(e.burkov):  Return error.
func newManager(_ context.Context, conf *ManagerConfig) (mgr *manager) {
	// Call chooseSystem explicitly to introduce platform-specific support for
	// service package.  It's a noop for other GOOS values.
	chooseSystem()

	return &manager{
		logger:        conf.Logger,
		cmdCons:       conf.CommandConstructor,
		isOpenWrt:     aghos.IsOpenWrt(),
		isUnixSystemV: service.Platform() == "unix-systemv",
	}
}

// type check
var _ Manager = (*manager)(nil)

// Perform implements the [Manager] interface for *manager.
func (m *manager) Perform(ctx context.Context, action Action) (err error) {
	switch action := action.(type) {
	case *ActionInstall:
		err = m.install(ctx, action)
	case *ActionRestart:
		err = m.restart(ctx, action)
	case *ActionStart:
		err = m.start(ctx, action)
	case *ActionStop:
		err = m.stop(ctx, action)
	case *ActionUninstall:
		err = m.uninstall(ctx, action)
	default:
		panic(fmt.Errorf("action: %w: %T(%[2]v)", errors.ErrBadEnumValue, action))
	}
	if err != nil {
		// Don't wrap the error, since it's informative enough as is.
		return err
	}

	m.logger.DebugContext(
		ctx,
		"performed service action",
		"action", action.Name(),
		"system", service.ChosenSystem(),
	)

	return nil
}

// statusRestartOnFail is a custom status value used to indicate the service's
// state of restarting after failed start.
const statusRestartOnFail = service.StatusStopped + 1

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
	if err != nil && m.isUnixSystemV {
		var code int
		code, err = m.runInitdCommand(ctx, string(name), "status")
		if err != nil || code != 0 {
			// Treat an error or non-zero exit code as stopped status on Unix
			// System V.
			//
			// TODO(e.burkov):  Investigate if it's a valid assumption, and
			// properly handle errors in similar cases.
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

	return statusToInternal(svcStatus)
}

// type check
var _ ReloadManager = (*manager)(nil)

// Reload implements the [ReloadManager] interface for *manager.
func (m *manager) Reload(ctx context.Context, name ServiceName) (err error) {
	return m.reload(ctx, name)
}

// install installs the service in the service manager.
func (m *manager) install(ctx context.Context, action *ActionInstall) (err error) {
	m.logger.InfoContext(ctx, "installing service", "name", action.ServiceName)

	conf := &service.Config{
		Name:             string(action.ServiceName),
		DisplayName:      action.DisplayName,
		Description:      action.Description,
		WorkingDirectory: action.WorkingDirectory,
		Arguments:        action.Arguments,
	}
	ConfigureServiceOptions(conf, action.Version)

	s, err := service.New(emptyInterface{}, conf)
	if err != nil {
		return fmt.Errorf("creating service: %w", err)
	}

	err = s.Install()
	if err != nil {
		return fmt.Errorf("installing service: %w", err)
	}

	if m.isOpenWrt {
		// On OpenWrt it is important to run enable after the service
		// installation.  Otherwise, the service won't start on the system
		// startup.
		_, err = m.runInitdCommand(ctx, string(action.ServiceName), "enable")
		if err != nil {
			return fmt.Errorf("enabling service on openwrt: %w", err)
		}
	}

	return nil
}

// restart stops, if not yet, and starts the configured service in the service
// manager.
func (m *manager) restart(ctx context.Context, action *ActionRestart) (err error) {
	m.logger.InfoContext(ctx, "restarting service", "name", action.ServiceName)

	s, err := service.New(emptyInterface{}, &service.Config{
		Name: string(action.ServiceName),
	})
	if err != nil {
		return fmt.Errorf("creating service: %w", err)
	}

	err = s.Restart()
	if err != nil && m.isUnixSystemV {
		_, initdErr := m.runInitdCommand(ctx, string(action.ServiceName), "restart")
		if initdErr != nil {
			return fmt.Errorf("%w (restarting via init.d: %w)", err, initdErr)
		}
	}

	return err
}

// start starts the configured service in the service manager.
func (m *manager) start(ctx context.Context, action *ActionStart) (err error) {
	m.logger.InfoContext(ctx, "starting service", "name", action.ServiceName)

	// Perform pre-check before starting service.
	if err = aghos.PreCheckActionStart(); err != nil {
		m.logger.ErrorContext(ctx, "pre-check failed", "err", err)
	}

	s, err := service.New(emptyInterface{}, &service.Config{
		Name: string(action.ServiceName),
	})
	if err != nil {
		return fmt.Errorf("creating service: %w", err)
	}

	err = s.Start()
	if err != nil && m.isUnixSystemV {
		_, initdErr := m.runInitdCommand(ctx, string(action.ServiceName), "start")
		if initdErr != nil {
			return fmt.Errorf("%w (starting via init.d: %w)", err, initdErr)
		}
	}

	return err
}

// stop stops the service in the service manager.
func (m *manager) stop(ctx context.Context, action *ActionStop) (err error) {
	m.logger.InfoContext(ctx, "stopping service", "name", action.ServiceName)

	conf := &service.Config{
		Name: string(action.ServiceName),
	}

	s, err := service.New(emptyInterface{}, conf)
	if err != nil {
		return fmt.Errorf("creating service: %w", err)
	}

	err = s.Stop()
	if err != nil && m.isUnixSystemV {
		_, initdErr := m.runInitdCommand(ctx, string(action.ServiceName), "stop")
		if initdErr != nil {
			return fmt.Errorf("%w (stopping via init.d: %w)", err, initdErr)
		}
	}

	return err
}

// uninstall uninstalls the service from the service manager.
func (m *manager) uninstall(ctx context.Context, action *ActionUninstall) (err error) {
	m.logger.InfoContext(ctx, "uninstalling service", "name", action.ServiceName)

	if m.isOpenWrt {
		// On OpenWrt it is important to run disable command first as it will
		// remove the symlink.
		_, err = m.runInitdCommand(ctx, string(action.ServiceName), "disable")
		if err != nil {
			return fmt.Errorf("disabling service on openwrt: %w", err)
		}
	}

	s, err := service.New(emptyInterface{}, &service.Config{
		Name: string(action.ServiceName),
	})
	if err != nil {
		return fmt.Errorf("creating service: %w", err)
	}

	err = s.Stop()
	if err != nil {
		m.logger.DebugContext(ctx, "stopping service", "err", err)
	}

	err = s.Uninstall()
	if err != nil {
		return fmt.Errorf("uninstalling service: %w", err)
	}

	if runtime.GOOS == "darwin" {
		removeLaunchdStdLogs(ctx, m.logger)
	}

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
	confPath := filepath.Join("/etc", "init.d", serviceName)
	// Pass the script and action as a single string argument.
	//
	// TODO(e.burkov):  Use CommandConstructor.
	code, _, err = aghos.RunCommand(ctx, m.cmdCons, "sh", "-c", confPath, action)

	return code, err
}

// emptyInterface is an empty implementation of the [service.Interface], as the
// actual implementation is only needed for the [service.Service.Run] method.
type emptyInterface struct{}

// type check
var _ service.Interface = emptyInterface{}

// Start implements the [service.Interface] interface for emptyInterface.
func (emptyInterface) Start(_ service.Service) (err error) { return nil }

// Stop implements the [service.Interface] interface for emptyInterface.
func (emptyInterface) Stop(_ service.Service) (err error) { return nil }

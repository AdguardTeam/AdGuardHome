package home

import (
	"context"
	"fmt"
	"io/fs"
	"log/slog"
	"os"

	"github.com/AdguardTeam/AdGuardHome/internal/aghos"
	"github.com/AdguardTeam/AdGuardHome/internal/ossvc"
	"github.com/AdguardTeam/AdGuardHome/internal/version"
	"github.com/AdguardTeam/golibs/errors"
	"github.com/AdguardTeam/golibs/logutil/slogutil"
	"github.com/AdguardTeam/golibs/netutil/urlutil"
	"github.com/AdguardTeam/golibs/osutil/executil"
	"github.com/kardianos/service"
)

const (
	serviceName        = "AdGuardHome"
	serviceDisplayName = "AdGuard Home service"
	serviceDescription = "AdGuard Home: Network-level blocker"
)

// svcLogPrefix is the prefix for logging from service manager.
const svcLogPrefix = "service_manager"

// program represents the program that will be launched by as a service or a
// daemon.
//
// TODO(e.burkov):  Handle the run action as a direct execution instead of
// constructing a service instance and running it.  Perhaps, deprecate the
// action.
type program struct {
	ctx           context.Context
	clientBuildFS fs.FS
	signals       chan os.Signal
	done          chan struct{}
	opts          options
	baseLogger    *slog.Logger
	logger        *slog.Logger
	sigHdlr       *signalHandler
	workDir       string
	confPath      string
}

// type check
var _ service.Interface = (*program)(nil)

// Start implements service.Interface interface for *program.
func (p *program) Start(_ service.Service) (err error) {
	// Start should not block.  Do the actual work async.
	args := p.opts
	args.runningAsService = true

	go run(p.ctx, p.baseLogger, args, p.clientBuildFS, p.done, p.sigHdlr, p.workDir, p.confPath)

	return nil
}

// Stop implements service.Interface interface for *program.
func (p *program) Stop(_ service.Service) (err error) {
	p.logger.InfoContext(p.ctx, "stopping: waiting for cleanup")

	aghos.SendShutdownSignal(p.signals)

	// Wait for other goroutines to complete their job.
	<-p.done

	return nil
}

// handleRun runs p.
func (p *program) handleRun(
	ctx context.Context,
	baseLogger *slog.Logger,
	opts options,
) (err error) {
	pwd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("getting current directory: %w", err)
	}

	args := optsToArgs(opts)
	baseLogger.DebugContext(ctx, "using", "args", args)

	svcConfig := &service.Config{
		Name:             serviceName,
		DisplayName:      serviceDisplayName,
		Description:      serviceDescription,
		WorkingDirectory: pwd,
		Arguments:        args,
	}
	ossvc.ConfigureServiceOptions(svcConfig, version.Full())

	s, err := service.New(p, svcConfig)
	if err != nil {
		return fmt.Errorf("initializing service: %w", err)
	}

	return s.Run()
}

// restartService restarts the service.  It returns error if the service is not
// running.  l must not be nil.
func restartService(ctx context.Context, baseLogger *slog.Logger) (err error) {
	svcMgr, err := ossvc.NewManager(ctx, &ossvc.ManagerConfig{
		Logger:             baseLogger.With(slogutil.KeyPrefix, svcLogPrefix),
		CommandConstructor: executil.SystemCommandConstructor{},
	})
	if err != nil {
		return fmt.Errorf("initializing service manager: %w", err)
	}

	act := &ossvc.ActionRestart{
		ServiceName: serviceName,
	}

	err = svcMgr.Perform(ctx, act)
	if err != nil {
		return fmt.Errorf("restarting service: %w", err)
	}

	return nil
}

// handleServiceControlAction one of the possible control actions:
//
//   - install:  Installs a service/daemon.
//   - uninstall:  Uninstalls it.
//   - status:  Prints the service status.
//   - start:  Starts the previously installed service.
//   - stop:  Stops the previously installed service.
//   - restart:  Restarts the previously installed service.
//   - run:  This is a special command that is not supposed to be used directly
//     it is specified when we register a service, and it indicates to the app
//     that it is being run as a service/daemon.
func handleServiceControlAction(
	ctx context.Context,
	baseLogger *slog.Logger,
	l *slog.Logger,
	opts options,
	clientBuildFS fs.FS,
	signals chan os.Signal,
	done chan struct{},
	sigHdlr *signalHandler,
	workDir string,
	confPath string,
) (err error) {
	actionName := opts.serviceControlAction
	l.InfoContext(ctx, version.Full())
	l.InfoContext(ctx, "control", "action", actionName)

	// Create a service manager before even a run action, since it picks the
	// correct system implementation.
	svcMgr, err := ossvc.NewManager(ctx, &ossvc.ManagerConfig{
		Logger:             baseLogger.With(slogutil.KeyPrefix, svcLogPrefix),
		CommandConstructor: executil.SystemCommandConstructor{},
	})
	if err != nil {
		return fmt.Errorf("initializing service manager: %w", err)
	}

	if actionName == "run" {
		runOpts := opts
		runOpts.serviceControlAction = "run"

		p := &program{
			ctx:           ctx,
			clientBuildFS: clientBuildFS,
			signals:       signals,
			done:          done,
			opts:          runOpts,
			baseLogger:    baseLogger,
			logger:        baseLogger.With(slogutil.KeyPrefix, "service"),
			sigHdlr:       sigHdlr,
			workDir:       workDir,
			confPath:      confPath,
		}

		return p.handleRun(ctx, baseLogger, runOpts)
	}

	switch actionName {
	case "reload":
		err = handleServiceReloadCmd(ctx, l, svcMgr)
	case "status":
		err = handleServiceStatusCmd(ctx, l, svcMgr)
	default:
		err = handleServiceCommand(ctx, baseLogger, svcMgr, opts, workDir, confPath)
	}
	if err != nil {
		return fmt.Errorf("action %q: %w", actionName, err)
	}

	return nil
}

// handleServiceCommand handles service command.
func handleServiceCommand(
	ctx context.Context,
	l *slog.Logger,
	mgr ossvc.Manager,
	opts options,
	workDir string,
	confPath string,
) (err error) {
	var action ossvc.Action
	switch opts.serviceControlAction {
	case "install":
		return handleServiceInstallCmd(ctx, l, mgr, opts, workDir, confPath)
	case "uninstall":
		action = &ossvc.ActionUninstall{
			ServiceName: serviceName,
		}
	case "start":
		action = &ossvc.ActionStart{
			ServiceName: serviceName,
		}
	case "stop":
		action = &ossvc.ActionStop{
			ServiceName: serviceName,
		}
	case "restart":
		action = &ossvc.ActionRestart{
			ServiceName: serviceName,
		}
	default:
		return fmt.Errorf("%w: %q", errors.ErrBadEnumValue, opts.serviceControlAction)
	}

	return mgr.Perform(ctx, action)
}

// handleServiceStatusCmd logs the service's status.  l and mgr must not be
// nil.
func handleServiceStatusCmd(ctx context.Context, l *slog.Logger, mgr ossvc.Manager) (err error) {
	status, err := mgr.Status(ctx, serviceName)
	if err != nil {
		// Don't wrap the error, since it's informative enough as is.
		return err
	}

	switch status {
	case ossvc.StatusNotInstalled:
		l.InfoContext(ctx, "not installed")
	case ossvc.StatusStopped:
		l.InfoContext(ctx, "stopped")
	case ossvc.StatusRunning:
		l.InfoContext(ctx, "running")
	case ossvc.StatusRestartOnFail:
		l.InfoContext(ctx, "restarting after failed start")
	}

	return nil
}

// handleServiceReloadCmd reloads the service, if it's running.  l must not be
// nil, mgr must be a ReloadManager.
func handleServiceReloadCmd(ctx context.Context, l *slog.Logger, mgr ossvc.Manager) (err error) {
	relSvcMgr, ok := mgr.(ossvc.ReloadManager)
	if !ok {
		return fmt.Errorf("service manager can't reload: %w", errors.ErrUnsupported)
	}

	err = relSvcMgr.Reload(ctx, serviceName)
	if err != nil {
		return fmt.Errorf("reloading service: %w", err)
	}

	l.InfoContext(ctx, "service reloaded successfully")

	return nil
}

// handleServiceInstallCmd handles the service "install" command.  l must
// not be nil.
func handleServiceInstallCmd(
	ctx context.Context,
	l *slog.Logger,
	mgr ossvc.Manager,
	opts options,
	workDir string,
	confPath string,
) (err error) {
	pwd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("getting current directory: %w", err)
	}

	runOpts := opts
	runOpts.serviceControlAction = "run"

	args := optsToArgs(runOpts)
	l.DebugContext(ctx, "using", "args", args)

	err = mgr.Perform(ctx, &ossvc.ActionInstall{
		ServiceName:      serviceName,
		DisplayName:      serviceDisplayName,
		Description:      serviceDescription,
		WorkingDirectory: pwd,
		Version:          version.Full(),
		Arguments:        args,
	})
	if err != nil {
		return fmt.Errorf("installing service: %w", err)
	}

	err = mgr.Perform(ctx, &ossvc.ActionStart{
		ServiceName: serviceName,
	})
	if err != nil {
		return fmt.Errorf("starting service: %w", err)
	}

	if detectFirstRun(ctx, l, workDir, confPath) {
		slogutil.PrintLines(ctx, l, slog.LevelInfo, "", "Almost ready!\n"+
			"AdGuard Home is successfully installed and will automatically start on boot.\n"+
			"There are a few more things that must be configured before you can use it.\n"+
			"Click on the link below and follow the Installation Wizard steps to finish setup.\n"+
			"AdGuard Home is now available at the following addresses:")
		printHTTPAddresses(urlutil.SchemeHTTP, nil)
	}

	return nil
}

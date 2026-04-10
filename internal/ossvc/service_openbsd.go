//go:build openbsd

package ossvc

import (
	"bytes"
	"cmp"
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"text/template"

	"github.com/AdguardTeam/AdGuardHome/internal/aghos"
	"github.com/AdguardTeam/golibs/errors"
	"github.com/AdguardTeam/golibs/ioutil"
	"github.com/AdguardTeam/golibs/osutil/executil"
	"github.com/kardianos/service"
)

// OpenBSD Service Implementation
//
// The file contains OpenBSD implementations for service.System and
// service.Service interfaces.  It uses the default approach for RunCom-based
// services systems, e.g. rc.d script.  It's written as if it was in a separate
// package and has only one internal dependency.
//
// TODO(e.burkov):  Perhaps, file a PR to github.com/kardianos/service.

// sysVersion is the version of local service.System interface implementation.
const sysVersion = "openbsd-runcom"

// chooseSystem checks the current system detected and substitutes it with local
// implementation if needed.  cmdCons and l must not be nil.
func chooseSystem(_ context.Context, l *slog.Logger, cmdCons executil.CommandConstructor) {
	service.ChooseSystem(&openbsdSystem{
		cmdCons: cmdCons,
		logger:  l,
	})
}

// openbsdSystem is an implementation of the [service.System] interface to be
// used on the OpenBSD operating system.
type openbsdSystem struct {
	cmdCons executil.CommandConstructor
	logger  *slog.Logger
}

// type check
var _ service.System = (*openbsdSystem)(nil)

// String implements the [service.System] interface for *openbsdSystem.
func (sys *openbsdSystem) String() (s string) {
	return sysVersion
}

// Detect implements the [service.System] interface for *openbsdSystem.
func (sys *openbsdSystem) Detect() (ok bool) {
	return true
}

// Interactive implements the [service.System] interface for *openbsdSystem.
func (sys *openbsdSystem) Interactive() (ok bool) {
	return os.Getppid() != 1
}

// New implements the [service.System] interface for *openbsdSystem.
func (sys *openbsdSystem) New(
	i service.Interface,
	c *service.Config,
) (s service.Service, err error) {
	return &openbsdRunComService{
		cmdCons: sys.cmdCons,
		logger:  sys.logger,
		i:       i,
		cfg:     c,
	}, nil
}

// openbsdRunComService is the RunCom-based [service.Service] interface
// implementation to be used on the OpenBSD operating system.
type openbsdRunComService struct {
	cmdCons executil.CommandConstructor
	i       service.Interface
	cfg     *service.Config
	logger  *slog.Logger
}

// type check
var _ service.Service = (*openbsdRunComService)(nil)

// Platform implements the [service.Service] interface for *openbsdRunComService.
func (*openbsdRunComService) Platform() (p string) {
	return "openbsd"
}

// String implements the [service.Service] interface for *openbsdRunComService.
func (s *openbsdRunComService) String() (str string) {
	return cmp.Or(s.cfg.DisplayName, s.cfg.Name)
}

// stringFromKV returns the value of the given name from kv, assuming the value
// is a string.  If the value isn't found or is not of the type, the
// defaultValue is returned.
func stringFromKV(kv service.KeyValue, name, defaultValue string) (val string) {
	var ok bool
	if val, ok = kv[name].(string); ok {
		return val
	}

	return defaultValue
}

const (
	// optionUserService is the UserService option name.
	optionUserService = "UserService"

	// optionSvcInfo is the name of the option associated with service info.
	optionSvcInfo = "SvcInfo"

	// errNoUserServiceRunCom is returned when the service uses some custom
	// path to script.
	errNoUserServiceRunCom errors.Error = "user services are not supported on " + sysVersion
)

// scriptPath returns the absolute path to the script.  It's commonly used to
// send commands to the service.
func (s *openbsdRunComService) scriptPath() (cp string, err error) {
	if usesCustomPath, ok := s.cfg.Option[optionUserService].(bool); ok && usesCustomPath {
		return "", errNoUserServiceRunCom
	}

	const scriptPathPref = "/etc/rc.d"

	return filepath.Join(scriptPathPref, s.cfg.Name), nil
}

const (
	// optionRunComScript is the RunCom script option name.
	optionRunComScript = "RunComScript"

	// runComScript is the default RunCom script.
	runComScript = `#!/bin/sh
#
# $OpenBSD: {{ .SvcInfo }}

daemon="{{.Path}}"
daemon_flags={{ .Arguments | args }}

. /etc/rc.d/rc.subr

rc_bg=YES

rc_cmd $1
`
)

// template returns the script template to put into rc.d.
func (s *openbsdRunComService) template() (t *template.Template) {
	tf := map[string]any{
		"args": func(sl []string) string {
			return `"` + strings.Join(sl, " ") + `"`
		},
	}

	script := stringFromKV(s.cfg.Option, optionRunComScript, runComScript)

	return template.Must(template.New("").Funcs(tf).Parse(script))
}

// execPath returns the absolute path to the executable to be run as a service.
func (s *openbsdRunComService) execPath() (path string, err error) {
	if c := s.cfg; c != nil && len(c.Executable) != 0 {
		return filepath.Abs(c.Executable)
	}

	if path, err = os.Executable(); err != nil {
		return "", err
	}

	return filepath.Abs(path)
}

// annotate wraps errors.Annotate applying a common error format.
func (s *openbsdRunComService) annotate(action string, err error) (annotated error) {
	return errors.Annotate(err, "%s %s %s service: %w", action, sysVersion, s.cfg.Name)
}

// Install implements the [service.Service] interface for *openbsdRunComService.
func (s *openbsdRunComService) Install() (err error) {
	defer func() { err = s.annotate("installing", err) }()

	if err = s.writeScript(); err != nil {
		return err
	}

	return s.configureSysStartup(true)
}

// configureSysStartup adds s into the group of packages started with system.
func (s *openbsdRunComService) configureSysStartup(enable bool) (err error) {
	cmd := "enable"
	if !enable {
		cmd = "disable"
	}

	// TODO(s.chzhen):  Pass context.
	return executil.RunWithPeek(
		context.TODO(),
		s.cmdCons,
		aghos.MaxCmdOutputSize,
		"rcctl",
		cmd,
		s.cfg.Name,
	)
}

// writeScript tries to write the script for the service.
func (s *openbsdRunComService) writeScript() (err error) {
	var scriptPath string
	if scriptPath, err = s.scriptPath(); err != nil {
		return err
	}

	if _, err = os.Stat(scriptPath); !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("script already exists at %s", scriptPath)
	}

	var execPath string
	if execPath, err = s.execPath(); err != nil {
		return err
	}

	t := s.template()
	f, err := os.Create(scriptPath)
	if err != nil {
		return fmt.Errorf("creating rc.d script file: %w", err)
	}
	defer f.Close()

	err = t.Execute(f, &struct {
		*service.Config
		Path    string
		SvcInfo string
	}{
		Config:  s.cfg,
		Path:    execPath,
		SvcInfo: stringFromKV(s.cfg.Option, optionSvcInfo, s.String()),
	})
	if err != nil {
		return err
	}

	return errors.Annotate(
		os.Chmod(scriptPath, 0o755),
		"changing rc.d script file permissions: %w",
	)
}

// Uninstall implements the [service.Service] interface for *openbsdRunComService.
func (s *openbsdRunComService) Uninstall() (err error) {
	defer func() { err = s.annotate("uninstalling", err) }()

	if err = s.configureSysStartup(false); err != nil {
		return err
	}

	var scriptPath string
	if scriptPath, err = s.scriptPath(); err != nil {
		return err
	}

	if err = os.Remove(scriptPath); errors.Is(err, os.ErrNotExist) {
		return service.ErrNotInstalled
	}

	return errors.Annotate(err, "removing rc.d script: %w")
}

// runWait is the default function to wait for service to be stopped.
func runWait() {
	sigChan := make(chan os.Signal, 3)

	// TODO(m.kazantsev):  Replace with osutil interface.
	signal.Notify(sigChan, syscall.SIGTERM, os.Interrupt)
	<-sigChan
}

// Run implements the [service.Service] interface for *openbsdRunComService.
func (s *openbsdRunComService) Run() (err error) {
	if err = s.i.Start(s); err != nil {
		return err
	}

	runWait()

	return s.i.Stop(s)
}

// runCom calls the script with the specified cmd.
func (s *openbsdRunComService) runCom(cmd string) (out string, err error) {
	var scriptPath string
	if scriptPath, err = s.scriptPath(); err != nil {
		// Don't wrap the error because it is informative as is.
		return "", err
	}

	stdoutBuf := bytes.Buffer{}
	stderrBuf := bytes.Buffer{}

	// TODO(s.chzhen):  Pass context.
	err = executil.Run(context.TODO(), s.cmdCons, &executil.CommandConfig{
		Stderr: &stderrBuf,
		Stdout: ioutil.NewTruncatedWriter(&stdoutBuf, aghos.MaxCmdOutputSize),
		Path:   scriptPath,
		Args:   []string{cmd},
	})

	switch {
	case errors.Is(err, os.ErrNotExist):
		return "", service.ErrNotInstalled
	case err != nil:
		return stderrBuf.String(), err
	}

	return stdoutBuf.String(), nil
}

// Status implements the [service.Service] interface for *openbsdRunComService.
func (s *openbsdRunComService) Status() (status service.Status, err error) {
	defer func() { err = s.annotate("getting status of", err) }()

	var out string
	if out, err = s.runCom("check"); err != nil {
		return service.StatusUnknown, err
	}

	name := s.cfg.Name
	switch out {
	case fmt.Sprintf("%s(ok)\n", name):
		return service.StatusRunning, nil
	case fmt.Sprintf("%s(failed)\n", name):
		return service.StatusStopped, nil
	default:
		return service.StatusUnknown, service.ErrNotInstalled
	}
}

// Start implements the [service.Service] interface for *openbsdRunComService.
func (s *openbsdRunComService) Start() (err error) {
	_, err = s.runCom("start")

	return s.annotate("starting", err)
}

// Stop implements the [service.Service] interface for *openbsdRunComService.
func (s *openbsdRunComService) Stop() (err error) {
	_, err = s.runCom("stop")

	return s.annotate("stopping", err)
}

// Restart implements the [service.Service] interface for *openbsdRunComService.
func (s *openbsdRunComService) Restart() (err error) {
	if err = s.Stop(); err != nil {
		// Don't wrap the error because it is informative as is.
		return err
	}

	return s.Start()
}

// Logger implements the [service.Service] interface for *openbsdRunComService.
func (s *openbsdRunComService) Logger(errs chan<- error) (l service.Logger, err error) {
	if service.ChosenSystem().Interactive() {
		return service.ConsoleLogger, nil
	}

	return s.SystemLogger(errs)
}

// SystemLogger implements the [service.Service] interface for *openbsdRunComService.
func (s *openbsdRunComService) SystemLogger(errs chan<- error) (l service.Logger, err error) {
	return newSysLogger(s.logger, s.cfg.Name, errs)
}

// newSysLogger returns a the [service.Logger] interface implementation.
func newSysLogger(l *slog.Logger, _ string, _ chan<- error) (sl service.Logger, err error) {
	return &sysLogger{l: l}, nil
}

// sysLogger is an implementation of [service.Logger] interface that wraps calls
// of the logging functions in a way that is understandable for service
// interfaces.
type sysLogger struct {
	l *slog.Logger
}

// type check
var _ service.Logger = (*sysLogger)(nil)

// Error implements the [service.Logger] interface for sysLogger.
func (s *sysLogger) Error(v ...any) (err error) {
	s.l.Error(fmt.Sprint(v...))

	return nil
}

// Warning implements the [service.Logger] interface for sysLogger.
func (s *sysLogger) Warning(v ...any) (err error) {
	s.l.Warn(fmt.Sprint(v...))

	return nil
}

// Info implements the [service.Logger] interface for sysLogger.
func (s *sysLogger) Info(v ...any) (err error) {
	s.l.Info(fmt.Sprint(v...))

	return nil
}

// Errorf implements the [service.Logger] interface for sysLogger.
func (s *sysLogger) Errorf(format string, a ...any) (err error) {
	s.l.Error(fmt.Sprintf(format, a...))

	return nil
}

// Warningf implements the [service.Logger] interface for sysLogger.
func (s *sysLogger) Warningf(format string, a ...any) (err error) {
	s.l.Warn(fmt.Sprintf(format, a...))

	return nil
}

// Infof implements the [service.Logger] interface for sysLogger.
func (s *sysLogger) Infof(format string, a ...any) (err error) {
	s.l.Info(fmt.Sprintf(format, a...))

	return nil
}

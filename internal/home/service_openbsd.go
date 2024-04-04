//go:build openbsd

package home

import (
	"cmp"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"text/template"

	"github.com/AdguardTeam/AdGuardHome/internal/aghos"
	"github.com/AdguardTeam/golibs/errors"
	"github.com/AdguardTeam/golibs/log"
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
// implementation if needed.
func chooseSystem() {
	service.ChooseSystem(openbsdSystem{})
}

// openbsdSystem is the service.System to be used on the OpenBSD.
type openbsdSystem struct{}

// String implements service.System interface for openbsdSystem.
func (openbsdSystem) String() string {
	return sysVersion
}

// Detect implements service.System interface for openbsdSystem.
func (openbsdSystem) Detect() (ok bool) {
	return true
}

// Interactive implements service.System interface for openbsdSystem.
func (openbsdSystem) Interactive() (ok bool) {
	return os.Getppid() != 1
}

// New implements service.System interface for openbsdSystem.
func (openbsdSystem) New(i service.Interface, c *service.Config) (s service.Service, err error) {
	return &openbsdRunComService{
		i:   i,
		cfg: c,
	}, nil
}

// openbsdRunComService is the RunCom-based service.Service to be used on the
// OpenBSD.
type openbsdRunComService struct {
	i   service.Interface
	cfg *service.Config
}

// Platform implements service.Service interface for *openbsdRunComService.
func (*openbsdRunComService) Platform() (p string) {
	return "openbsd"
}

// String implements service.Service interface for *openbsdRunComService.
func (s *openbsdRunComService) String() string {
	return cmp.Or(s.cfg.DisplayName, s.cfg.Name)
}

// getBool returns the value of the given name from kv, assuming the value is a
// boolean.  If the value isn't found or is not of the type, the defaultValue is
// returned.
func getBool(kv service.KeyValue, name string, defaultValue bool) (val bool) {
	var ok bool
	if val, ok = kv[name].(bool); ok {
		return val
	}

	return defaultValue
}

// getString returns the value of the given name from kv, assuming the value is
// a string.  If the value isn't found or is not of the type, the defaultValue
// is returned.
func getString(kv service.KeyValue, name, defaultValue string) (val string) {
	var ok bool
	if val, ok = kv[name].(string); ok {
		return val
	}

	return defaultValue
}

// getFuncNiladic returns the value of the given name from kv, assuming the
// value is a func().  If the value isn't found or is not of the type, the
// defaultValue is returned.
func getFuncNiladic(kv service.KeyValue, name string, defaultValue func()) (val func()) {
	var ok bool
	if val, ok = kv[name].(func()); ok {
		return val
	}

	return defaultValue
}

const (
	// optionUserService is the UserService option name.
	optionUserService = "UserService"

	// optionUserServiceDefault is the UserService option default value.
	optionUserServiceDefault = false

	// errNoUserServiceRunCom is returned when the service uses some custom
	// path to script.
	errNoUserServiceRunCom errors.Error = "user services are not supported on " + sysVersion
)

// scriptPath returns the absolute path to the script.  It's commonly used to
// send commands to the service.
func (s *openbsdRunComService) scriptPath() (cp string, err error) {
	if getBool(s.cfg.Option, optionUserService, optionUserServiceDefault) {
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

	return template.Must(template.New("").Funcs(tf).Parse(getString(
		s.cfg.Option,
		optionRunComScript,
		runComScript,
	)))
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

// Install implements service.Service interface for *openbsdRunComService.
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

	var code int
	code, _, err = aghos.RunCommand("rcctl", cmd, s.cfg.Name)
	if err != nil {
		return err
	} else if code != 0 {
		return fmt.Errorf("rcctl finished with code %d", code)
	}

	return nil
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
		SvcInfo: getString(s.cfg.Option, "SvcInfo", s.String()),
	})
	if err != nil {
		return err
	}

	return errors.Annotate(
		os.Chmod(scriptPath, 0o755),
		"changing rc.d script file permissions: %w",
	)
}

// Uninstall implements service.Service interface for *openbsdRunComService.
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

// optionRunWait is the name of the option associated with function which waits
// for the service to be stopped.
const optionRunWait = "RunWait"

// runWait is the default function to wait for service to be stopped.
func runWait() {
	sigChan := make(chan os.Signal, 3)
	signal.Notify(sigChan, syscall.SIGTERM, os.Interrupt)
	<-sigChan
}

// Run implements service.Service interface for *openbsdRunComService.
func (s *openbsdRunComService) Run() (err error) {
	if err = s.i.Start(s); err != nil {
		return err
	}

	getFuncNiladic(s.cfg.Option, optionRunWait, runWait)()

	return s.i.Stop(s)
}

// runCom calls the script with the specified cmd.
func (s *openbsdRunComService) runCom(cmd string) (out string, err error) {
	var scriptPath string
	if scriptPath, err = s.scriptPath(); err != nil {
		return "", err
	}

	// TODO(e.burkov):  It's possible that os.ErrNotExist is caused by
	// something different than the service script's non-existence.  Keep it
	// in mind, when replace the aghos.RunCommand.
	var outData []byte
	_, outData, err = aghos.RunCommand(scriptPath, cmd)
	if errors.Is(err, os.ErrNotExist) {
		return "", service.ErrNotInstalled
	}

	return string(outData), err
}

// Status implements service.Service interface for *openbsdRunComService.
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

// Start implements service.Service interface for *openbsdRunComService.
func (s *openbsdRunComService) Start() (err error) {
	_, err = s.runCom("start")

	return s.annotate("starting", err)
}

// Stop implements service.Service interface for *openbsdRunComService.
func (s *openbsdRunComService) Stop() (err error) {
	_, err = s.runCom("stop")

	return s.annotate("stopping", err)
}

// Restart implements service.Service interface for *openbsdRunComService.
func (s *openbsdRunComService) Restart() (err error) {
	if err = s.Stop(); err != nil {
		return err
	}

	return s.Start()
}

// Logger implements service.Service interface for *openbsdRunComService.
func (s *openbsdRunComService) Logger(errs chan<- error) (l service.Logger, err error) {
	if service.ChosenSystem().Interactive() {
		return service.ConsoleLogger, nil
	}

	return s.SystemLogger(errs)
}

// SystemLogger implements service.Service interface for *openbsdRunComService.
func (s *openbsdRunComService) SystemLogger(errs chan<- error) (l service.Logger, err error) {
	return newSysLogger(s.cfg.Name, errs)
}

// newSysLogger returns a stub service.Logger implementation.
func newSysLogger(_ string, _ chan<- error) (service.Logger, error) {
	return sysLogger{}, nil
}

// sysLogger wraps calls of the logging functions understandable for service
// interfaces.
type sysLogger struct{}

// Error implements service.Logger interface for sysLogger.
func (sysLogger) Error(v ...any) error {
	log.Error(fmt.Sprint(v...))

	return nil
}

// Warning implements service.Logger interface for sysLogger.
func (sysLogger) Warning(v ...any) error {
	log.Info("warning: %s", fmt.Sprint(v...))

	return nil
}

// Info implements service.Logger interface for sysLogger.
func (sysLogger) Info(v ...any) error {
	log.Info(fmt.Sprint(v...))

	return nil
}

// Errorf implements service.Logger interface for sysLogger.
func (sysLogger) Errorf(format string, a ...any) error {
	log.Error(format, a...)

	return nil
}

// Warningf implements service.Logger interface for sysLogger.
func (sysLogger) Warningf(format string, a ...any) error {
	log.Info("warning: %s", fmt.Sprintf(format, a...))

	return nil
}

// Infof implements service.Logger interface for sysLogger.
func (sysLogger) Infof(format string, a ...any) error {
	log.Info(format, a...)

	return nil
}

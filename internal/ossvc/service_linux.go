//go:build linux

package ossvc

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"io"
	"strings"

	"github.com/AdguardTeam/AdGuardHome/internal/aghos"
	"github.com/AdguardTeam/golibs/osutil/executil"
	"github.com/kardianos/service"
)

// chooseSystem checks the current system detected and substitutes it with local
// implementation if needed.
func chooseSystem() {
	sys := service.ChosenSystem()
	switch sys.String() {
	case "unix-systemv":
		// By default, package service uses the SysV system if it cannot detect
		// anything other, but the update-rc.d fix should not be applied on
		// OpenWrt, so exclude it explicitly.
		//
		// See https://github.com/AdguardTeam/AdGuardHome/issues/4480 and
		// https://github.com/AdguardTeam/AdGuardHome/issues/4677.
		if !aghos.IsOpenWrt() {
			service.ChooseSystem(&sysvSystem{System: sys})
		}
	case "linux-systemd":
		service.ChooseSystem(&systemdSystem{System: sys})
	default:
		// Do nothing.
	}
}

// sysvSystem is a wrapper for a [service.System] that returns the custom
// implementation of the [service.Service] interface.
//
// TODO(e.burkov):  File a PR to github.com/kardianos/service.
type sysvSystem struct {
	// System must have an unexported type *service.linuxSystemService.
	service.System
}

// type check
var _ service.System = (*sysvSystem)(nil)

// New implements the [service.System] interface for *sysvSystem.  i and c must
// not be nil.
func (sys *sysvSystem) New(i service.Interface, c *service.Config) (s service.Service, err error) {
	s, err = sys.System.New(i, c)
	if err != nil {
		// Don't wrap the error to keep it as close to the original one as
		// possible.
		return s, err
	}

	return &sysvService{
		cmdCons: executil.SystemCommandConstructor{},
		Service: s,
		name:    c.Name,
	}, nil
}

// sysvService is a wrapper for a SysV [service.Service] that supplements the
// installation and uninstallation.
type sysvService struct {
	// cmdCons is used to run external commands.  It must not be nil.
	cmdCons executil.CommandConstructor

	// Service must have an unexported type *service.sysv.
	service.Service

	// name stores the name of the service to call updating script with it.
	name string
}

// type check
var _ service.Service = (*sysvService)(nil)

// Install implements the [service.Service] interface for *sysvService.
func (svc *sysvService) Install() (err error) {
	err = svc.Service.Install()
	if err != nil {
		// Don't wrap the error to keep it as close to the original one as
		// possible.
		return err
	}

	// TODO(s.chzhen):  Pass context.
	_, _, err = aghos.RunCommand(context.TODO(), svc.cmdCons, "update-rc.d", svc.name, "defaults")

	// Don't wrap an error since it's informative enough as is.
	return err
}

// Uninstall implements the [service.Service] interface for *sysvService.
func (svc *sysvService) Uninstall() (err error) {
	err = svc.Service.Uninstall()
	if err != nil {
		// Don't wrap the error to keep it as close to the original one as
		// possible.
		return err
	}

	// TODO(s.chzhen):  Pass context.
	_, _, err = aghos.RunCommand(context.TODO(), svc.cmdCons, "update-rc.d", svc.name, "remove")

	// Don't wrap an error since it's informative enough as is.
	return err
}

// systemdSystem is a wrapper for a [service.System] that returns the custom
// implementation of the [service.Service] interface.
type systemdSystem struct {
	// System must have an unexported type *service.linuxSystemService.
	service.System
}

// type check
var _ service.System = (*systemdSystem)(nil)

// New implements the [service.System] interface for *systemdSystem.  i and c
// must not be nil.
func (sys *systemdSystem) New(i service.Interface, c *service.Config) (s service.Service, err error) {
	s, err = sys.System.New(i, c)
	if err != nil {
		// Don't wrap the error to keep it as close to the original one as
		// possible.
		return s, err
	}

	return &systemdService{
		cmdCons:  executil.SystemCommandConstructor{},
		Service:  s,
		unitName: fmt.Sprintf("%s.service", c.Name),
	}, nil
}

// type check
var _ service.Service = (*systemdService)(nil)

// systemdService is a wrapper for a systemd [service.Service] that enriches the
// service status information.
type systemdService struct {
	// cmdCons is used to run external commands.  It must not be nil.
	cmdCons executil.CommandConstructor

	// Service is expected to have an unexported type *service.systemd.
	service.Service

	// unitName stores the name of the systemd daemon.
	unitName string
}

// type check
var _ service.Service = (*systemdService)(nil)

// Status implements the [service.Service] interface for *systemdService.
func (s *systemdService) Status() (status service.Status, err error) {
	const systemctlCmd = "systemctl"

	var (
		systemctlArgs   = []string{"show", s.unitName}
		systemctlStdout bytes.Buffer
	)

	// TODO(s.chzhen):  Consider streaming the output if needed.  Using
	// [io.Pipe] here is unnecessary; it complicates lifecycle management
	// because the output must be read concurrently, and the PipeWriter must be
	// explicitly closed to signal EOF.  Since this command's output is small, a
	// bytes.Buffer via executil.Run is sufficient.
	err = executil.Run(
		// TODO(s.chzhen):  Pass context.
		context.TODO(),
		s.cmdCons,
		&executil.CommandConfig{
			Path:   systemctlCmd,
			Args:   systemctlArgs,
			Stdout: &systemctlStdout,
		},
	)
	if err != nil {
		return service.StatusUnknown, fmt.Errorf("executing command: %w", err)
	}

	status, err = parseSystemctlShow(&systemctlStdout)
	if err != nil {
		return service.StatusUnknown, fmt.Errorf("parsing command output: %w", err)
	}

	return status, nil
}

// Searched property names.  See man systemctl(1).
const (
	propNameLoadState   = "LoadState"
	propNameActiveState = "ActiveState"
	propNameSubState    = "SubState"
)

// parseSystemctlShow parses the output of the systemctl show command.  It
// expects the key=value pairs separated by newlines.
func parseSystemctlShow(output io.Reader) (status service.Status, err error) {
	var loadState, activeState, subState string

	scanner := bufio.NewScanner(output)
	for scanner.Scan() {
		line := scanner.Text()

		propName, propValue, ok := strings.Cut(line, "=")
		if !ok {
			return service.StatusUnknown, fmt.Errorf("unexpected line format: %q", line)
		}

		switch propName {
		case propNameLoadState:
			loadState = propValue
		case propNameActiveState:
			activeState = propValue
		case propNameSubState:
			subState = propValue
		default:
			// Go on.
		}
	}
	if err = scanner.Err(); err != nil {
		return service.StatusUnknown, err
	}

	return statusFromState(loadState, activeState, subState)
}

// statusFromState returns the service status based on the systemctl state
// property values.
func statusFromState(loadState, activeState, subState string) (status service.Status, err error) {
	// Desired property values.  See man systemctl(1).
	const (
		propValueLoadStateNotFound   = "not-found"
		propValueActiveStateActive   = "active"
		propValueActiveStateInactive = "inactive"
		propValueSubStateAutoRestart = "auto-restart"
	)

	switch {
	case loadState == propValueLoadStateNotFound:
		return service.StatusUnknown, service.ErrNotInstalled
	case activeState == propValueActiveStateActive:
		return service.StatusRunning, nil
	case activeState == propValueActiveStateInactive:
		return service.StatusStopped, nil
	case subState == propValueSubStateAutoRestart:
		return statusRestartOnFail, nil
	default:
		return service.StatusUnknown, fmt.Errorf(
			"unexpected state: %s=%q, %s=%q, %s=%q",
			propNameLoadState, loadState,
			propNameActiveState, activeState,
			propNameSubState, subState,
		)
	}
}

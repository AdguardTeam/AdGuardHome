//go:build darwin

package ossvc

import (
	"bufio"
	"bytes"
	"context"
	"io"
	"log/slog"
	"os"
	"path"
	"strings"

	"github.com/AdguardTeam/golibs/errors"
	"github.com/AdguardTeam/golibs/logutil/slogutil"
	"github.com/AdguardTeam/golibs/osutil/executil"
	"github.com/kardianos/service"
)

// chooseSystem replaces the currently selected system with a wrapper.  l and
// cmdCons must not be nil.
func chooseSystem(_ context.Context, l *slog.Logger, cmdCons executil.CommandConstructor) {
	sys := service.ChosenSystem()
	service.ChooseSystem(&darwinSystem{
		System:  sys,
		cmdCons: cmdCons,
		logger:  l,
	})
}

// darwinSystem is the wrapper for [service.System] that returns the custom
// implementation of the [service.Service] interface.
type darwinSystem struct {
	service.System
	cmdCons executil.CommandConstructor
	logger  *slog.Logger
}

// type check
var _ service.System = (*darwinSystem)(nil)

// New implements the [service.System] interface for *darwinSystem.  i and c
// must not be nil.
func (d *darwinSystem) New(i service.Interface, c *service.Config) (s service.Service, err error) {
	s, err = d.System.New(i, c)
	if err != nil {
		// Don't wrap the error to keep it as close to the original one as
		// possible.
		return s, err
	}

	return newDarwinService(&darwinServiceConfig{
		svc:      s,
		logger:   d.logger,
		cmdCons:  d.cmdCons,
		name:     c.Name,
		plistDir: "/Library/LaunchDaemons",
	}), nil
}

// darwinServiceConfig is the configuration structure for [*darwinService].
type darwinServiceConfig struct {
	// svc is the base service to extend.  It must not be nil.
	svc service.Service

	// logger is used for logging service operations.  It must not be nil.
	logger *slog.Logger

	// cmdCons is used to create system commands.  It must not be nil.
	cmdCons executil.CommandConstructor

	// name is the launchd service name.
	name string

	// plistDir is the path to the directory that contains launchd plist files.
	plistDir string
}

// darwinService is a wrapper for a darwin [service.Service] that enriches the
// service status information.
type darwinService struct {
	service.Service
	logger   *slog.Logger
	cmdCons  executil.CommandConstructor
	name     string
	plistDir string
}

// newDarwinService returns properly initialized *darwinService.  c must be
// non-nil and valid.
func newDarwinService(c *darwinServiceConfig) (d *darwinService) {
	return &darwinService{
		Service:  c.svc,
		logger:   c.logger,
		cmdCons:  c.cmdCons,
		name:     c.name,
		plistDir: c.plistDir,
	}
}

// type check
var _ service.Service = (*darwinService)(nil)

// Status implements the [service.Service] interface for *darwinService.
func (d *darwinService) Status() (status service.Status, err error) {
	// TODO(f.setrakov): Pass context.
	ctx := context.TODO()

	if !d.isInstalled(ctx) {
		return service.StatusUnknown, service.ErrNotInstalled
	}

	const launchctlCmd = "launchctl"
	var (
		launchctlArgs   = []string{"list", d.name}
		launchctlStdout bytes.Buffer
	)

	err = executil.Run(ctx, d.cmdCons, &executil.CommandConfig{
		Path:   launchctlCmd,
		Args:   launchctlArgs,
		Stdout: &launchctlStdout,
	})
	if err != nil {
		return service.StatusStopped, nil
	}

	return parseLaunchctlList(&launchctlStdout)
}

// isInstalled returns true if there is actual service .plist file.
func (d *darwinService) isInstalled(ctx context.Context) (ok bool) {
	plistPath := path.Join(d.plistDir, d.name+".plist")
	_, err := os.Stat(plistPath)
	if err != nil {
		if !errors.Is(err, os.ErrNotExist) {
			d.logger.WarnContext(ctx, "checking plist file", slogutil.KeyError, err)
		}

		return false
	}

	return true
}

// propNamePID represents the PID prop name in the launchctl list output.
const propNamePID = `"PID"`

// parseLaunchctlList parses the output of the launchctl list command.  It
// expects that output contains default launchctl tree-like output with
// prop=value pairs.
func parseLaunchctlList(output io.Reader) (status service.Status, err error) {
	scanner := bufio.NewScanner(output)
	for scanner.Scan() {
		line := scanner.Text()
		propName, propValue, ok := strings.Cut(line, "=")
		if !ok {
			continue
		}

		propName = strings.TrimSpace(propName)
		propValue = strings.TrimSpace(propValue)

		if propName == propNamePID && propValue != "" {
			return service.StatusRunning, nil
		}
	}

	if err = scanner.Err(); err != nil {
		return service.StatusUnknown, err
	}

	return statusRestartOnFail, nil
}

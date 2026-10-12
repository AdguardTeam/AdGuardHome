package ossvc

import (
	"context"
	"log/slog"
	"os"

	"github.com/AdguardTeam/golibs/errors"
	"github.com/AdguardTeam/golibs/osutil/executil"
	"github.com/AdguardTeam/golibs/timeutil"
	"github.com/AdguardTeam/golibs/validate"
)

// Manager is the interface for communication with the OS service manager.
//
// TODO(e.burkov):  Move to golibs.
type Manager interface {
	// Perform performs the specified action.
	Perform(ctx context.Context, action Action) (err error)

	// Status returns the status of the service with the given name.
	Status(ctx context.Context, name ServiceName) (status Status, err error)
}

// ManagerConfig contains the configuration for [Manager].
type ManagerConfig struct {
	// Clock is used to get the current time.  It must not be nil.
	Clock timeutil.Clock

	// Logger is the logger to use.  It must not be nil.
	Logger *slog.Logger

	// CommandConstructor is the constructor to use for creating commands.  It
	// must not be nil.
	CommandConstructor executil.CommandConstructor

	// RootDir is the root directory of the current OS.  It's used to access the
	// configuration files of the services.  It must not be nil.
	RootDir *os.Root
}

// type check
var _ validate.Interface = (*ManagerConfig)(nil)

// Validate implements the [validate.Interface] interface for *ManagerConfig.
func (conf *ManagerConfig) Validate() (err error) {
	if conf == nil {
		return errors.ErrNoValue
	}

	return errors.Join(
		validate.NotNilInterface("Clock", conf.Clock),
		validate.NotNil("Logger", conf.Logger),
		validate.NotNilInterface("CommandConstructor", conf.CommandConstructor),
		validate.NotNil("RootDir", conf.RootDir),
	)
}

// NewManager returns a new properly initialized [Manager], appropriate for the
// current platform.  conf must be valid.
func NewManager(ctx context.Context, conf *ManagerConfig) (mgr Manager) {
	return newManager(ctx, conf)
}

// EmptyManager is an empty implementation of [Manager] that does nothing.
type EmptyManager struct{}

// type check
var _ Manager = EmptyManager{}

// Perform implements the [Manager] interface for EmptyManager.
func (EmptyManager) Perform(_ context.Context, _ Action) (err error) {
	return nil
}

// Status implements the [Manager] interface for EmptyManager.
func (EmptyManager) Status(_ context.Context, _ ServiceName) (status Status, err error) {
	return StatusNotInstalled, nil
}

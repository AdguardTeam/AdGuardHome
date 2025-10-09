package ossvc

import (
	"context"
	"log/slog"

	"github.com/AdguardTeam/golibs/osutil/executil"
)

// Manager is the interface for communication with the OS service manager.
//
// TODO(e.burkov):  Move to golibs.
//
// TODO(e.burkov):  Use.
type Manager interface {
	// Perform performs the specified action.
	Perform(ctx context.Context, action Action) (err error)

	// Status returns the status of the service with the given name.
	Status(ctx context.Context, name ServiceName) (status Status, err error)
}

// ManagerConfig contains the configuration for [Manager].
type ManagerConfig struct {
	// Logger is the logger to use.
	Logger *slog.Logger

	// CommandConstructor is the constructor to use for creating commands.
	CommandConstructor executil.CommandConstructor
}

// NewManager returns a new properly initialized [Manager], appropriate for the
// current platform.
func NewManager(ctx context.Context, conf *ManagerConfig) (mgr Manager, err error) {
	return newManager(ctx, conf), nil
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

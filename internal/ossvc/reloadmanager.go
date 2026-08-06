package ossvc

import "context"

// ReloadManager is the extension interface for [Manager] that provides an
// ability to reload a service.
type ReloadManager interface {
	Manager

	// Reload reloads the service with the given name.  As opposed to
	// [ActionRestart], this method does not stop the service.
	Reload(ctx context.Context, name ServiceName) (err error)
}

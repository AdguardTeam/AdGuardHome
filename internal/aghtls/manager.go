package aghtls

import (
	"context"

	"github.com/AdguardTeam/golibs/service"
)

// TLSPair is a pair of paths to a certificate and a key.
type TLSPair struct {
	// CertPath is the path to the certificate.  If empty, the certificate will
	// not be tracked.
	CertPath string

	// KeyPath is the path to the key.  If empty, the key will not be tracked.
	KeyPath string
}

// UpdateSignal is the signal that the TLS certificate and key have been
// updated.
type UpdateSignal struct{}

// Manager manages TLS certificates and keys updates.
type Manager interface {
	service.Interface
	service.Refresher

	// Set sets the TLS certificate and key.  certKey may have unset fields,
	// in which case the corresponding files will not be tracked.
	Set(ctx context.Context, certKey TLSPair) (err error)

	// Updates returns a channel that emits signals when the TLS certificate
	// and/or key have been updated.
	//
	// TODO(e.burkov):  Move reloading logic to the manager and get rid of this
	// method.
	Updates(ctx context.Context) (updates <-chan UpdateSignal)
}

// EmptyManager is an empty implementation of the [Manager] interface.
type EmptyManager struct{}

// type check
var _ Manager = (*EmptyManager)(nil)

// Start implements the [service.Interface] interface for EmptyManager.  It
// always returns nil.
func (EmptyManager) Start(_ context.Context) (err error) { return nil }

// Shutdown implements the [service.Interface] interface for EmptyManager.  It
// always returns nil.
func (EmptyManager) Shutdown(_ context.Context) (err error) { return nil }

// Refresh implements the [service.Refresher] interface for EmptyManager.  It
// always returns nil.
func (EmptyManager) Refresh(_ context.Context) (err error) { return nil }

// Set implements the [Manager] interface for EmptyManager.  It always returns
// nil.
func (EmptyManager) Set(_ context.Context, _ TLSPair) (err error) { return nil }

// Updates implements the [Manager] interface for EmptyManager.  It always
// returns a nil channel.
func (EmptyManager) Updates(_ context.Context) (updates <-chan UpdateSignal) { return nil }

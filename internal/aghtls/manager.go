package aghtls

import (
	"context"

	"github.com/AdguardTeam/golibs/service"
)

// TLSPair is a pair of paths to a certificate and a key.
type TLSPair struct {
	// CertPath is the path to the certificate.
	CertPath string

	// KeyPath is the path to the key.
	KeyPath string
}

// UpdateSignal is the signal that the TLS certificate and key have been
// updated.
type UpdateSignal struct{}

// Manager manages TLS certificates and keys updates.
type Manager interface {
	service.Interface
	service.Refresher

	// Set sets the TLS certificate and key.
	Set(ctx context.Context, certKey *TLSPair) (err error)

	// Updates returns a channel that emits signals when the TLS certificate
	// and/or key have been updated.
	//
	// TODO(e.burkov):  Move reloading logic to the manager and get rid of this
	// method.
	Updates(ctx context.Context) (updates <-chan UpdateSignal)
}

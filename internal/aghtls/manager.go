package aghtls

import (
	"context"
	"crypto/tls"
	"crypto/x509"

	"github.com/AdguardTeam/AdGuardHome/internal/aghalg"
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

	// CipherSuites returns the list of supported TLS cipher suites.
	//
	// TODO(m.kazantsev): Remove.
	CipherSuites() (cs []uint16)

	// Set sets the TLS certificate and key.  certKey may have unset fields,
	// in which case the corresponding files will not be tracked.
	Set(ctx context.Context, certKey TLSPair) (err error)

	// Updates returns a channel that emits signals when the TLS certificate
	// and/or key have been updated.
	//
	// TODO(e.burkov):  Move reloading logic to the manager and get rid of this
	// method.
	Updates(ctx context.Context) (updates <-chan UpdateSignal)

	// TLSConfig returns a clone of the current TLS configuration.  conf
	// provides its certificates via GetCertificate method.
	TLSConfig() (conf *tls.Config)

	// RootCAs returns the current root CA pool.
	RootCAs() (root *x509.CertPool)

	// HasIPAddrs returns true if the current TLS configuration has at least one
	// certificate with an IP address in its SAN extension.
	HasIPAddrs() (ok bool)

	// ExtendedTLSConfig returns a clone of the current extended TLS
	// configuration.
	ExtendedTLSConfig() (conf *ExtendedTLSConfig)

	// SetExtendedTLSConfig updates the current extended TLS configuration.  It
	// returns true if the configuration has changed.  Note that changing only
	// the plain-DNS setting also causes this method to return true.  Also, this
	// method only updates the manager's own state, persisting the plain-DNS
	// setting to the global configuration is the caller's responsibility.
	SetExtendedTLSConfig(
		ctx context.Context,
		servePlainDNS aghalg.NullBool,
		conf *ExtendedTLSConfig,
	) (changed bool, err error)
}

// EmptyManager is an empty implementation of the [Manager] interface.
type EmptyManager struct{}

// type check
var _ Manager = EmptyManager{}

// Start implements the [service.Interface] interface for EmptyManager.  It
// always returns nil.
func (EmptyManager) Start(_ context.Context) (err error) { return nil }

// Shutdown implements the [service.Interface] interface for EmptyManager.  It
// always returns nil.
func (EmptyManager) Shutdown(_ context.Context) (err error) { return nil }

// CipherSuites implements the [Manager] interface for EmptyManager.  It always
// returns nil.
func (EmptyManager) CipherSuites() (cs []uint16) { return nil }

// Refresh implements the [service.Refresher] interface for EmptyManager.  It
// always returns nil.
func (EmptyManager) Refresh(_ context.Context) (err error) { return nil }

// Set implements the [Manager] interface for EmptyManager.  It always returns
// nil.
func (EmptyManager) Set(_ context.Context, _ TLSPair) (err error) { return nil }

// Updates implements the [Manager] interface for EmptyManager.  It always
// returns a nil channel.
func (EmptyManager) Updates(_ context.Context) (updates <-chan UpdateSignal) { return nil }

// TLSConfig implements the [Manager] interface for EmptyManager.  It always
// returns nil.
func (EmptyManager) TLSConfig() (conf *tls.Config) {
	return nil
}

// RootCAs implements the [Manager] interface for EmptyManager.  It always
// returns nil.
func (EmptyManager) RootCAs() (root *x509.CertPool) {
	return nil
}

// HasIPAddrs implements the [Manager] interface for EmptyManager.  It always
// returns false.
func (EmptyManager) HasIPAddrs() (ok bool) {
	return false
}

// ExtendedTLSConfig implements the [Manager] interface for EmptyManager.  It
// always returns nil.
func (EmptyManager) ExtendedTLSConfig() (conf *ExtendedTLSConfig) {
	return nil
}

// SetExtendedTLSConfig implements the [Manager] interface for EmptyManager.  It
// always returns false and nil.
func (EmptyManager) SetExtendedTLSConfig(
	_ context.Context,
	_ aghalg.NullBool,
	_ *ExtendedTLSConfig,
) (restartHTTPS bool, err error) {
	return false, nil
}

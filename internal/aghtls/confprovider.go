package aghtls

import (
	"crypto/tls"
	"crypto/x509"
)

// TLSConfigProvider provides TLS configuration to consumers.  Implementations
// must be safe for concurrent use.
//
// TODO(m.kazantsev):  Merge with the Manager interface.
type TLSConfigProvider interface {
	// TLSConfig returns a clone of the current TLS configuration.  conf uses
	// GetConfigForClient for automatic updates.
	TLSConfig() (conf *tls.Config)

	// RootCAs returns the current root CA pool.
	RootCAs() (root *x509.CertPool)
}

// type check
var _ TLSConfigProvider = (*EmptyTLSConfigProvider)(nil)

// EmptyTLSConfigProvider is the implementation of the [TLSConfigProvider]
// interface that does nothing.
type EmptyTLSConfigProvider struct{}

// TLSConfig implements the [TLSConfigProvider] interface for
// *EmptyTLSConfigProvider.
func (t *EmptyTLSConfigProvider) TLSConfig() (conf *tls.Config) {
	return nil
}

// RootCAs implements the [TLSConfigProvider] interface for
// *EmptyTLSConfigProvider.
func (t *EmptyTLSConfigProvider) RootCAs() (root *x509.CertPool) {
	return nil
}

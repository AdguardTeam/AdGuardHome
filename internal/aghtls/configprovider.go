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
	// TLSConfig returns a clone of the current TLS configuration.  conf
	// provides its certificates via GetConfigForClient method.
	TLSConfig() (conf *tls.Config)

	// RootCAs returns the current root CA pool.
	RootCAs() (root *x509.CertPool)

	// HasIPAddrs returns true if the current TLS configuration has at least one
	// certificate with an IP address in its SAN extension.
	HasIPAddrs() (ok bool)
}

// type check
var _ TLSConfigProvider = EmptyTLSConfigProvider{}

// EmptyTLSConfigProvider is the implementation of the [TLSConfigProvider]
// interface that does nothing.
type EmptyTLSConfigProvider struct{}

// TLSConfig implements the [TLSConfigProvider] interface for
// EmptyTLSConfigProvider.  It always returns nil.
func (EmptyTLSConfigProvider) TLSConfig() (conf *tls.Config) {
	return nil
}

// RootCAs implements the [TLSConfigProvider] interface for
// EmptyTLSConfigProvider.  It always returns nil.
func (EmptyTLSConfigProvider) RootCAs() (root *x509.CertPool) {
	return nil
}

// HasIPAddrs implements the [TLSConfigProvider] interface for
// EmptyTLSConfigProvider.  It always returns false.
func (EmptyTLSConfigProvider) HasIPAddrs() (ok bool) {
	return false
}

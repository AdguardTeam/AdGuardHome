package aghtls

import (
	"crypto/x509"
)

// SystemRootCAs tries to load root certificates from the operating system.  It
// returns nil in case nothing is found so that Go' crypto/x509 can use its
// default algorithm to find system root CA list.
//
// See https://github.com/AdguardTeam/AdGuardHome/issues/1311.
func SystemRootCAs() (roots *x509.CertPool) {
	return rootCAs()
}

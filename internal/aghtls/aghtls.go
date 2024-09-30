// Package aghtls contains utilities for work with TLS.
package aghtls

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"slices"

	"github.com/AdguardTeam/golibs/log"
	"github.com/AdguardTeam/golibs/netutil"
)

// init makes sure that the cipher name map is filled.
//
// TODO(a.garipov): Propose a similar API to crypto/tls.
func init() {
	suites := tls.CipherSuites()
	cipherSuites = make(map[string]uint16, len(suites))
	for _, s := range suites {
		cipherSuites[s.Name] = s.ID
	}

	log.Debug("tls: known ciphers: %q", cipherSuites)
}

// cipherSuites are a name-to-ID mapping of cipher suites from crypto/tls.  It
// is filled by init.  It must not be modified.
var cipherSuites map[string]uint16

// ParseCiphers parses a slice of cipher suites from cipher names.
func ParseCiphers(cipherNames []string) (cipherIDs []uint16, err error) {
	if cipherNames == nil {
		return nil, nil
	}

	cipherIDs = make([]uint16, 0, len(cipherNames))
	for _, name := range cipherNames {
		id, ok := cipherSuites[name]
		if !ok {
			return nil, fmt.Errorf("unknown cipher %q", name)
		}

		cipherIDs = append(cipherIDs, id)
	}

	return cipherIDs, nil
}

// SaferCipherSuites returns a set of default cipher suites with vulnerable and
// weak cipher suites removed.
func SaferCipherSuites() (safe []uint16) {
	for _, s := range tls.CipherSuites() {
		switch s.ID {
		case
			tls.TLS_RSA_WITH_3DES_EDE_CBC_SHA,
			tls.TLS_RSA_WITH_AES_128_CBC_SHA,
			tls.TLS_RSA_WITH_AES_256_CBC_SHA,
			tls.TLS_RSA_WITH_AES_128_CBC_SHA256,
			tls.TLS_ECDHE_ECDSA_WITH_AES_128_CBC_SHA,
			tls.TLS_ECDHE_ECDSA_WITH_AES_256_CBC_SHA,
			tls.TLS_ECDHE_RSA_WITH_3DES_EDE_CBC_SHA,
			tls.TLS_ECDHE_RSA_WITH_AES_128_CBC_SHA,
			tls.TLS_ECDHE_RSA_WITH_AES_256_CBC_SHA,
			tls.TLS_ECDHE_ECDSA_WITH_AES_128_CBC_SHA256,
			tls.TLS_ECDHE_RSA_WITH_AES_128_CBC_SHA256:
			// Less safe 3DES and CBC suites, go on.
		default:
			safe = append(safe, s.ID)
		}
	}

	return safe
}

// CertificateHasIP returns true if cert has at least a single IP address among
// its subjectAltNames.
func CertificateHasIP(cert *x509.Certificate) (ok bool) {
	return len(cert.IPAddresses) > 0 || slices.ContainsFunc(cert.DNSNames, netutil.IsValidIPString)
}

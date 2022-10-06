// Package aghtls contains utilities for work with TLS.
package aghtls

import (
	"crypto/tls"
	"fmt"
)

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

// ParseCipherIDs returns a set of cipher suites with the cipher names provided
func ParseCipherIDs(ciphers []string) (userCiphers []uint16, err error) {
	for _, cipher := range ciphers {
		exists, cipherID := CipherExists(cipher)
		if exists {
			userCiphers = append(userCiphers, cipherID)
		} else {
			return nil, fmt.Errorf("unknown cipher : %s ", cipher)
		}
	}

	return userCiphers, nil
}

// CipherExists returns cipherid if exists, else return false in boolean
func CipherExists(cipher string) (exists bool, cipherID uint16) {
	for _, s := range tls.CipherSuites() {
		if s.Name == cipher {
			return true, s.ID
		}
	}

	return false, 0
}

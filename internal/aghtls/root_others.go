//go:build !linux

package aghtls

import "crypto/x509"

func rootCAs() (roots *x509.CertPool) {
	return nil
}

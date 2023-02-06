//go:build linux

package aghtls

import (
	"crypto/x509"
	"os"
	"path/filepath"

	"github.com/AdguardTeam/golibs/errors"
	"github.com/AdguardTeam/golibs/log"
)

func rootCAs() (roots *x509.CertPool) {
	// Directories with the system root certificates, which aren't supported by
	// Go's crypto/x509.
	dirs := []string{
		// Entware.
		"/opt/etc/ssl/certs",
	}

	roots = x509.NewCertPool()
	for _, dir := range dirs {
		dirEnts, err := os.ReadDir(dir)
		if err != nil {
			if errors.Is(err, os.ErrNotExist) {
				continue
			}

			// TODO(a.garipov): Improve error handling here and in other places.
			log.Error("aghtls: opening directory %q: %s", dir, err)
		}

		var rootsAdded bool
		for _, de := range dirEnts {
			var certData []byte
			rootFile := filepath.Join(dir, de.Name())
			certData, err = os.ReadFile(rootFile)
			if err != nil {
				log.Error("aghtls: reading root cert: %s", err)
			} else {
				if roots.AppendCertsFromPEM(certData) {
					rootsAdded = true
				} else {
					log.Error("aghtls: could not add root from %q", rootFile)
				}
			}
		}

		if rootsAdded {
			return roots
		}
	}

	return nil
}

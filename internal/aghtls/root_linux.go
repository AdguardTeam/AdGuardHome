//go:build linux

package aghtls

import (
	"context"
	"crypto/x509"
	"log/slog"
	"os"
	"path/filepath"

	"github.com/AdguardTeam/golibs/errors"
	"github.com/AdguardTeam/golibs/logutil/slogutil"
)

func rootCAs(ctx context.Context, l *slog.Logger) (roots *x509.CertPool) {
	// Directories with the system root certificates, which aren't supported by
	// Go's crypto/x509.
	dirs := []string{
		// Entware.
		"/opt/etc/ssl/certs",
	}

	roots = x509.NewCertPool()
	for _, dir := range dirs {
		if addCertsFromDir(ctx, l, roots, dir) {
			return roots
		}
	}

	return nil
}

// addCertsFromDir appends all readable PEM files from dir to pool.  It returns
// true if at least one certificate was accepted.
func addCertsFromDir(
	ctx context.Context,
	l *slog.Logger,
	pool *x509.CertPool,
	dir string,
) (ok bool) {
	dirEnts, err := os.ReadDir(dir)
	if err != nil {
		if !errors.Is(err, os.ErrNotExist) {
			// TODO(a.garipov): Improve error handling here and in other places.
			l.ErrorContext(ctx, "opening directory", slogutil.KeyError, err)
		}

		return false
	}

	var rootsAdded bool
	for _, de := range dirEnts {
		var certData []byte
		rootFile := filepath.Join(dir, de.Name())
		certData, err = os.ReadFile(rootFile)
		if err != nil {
			l.ErrorContext(ctx, "reading root cert", slogutil.KeyError, err)

			continue
		}

		if !pool.AppendCertsFromPEM(certData) {
			l.ErrorContext(ctx, "adding root cert", "file", rootFile, slogutil.KeyError, err)

			continue
		}

		rootsAdded = true
	}

	return rootsAdded
}

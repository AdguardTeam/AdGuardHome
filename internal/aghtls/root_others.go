//go:build !linux

package aghtls

import (
	"context"
	"crypto/x509"
	"log/slog"
)

func rootCAs(_ context.Context, _ *slog.Logger) (roots *x509.CertPool) {
	return nil
}

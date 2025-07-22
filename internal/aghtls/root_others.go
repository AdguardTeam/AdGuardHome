//go:build !linux

package aghtls

import (
	"context"
	"crypto/x509"
	"log/slog"
)

func rootCAs(ctx context.Context, l *slog.Logger) (roots *x509.CertPool) {
	return nil
}

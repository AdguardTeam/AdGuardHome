//go:build !openbsd && !linux && !darwin

package ossvc

import (
	"context"
	"log/slog"
)

// chooseSystem checks the current system detected and substitutes it with local
// implementation if needed.
func chooseSystem(_ context.Context, _ *slog.Logger) {}

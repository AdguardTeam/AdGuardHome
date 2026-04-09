//go:build !openbsd && !linux && !darwin

package ossvc

import (
	"context"
	"log/slog"

	"github.com/AdguardTeam/golibs/osutil/executil"
)

// chooseSystem checks the current system detected and substitutes it with local
// implementation if needed.
func chooseSystem(_ context.Context, _ *slog.Logger, _ executil.CommandConstructor) {}

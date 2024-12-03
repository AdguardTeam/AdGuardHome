//go:build windows

package permcheck

import (
	"context"
	"log/slog"

	"github.com/AdguardTeam/golibs/logutil/slogutil"
	"golang.org/x/sys/windows"
)

// check is the Windows-specific implementation of [Check].
//
// Note, that it only checks the owner and the ACEs of the working directory.
// This is due to the assumption that the working directory ACEs are inherited
// by the underlying files and directories, since at least [migrate] sets this
// inheritance mode.
func check(ctx context.Context, l *slog.Logger, workDir, _, _, _, _ string) {
	l = l.With("type", typeDir, "path", workDir)

	dacl, owner, err := getSecurityInfo(workDir)
	if err != nil {
		l.ErrorContext(ctx, "getting security info", slogutil.KeyError, err)

		return
	}

	if !owner.IsWellKnown(windows.WinBuiltinAdministratorsSid) {
		l.WarnContext(ctx, "owner is not in administrators group")
	}

	err = rangeACEs(dacl, func(
		hdr windows.ACE_HEADER,
		mask windows.ACCESS_MASK,
		sid *windows.SID,
	) (cont bool) {
		l.DebugContext(ctx, "checking access control entry", "mask", mask, "sid", sid)

		warn := false
		switch {
		case hdr.AceType != windows.ACCESS_ALLOWED_ACE_TYPE:
			// Skip non-allowed ACEs.
		case !sid.IsWellKnown(windows.WinBuiltinAdministratorsSid):
			// Non-administrator ACEs should not have any access rights.
			warn = mask > 0
		default:
			// Administrators should full control access rights.
			warn = mask&fullControlMask != fullControlMask
		}
		if warn {
			l.WarnContext(ctx, "unexpected access control entry", "mask", mask, "sid", sid)
		}

		return true
	})
	if err != nil {
		l.ErrorContext(ctx, "checking access control entries", slogutil.KeyError, err)
	}
}

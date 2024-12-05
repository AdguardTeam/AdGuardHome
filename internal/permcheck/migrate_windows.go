//go:build windows

package permcheck

import (
	"context"
	"log/slog"

	"github.com/AdguardTeam/golibs/logutil/slogutil"
	"golang.org/x/sys/windows"
)

// needsMigration is the Windows-specific implementation of [NeedsMigration].
func needsMigration(ctx context.Context, l *slog.Logger, workDir, _ string) (ok bool) {
	l = l.With("type", typeDir, "path", workDir)

	dacl, owner, err := getSecurityInfo(workDir)
	if err != nil {
		l.ErrorContext(ctx, "getting security info", slogutil.KeyError, err)

		return true
	}

	if !owner.IsWellKnown(windows.WinBuiltinAdministratorsSid) {
		return true
	}

	err = rangeACEs(dacl, func(
		hdr windows.ACE_HEADER,
		mask windows.ACCESS_MASK,
		sid *windows.SID,
	) (cont bool) {
		switch {
		case hdr.AceType != windows.ACCESS_ALLOWED_ACE_TYPE:
			// Skip non-allowed access control entries.
			l.DebugContext(ctx, "skipping deny access control entry", "sid", sid)
		case !sid.IsWellKnown(windows.WinBuiltinAdministratorsSid):
			// Non-administrator access control entries should not have any
			// access rights.
			ok = mask > 0
		default:
			// Administrators should have full control.
			ok = mask&fullControlMask != fullControlMask
		}

		// Stop ranging if the access control entry is unexpected.
		return !ok
	})
	if err != nil {
		l.ErrorContext(ctx, "checking access control entries", slogutil.KeyError, err)

		return true
	}

	return ok
}

// migrate is the Windows-specific implementation of [Migrate].
//
// It sets the owner to administrators and adds a full control access control
// entry for the account.  It also removes all non-administrator access control
// entries, and keeps deny access control entries.  For any created or modified
// entry it sets the propagation flags to be inherited by child objects.
func migrate(ctx context.Context, logger *slog.Logger, workDir, _, _, _, _ string) {
	l := logger.With("type", typeDir, "path", workDir)

	dacl, owner, err := getSecurityInfo(workDir)
	if err != nil {
		l.ErrorContext(ctx, "getting security info", slogutil.KeyError, err)

		return
	}

	admins, err := windows.CreateWellKnownSid(windows.WinBuiltinAdministratorsSid)
	if err != nil {
		l.ErrorContext(ctx, "creating administrators sid", slogutil.KeyError, err)

		return
	}

	// TODO(e.burkov):  Check for duplicates?
	var accessEntries []windows.EXPLICIT_ACCESS
	var setACL bool
	// Iterate over the access control entries in DACL to determine if its
	// migration is needed.
	err = rangeACEs(dacl, func(
		hdr windows.ACE_HEADER,
		mask windows.ACCESS_MASK,
		sid *windows.SID,
	) (cont bool) {
		switch {
		case hdr.AceType != windows.ACCESS_ALLOWED_ACE_TYPE:
			// Add non-allowed access control entries as is, since they specify
			// the access restrictions, which shouldn't be lost.
			l.InfoContext(ctx, "migrating deny access control entry", "sid", sid)
			accessEntries = append(accessEntries, newDenyExplicitAccess(sid, mask))
			setACL = true
		case !sid.IsWellKnown(windows.WinBuiltinAdministratorsSid):
			// Remove non-administrator ACEs, since such accounts should not
			// have any access rights.
			l.InfoContext(ctx, "removing access control entry", "sid", sid)
			setACL = true
		default:
			// Administrators should have full control.  Don't add a new entry
			// here since it will be added later in case there are other
			// required entries.
			l.InfoContext(ctx, "migrating access control entry", "sid", sid, "mask", mask)
			setACL = setACL || mask&fullControlMask != fullControlMask
		}

		return true
	})
	if err != nil {
		l.ErrorContext(ctx, "ranging through access control entries", slogutil.KeyError, err)

		return
	}

	if setACL {
		accessEntries = append(accessEntries, newFullExplicitAccess(admins))
	}

	if !owner.IsWellKnown(windows.WinBuiltinAdministratorsSid) {
		l.InfoContext(ctx, "migrating owner", "sid", owner)
		owner = admins
	} else {
		l.DebugContext(ctx, "owner is already an administrator")
		owner = nil
	}

	err = setSecurityInfo(workDir, owner, accessEntries)
	if err != nil {
		l.ErrorContext(ctx, "setting security info", slogutil.KeyError, err)
	}
}

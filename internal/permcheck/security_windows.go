//go:build windows

package permcheck

import (
	"fmt"
	"unsafe"

	"github.com/AdguardTeam/golibs/errors"
	"golang.org/x/sys/windows"
)

// objectType is the type of the object for directories in context of security
// API.
const objectType windows.SE_OBJECT_TYPE = windows.SE_FILE_OBJECT

// fileDeleteChildRight is the mask bit for the right to delete a child object.
// It seems to be missing from the [windows] package.
//
// See https://learn.microsoft.com/en-us/windows-hardware/drivers/ifs/access-mask.
const fileDeleteChildRight windows.ACCESS_MASK = 0b0100_0000

// fullControlMask is the mask for full control access rights.
const fullControlMask windows.ACCESS_MASK = windows.FILE_LIST_DIRECTORY |
	windows.FILE_WRITE_DATA |
	windows.FILE_APPEND_DATA |
	windows.FILE_READ_EA |
	windows.FILE_WRITE_EA |
	windows.FILE_TRAVERSE |
	fileDeleteChildRight |
	windows.FILE_READ_ATTRIBUTES |
	windows.FILE_WRITE_ATTRIBUTES |
	windows.DELETE |
	windows.READ_CONTROL |
	windows.WRITE_DAC |
	windows.WRITE_OWNER |
	windows.SYNCHRONIZE

// aceFunc is a function that handles access control entries in the
// discretionary access control list.  It should return true to continue
// iterating over the entries, or false to stop.
type aceFunc = func(
	hdr windows.ACE_HEADER,
	mask windows.ACCESS_MASK,
	sid *windows.SID,
) (cont bool)

// rangeACEs ranges over the access control entries in the discretionary access
// control list of the specified security descriptor and calls f for each one.
func rangeACEs(dacl *windows.ACL, f aceFunc) (err error) {
	var errs []error
	for i := range uint32(dacl.AceCount) {
		var ace *windows.ACCESS_ALLOWED_ACE
		err = windows.GetAce(dacl, i, &ace)
		if err != nil {
			errs = append(errs, fmt.Errorf("getting entry at index %d: %w", i, err))

			continue
		}

		sid := (*windows.SID)(unsafe.Pointer(&ace.SidStart))
		if !f(ace.Header, ace.Mask, sid) {
			break
		}
	}

	if err = errors.Join(errs...); err != nil {
		return fmt.Errorf("checking access control entries: %w", err)
	}

	return nil
}

// setSecurityInfo sets the security information on the specified file, using
// ents to create a discretionary access control list.  Either owner or ents can
// be nil, in which case the corresponding information is not set, but at least
// one of them should be specified.
func setSecurityInfo(fname string, owner *windows.SID, ents []windows.EXPLICIT_ACCESS) (err error) {
	var secInfo windows.SECURITY_INFORMATION

	var acl *windows.ACL
	if len(ents) > 0 {
		// TODO(e.burkov):  Investigate if this whole set is necessary.
		secInfo |= windows.DACL_SECURITY_INFORMATION |
			windows.PROTECTED_DACL_SECURITY_INFORMATION |
			windows.UNPROTECTED_DACL_SECURITY_INFORMATION

		acl, err = windows.ACLFromEntries(ents, nil)
		if err != nil {
			return fmt.Errorf("creating access control list: %w", err)
		}
	}

	if owner != nil {
		secInfo |= windows.OWNER_SECURITY_INFORMATION
	}

	if secInfo == 0 {
		return errors.Error("no security information to set")
	}

	err = windows.SetNamedSecurityInfo(fname, objectType, secInfo, owner, nil, acl, nil)
	if err != nil {
		return fmt.Errorf("setting security info: %w", err)
	}

	return nil
}

// getSecurityInfo retrieves the security information for the specified file.
func getSecurityInfo(fname string) (dacl *windows.ACL, owner *windows.SID, err error) {
	// desiredSecInfo defines the parts of a security descriptor to retrieve.
	const desiredSecInfo windows.SECURITY_INFORMATION = windows.OWNER_SECURITY_INFORMATION |
		windows.DACL_SECURITY_INFORMATION |
		windows.PROTECTED_DACL_SECURITY_INFORMATION |
		windows.UNPROTECTED_DACL_SECURITY_INFORMATION

	sd, err := windows.GetNamedSecurityInfo(fname, objectType, desiredSecInfo)
	if err != nil {
		return nil, nil, fmt.Errorf("getting security descriptor: %w", err)
	}

	owner, _, err = sd.Owner()
	if err != nil {
		return nil, nil, fmt.Errorf("getting owner sid: %w", err)
	}

	dacl, _, err = sd.DACL()
	if err != nil {
		return nil, nil, fmt.Errorf("getting discretionary access control list: %w", err)
	}

	return dacl, owner, nil
}

// newFullExplicitAccess creates a new explicit access entry with full control
// permissions.
func newFullExplicitAccess(sid *windows.SID) (accEnt windows.EXPLICIT_ACCESS) {
	return windows.EXPLICIT_ACCESS{
		AccessPermissions: fullControlMask,
		AccessMode:        windows.GRANT_ACCESS,
		Inheritance:       windows.SUB_CONTAINERS_AND_OBJECTS_INHERIT,
		Trustee: windows.TRUSTEE{
			TrusteeForm:  windows.TRUSTEE_IS_SID,
			TrusteeType:  windows.TRUSTEE_IS_UNKNOWN,
			TrusteeValue: windows.TrusteeValueFromSID(sid),
		},
	}
}

// newDenyExplicitAccess creates a new explicit access entry with specified deny
// permissions.
func newDenyExplicitAccess(
	sid *windows.SID,
	mask windows.ACCESS_MASK,
) (accEnt windows.EXPLICIT_ACCESS) {
	return windows.EXPLICIT_ACCESS{
		AccessPermissions: mask,
		AccessMode:        windows.DENY_ACCESS,
		Inheritance:       windows.SUB_CONTAINERS_AND_OBJECTS_INHERIT,
		Trustee: windows.TRUSTEE{
			TrusteeForm:  windows.TRUSTEE_IS_SID,
			TrusteeType:  windows.TRUSTEE_IS_UNKNOWN,
			TrusteeValue: windows.TrusteeValueFromSID(sid),
		},
	}
}

//go:build windows

package aghos

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"unsafe"

	"github.com/AdguardTeam/golibs/errors"
	"golang.org/x/sys/windows"
)

// fileInfo is a Windows implementation of [fs.FileInfo], that contains the
// filemode converted from the security descriptor.
type fileInfo struct {
	// fs.FileInfo is embedded to provide the default implementations and data
	// successfully retrieved by [os.Stat].
	fs.FileInfo

	// mode is the file mode converted from the security descriptor.
	mode fs.FileMode
}

// type check
var _ fs.FileInfo = (*fileInfo)(nil)

// Mode implements [fs.FileInfo.Mode] for [*fileInfo].
func (fi *fileInfo) Mode() (mode fs.FileMode) { return fi.mode }

// stat is a Windows implementation of [Stat].
func stat(name string) (fi os.FileInfo, err error) {
	absName, err := filepath.Abs(name)
	if err != nil {
		return nil, fmt.Errorf("computing absolute path: %w", err)
	}

	fi, err = os.Stat(absName)
	if err != nil {
		// Don't wrap the error, since it's informative enough as is.
		return nil, err
	}

	dacl, owner, group, err := retrieveDACL(absName)
	if err != nil {
		// Don't wrap the error, since it's informative enough as is.
		return nil, err
	}

	var ownerMask, groupMask, otherMask windows.ACCESS_MASK
	for i := range uint32(dacl.AceCount) {
		var ace *windows.ACCESS_ALLOWED_ACE
		err = windows.GetAce(dacl, i, &ace)
		if err != nil {
			return nil, fmt.Errorf("getting access control entry at index %d: %w", i, err)
		}

		entrySid := (*windows.SID)(unsafe.Pointer(&ace.SidStart))
		switch {
		case entrySid.Equals(owner):
			ownerMask |= ace.Mask
		case entrySid.Equals(group):
			groupMask |= ace.Mask
		default:
			otherMask |= ace.Mask
		}
	}

	mode := fi.Mode()
	perm := masksToPerm(ownerMask, groupMask, otherMask, mode.IsDir())

	return &fileInfo{
		FileInfo: fi,
		// Use the file mode from the security descriptor, but use the
		// calculated permission bits.
		mode: perm | mode&^fs.FileMode(0o777),
	}, nil
}

// retrieveDACL retrieves the discretionary access control list, owner, and
// group from the security descriptor of the file with the specified absolute
// name.
func retrieveDACL(absName string) (dacl *windows.ACL, owner, group *windows.SID, err error) {
	// desiredSecInfo defines the parts of a security descriptor to retrieve.
	const desiredSecInfo windows.SECURITY_INFORMATION = windows.OWNER_SECURITY_INFORMATION |
		windows.GROUP_SECURITY_INFORMATION |
		windows.DACL_SECURITY_INFORMATION |
		windows.PROTECTED_DACL_SECURITY_INFORMATION |
		windows.UNPROTECTED_DACL_SECURITY_INFORMATION

	sd, err := windows.GetNamedSecurityInfo(absName, windows.SE_FILE_OBJECT, desiredSecInfo)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("getting security descriptor: %w", err)
	}

	dacl, _, err = sd.DACL()
	if err != nil {
		return nil, nil, nil, fmt.Errorf("getting discretionary access control list: %w", err)
	}

	owner, _, err = sd.Owner()
	if err != nil {
		return nil, nil, nil, fmt.Errorf("getting owner sid: %w", err)
	}

	group, _, err = sd.Group()
	if err != nil {
		return nil, nil, nil, fmt.Errorf("getting group sid: %w", err)
	}

	return dacl, owner, group, nil
}

// chmod is a Windows implementation of [Chmod].
func chmod(name string, perm fs.FileMode) (err error) {
	fi, err := os.Stat(name)
	if err != nil {
		return fmt.Errorf("getting file info: %w", err)
	}

	entries := make([]windows.EXPLICIT_ACCESS, 0, 3)
	creatorMask, groupMask, worldMask := permToMasks(perm, fi.IsDir())

	sidMasks := []struct {
		Key   windows.WELL_KNOWN_SID_TYPE
		Value windows.ACCESS_MASK
	}{{
		Key:   windows.WinCreatorOwnerSid,
		Value: creatorMask,
	}, {
		Key:   windows.WinCreatorGroupSid,
		Value: groupMask,
	}, {
		Key:   windows.WinWorldSid,
		Value: worldMask,
	}}

	var errs []error
	for _, sidMask := range sidMasks {
		if sidMask.Value == 0 {
			continue
		}

		var trustee windows.TRUSTEE
		trustee, err = newWellKnownTrustee(sidMask.Key)
		if err != nil {
			errs = append(errs, err)

			continue
		}

		entries = append(entries, windows.EXPLICIT_ACCESS{
			AccessPermissions: sidMask.Value,
			AccessMode:        windows.GRANT_ACCESS,
			Inheritance:       windows.NO_INHERITANCE,
			Trustee:           trustee,
		})
	}

	if err = errors.Join(errs...); err != nil {
		return fmt.Errorf("creating access control entries: %w", err)
	}

	acl, err := windows.ACLFromEntries(entries, nil)
	if err != nil {
		return fmt.Errorf("creating access control list: %w", err)
	}

	// secInfo defines the parts of a security descriptor to set.
	const secInfo windows.SECURITY_INFORMATION = windows.DACL_SECURITY_INFORMATION |
		windows.PROTECTED_DACL_SECURITY_INFORMATION

	err = windows.SetNamedSecurityInfo(name, windows.SE_FILE_OBJECT, secInfo, nil, nil, acl, nil)
	if err != nil {
		return fmt.Errorf("setting security descriptor: %w", err)
	}

	return nil
}

// mkdir is a Windows implementation of [Mkdir].
//
// TODO(e.burkov):  Consider using [windows.CreateDirectory] instead of
// [os.Mkdir] to reduce the number of syscalls.
func mkdir(name string, perm os.FileMode) (err error) {
	name, err = filepath.Abs(name)
	if err != nil {
		return fmt.Errorf("computing absolute path: %w", err)
	}

	err = os.Mkdir(name, perm)
	if err != nil {
		return fmt.Errorf("creating directory: %w", err)
	}

	defer func() {
		if err != nil {
			err = errors.WithDeferred(err, os.Remove(name))
		}
	}()

	return chmod(name, perm)
}

// mkdirAll is a Windows implementation of [MkdirAll].
func mkdirAll(path string, perm os.FileMode) (err error) {
	parent, _ := filepath.Split(path)

	if parent != "" {
		err = os.MkdirAll(parent, perm)
		if err != nil && !errors.Is(err, os.ErrExist) {
			return fmt.Errorf("creating parent directories: %w", err)
		}
	}

	err = mkdir(path, perm)
	if errors.Is(err, os.ErrExist) {
		return nil
	}

	return err
}

// writeFile is a Windows implementation of [WriteFile].
func writeFile(filename string, data []byte, perm os.FileMode) (err error) {
	file, err := openFile(filename, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, perm)
	if err != nil {
		return fmt.Errorf("opening file: %w", err)
	}
	defer func() { err = errors.WithDeferred(err, file.Close()) }()

	_, err = file.Write(data)
	if err != nil {
		return fmt.Errorf("writing data: %w", err)
	}

	return nil
}

// openFile is a Windows implementation of [OpenFile].
func openFile(name string, flag int, perm os.FileMode) (file *os.File, err error) {
	// Only change permissions if the file not yet exists, but should be
	// created.
	if flag&os.O_CREATE == 0 {
		return os.OpenFile(name, flag, perm)
	}

	_, err = stat(name)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			defer func() { err = errors.WithDeferred(err, chmod(name, perm)) }()
		} else {
			return nil, fmt.Errorf("getting file info: %w", err)
		}
	}

	return os.OpenFile(name, flag, perm)
}

// newWellKnownTrustee returns a trustee for a well-known SID.
func newWellKnownTrustee(stype windows.WELL_KNOWN_SID_TYPE) (t windows.TRUSTEE, err error) {
	sid, err := windows.CreateWellKnownSid(stype)
	if err != nil {
		return windows.TRUSTEE{}, fmt.Errorf("creating sid for type %d: %w", stype, err)
	}

	return windows.TRUSTEE{
		TrusteeForm:  windows.TRUSTEE_IS_SID,
		TrusteeValue: windows.TrusteeValueFromSID(sid),
	}, nil
}

// UNIX file mode permission bits.
const (
	permRead    = 0b100
	permWrite   = 0b010
	permExecute = 0b001
)

// Windows access masks for appropriate UNIX file mode permission bits and
// file types.
const (
	fileReadRights windows.ACCESS_MASK = windows.READ_CONTROL |
		windows.FILE_READ_DATA |
		windows.FILE_READ_ATTRIBUTES |
		windows.FILE_READ_EA |
		windows.SYNCHRONIZE |
		windows.ACCESS_SYSTEM_SECURITY

	fileWriteRights windows.ACCESS_MASK = windows.WRITE_DAC |
		windows.WRITE_OWNER |
		windows.FILE_WRITE_DATA |
		windows.FILE_WRITE_ATTRIBUTES |
		windows.FILE_WRITE_EA |
		windows.DELETE |
		windows.FILE_APPEND_DATA |
		windows.SYNCHRONIZE |
		windows.ACCESS_SYSTEM_SECURITY

	fileExecuteRights windows.ACCESS_MASK = windows.FILE_EXECUTE

	dirReadRights windows.ACCESS_MASK = windows.READ_CONTROL |
		windows.FILE_LIST_DIRECTORY |
		windows.FILE_READ_EA |
		windows.FILE_READ_ATTRIBUTES<<1 |
		windows.SYNCHRONIZE |
		windows.ACCESS_SYSTEM_SECURITY

	dirWriteRights windows.ACCESS_MASK = windows.WRITE_DAC |
		windows.WRITE_OWNER |
		windows.DELETE |
		windows.FILE_WRITE_DATA |
		windows.FILE_APPEND_DATA |
		windows.FILE_WRITE_EA |
		windows.FILE_WRITE_ATTRIBUTES<<1 |
		windows.SYNCHRONIZE |
		windows.ACCESS_SYSTEM_SECURITY

	dirExecuteRights windows.ACCESS_MASK = windows.FILE_TRAVERSE
)

// permToMasks converts a UNIX file mode permissions to the corresponding
// Windows access masks.  The [isDir] argument is used to set specific access
// bits for directories.
func permToMasks(fm os.FileMode, isDir bool) (owner, group, world windows.ACCESS_MASK) {
	mask := fm.Perm()

	owner = permToMask(byte((mask>>6)&0b111), isDir)
	group = permToMask(byte((mask>>3)&0b111), isDir)
	world = permToMask(byte(mask&0b111), isDir)

	return owner, group, world
}

// permToMask converts a UNIX file mode permission bits within p byte to the
// corresponding Windows access mask.  The [isDir] argument is used to set
// specific access bits for directories.
func permToMask(p byte, isDir bool) (mask windows.ACCESS_MASK) {
	readRights, writeRights, executeRights := fileReadRights, fileWriteRights, fileExecuteRights
	if isDir {
		readRights, writeRights, executeRights = dirReadRights, dirWriteRights, dirExecuteRights
	}

	if p&permRead != 0 {
		mask |= readRights
	}
	if p&permWrite != 0 {
		mask |= writeRights
	}
	if p&permExecute != 0 {
		mask |= executeRights
	}

	return mask
}

// masksToPerm converts Windows access masks to the corresponding UNIX file
// mode permission bits.
func masksToPerm(u, g, o windows.ACCESS_MASK, isDir bool) (perm fs.FileMode) {
	perm |= fs.FileMode(maskToPerm(u, isDir)) << 6
	perm |= fs.FileMode(maskToPerm(g, isDir)) << 3
	perm |= fs.FileMode(maskToPerm(o, isDir))

	return perm
}

// maskToPerm converts a Windows access mask to the corresponding UNIX file
// mode permission bits.
func maskToPerm(mask windows.ACCESS_MASK, isDir bool) (perm byte) {
	readMask, writeMask, executeMask := fileReadRights, fileWriteRights, fileExecuteRights
	if isDir {
		readMask, writeMask, executeMask = dirReadRights, dirWriteRights, dirExecuteRights
	}

	// Remove common bits to avoid false positive detection of unset rights.
	readMask ^= windows.SYNCHRONIZE | windows.ACCESS_SYSTEM_SECURITY
	writeMask ^= windows.SYNCHRONIZE | windows.ACCESS_SYSTEM_SECURITY

	if mask&readMask != 0 {
		perm |= permRead
	}
	if mask&writeMask != 0 {
		perm |= permWrite
	}
	if mask&executeMask != 0 {
		perm |= permExecute
	}

	return perm
}

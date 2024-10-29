//go:build windows

package aghos

import (
	"io/fs"
	"testing"

	"github.com/stretchr/testify/assert"
	"golang.org/x/sys/windows"
)

func TestPermToMasks(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name      string
		perm      fs.FileMode
		wantUser  windows.ACCESS_MASK
		wantGroup windows.ACCESS_MASK
		wantOther windows.ACCESS_MASK
		isDir     bool
	}{{
		name:      "all",
		perm:      0b111_111_111,
		wantUser:  fileReadRights | fileWriteRights | fileExecuteRights,
		wantGroup: fileReadRights | fileWriteRights | fileExecuteRights,
		wantOther: fileReadRights | fileWriteRights | fileExecuteRights,
		isDir:     false,
	}, {
		name:      "user_write",
		perm:      0b010_000_000,
		wantUser:  fileWriteRights,
		wantGroup: 0,
		wantOther: 0,
		isDir:     false,
	}, {
		name:      "group_read",
		perm:      0b000_100_000,
		wantUser:  0,
		wantGroup: fileReadRights,
		wantOther: 0,
		isDir:     false,
	}, {
		name:      "all_dir",
		perm:      0b111_111_111,
		wantUser:  dirReadRights | dirWriteRights | dirExecuteRights,
		wantGroup: dirReadRights | dirWriteRights | dirExecuteRights,
		wantOther: dirReadRights | dirWriteRights | dirExecuteRights,
		isDir:     true,
	}, {
		name:      "user_write_dir",
		perm:      0b010_000_000,
		wantUser:  dirWriteRights,
		wantGroup: 0,
		wantOther: 0,
		isDir:     true,
	}}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			user, group, other := permToMasks(tc.perm, tc.isDir)
			assert.Equal(t, tc.wantUser, user)
			assert.Equal(t, tc.wantGroup, group)
			assert.Equal(t, tc.wantOther, other)
		})
	}
}

func TestMasksToPerm(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name     string
		user     windows.ACCESS_MASK
		group    windows.ACCESS_MASK
		other    windows.ACCESS_MASK
		wantPerm fs.FileMode
		isDir    bool
	}{{
		name:     "all",
		user:     fileReadRights | fileWriteRights | fileExecuteRights,
		group:    fileReadRights | fileWriteRights | fileExecuteRights,
		other:    fileReadRights | fileWriteRights | fileExecuteRights,
		wantPerm: 0b111_111_111,
		isDir:    false,
	}, {
		name:     "user_write",
		user:     fileWriteRights,
		group:    0,
		other:    0,
		wantPerm: 0b010_000_000,
		isDir:    false,
	}, {
		name:     "group_read",
		user:     0,
		group:    fileReadRights,
		other:    0,
		wantPerm: 0b000_100_000,
		isDir:    false,
	}, {
		name:     "no_access",
		user:     0,
		group:    0,
		other:    0,
		wantPerm: 0,
		isDir:    false,
	}, {
		name:     "all_dir",
		user:     dirReadRights | dirWriteRights | dirExecuteRights,
		group:    dirReadRights | dirWriteRights | dirExecuteRights,
		other:    dirReadRights | dirWriteRights | dirExecuteRights,
		wantPerm: 0b111_111_111,
		isDir:    true,
	}, {
		name:     "user_write_dir",
		user:     dirWriteRights,
		group:    0,
		other:    0,
		wantPerm: 0b010_000_000,
		isDir:    true,
	}}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			// Don't call [fs.FileMode.Perm] since the result is expected to
			// contain only the permission bits.
			assert.Equal(t, tc.wantPerm, masksToPerm(tc.user, tc.group, tc.other, tc.isDir))
		})
	}
}

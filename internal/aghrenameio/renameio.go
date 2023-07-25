// Package aghrenameio is a wrapper around package github.com/google/renameio/v2
// that provides a similar stream-based API for both Unix and Windows systems.
// While the Windows API is not technically atomic, it still provides a
// consistent stream-based interface, and atomic renames of files do not seem to
// be possible in all cases anyway.
//
// See https://github.com/google/renameio/issues/1.
//
// TODO(a.garipov): Consider moving to golibs/renameioutil once tried and
// tested.
package aghrenameio

import (
	"io/fs"

	"github.com/AdguardTeam/golibs/errors"
)

// PendingFile is the interface for pending temporary files.
type PendingFile interface {
	// Cleanup closes the file, and removes it without performing the renaming.
	// To close and rename the file, use CloseReplace.
	Cleanup() (err error)

	// CloseReplace closes the temporary file and replaces the destination file
	// with it, possibly atomically.
	//
	// This method is not safe for concurrent use by multiple goroutines.
	CloseReplace() (err error)

	// Write writes len(b) bytes from b to the File.  It returns the number of
	// bytes written and an error, if any.  Write returns a non-nil error when n
	// != len(b).
	Write(b []byte) (n int, err error)
}

// NewPendingFile is a wrapper around [renameio.NewPendingFile] on Unix systems
// and [os.CreateTemp] on Windows.
func NewPendingFile(filePath string, mode fs.FileMode) (f PendingFile, err error) {
	return newPendingFile(filePath, mode)
}

// WithDeferredCleanup is a helper that performs the necessary cleanups and
// finalizations of the temporary files based on the returned error.
func WithDeferredCleanup(returned error, file PendingFile) (err error) {
	// Make sure that any error returned from here is marked as a deferred one.
	if returned != nil {
		return errors.WithDeferred(returned, file.Cleanup())
	}

	return errors.WithDeferred(nil, file.CloseReplace())
}

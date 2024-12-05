//go:build windows

package aghrenameio

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"

	"github.com/AdguardTeam/golibs/errors"
)

// pendingFile is a wrapper around [*os.File] calling [os.Rename] in its Close
// method.
type pendingFile struct {
	file       *os.File
	targetPath string
}

// type check
var _ PendingFile = (*pendingFile)(nil)

// Cleanup implements the [PendingFile] interface for *pendingFile.
func (f *pendingFile) Cleanup() (err error) {
	closeErr := f.file.Close()
	err = os.Remove(f.file.Name())

	// Put closeErr into the deferred error because that's where it is usually
	// expected.
	return errors.WithDeferred(err, closeErr)
}

// CloseReplace implements the [PendingFile] interface for *pendingFile.
func (f *pendingFile) CloseReplace() (err error) {
	err = f.file.Close()
	if err != nil {
		return fmt.Errorf("closing: %w", err)
	}

	err = os.Rename(f.file.Name(), f.targetPath)
	if err != nil {
		return fmt.Errorf("renaming: %w", err)
	}

	return nil
}

// Write implements the [PendingFile] interface for *pendingFile.
func (f *pendingFile) Write(b []byte) (n int, err error) {
	return f.file.Write(b)
}

// NewPendingFile is a wrapper around [os.CreateTemp].
//
// f.Close must be called to finish the renaming.
func newPendingFile(filePath string, mode fs.FileMode) (f PendingFile, err error) {
	// Use the same directory as the file itself, because moves across
	// filesystems can be especially problematic.
	file, err := os.CreateTemp(filepath.Dir(filePath), "")
	if err != nil {
		return nil, fmt.Errorf("opening pending file: %w", err)
	}

	// TODO(e.burkov):  The [os.Chmod] implementation is useless on Windows,
	// investigate if it can be removed.
	err = os.Chmod(file.Name(), mode)
	if err != nil {
		return nil, fmt.Errorf("preparing pending file: %w", err)
	}

	return &pendingFile{
		file:       file,
		targetPath: filePath,
	}, nil
}

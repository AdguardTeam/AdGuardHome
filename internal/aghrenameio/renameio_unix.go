//go:build unix

package aghrenameio

import (
	"io/fs"

	"github.com/google/renameio/v2"
)

// pendingFile is a wrapper around [*renameio.PendingFile] making it an
// [io.WriteCloser].
type pendingFile struct {
	file *renameio.PendingFile
}

// type check
var _ PendingFile = pendingFile{}

// Cleanup implements the [PendingFile] interface for pendingFile.
func (f pendingFile) Cleanup() (err error) {
	return f.file.Cleanup()
}

// CloseReplace implements the [PendingFile] interface for pendingFile.
func (f pendingFile) CloseReplace() (err error) {
	return f.file.CloseAtomicallyReplace()
}

// Write implements the [PendingFile] interface for pendingFile.
func (f pendingFile) Write(b []byte) (n int, err error) {
	return f.file.Write(b)
}

// NewPendingFile is a wrapper around [renameio.NewPendingFile].
//
// f.Close must be called to finish the renaming.
func newPendingFile(filePath string, mode fs.FileMode) (f PendingFile, err error) {
	file, err := renameio.NewPendingFile(filePath, renameio.WithPermissions(mode))
	if err != nil {
		// Don't wrap the error since it's informative enough as is.
		return nil, err
	}

	return pendingFile{
		file: file,
	}, nil
}

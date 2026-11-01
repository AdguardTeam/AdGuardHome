//go:build !unix

package domainhash

import (
	"errors"
	"os"
)

// errMmapUnsupported is returned by mapFile on platforms without support for
// memory mapping, so that [Open] reads the file into memory instead.
var errMmapUnsupported = errors.New("memory mapping is not supported")

// mapFile is not supported on this platform, and always returns
// errMmapUnsupported.
func mapFile(f *os.File, size int64) (data []byte, unmap func() (err error), err error) {
	return nil, nil, errMmapUnsupported
}

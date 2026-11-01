//go:build unix

package domainhash

import (
	"fmt"
	"os"

	"golang.org/x/sys/unix"
)

// mapFile maps the first size bytes of the file f into memory as a read-only
// slice.  unmap releases the mapping.
func mapFile(f *os.File, size int64) (data []byte, unmap func() (err error), err error) {
	if size == 0 {
		return nil, noUnmap, nil
	}

	if int64(int(size)) != size {
		return nil, nil, fmt.Errorf("blocklist file is too large: %d bytes", size)
	}

	data, err = unix.Mmap(int(f.Fd()), 0, int(size), unix.PROT_READ, unix.MAP_SHARED)
	if err != nil {
		return nil, nil, fmt.Errorf("mapping blocklist: %w", err)
	}

	return data, func() (err error) {
		return unix.Munmap(data)
	}, nil
}

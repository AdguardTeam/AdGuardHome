package domainhash

import (
	"errors"
	"fmt"
	"os"
)

// ErrBadIndexSize is returned by [Open] when the size of the blocklist file is
// not a multiple of the hash size.
var ErrBadIndexSize = errors.New("blocklist file size is not a multiple of the hash size")

// noUnmap is the unmap function used when the blocklist file is not mapped.
func noUnmap() (err error) {
	return nil
}

// Index is a read-only set of truncated domain hashes, normally backed by a
// memory-mapped blocklist file.
type Index struct {
	// data is the blocklist file contents.  It must not be modified.
	data []byte

	// unmap releases the memory mapped by Open.  It is never nil while the
	// index is open.
	unmap func() (err error)

	// hashBytes is the number of bytes of a single hash in data.
	hashBytes int

	// count is the number of hashes in data.
	count int
}

// Open opens the blocklist file at path and prepares it for lookups.  hashBytes
// is the number of bytes of a single hash in the file, and zero means
// [DefaultHashBytes].  The file is memory-mapped when possible, and read into
// memory otherwise.
func Open(path string, hashBytes int) (idx *Index, err error) {
	hb := ClampHashBytes(hashBytes)

	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("opening blocklist %q: %w", path, err)
	}
	defer func() {
		// The file is closed after the mapping is created, which is fine: the
		// mapping stays valid.
		_ = f.Close()
	}()

	fi, err := f.Stat()
	if err != nil {
		return nil, fmt.Errorf("stating blocklist %q: %w", path, err)
	}

	if fi.IsDir() {
		return nil, fmt.Errorf("opening blocklist %q: is a directory", path)
	}

	size := fi.Size()
	if size%int64(hb) != 0 {
		return nil, fmt.Errorf("%w: %q has %d bytes", ErrBadIndexSize, path, size)
	}

	data, unmap, err := mapFile(f, size)
	if err != nil {
		data, err = os.ReadFile(path)
		if err != nil {
			return nil, fmt.Errorf("reading blocklist %q: %w", path, err)
		}

		unmap = noUnmap
	}

	return &Index{
		data:      data,
		unmap:     unmap,
		hashBytes: hb,
		count:     len(data) / hb,
	}, nil
}

// Len returns the number of hashes in the index.
func (idx *Index) Len() (n int) {
	if idx == nil {
		return 0
	}

	return idx.count
}

// HashBytes returns the number of bytes of a single hash in the index.
func (idx *Index) HashBytes() (n int) {
	if idx == nil {
		return 0
	}

	return idx.hashBytes
}

// Close releases the memory used by the index.  It is safe to call it more than
// once.
func (idx *Index) Close() (err error) {
	if idx == nil || idx.unmap == nil {
		return nil
	}

	unmap := idx.unmap
	idx.data = nil
	idx.unmap = nil
	idx.count = 0

	return unmap()
}

// ContainsHash reports whether the truncated hash h is in the index.
func (idx *Index) ContainsHash(h uint64) (ok bool) {
	if idx == nil || idx.count == 0 {
		return false
	}

	h &= truncMask(idx.hashBytes)

	hb := idx.hashBytes
	lo, hi := 0, idx.count
	for lo < hi {
		mid := int(uint(lo+hi) >> 1)
		v := GetHash(idx.data[mid*hb : (mid+1)*hb])
		switch {
		case v < h:
			lo = mid + 1
		case v > h:
			hi = mid
		default:
			return true
		}
	}

	return false
}

// ContainsDomain reports whether domain or any of its parent domains is in the
// index.  For example, if "example.com" is blocked, both "tracker.example.com"
// and "example.com" are reported as blocked, while "com" alone is never
// matched.
func (idx *Index) ContainsDomain(domain string) (ok bool) {
	if idx == nil || idx.count == 0 {
		return false
	}

	var buf [maxDomainLen]byte

	norm, ok := normalizeTo(buf[:], domain)
	if !ok {
		return false
	}

	mask := truncMask(idx.hashBytes)
	found := false
	forEachSuffix(norm, func(s []byte) (cont bool) {
		if idx.ContainsHash(sum(s) & mask) {
			found = true

			return false
		}

		return true
	})

	return found
}

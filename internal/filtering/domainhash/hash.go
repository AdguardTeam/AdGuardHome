// Package domainhash implements a memory-efficient blocklist of blocked domain
// names.  Domains are stored as sorted, truncated FNV-1a-64 hashes, so that a
// large blocklist takes only a few bytes per domain and is looked up with a
// binary search over a memory-mapped file, without loading the domain names
// themselves into memory.
//
// The blocklist file is a sequence of fixed-size entries, each holding the low
// HashBytes bytes of the hash of a blocked domain in little-endian order.  The
// entries are sorted in ascending order of the numeric hash value.
package domainhash

import (
	"bytes"
	"encoding/binary"
	"strings"
)

const (
	// DefaultHashBytes is the default number of bytes of a single hash stored
	// in the blocklist.
	DefaultHashBytes = 5

	// MinHashBytes is the minimum number of bytes of a single hash.
	MinHashBytes = 2

	// MaxHashBytes is the maximum number of bytes of a single hash.
	MaxHashBytes = 8

	// maxDomainLen is the maximum length of a domain name, in bytes, as
	// defined by RFC 1035.
	maxDomainLen = 253

	// maxLabelLen is the maximum length of a single label of a domain name, in
	// bytes, as defined by RFC 1035.
	maxLabelLen = 63
)

// Parameters of the FNV-1a-64 hash function.
const (
	fnvOffset64 uint64 = 14695981039346656037
	fnvPrime64  uint64 = 1099511628211
)

// ClampHashBytes returns n if it is a valid number of hash bytes, and the
// closest valid one otherwise.  Zero and negative values are replaced with
// [DefaultHashBytes].
func ClampHashBytes(n int) (b int) {
	switch {
	case n <= 0:
		return DefaultHashBytes
	case n < MinHashBytes:
		return MinHashBytes
	case n > MaxHashBytes:
		return MaxHashBytes
	default:
		return n
	}
}

// truncMask returns the mask of the lowest hashBytes bytes.
func truncMask(hashBytes int) (m uint64) {
	if hashBytes >= MaxHashBytes {
		return ^uint64(0)
	}

	return 1<<(uint(hashBytes)*8) - 1
}

// sum returns the FNV-1a-64 hash of s.
func sum[S ~string | ~[]byte](s S) (h uint64) {
	h = fnvOffset64
	for i := range len(s) {
		h = (h ^ uint64(s[i])) * fnvPrime64
	}

	return h
}

// Hash returns the FNV-1a-64 hash of s, truncated to ClampHashBytes(hashBytes)
// bytes.  s must already be in the normalized form; use [HashDomain] to
// normalize a domain name first.
func Hash(s string, hashBytes int) (h uint64) {
	return sum(s) & truncMask(ClampHashBytes(hashBytes))
}

// HashDomain returns the truncated hash of domain, which is normalized first.
// ok is false if domain is not a valid domain name.
func HashDomain(domain string, hashBytes int) (h uint64, ok bool) {
	var buf [maxDomainLen]byte

	norm, ok := normalizeTo(buf[:], domain)
	if !ok {
		return 0, false
	}

	return sum(norm) & truncMask(ClampHashBytes(hashBytes)), true
}

// NormalizeDomain returns the canonical form of domain used for hashing: it is
// lowercased, and the trailing dot and the leading "www." label are removed.
// ok is false if domain is not a valid domain name.
func NormalizeDomain(domain string) (norm string, ok bool) {
	var buf [maxDomainLen]byte

	b, ok := normalizeTo(buf[:], domain)
	if !ok {
		return "", false
	}

	return string(b), true
}

// normalizeTo writes the canonical form of domain to dst, which must be at
// least maxDomainLen bytes long, and returns it.  ok is false if domain is not
// a valid domain name.
func normalizeTo(dst []byte, domain string) (norm []byte, ok bool) {
	s := strings.TrimSpace(domain)
	if s == "" || len(s) > maxDomainLen {
		return nil, false
	}

	// Lowercase the domain and reject non-ASCII bytes at the same time.
	n := copy(dst, s)
	for i := range n {
		c := dst[i]
		switch {
		case c >= 'A' && c <= 'Z':
			dst[i] = c + 'a' - 'A'
		case c > 0x7f:
			return nil, false
		}
	}

	if dst[n-1] == '.' {
		n--
		if n == 0 {
			return nil, false
		}
	}

	// Strip the leading "www." label, unless it would leave a single-label
	// name, since stripping it from "www.com" would block all of "com".
	if n > len("www.") && string(dst[:len("www.")]) == "www." &&
		bytes.IndexByte(dst[len("www."):n], '.') >= 0 {
		n = copy(dst, dst[len("www."):n])
	}

	// Validate the labels.
	labelLen := 0
	for i := range n {
		c := dst[i]
		if c == '.' {
			if labelLen == 0 || dst[i-1] == '-' {
				return nil, false
			}

			labelLen = 0

			continue
		}

		labelLen++
		if labelLen > maxLabelLen {
			return nil, false
		}

		switch {
		case c >= 'a' && c <= 'z', c >= '0' && c <= '9', c == '_':
		case c == '-':
			if labelLen == 1 {
				return nil, false
			}
		default:
			return nil, false
		}
	}

	if labelLen == 0 || dst[n-1] == '-' {
		return nil, false
	}

	return dst[:n], true
}

// forEachSuffix calls fn with domain and with each of its parent suffixes that
// still have at least two labels.  For example, for "ads.tracker.example.com"
// fn is called with "ads.tracker.example.com", "tracker.example.com", and
// "example.com", and not with "com".  domain must be normalized.  The
// iteration stops when fn returns false.
func forEachSuffix(domain []byte, fn func(s []byte) (cont bool)) {
	if len(domain) == 0 || !fn(domain) {
		return
	}

	s := domain
	for {
		i := bytes.IndexByte(s, '.')
		if i < 0 {
			return
		}

		s = s[i+1:]
		if bytes.IndexByte(s, '.') < 0 {
			return
		}

		if !fn(s) {
			return
		}
	}
}

// PutHash writes the low ClampHashBytes(hashBytes) bytes of h to dst in
// little-endian order.  dst must be at least that long.
func PutHash(dst []byte, h uint64, hashBytes int) {
	n := ClampHashBytes(hashBytes)

	var buf [MaxHashBytes]byte
	binary.LittleEndian.PutUint64(buf[:], h)

	copy(dst[:n], buf[:n])
}

// GetHash returns the value encoded in b, which is a little-endian
// representation of between [MinHashBytes] and [MaxHashBytes] bytes.
func GetHash(b []byte) (h uint64) {
	if len(b) >= MaxHashBytes {
		return binary.LittleEndian.Uint64(b)
	}

	var buf [MaxHashBytes]byte
	copy(buf[:], b)

	return binary.LittleEndian.Uint64(buf[:])
}

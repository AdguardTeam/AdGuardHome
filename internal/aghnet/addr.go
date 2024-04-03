package aghnet

import (
	"strings"
)

// NormalizeDomain returns a lowercased version of host without the final dot,
// unless host is ".", in which case it returns it unchanged.  That is a special
// case that to allow matching queries like:
//
//	dig IN NS '.'
func NormalizeDomain(host string) (norm string) {
	if host == "." {
		return host
	}

	return strings.ToLower(strings.TrimSuffix(host, "."))
}

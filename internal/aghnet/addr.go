package aghnet

import (
	"fmt"
	"strings"

	"github.com/AdguardTeam/golibs/stringutil"
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

// NewDomainNameSet returns nil and error, if list has duplicate or empty domain
// name.  Otherwise returns a set, which contains domain names normalized using
// [NormalizeDomain].
func NewDomainNameSet(list []string) (set *stringutil.Set, err error) {
	set = stringutil.NewSet()

	for i, host := range list {
		if host == "" {
			return nil, fmt.Errorf("at index %d: hostname is empty", i)
		}

		host = NormalizeDomain(host)
		if set.Has(host) {
			return nil, fmt.Errorf("duplicate hostname %q at index %d", host, i)
		}

		set.Add(host)
	}

	return set, nil
}

package aghnet

import (
	"fmt"
	"net/netip"
	"strings"

	"github.com/AdguardTeam/golibs/errors"
	"github.com/AdguardTeam/golibs/stringutil"
)

// GenerateHostname generates the hostname from ip.  In case of using IPv4 the
// result should be like:
//
//	192-168-10-1
//
// In case of using IPv6, the result is like:
//
//	ff80-f076-0000-0000-0000-0000-0000-0010
//
// ip must be either an IPv4 or an IPv6.
func GenerateHostname(ip netip.Addr) (hostname string) {
	if !ip.IsValid() {
		// TODO(s.chzhen):  Get rid of it.
		panic("aghnet generate hostname: invalid ip")
	}

	ip = ip.Unmap()
	hostname = ip.StringExpanded()

	if ip.Is4() {
		return strings.Replace(hostname, ".", "-", -1)
	}

	return strings.Replace(hostname, ":", "-", -1)
}

// NewDomainNameSet returns nil and error, if list has duplicate or empty
// domain name.  Otherwise returns a set, which contains non-FQDN domain names,
// and nil error.
func NewDomainNameSet(list []string) (set *stringutil.Set, err error) {
	set = stringutil.NewSet()

	for i, v := range list {
		host := strings.ToLower(strings.TrimSuffix(v, "."))
		// TODO(a.garipov): Think about ignoring empty (".") names in the
		// future.
		if host == "" {
			return nil, errors.Error("host name is empty")
		}

		if set.Has(host) {
			return nil, fmt.Errorf("duplicate host name %q at index %d", host, i)
		}

		set.Add(host)
	}

	return set, nil
}

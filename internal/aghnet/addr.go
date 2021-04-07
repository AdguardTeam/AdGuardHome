package aghnet

import (
	"fmt"
	"net"
	"strings"

	"github.com/AdguardTeam/AdGuardHome/internal/agherr"
	"golang.org/x/net/idna"
)

// ValidateHardwareAddress returns an error if hwa is not a valid EUI-48,
// EUI-64, or 20-octet InfiniBand link-layer address.
func ValidateHardwareAddress(hwa net.HardwareAddr) (err error) {
	defer agherr.Annotate("validating hardware address %q: %w", &err, hwa)

	switch l := len(hwa); l {
	case 0:
		return agherr.Error("address is empty")
	case 6, 8, 20:
		return nil
	default:
		return fmt.Errorf("bad len: %d", l)
	}
}

// maxDomainLabelLen is the maximum allowed length of a domain name label
// according to RFC 1035.
const maxDomainLabelLen = 63

// maxDomainNameLen is the maximum allowed length of a full domain name
// according to RFC 1035.
//
// See https://stackoverflow.com/a/32294443/1892060.
const maxDomainNameLen = 253

const invalidCharMsg = "invalid char %q at index %d in %q"

// isValidHostFirstRune returns true if r is a valid first rune for a hostname
// label.
func isValidHostFirstRune(r rune) (ok bool) {
	return (r >= 'a' && r <= 'z') ||
		(r >= 'A' && r <= 'Z') ||
		(r >= '0' && r <= '9')
}

// isValidHostRune returns true if r is a valid rune for a hostname label.
func isValidHostRune(r rune) (ok bool) {
	return r == '-' || isValidHostFirstRune(r)
}

// ValidateDomainNameLabel returns an error if label is not a valid label of
// a domain name.
func ValidateDomainNameLabel(label string) (err error) {
	if len(label) > maxDomainLabelLen {
		return fmt.Errorf("%q is too long, max: %d", label, maxDomainLabelLen)
	} else if len(label) == 0 {
		return agherr.Error("label is empty")
	}

	if r := label[0]; !isValidHostFirstRune(rune(r)) {
		return fmt.Errorf(invalidCharMsg, r, 0, label)
	}

	for i, r := range label[1:] {
		if !isValidHostRune(r) {
			return fmt.Errorf(invalidCharMsg, r, i+1, label)
		}
	}

	return nil
}

// ValidateDomainName validates the domain name in accordance to RFC 952, RFC
// 1035, and with RFC-1123's inclusion of digits at the start of the host.  It
// doesn't validate against two or more hyphens to allow punycode and
// internationalized domains.
//
// TODO(a.garipov): After making sure that this works correctly, port this into
// module golibs.
func ValidateDomainName(name string) (err error) {
	name, err = idna.ToASCII(name)
	if err != nil {
		return err
	}

	l := len(name)
	if l == 0 || l > maxDomainNameLen {
		return fmt.Errorf("%q is too long, max: %d", name, maxDomainNameLen)
	}

	labels := strings.Split(name, ".")
	for i, l := range labels {
		err = ValidateDomainNameLabel(l)
		if err != nil {
			return fmt.Errorf("invalid domain name label at index %d: %w", i, err)
		}
	}

	return nil
}

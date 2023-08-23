// Package aghtest contains utilities for testing.
package aghtest

import (
	"crypto/sha256"
	"io"
	"net/netip"
	"testing"

	"github.com/AdguardTeam/golibs/log"
)

const (
	// ReqHost is the common request host for filtering tests.
	ReqHost = "www.host.example"

	// ReqFQDN is the common request FQDN for filtering tests.
	ReqFQDN = ReqHost + "."
)

// ReplaceLogWriter moves logger output to w and uses Cleanup method of t to
// revert changes.
func ReplaceLogWriter(t testing.TB, w io.Writer) {
	t.Helper()

	prev := log.Writer()
	t.Cleanup(func() { log.SetOutput(prev) })
	log.SetOutput(w)
}

// ReplaceLogLevel sets logging level to l and uses Cleanup method of t to
// revert changes.
func ReplaceLogLevel(t testing.TB, l log.Level) {
	t.Helper()

	switch l {
	case log.INFO, log.DEBUG, log.ERROR:
		// Go on.
	default:
		t.Fatalf("wrong l value (must be one of %v, %v, %v)", log.INFO, log.DEBUG, log.ERROR)
	}

	prev := log.GetLevel()
	t.Cleanup(func() { log.SetLevel(prev) })
	log.SetLevel(l)
}

// HostToIPs is a helper that generates one IPv4 and one IPv6 address from host.
func HostToIPs(host string) (ipv4, ipv6 netip.Addr) {
	hash := sha256.Sum256([]byte(host))

	return netip.AddrFrom4([4]byte(hash[:4])), netip.AddrFrom16([16]byte(hash[4:20]))
}

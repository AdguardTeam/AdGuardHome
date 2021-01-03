package dnsforward

import (
	"math/rand"
	"net"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestRebindingPrivateAddresses(t *testing.T) {
	c, _ := newRebindChecker(nil)

	r1 := byte(rand.Int31() & 0xFF)
	r2 := byte(rand.Int31() & 0xFF)
	r3 := byte(rand.Int31() & 0xFF)

	for _, ip := range []net.IP{
		net.IPv4(0, r1, r2, r3),             /* 0.0.0.0/8 (RFC 5735 section 3. "here" network) */
		net.IPv4(127, r1, r2, r3),           /* 127.0.0.0/8    (loopback) */
		net.IPv4(10, r1, r2, r3),            /* 10.0.0.0/8     (private)  */
		net.IPv4(172, 16|(0x0F&r1), r2, r3), /* 172.16.0.0/12  (private)  */
		net.IPv4(192, 168, r2, r3),          /* 192.168.0.0/16 (private)  */
		net.IPv4(169, 254, r2, r3),          /* 169.254.0.0/16 (zeroconf) */
		net.IPv4(192, 0, 2, r3),             /* 192.0.2.0/24   (test-net) */
		net.IPv4(198, 51, 100, r3),          /* 198.51.100.0/24(test-net) */
		net.IPv4(203, 0, 113, r3),           /* 203.0.113.0/24 (test-net) */
		net.IPv4(255, 255, 255, 255),        /* 255.255.255.255/32 (broadcast)*/

		/* RFC 6303 4.3 (unspecified & loopback) */
		net.IPv6zero,
		net.IPv6unspecified,

		/* RFC 6303 4.4 */
		/* RFC 6303 4.5 */
		/* RFC 6303 4.6 */
		net.IPv6interfacelocalallnodes,
		net.IPv6linklocalallnodes,
		net.IPv6linklocalallrouters,

		/* (TODO) Check IPv4-mapped IPv6 addresses */
	} {
		assert.Truef(t, c.isRebindIP(ip), "%s is not a rebind", ip)
	}
}

func TestRebindLocalhost(t *testing.T) {
	c := &dnsRebindChecker{}
	assert.False(t, c.isRebindHost("example.com"))
	assert.False(t, c.isRebindHost("200.0.0.1"))
	assert.True(t, c.isRebindHost("127.0.0.1"))
	assert.True(t, c.isRebindHost("localhost"))
}

func TestIsResponseRebind(t *testing.T) {
	c, _ := newRebindChecker([]string{
		"||totally-safe.com^",
	})
	s := &Server{
		rebinding: c,
	}

	for _, host := range []string{
		"0.1.2.3",         /* 0.0.0.0/8 (RFC 5735 section 3. "here" network) */
		"127.1.2.3",       /* 127.0.0.0/8    (loopback) */
		"10.1.2.3",        /* 10.0.0.0/8     (private)  */
		"172.16.2.3",      /* 172.16.0.0/12  (private)  */
		"192.168.2.3",     /* 192.168.0.0/16 (private)  */
		"169.254.2.3",     /* 169.254.0.0/16 (zeroconf) */
		"192.0.2.3",       /* 192.0.2.0/24   (test-net) */
		"198.51.100.3",    /* 198.51.100.0/24(test-net) */
		"203.0.113.3",     /* 203.0.113.0/24 (test-net) */
		"255.255.255.255", /* 255.255.255.255/32 (broadcast)*/

		/* RFC 6303 4.3 (unspecified & loopback) */
		net.IPv6zero.String(),
		net.IPv6unspecified.String(),

		/* RFC 6303 4.4 */
		/* RFC 6303 4.5 */
		/* RFC 6303 4.6 */
		net.IPv6interfacelocalallnodes.String(),
		net.IPv6linklocalallnodes.String(),
		net.IPv6linklocalallrouters.String(),

		"localhost",
	} {
		s.conf.RebindingProtectionEnabled = true
		assert.Truef(t, s.isResponseRebind("example.com", host), "host: %s", host)
		assert.Falsef(t, s.isResponseRebind("totally-safe.com", host), "host: %s", host)
		assert.Falsef(t, s.isResponseRebind("absolutely.totally-safe.com", host), "host: %s", host)

		s.conf.RebindingProtectionEnabled = false
		assert.Falsef(t, s.isResponseRebind("example.com", host), "host: %s", host)
		assert.Falsef(t, s.isResponseRebind("totally-safe.com", host), "host: %s", host)
		assert.Falsef(t, s.isResponseRebind("absolutely.totally-safe.com", host), "host: %s", host)
	}

	for _, host := range []string{
		"200.168.2.3",
		"another-example.com",
	} {
		s.conf.RebindingProtectionEnabled = true
		assert.Falsef(t, s.isResponseRebind("example.com", host), "host: %s", host)
		assert.Falsef(t, s.isResponseRebind("totally-safe.com", host), "host: %s", host)
		assert.Falsef(t, s.isResponseRebind("absolutely.totally-legit.com", host), "host: %s", host)

		s.conf.RebindingProtectionEnabled = false
		assert.Falsef(t, s.isResponseRebind("example.com", host), "host: %s", host)
		assert.Falsef(t, s.isResponseRebind("totally-safe.com", host), "host: %s", host)
		assert.Falsef(t, s.isResponseRebind("absolutely.totally-legit.com", host), "host: %s", host)
	}
}

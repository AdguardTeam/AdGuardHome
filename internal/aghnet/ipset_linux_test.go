//go:build linux

package aghnet

import (
	"net"
	"strings"
	"testing"

	"github.com/AdguardTeam/golibs/errors"
	"github.com/digineo/go-ipset/v2"
	"github.com/mdlayher/netlink"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/ti-mo/netfilter"
)

// fakeIpsetConn is a fake ipsetConn for tests.
type fakeIpsetConn struct {
	ipv4Header  *ipset.HeaderPolicy
	ipv4Entries *[]*ipset.Entry
	ipv6Header  *ipset.HeaderPolicy
	ipv6Entries *[]*ipset.Entry
}

// Add implements the ipsetConn interface for *fakeIpsetConn.
func (c *fakeIpsetConn) Add(name string, entries ...*ipset.Entry) (err error) {
	if strings.Contains(name, "ipv4") {
		*c.ipv4Entries = append(*c.ipv4Entries, entries...)

		return nil
	} else if strings.Contains(name, "ipv6") {
		*c.ipv6Entries = append(*c.ipv6Entries, entries...)

		return nil
	}

	return errors.Error("test: ipset not found")
}

// Close implements the ipsetConn interface for *fakeIpsetConn.
func (c *fakeIpsetConn) Close() (err error) {
	return nil
}

// Header implements the ipsetConn interface for *fakeIpsetConn.
func (c *fakeIpsetConn) Header(name string) (p *ipset.HeaderPolicy, err error) {
	if strings.Contains(name, "ipv4") {
		return c.ipv4Header, nil
	} else if strings.Contains(name, "ipv6") {
		return c.ipv6Header, nil
	}

	return nil, errors.Error("test: ipset not found")
}

func TestIpsetMgr_Add(t *testing.T) {
	ipsetConf := []string{
		"example.com,example.net/ipv4set",
		"example.org,example.biz/ipv6set",
	}

	var ipv4Entries []*ipset.Entry
	var ipv6Entries []*ipset.Entry

	fakeDial := func(
		pf netfilter.ProtoFamily,
		conf *netlink.Config,
	) (conn ipsetConn, err error) {
		return &fakeIpsetConn{
			ipv4Header: &ipset.HeaderPolicy{
				Family: ipset.NewUInt8Box(uint8(netfilter.ProtoIPv4)),
			},
			ipv4Entries: &ipv4Entries,
			ipv6Header: &ipset.HeaderPolicy{
				Family: ipset.NewUInt8Box(uint8(netfilter.ProtoIPv6)),
			},
			ipv6Entries: &ipv6Entries,
		}, nil
	}

	m, err := newIpsetMgrWithDialer(ipsetConf, fakeDial)
	require.NoError(t, err)

	ip4 := net.IP{1, 2, 3, 4}
	ip6 := net.IP{
		0x12, 0x34, 0x00, 0x00,
		0x00, 0x00, 0x00, 0x00,
		0x00, 0x00, 0x00, 0x00,
		0x00, 0x00, 0x56, 0x78,
	}

	n, err := m.Add("example.net", []net.IP{ip4}, nil)
	require.NoError(t, err)

	assert.Equal(t, 1, n)

	require.Len(t, ipv4Entries, 1)

	gotIP4 := ipv4Entries[0].IP.Value
	assert.Equal(t, ip4, gotIP4)

	n, err = m.Add("example.biz", nil, []net.IP{ip6})
	require.NoError(t, err)

	assert.Equal(t, 1, n)

	require.Len(t, ipv6Entries, 1)

	gotIP6 := ipv6Entries[0].IP.Value
	assert.Equal(t, ip6, gotIP6)

	err = m.Close()
	assert.NoError(t, err)
}

var ipsetPropsSink []ipsetProps

func BenchmarkIpsetMgr_lookupHost(b *testing.B) {
	propsLong := []ipsetProps{{
		name:   "example.com",
		family: netfilter.ProtoIPv4,
	}}

	propsShort := []ipsetProps{{
		name:   "example.net",
		family: netfilter.ProtoIPv4,
	}}

	m := &ipsetMgr{
		domainToIpsets: map[string][]ipsetProps{
			"":            propsLong,
			"example.net": propsShort,
		},
	}

	b.Run("long", func(b *testing.B) {
		const name = "a.very.long.domain.name.inside.the.domain.example.com"
		for i := 0; i < b.N; i++ {
			ipsetPropsSink = m.lookupHost(name)
		}

		require.Equal(b, propsLong, ipsetPropsSink)
	})

	b.Run("short", func(b *testing.B) {
		const name = "example.net"
		for i := 0; i < b.N; i++ {
			ipsetPropsSink = m.lookupHost(name)
		}

		require.Equal(b, propsShort, ipsetPropsSink)
	})
}

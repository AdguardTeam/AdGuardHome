//go:build linux

package ipset

import (
	"net"
	"strings"
	"testing"
	"time"

	"github.com/AdguardTeam/golibs/errors"
	"github.com/AdguardTeam/golibs/logutil/slogutil"
	"github.com/AdguardTeam/golibs/testutil"
	"github.com/digineo/go-ipset/v2"
	"github.com/mdlayher/netlink"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/ti-mo/netfilter"
)

// testTimeout is a common timeout for tests and contexts.
const testTimeout = 1 * time.Second

// fakeConn is a fake ipsetConn for tests.
type fakeConn struct {
	ipv4Header  *ipset.HeaderPolicy
	ipv4Entries *[]*ipset.Entry
	ipv6Header  *ipset.HeaderPolicy
	ipv6Entries *[]*ipset.Entry
	sets        []props
}

// type check
var _ ipsetConn = (*fakeConn)(nil)

// Add implements the [ipsetConn] interface for *fakeConn.
func (c *fakeConn) Add(name string, entries ...*ipset.Entry) (err error) {
	if strings.Contains(name, "ipv4") {
		*c.ipv4Entries = append(*c.ipv4Entries, entries...)

		return nil
	} else if strings.Contains(name, "ipv6") {
		*c.ipv6Entries = append(*c.ipv6Entries, entries...)

		return nil
	}

	return errors.Error("test: ipset not found")
}

// Close implements the [ipsetConn] interface for *fakeConn.
func (c *fakeConn) Close() (err error) {
	return nil
}

// Header implements the [ipsetConn] interface for *fakeConn.
func (c *fakeConn) Header(_ string) (_ *ipset.HeaderPolicy, _ error) {
	return nil, nil
}

// listAll implements the [ipsetConn] interface for *fakeConn.
func (c *fakeConn) listAll() (sets []props, err error) {
	return c.sets, nil
}

func TestManager_Add(t *testing.T) {
	ipsetList := []string{
		"example.com,example.net/ipv4set",
		"example.org,example.biz/ipv6set",
	}

	var ipv4Entries []*ipset.Entry
	var ipv6Entries []*ipset.Entry

	fakeDial := func(
		pf netfilter.ProtoFamily,
		conf *netlink.Config,
	) (conn ipsetConn, err error) {
		return &fakeConn{
			ipv4Header: &ipset.HeaderPolicy{
				Family: ipset.NewUInt8Box(uint8(netfilter.ProtoIPv4)),
			},
			ipv4Entries: &ipv4Entries,
			ipv6Header: &ipset.HeaderPolicy{
				Family: ipset.NewUInt8Box(uint8(netfilter.ProtoIPv6)),
			},
			ipv6Entries: &ipv6Entries,
			sets: []props{{
				name:   "ipv4set",
				family: netfilter.ProtoIPv4,
			}, {
				name:   "ipv6set",
				family: netfilter.ProtoIPv6,
			}},
		}, nil
	}

	conf := &Config{
		Logger: slogutil.NewDiscardLogger(),
		Lines:  ipsetList,
	}
	m, err := newManagerWithDialer(testutil.ContextWithTimeout(t, testTimeout), conf, fakeDial)
	require.NoError(t, err)

	ip4 := net.IP{1, 2, 3, 4}
	ip6 := net.IP{
		0x12, 0x34, 0x00, 0x00,
		0x00, 0x00, 0x00, 0x00,
		0x00, 0x00, 0x00, 0x00,
		0x00, 0x00, 0x56, 0x78,
	}

	n, err := m.Add(testutil.ContextWithTimeout(t, testTimeout), "example.net", []net.IP{ip4}, nil)
	require.NoError(t, err)

	assert.Equal(t, 1, n)

	require.Len(t, ipv4Entries, 1)

	gotIP4 := ipv4Entries[0].IP.Value
	assert.Equal(t, ip4, gotIP4)

	n, err = m.Add(testutil.ContextWithTimeout(t, testTimeout), "example.biz", nil, []net.IP{ip6})
	require.NoError(t, err)

	assert.Equal(t, 1, n)

	require.Len(t, ipv6Entries, 1)

	gotIP6 := ipv6Entries[0].IP.Value
	assert.Equal(t, ip6, gotIP6)

	err = m.Close()
	assert.NoError(t, err)
}

// ipsetPropsSink is the typed sink for benchmark results.
var ipsetPropsSink []props

func BenchmarkManager_LookupHost(b *testing.B) {
	propsLong := []props{{
		name:   "example.com",
		family: netfilter.ProtoIPv4,
	}}

	propsShort := []props{{
		name:   "example.net",
		family: netfilter.ProtoIPv4,
	}}

	m := &manager{
		domainToIpsets: map[string][]props{
			"":            propsLong,
			"example.net": propsShort,
		},
	}

	b.Run("long", func(b *testing.B) {
		const name = "a.very.long.domain.name.inside.the.domain.example.com"
		for range b.N {
			ipsetPropsSink = m.lookupHost(name)
		}

		require.Equal(b, propsLong, ipsetPropsSink)
	})

	b.Run("short", func(b *testing.B) {
		const name = "example.net"
		for range b.N {
			ipsetPropsSink = m.lookupHost(name)
		}

		require.Equal(b, propsShort, ipsetPropsSink)
	})
}

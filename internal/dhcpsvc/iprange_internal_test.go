package dhcpsvc

import (
	"net/netip"
	"strconv"
	"testing"

	"github.com/AdguardTeam/golibs/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	// testRangeStartV4Str is the string representation of the start of the test
	// range for IPv4.
	testRangeStartV4Str = "192.0.2.1"

	// testRangeEndV4Str is the string representation of the end of the test
	// range for IPv4.
	testRangeEndV4Str = "192.0.2.5"

	// testRangeStartV6Str is the string representation of the start of the
	// test range for IPv6.
	testRangeStartV6Str = "2001:db8::1"

	// testRangeEndV6Str is the string representation of the end of the test
	// range for IPv6.
	testRangeEndV6Str = "2001:db8::3"

	// testRangeEndV6LargeStr is the string representation of the end of the
	// test range for IPv6 that is too large.
	testRangeEndV6LargeStr = "2001:db9::4"
)

var (
	// testRangeStartV4 is the start of the test range for IPv4.
	testRangeStartV4 = netip.MustParseAddr(testRangeStartV4Str)

	// testRangeEndV4 is the end of the test range for IPv4.
	testRangeEndV4 = netip.MustParseAddr(testRangeEndV4Str)

	// testRangeStartV6 is the start of the test range for IPv6.
	testRangeStartV6 = netip.MustParseAddr(testRangeStartV6Str)

	// testRangeEndV6 is the end of the test range for IPv6.
	testRangeEndV6 = netip.MustParseAddr(testRangeEndV6Str)

	// testRangeEndV6Large is the end of the test range for IPv6 that is too
	// large.
	testRangeEndV6Large = netip.MustParseAddr(testRangeEndV6LargeStr)
)

func TestNewIPRange(t *testing.T) {
	testCases := []struct {
		start      netip.Addr
		end        netip.Addr
		name       string
		wantErrMsg string
	}{{
		start:      testRangeStartV4,
		end:        testRangeEndV4,
		name:       "success_ipv4",
		wantErrMsg: "",
	}, {
		start:      testRangeStartV6,
		end:        testRangeEndV6,
		name:       "success_ipv6",
		wantErrMsg: "",
	}, {
		start: testRangeEndV4,
		end:   testRangeStartV4,
		name:  "start_gt_end",
		wantErrMsg: "invalid ip range: start " + testRangeEndV4Str +
			" is greater than or equal to end " + testRangeStartV4Str,
	}, {
		start: testRangeStartV4,
		end:   testRangeStartV4,
		name:  "start_eq_end",
		wantErrMsg: "invalid ip range: start " + testRangeStartV4Str +
			" is greater than or equal to end " + testRangeStartV4Str,
	}, {
		start: testRangeStartV6,
		end:   testRangeEndV6Large,
		name:  "too_large",
		wantErrMsg: "invalid ip range: range length must be within " +
			strconv.FormatUint(maxRangeLen, 10),
	}, {
		start: testRangeStartV4,
		end:   testRangeEndV6,
		name:  "different_family",
		wantErrMsg: "invalid ip range: " + testRangeStartV4Str + " and " +
			testRangeEndV6Str + " must be within the same address family",
	}}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := newIPRange(tc.start, tc.end)
			testutil.AssertErrorMsg(t, tc.wantErrMsg, err)
		})
	}
}

func TestIPRange_Contains(t *testing.T) {
	r, err := newIPRange(testRangeStartV4, testRangeEndV4)
	require.NoError(t, err)

	testCases := []struct {
		in   netip.Addr
		want assert.BoolAssertionFunc
		name string
	}{{
		in:   testRangeStartV4,
		want: assert.True,
		name: "start",
	}, {
		in:   testRangeEndV4,
		want: assert.True,
		name: "end",
	}, {
		in:   testRangeStartV4.Next(),
		want: assert.True,
		name: "within",
	}, {
		in:   testRangeStartV4.Prev(),
		want: assert.False,
		name: "before",
	}, {
		in:   testRangeEndV4.Next(),
		want: assert.False,
		name: "after",
	}, {
		in:   testRangeStartV6,
		want: assert.False,
		name: "another_family",
	}}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			tc.want(t, r.contains(tc.in))
		})
	}
}

func TestIPRange_Find(t *testing.T) {
	r, err := newIPRange(testRangeStartV4, testRangeEndV4)
	require.NoError(t, err)

	num, ok := r.offset(testRangeEndV4)
	require.True(t, ok)

	testCases := []struct {
		predicate ipPredicate
		want      netip.Addr
		name      string
	}{{
		predicate: func(ip netip.Addr) (ok bool) {
			ipData := ip.AsSlice()

			return ipData[len(ipData)-1]%2 == 0
		},
		want: testRangeStartV4.Next(),
		name: "even",
	}, {
		predicate: func(ip netip.Addr) (ok bool) {
			ipData := ip.AsSlice()

			return ipData[len(ipData)-1]%10 == 0
		},
		want: netip.Addr{},
		name: "none",
	}, {
		predicate: func(ip netip.Addr) (ok bool) {
			return true
		},
		want: testRangeStartV4,
		name: "first",
	}, {
		predicate: func(ip netip.Addr) (ok bool) {
			off, _ := r.offset(ip)

			return off == num
		},
		want: testRangeEndV4,
		name: "last",
	}}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			got := r.find(tc.predicate)
			assert.Equal(t, tc.want, got)
		})
	}
}

func TestIPRange_Offset(t *testing.T) {
	r, err := newIPRange(testRangeStartV4, testRangeEndV4)
	require.NoError(t, err)

	testCases := []struct {
		wantOK     assert.BoolAssertionFunc
		in         netip.Addr
		name       string
		wantOffset uint64
	}{{
		wantOK:     assert.True,
		in:         testRangeStartV4.Next(),
		name:       "in",
		wantOffset: 1,
	}, {
		wantOK:     assert.True,
		in:         testRangeStartV4,
		name:       "in_start",
		wantOffset: 0,
	}, {
		wantOK:     assert.True,
		in:         testRangeEndV4,
		name:       "in_end",
		wantOffset: 4,
	}, {
		wantOK:     assert.False,
		in:         testRangeEndV4.Next(),
		name:       "out_after",
		wantOffset: 0,
	}, {
		wantOK:     assert.False,
		in:         testRangeStartV4.Prev(),
		name:       "out_before",
		wantOffset: 0,
	}}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			offset, ok := r.offset(tc.in)
			assert.Equal(t, tc.wantOffset, offset)
			tc.wantOK(t, ok)
		})
	}
}

package dhcpsvc

import (
	"net/netip"
	"strconv"
	"testing"

	"github.com/AdguardTeam/golibs/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewIPRange(t *testing.T) {
	start4 := netip.MustParseAddr("0.0.0.1")
	end4 := netip.MustParseAddr("0.0.0.3")
	start6 := netip.MustParseAddr("1::1")
	end6 := netip.MustParseAddr("1::3")
	end6Large := netip.MustParseAddr("2::3")

	testCases := []struct {
		start      netip.Addr
		end        netip.Addr
		name       string
		wantErrMsg string
	}{{
		start:      start4,
		end:        end4,
		name:       "success_ipv4",
		wantErrMsg: "",
	}, {
		start:      start6,
		end:        end6,
		name:       "success_ipv6",
		wantErrMsg: "",
	}, {
		start:      end4,
		end:        start4,
		name:       "start_gt_end",
		wantErrMsg: "invalid ip range: start 0.0.0.3 is greater than or equal to end 0.0.0.1",
	}, {
		start:      start4,
		end:        start4,
		name:       "start_eq_end",
		wantErrMsg: "invalid ip range: start 0.0.0.1 is greater than or equal to end 0.0.0.1",
	}, {
		start: start6,
		end:   end6Large,
		name:  "too_large",
		wantErrMsg: "invalid ip range: range length must be within " +
			strconv.FormatUint(maxRangeLen, 10),
	}, {
		start:      start4,
		end:        end6,
		name:       "different_family",
		wantErrMsg: "invalid ip range: 0.0.0.1 and 1::3 must be within the same address family",
	}}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := newIPRange(tc.start, tc.end)
			testutil.AssertErrorMsg(t, tc.wantErrMsg, err)
		})
	}
}

func TestIPRange_Contains(t *testing.T) {
	start, end := netip.MustParseAddr("0.0.0.1"), netip.MustParseAddr("0.0.0.3")
	r, err := newIPRange(start, end)
	require.NoError(t, err)

	testCases := []struct {
		in   netip.Addr
		want assert.BoolAssertionFunc
		name string
	}{{
		in:   start,
		want: assert.True,
		name: "start",
	}, {
		in:   end,
		want: assert.True,
		name: "end",
	}, {
		in:   start.Next(),
		want: assert.True,
		name: "within",
	}, {
		in:   netip.MustParseAddr("0.0.0.0"),
		want: assert.False,
		name: "before",
	}, {
		in:   netip.MustParseAddr("0.0.0.4"),
		want: assert.False,
		name: "after",
	}, {
		in:   netip.MustParseAddr("::"),
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
	start, end := netip.MustParseAddr("0.0.0.1"), netip.MustParseAddr("0.0.0.5")
	r, err := newIPRange(start, end)
	require.NoError(t, err)

	num, ok := r.offset(end)
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
		want: netip.MustParseAddr("0.0.0.2"),
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
		want: start,
		name: "first",
	}, {
		predicate: func(ip netip.Addr) (ok bool) {
			off, _ := r.offset(ip)

			return off == num
		},
		want: end,
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
	start, end := netip.MustParseAddr("0.0.0.1"), netip.MustParseAddr("0.0.0.5")
	r, err := newIPRange(start, end)
	require.NoError(t, err)

	testCases := []struct {
		in         netip.Addr
		name       string
		wantOffset uint64
		wantOK     bool
	}{{
		in:         netip.MustParseAddr("0.0.0.2"),
		name:       "in",
		wantOffset: 1,
		wantOK:     true,
	}, {
		in:         start,
		name:       "in_start",
		wantOffset: 0,
		wantOK:     true,
	}, {
		in:         end,
		name:       "in_end",
		wantOffset: 4,
		wantOK:     true,
	}, {
		in:         netip.MustParseAddr("0.0.0.6"),
		name:       "out_after",
		wantOffset: 0,
		wantOK:     false,
	}, {
		in:         netip.MustParseAddr("0.0.0.0"),
		name:       "out_before",
		wantOffset: 0,
		wantOK:     false,
	}}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			offset, ok := r.offset(tc.in)
			assert.Equal(t, tc.wantOffset, offset)
			assert.Equal(t, tc.wantOK, ok)
		})
	}
}

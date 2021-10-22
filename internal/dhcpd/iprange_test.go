package dhcpd

import (
	"net"
	"testing"

	"github.com/AdguardTeam/golibs/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewIPRange(t *testing.T) {
	start4 := net.IP{0, 0, 0, 1}
	end4 := net.IP{0, 0, 0, 3}
	start6 := net.IP{
		0x01, 0x00, 0x00, 0x00,
		0x00, 0x00, 0x00, 0x00,
		0x00, 0x00, 0x00, 0x00,
		0x00, 0x00, 0x00, 0x01,
	}
	end6 := net.IP{
		0x01, 0x00, 0x00, 0x00,
		0x00, 0x00, 0x00, 0x00,
		0x00, 0x00, 0x00, 0x00,
		0x00, 0x00, 0x00, 0x03,
	}
	end6Large := net.IP{
		0x02, 0x00, 0x00, 0x00,
		0x00, 0x00, 0x00, 0x00,
		0x00, 0x00, 0x00, 0x00,
		0x00, 0x00, 0x00, 0x03,
	}

	testCases := []struct {
		name       string
		wantErrMsg string
		start      net.IP
		end        net.IP
	}{{
		name:       "success_ipv4",
		wantErrMsg: "",
		start:      start4,
		end:        end4,
	}, {
		name:       "success_ipv6",
		wantErrMsg: "",
		start:      start6,
		end:        end6,
	}, {
		name:       "start_gt_end",
		wantErrMsg: "invalid ip range: start is greater than or equal to end",
		start:      end4,
		end:        start4,
	}, {
		name:       "start_eq_end",
		wantErrMsg: "invalid ip range: start is greater than or equal to end",
		start:      start4,
		end:        start4,
	}, {
		name:       "too_large",
		wantErrMsg: "invalid ip range: range is too large",
		start:      start6,
		end:        end6Large,
	}}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := newIPRange(tc.start, tc.end)
			testutil.AssertErrorMsg(t, tc.wantErrMsg, err)
		})
	}
}

func TestIPRange_Contains(t *testing.T) {
	start, end := net.IP{0, 0, 0, 1}, net.IP{0, 0, 0, 3}
	r, err := newIPRange(start, end)
	require.NoError(t, err)

	assert.True(t, r.contains(start))
	assert.True(t, r.contains(net.IP{0, 0, 0, 2}))
	assert.True(t, r.contains(end))

	assert.False(t, r.contains(net.IP{0, 0, 0, 0}))
	assert.False(t, r.contains(net.IP{0, 0, 0, 4}))
}

func TestIPRange_Find(t *testing.T) {
	start, end := net.IP{0, 0, 0, 1}, net.IP{0, 0, 0, 5}
	r, err := newIPRange(start, end)
	require.NoError(t, err)

	want := net.IPv4(0, 0, 0, 2)
	got := r.find(func(ip net.IP) (ok bool) {
		return ip[len(ip)-1]%2 == 0
	})

	assert.Equal(t, want, got)

	got = r.find(func(ip net.IP) (ok bool) {
		return ip[len(ip)-1]%10 == 0
	})
	assert.Nil(t, got)
}

func TestIPRange_Offset(t *testing.T) {
	start, end := net.IP{0, 0, 0, 1}, net.IP{0, 0, 0, 5}
	r, err := newIPRange(start, end)
	require.NoError(t, err)

	testCases := []struct {
		name       string
		in         net.IP
		wantOffset uint64
		wantOK     bool
	}{{
		name:       "in",
		in:         net.IP{0, 0, 0, 2},
		wantOffset: 1,
		wantOK:     true,
	}, {
		name:       "in_start",
		in:         start,
		wantOffset: 0,
		wantOK:     true,
	}, {
		name:       "in_end",
		in:         end,
		wantOffset: 4,
		wantOK:     true,
	}, {
		name:       "out_after",
		in:         net.IP{0, 0, 0, 6},
		wantOffset: 0,
		wantOK:     false,
	}, {
		name:       "out_before",
		in:         net.IP{0, 0, 0, 0},
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

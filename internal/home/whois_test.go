package home

import (
	"context"
	"io"
	"net"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// fakeConn is a mock implementation of net.Conn to simplify testing.
//
// TODO(e.burkov): Search for other places in code where it may be used. Move
// into aghtest then.
type fakeConn struct {
	// Conn is embedded here simply to make *fakeConn a net.Conn without
	// actually implementing all methods.
	net.Conn
	data []byte
}

// Write implements net.Conn interface for *fakeConn.  It always returns 0 and a
// nil error without mutating the slice.
func (c *fakeConn) Write(_ []byte) (n int, err error) {
	return 0, nil
}

// Read implements net.Conn interface for *fakeConn.  It puts the content of
// c.data field into b up to the b's capacity.
func (c *fakeConn) Read(b []byte) (n int, err error) {
	return copy(b, c.data), io.EOF
}

// Close implements net.Conn interface for *fakeConn.  It always returns nil.
func (c *fakeConn) Close() (err error) {
	return nil
}

// SetReadDeadline implements net.Conn interface for *fakeConn.  It always
// returns nil.
func (c *fakeConn) SetReadDeadline(_ time.Time) (err error) {
	return nil
}

// fakeDial is a mock implementation of customDialContext to simplify testing.
func (c *fakeConn) fakeDial(ctx context.Context, network, addr string) (conn net.Conn, err error) {
	return c, nil
}

func TestWHOIS(t *testing.T) {
	const (
		nl   = "\n"
		data = `OrgName:        FakeOrg LLC` + nl +
			`City:           Nonreal` + nl +
			`Country:        Imagiland` + nl
	)

	fc := &fakeConn{
		data: []byte(data),
	}

	w := WHOIS{
		timeoutMsec: 5000,
		dialContext: fc.fakeDial,
	}
	resp, err := w.queryAll(context.Background(), "1.2.3.4")
	assert.NoError(t, err)

	m := whoisParse(resp)
	require.NotEmpty(t, m)

	assert.Equal(t, "FakeOrg LLC", m["orgname"])
	assert.Equal(t, "Imagiland", m["country"])
	assert.Equal(t, "Nonreal", m["city"])
}

func TestWHOISParse(t *testing.T) {
	const (
		city    = "Nonreal"
		country = "Imagiland"
		orgname = "FakeOrgLLC"
		whois   = "whois.example.net"
	)

	testCases := []struct {
		want strmap
		name string
		in   string
	}{{
		want: strmap{},
		name: "empty",
		in:   ``,
	}, {
		want: strmap{},
		name: "comments",
		in:   "%\n#",
	}, {
		want: strmap{},
		name: "no_colon",
		in:   "city",
	}, {
		want: strmap{},
		name: "no_value",
		in:   "city:",
	}, {
		want: strmap{"city": city},
		name: "city",
		in:   `city: ` + city,
	}, {
		want: strmap{"country": country},
		name: "country",
		in:   `country: ` + country,
	}, {
		want: strmap{"orgname": orgname},
		name: "orgname",
		in:   `orgname: ` + orgname,
	}, {
		want: strmap{"orgname": orgname},
		name: "orgname_hyphen",
		in:   `org-name: ` + orgname,
	}, {
		want: strmap{"orgname": orgname},
		name: "orgname_descr",
		in:   `descr: ` + orgname,
	}, {
		want: strmap{"orgname": orgname},
		name: "orgname_netname",
		in:   `netname: ` + orgname,
	}, {
		want: strmap{"whois": whois},
		name: "whois",
		in:   `whois: ` + whois,
	}, {
		want: strmap{"whois": whois},
		name: "referralserver",
		in:   `referralserver: whois://` + whois,
	}, {
		want: strmap{},
		name: "other",
		in:   `other: value`,
	}}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			got := whoisParse(tc.in)
			assert.Equal(t, tc.want, got)
		})
	}
}

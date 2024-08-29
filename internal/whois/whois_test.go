package whois_test

import (
	"context"
	"io"
	"net"
	"net/netip"
	"testing"
	"time"

	"github.com/AdguardTeam/AdGuardHome/internal/whois"
	"github.com/AdguardTeam/golibs/logutil/slogutil"
	"github.com/AdguardTeam/golibs/testutil/fakenet"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDefault_Process(t *testing.T) {
	const (
		nl             = "\n"
		city           = "Nonreal"
		country        = "Imagiland"
		orgname        = "FakeOrgLLC"
		referralserver = "whois.example.net"
	)

	ip := netip.MustParseAddr("1.2.3.4")

	testCases := []struct {
		want *whois.Info
		name string
		data string
	}{{
		want: nil,
		name: "empty",
		data: "",
	}, {
		want: nil,
		name: "comments",
		data: "%\n#",
	}, {
		want: nil,
		name: "no_colon",
		data: "city",
	}, {
		want: nil,
		name: "no_value",
		data: "city:",
	}, {
		want: &whois.Info{
			City: city,
		},
		name: "city",
		data: "city: " + city,
	}, {
		want: &whois.Info{
			Country: country,
		},
		name: "country",
		data: "country: " + country,
	}, {
		want: &whois.Info{
			Orgname: orgname,
		},
		name: "orgname",
		data: "orgname: " + orgname,
	}, {
		want: &whois.Info{
			Orgname: orgname,
		},
		name: "orgname_hyphen",
		data: "org-name: " + orgname,
	}, {
		want: &whois.Info{
			Orgname: orgname,
		},
		name: "orgname_descr",
		data: "descr: " + orgname,
	}, {
		want: &whois.Info{
			Orgname: orgname,
		},
		name: "orgname_netname",
		data: "netname: " + orgname,
	}, {
		want: &whois.Info{
			City:    city,
			Country: country,
			Orgname: orgname,
		},
		name: "full",
		data: "OrgName: " + orgname + nl + "City: " + city + nl + "Country: " + country,
	}, {
		want: nil,
		name: "whois",
		data: "whois: " + referralserver,
	}, {
		want: nil,
		name: "referralserver",
		data: "referralserver: whois://" + referralserver,
	}, {
		want: nil,
		name: "other",
		data: "other: value",
	}}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			hit := 0

			fakeConn := &fakenet.Conn{
				OnRead: func(b []byte) (n int, err error) {
					hit++

					return copy(b, tc.data), io.EOF
				},
				OnWrite:       func(b []byte) (n int, err error) { return len(b), nil },
				OnClose:       func() (err error) { return nil },
				OnSetDeadline: func(t time.Time) (err error) { return nil },
			}

			w := whois.New(&whois.Config{
				Logger:  slogutil.NewDiscardLogger(),
				Timeout: 5 * time.Second,
				DialContext: func(_ context.Context, _, _ string) (_ net.Conn, _ error) {
					hit = 0

					return fakeConn, nil
				},
				MaxConnReadSize: 1024,
				MaxRedirects:    3,
				MaxInfoLen:      250,
				CacheSize:       100,
				CacheTTL:        time.Hour,
			})

			got, changed := w.Process(context.Background(), ip)
			require.True(t, changed)

			assert.Equal(t, tc.want, got)
			assert.Equal(t, 1, hit)

			// From cache.
			got, changed = w.Process(context.Background(), ip)
			require.False(t, changed)

			assert.Equal(t, tc.want, got)
			assert.Equal(t, 1, hit)
		})
	}
}

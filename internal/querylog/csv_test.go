package querylog

import (
	"net"
	"testing"
	"time"

	"github.com/AdguardTeam/AdGuardHome/internal/filtering"
	"github.com/miekg/dns"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var testDate = time.Date(2022, 1, 1, 0, 0, 0, 0, time.UTC)

func TestLogEntry_toCSV(t *testing.T) {
	ans, err := dns.NewRR("www.example.org. IN A 127.0.0.1")
	require.NoError(t, err)

	ansBytes, err := (&dns.Msg{Answer: []dns.RR{ans}}).Pack()
	require.NoError(t, err)

	testCases := []struct {
		entry *logEntry
		want  *csvRow
		name  string
	}{{
		name: "simple",
		entry: &logEntry{
			Time:              testDate,
			QHost:             "test.host",
			QType:             "A",
			QClass:            "IN",
			ReqECS:            "",
			ClientID:          "test-client-id",
			ClientProto:       ClientProtoDoH,
			Upstream:          "https://test.upstream:443/dns-query",
			Answer:            ansBytes,
			OrigAnswer:        nil,
			IP:                net.IP{1, 2, 3, 4},
			Result:            filtering.Result{},
			Elapsed:           500 * time.Millisecond,
			Cached:            false,
			AuthenticatedData: false,
		},
		want: &[18]string{
			"false",
			"NOERROR",
			"A",
			"127.0.0.1",
			"false",
			"1.2.3.4",
			"test-client-id",
			"",
			"500",
			"",
			"",
			"doh",
			"IN",
			"test.host",
			"A",
			"NotFilteredNotFound",
			"2022-01-01T00:00:00Z",
			"https://test.upstream:443/dns-query",
		},
	}}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.want, tc.entry.toCSV())
		})
	}
}

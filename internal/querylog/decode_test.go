package querylog

import (
	"bytes"
	"encoding/base64"
	"log/slog"
	"net"
	"net/netip"
	"testing"
	"time"

	"github.com/AdguardTeam/AdGuardHome/internal/filtering"
	"github.com/AdguardTeam/golibs/logutil/slogutil"
	"github.com/AdguardTeam/golibs/netutil"
	"github.com/AdguardTeam/golibs/testutil"
	"github.com/AdguardTeam/urlfilter/rules"
	"github.com/miekg/dns"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Common constants for tests.
const testTimeout = 1 * time.Second

func TestDecodeLogEntry(t *testing.T) {
	logOutput := &bytes.Buffer{}
	l := &queryLog{
		logger: slog.New(slog.NewTextHandler(logOutput, &slog.HandlerOptions{
			Level:       slog.LevelDebug,
			ReplaceAttr: slogutil.RemoveTime,
		})),
	}

	ctx := testutil.ContextWithTimeout(t, testTimeout)

	t.Run("success", func(t *testing.T) {
		const ansStr = `Qz+BgAABAAEAAAAAAmFuBnlhbmRleAJydQAAAQABwAwAAQABAAAACgAEAAAAAA==`
		const data = `{"IP":"127.0.0.1",` +
			`"CID":"cli42",` +
			`"T":"2020-11-25T18:55:56.519796+03:00",` +
			`"QH":"an.yandex.ru",` +
			`"QT":"A",` +
			`"QC":"IN",` +
			`"CP":"",` +
			`"ECS":"1.2.3.0/24",` +
			`"Answer":"` + ansStr + `",` +
			`"Cached":true,` +
			`"AD":true,` +
			`"Result":{` +
			`"IsFiltered":true,` +
			`"Reason":3,` +
			`"IPList":["127.0.0.2"],` +
			`"Rules":[{"FilterListID":42,"Text":"||an.yandex.ru","IP":"127.0.0.2"},` +
			`{"FilterListID":43,"Text":"||an2.yandex.ru","IP":"127.0.0.3"}],` +
			`"CanonName":"example.com",` +
			`"ServiceName":"example.org",` +
			`"DNSRewriteResult":{"RCode":0,"Response":{"1":["127.0.0.2"]}}},` +
			`"Upstream":"https://some.upstream",` +
			`"Elapsed":837429}`

		ans, err := base64.StdEncoding.DecodeString(ansStr)
		require.NoError(t, err)

		want := &logEntry{
			IP:          net.IPv4(127, 0, 0, 1),
			Time:        time.Date(2020, 11, 25, 15, 55, 56, 519796000, time.UTC),
			QHost:       "an.yandex.ru",
			QType:       "A",
			QClass:      "IN",
			ClientID:    "cli42",
			ClientProto: "",
			ReqECS:      "1.2.3.0/24",
			Answer:      ans,
			Cached:      true,
			Result: filtering.Result{
				DNSRewriteResult: &filtering.DNSRewriteResult{
					RCode: dns.RcodeSuccess,
					Response: filtering.DNSRewriteResultResponse{
						dns.TypeA: []rules.RRValue{net.IPv4(127, 0, 0, 2)},
					},
				},
				CanonName:   "example.com",
				ServiceName: "example.org",
				IPList:      []netip.Addr{netip.AddrFrom4([4]byte{127, 0, 0, 2})},
				Rules: []*filtering.ResultRule{{
					FilterListID: 42,
					Text:         "||an.yandex.ru",
					IP:           netip.AddrFrom4([4]byte{127, 0, 0, 2}),
				}, {
					FilterListID: 43,
					Text:         "||an2.yandex.ru",
					IP:           netip.AddrFrom4([4]byte{127, 0, 0, 3}),
				}},
				Reason:     filtering.FilteredBlockList,
				IsFiltered: true,
			},
			Upstream:          "https://some.upstream",
			Elapsed:           837429,
			AuthenticatedData: true,
		}

		got := &logEntry{}
		l.decodeLogEntry(ctx, got, data)

		s := logOutput.String()
		assert.Empty(t, s)

		// Correct for time zones.
		got.Time = got.Time.UTC()
		assert.Equal(t, want, got)
	})

	testCases := []struct {
		name string
		log  string
		want string
	}{{
		name: "all_right_old_rule",
		log:  `{"IP":"127.0.0.1","T":"2020-11-25T18:55:56.519796+03:00","QH":"an.yandex.ru","QT":"A","QC":"IN","CP":"","Answer":"Qz+BgAABAAEAAAAAAmFuBnlhbmRleAJydQAAAQABwAwAAQABAAAACgAEAAAAAA==","Result":{"IsFiltered":true,"Reason":3,"Rule":"||an.yandex.","FilterID":1,"ReverseHosts":["example.com"],"IPList":["127.0.0.1"]},"Elapsed":837429}`,
		want: "",
	}, {
		name: "bad_filter_id_old_rule",
		log:  `{"IP":"127.0.0.1","T":"2020-11-25T18:55:56.519796+03:00","QH":"an.yandex.ru","QT":"A","QC":"IN","CP":"","Answer":"Qz+BgAABAAEAAAAAAmFuBnlhbmRleAJydQAAAQABwAwAAQABAAAACgAEAAAAAA==","Result":{"IsFiltered":true,"Reason":3,"FilterID":1.5},"Elapsed":837429}`,
		want: `level=DEBUG msg="decoding result; handler" err="strconv.ParseInt: parsing \"1.5\": invalid syntax"`,
	}, {
		name: "bad_is_filtered",
		log:  `{"IP":"127.0.0.1","T":"2020-11-25T18:55:56.519796+03:00","QH":"an.yandex.ru","QT":"A","QC":"IN","CP":"","Answer":"Qz+BgAABAAEAAAAAAmFuBnlhbmRleAJydQAAAQABwAwAAQABAAAACgAEAAAAAA==","Result":{"IsFiltered":trooe,"Reason":3},"Elapsed":837429}`,
		want: `level=DEBUG msg="decoding log entry; token" err="invalid character 'o' in literal true (expecting 'u')"`,
	}, {
		name: "bad_elapsed",
		log:  `{"IP":"127.0.0.1","T":"2020-11-25T18:55:56.519796+03:00","QH":"an.yandex.ru","QT":"A","QC":"IN","CP":"","Answer":"Qz+BgAABAAEAAAAAAmFuBnlhbmRleAJydQAAAQABwAwAAQABAAAACgAEAAAAAA==","Result":{"IsFiltered":true,"Reason":3},"Elapsed":-1}`,
		want: "",
	}, {
		name: "bad_ip",
		log:  `{"IP":127001,"T":"2020-11-25T18:55:56.519796+03:00","QH":"an.yandex.ru","QT":"A","QC":"IN","CP":"","Answer":"Qz+BgAABAAEAAAAAAmFuBnlhbmRleAJydQAAAQABwAwAAQABAAAACgAEAAAAAA==","Result":{"IsFiltered":true,"Reason":3},"Elapsed":837429}`,
		want: "",
	}, {
		name: "bad_time",
		log:  `{"IP":"127.0.0.1","T":"12/09/1998T15:00:00.000000+05:00","QH":"an.yandex.ru","QT":"A","QC":"IN","CP":"","Answer":"Qz+BgAABAAEAAAAAAmFuBnlhbmRleAJydQAAAQABwAwAAQABAAAACgAEAAAAAA==","Result":{"IsFiltered":true,"Reason":3},"Elapsed":837429}`,
		want: `level=DEBUG msg="decoding log entry; handler" err="parsing time \"12/09/1998T15:00:00.000000+05:00\" as \"2006-01-02T15:04:05Z07:00\": cannot parse \"12/09/1998T15:00:00.000000+05:00\" as \"2006\""`,
	}, {
		name: "bad_host",
		log:  `{"IP":"127.0.0.1","T":"2020-11-25T18:55:56.519796+03:00","QH":6,"QT":"A","QC":"IN","CP":"","Answer":"Qz+BgAABAAEAAAAAAmFuBnlhbmRleAJydQAAAQABwAwAAQABAAAACgAEAAAAAA==","Result":{"IsFiltered":true,"Reason":3},"Elapsed":837429}`,
		want: "",
	}, {
		name: "bad_type",
		log:  `{"IP":"127.0.0.1","T":"2020-11-25T18:55:56.519796+03:00","QH":"an.yandex.ru","QT":true,"QC":"IN","CP":"","Answer":"Qz+BgAABAAEAAAAAAmFuBnlhbmRleAJydQAAAQABwAwAAQABAAAACgAEAAAAAA==","Result":{"IsFiltered":true,"Reason":3},"Elapsed":837429}`,
		want: "",
	}, {
		name: "bad_class",
		log:  `{"IP":"127.0.0.1","T":"2020-11-25T18:55:56.519796+03:00","QH":"an.yandex.ru","QT":"A","QC":false,"CP":"","Answer":"Qz+BgAABAAEAAAAAAmFuBnlhbmRleAJydQAAAQABwAwAAQABAAAACgAEAAAAAA==","Result":{"IsFiltered":true,"Reason":3},"Elapsed":837429}`,
		want: "",
	}, {
		name: "bad_client_proto",
		log:  `{"IP":"127.0.0.1","T":"2020-11-25T18:55:56.519796+03:00","QH":"an.yandex.ru","QT":"A","QC":"IN","CP":8,"Answer":"Qz+BgAABAAEAAAAAAmFuBnlhbmRleAJydQAAAQABwAwAAQABAAAACgAEAAAAAA==","Result":{"IsFiltered":true,"Reason":3},"Elapsed":837429}`,
		want: "",
	}, {
		name: "very_bad_client_proto",
		log:  `{"IP":"127.0.0.1","T":"2020-11-25T18:55:56.519796+03:00","QH":"an.yandex.ru","QT":"A","QC":"IN","CP":"dog","Answer":"Qz+BgAABAAEAAAAAAmFuBnlhbmRleAJydQAAAQABwAwAAQABAAAACgAEAAAAAA==","Result":{"IsFiltered":true,"Reason":3},"Elapsed":837429}`,
		want: `level=DEBUG msg="decoding log entry; handler" err="invalid client proto: \"dog\""`,
	}, {
		name: "bad_answer",
		log:  `{"IP":"127.0.0.1","T":"2020-11-25T18:55:56.519796+03:00","QH":"an.yandex.ru","QT":"A","QC":"IN","CP":"","Answer":0.9,"Result":{"IsFiltered":true,"Reason":3},"Elapsed":837429}`,
		want: "",
	}, {
		name: "very_bad_answer",
		log:  `{"IP":"127.0.0.1","T":"2020-11-25T18:55:56.519796+03:00","QH":"an.yandex.ru","QT":"A","QC":"IN","CP":"","Answer":"Qz+BgAABAAEAAAAAAmuBnlhbmRleAJydQAAAQABwAwAAQABAAAACgAEAAAAAA==","Result":{"IsFiltered":true,"Reason":3},"Elapsed":837429}`,
		want: `level=DEBUG msg="decoding log entry; handler" err="illegal base64 data at input byte 61"`,
	}, {
		name: "bad_rule",
		log:  `{"IP":"127.0.0.1","T":"2020-11-25T18:55:56.519796+03:00","QH":"an.yandex.ru","QT":"A","QC":"IN","CP":"","Answer":"Qz+BgAABAAEAAAAAAmFuBnlhbmRleAJydQAAAQABwAwAAQABAAAACgAEAAAAAA==","Result":{"IsFiltered":true,"Reason":3,"Rule":false},"Elapsed":837429}`,
		want: "",
	}, {
		name: "bad_reason",
		log:  `{"IP":"127.0.0.1","T":"2020-11-25T18:55:56.519796+03:00","QH":"an.yandex.ru","QT":"A","QC":"IN","CP":"","Answer":"Qz+BgAABAAEAAAAAAmFuBnlhbmRleAJydQAAAQABwAwAAQABAAAACgAEAAAAAA==","Result":{"IsFiltered":true,"Reason":true},"Elapsed":837429}`,
		want: "",
	}, {
		name: "bad_reverse_hosts",
		log:  `{"IP":"127.0.0.1","T":"2020-11-25T18:55:56.519796+03:00","QH":"an.yandex.ru","QT":"A","QC":"IN","CP":"","Answer":"Qz+BgAABAAEAAAAAAmFuBnlhbmRleAJydQAAAQABwAwAAQABAAAACgAEAAAAAA==","Result":{"IsFiltered":true,"Reason":3,"ReverseHosts":[{}]},"Elapsed":837429}`,
		want: `level=DEBUG msg="decoding result reverse hosts" err="unexpected delimiter: \"{\""`,
	}, {
		name: "bad_ip_list",
		log:  `{"IP":"127.0.0.1","T":"2020-11-25T18:55:56.519796+03:00","QH":"an.yandex.ru","QT":"A","QC":"IN","CP":"","Answer":"Qz+BgAABAAEAAAAAAmFuBnlhbmRleAJydQAAAQABwAwAAQABAAAACgAEAAAAAA==","Result":{"IsFiltered":true,"Reason":3,"ReverseHosts":["example.net"],"IPList":[{}]},"Elapsed":837429}`,
		want: `level=DEBUG msg="decoding result ip list" err="unexpected delimiter: \"{\""`,
	}}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			l.decodeLogEntry(ctx, new(logEntry), tc.log)
			got := logOutput.String()
			if tc.want == "" {
				assert.Empty(t, got)
			} else {
				require.NotEmpty(t, got)

				// Remove newline.
				got = got[:len(got)-1]
				assert.Equal(t, tc.want, got)
			}

			logOutput.Reset()
		})
	}
}

func TestDecodeLogEntry_backwardCompatability(t *testing.T) {
	var (
		a1    = netutil.IPv4Localhost()
		a2    = a1.Next()
		aaaa1 = netutil.IPv6Localhost()
		aaaa2 = aaaa1.Next()
	)

	l := &queryLog{
		logger: slogutil.NewDiscardLogger(),
	}

	ctx := testutil.ContextWithTimeout(t, testTimeout)

	testCases := []struct {
		want  *logEntry
		entry string
		name  string
	}{{
		entry: `{"Result":{"ReverseHosts":["example.net","example.org"]}`,
		want: &logEntry{
			Result: filtering.Result{DNSRewriteResult: &filtering.DNSRewriteResult{
				RCode: dns.RcodeSuccess,
				Response: filtering.DNSRewriteResultResponse{
					dns.TypePTR: []rules.RRValue{"example.net.", "example.org."},
				},
			}},
		},
		name: "reverse_hosts",
	}, {
		entry: `{"Result":{"IPList":["127.0.0.1","127.0.0.2","::1","::2"],"Reason":10}}`,
		want: &logEntry{
			Result: filtering.Result{
				DNSRewriteResult: &filtering.DNSRewriteResult{
					RCode: dns.RcodeSuccess,
					Response: filtering.DNSRewriteResultResponse{
						dns.TypeA:    []rules.RRValue{a1, a2},
						dns.TypeAAAA: []rules.RRValue{aaaa1, aaaa2},
					},
				},
				Reason: filtering.RewrittenAutoHosts,
			},
		},
		name: "iplist_autohosts",
	}, {
		entry: `{"Result":{"IPList":["127.0.0.1","127.0.0.2","::1","::2"],"Reason":9}}`,
		want: &logEntry{
			Result: filtering.Result{
				IPList: []netip.Addr{
					a1,
					a2,
					aaaa1,
					aaaa2,
				},
				Reason: filtering.Rewritten,
			},
		},
		name: "iplist_rewritten",
	}}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			e := &logEntry{}
			l.decodeLogEntry(ctx, e, tc.entry)

			assert.Equal(t, tc.want, e)
		})
	}
}

// anonymizeIPSlow masks ip to anonymize the client if the ip is a valid one.
// It only exists in purposes of benchmark comparison, see BenchmarkAnonymizeIP.
func anonymizeIPSlow(ip net.IP) {
	if ip4 := ip.To4(); ip4 != nil {
		copy(ip4[net.IPv4len-2:net.IPv4len], []byte{0, 0})
	} else if len(ip) == net.IPv6len {
		copy(ip[net.IPv6len-10:net.IPv6len], []byte{0, 0, 0, 0, 0, 0, 0, 0, 0, 0})
	}
}

func BenchmarkAnonymizeIP(b *testing.B) {
	benchCases := []struct {
		name string
		ip   net.IP
		want net.IP
	}{{
		name: "v4",
		ip:   net.IP{1, 2, 3, 4},
		want: net.IP{1, 2, 0, 0},
	}, {
		name: "v4_mapped",
		ip:   net.IP{1, 2, 3, 4}.To16(),
		want: net.IP{1, 2, 0, 0}.To16(),
	}, {
		name: "v6",
		ip: net.IP{
			0xa, 0xb, 0x0, 0x0,
			0x0, 0xb, 0xa, 0x9,
			0x8, 0x7, 0x6, 0x5,
			0x4, 0x3, 0x2, 0x1,
		},
		want: net.IP{
			0xa, 0xb, 0x0, 0x0,
			0x0, 0xb, 0x0, 0x0,
			0x0, 0x0, 0x0, 0x0,
			0x0, 0x0, 0x0, 0x0,
		},
	}, {
		name: "invalid",
		ip:   net.IP{1, 2, 3},
		want: net.IP{1, 2, 3},
	}}

	for _, bc := range benchCases {
		b.Run(bc.name, func(b *testing.B) {
			b.ReportAllocs()

			for range b.N {
				AnonymizeIP(bc.ip)
			}

			assert.Equal(b, bc.want, bc.ip)
		})

		b.Run(bc.name+"_slow", func(b *testing.B) {
			b.ReportAllocs()

			for range b.N {
				anonymizeIPSlow(bc.ip)
			}

			assert.Equal(b, bc.want, bc.ip)
		})
	}
}

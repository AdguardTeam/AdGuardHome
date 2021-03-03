package querylog

import (
	"bytes"
	"encoding/base64"
	"net"
	"strings"
	"testing"
	"time"

	"github.com/AdguardTeam/AdGuardHome/internal/aghtest"
	"github.com/AdguardTeam/AdGuardHome/internal/dnsfilter"
	"github.com/AdguardTeam/golibs/log"
	"github.com/AdguardTeam/urlfilter/rules"
	"github.com/miekg/dns"
	"github.com/stretchr/testify/assert"
)

func TestDecodeLogEntry(t *testing.T) {
	logOutput := &bytes.Buffer{}

	aghtest.ReplaceLogWriter(t, logOutput)
	aghtest.ReplaceLogLevel(t, log.DEBUG)

	t.Run("success", func(t *testing.T) {
		const ansStr = `Qz+BgAABAAEAAAAAAmFuBnlhbmRleAJydQAAAQABwAwAAQABAAAACgAEAAAAAA==`
		const data = `{"IP":"127.0.0.1",` +
			`"CID":"cli42",` +
			`"T":"2020-11-25T18:55:56.519796+03:00",` +
			`"QH":"an.yandex.ru",` +
			`"QT":"A",` +
			`"QC":"IN",` +
			`"CP":"",` +
			`"Answer":"` + ansStr + `",` +
			`"Result":{` +
			`"IsFiltered":true,` +
			`"Reason":3,` +
			`"ReverseHosts":["example.net"],` +
			`"IPList":["127.0.0.2"],` +
			`"Rules":[{"FilterListID":42,"Text":"||an.yandex.ru","IP":"127.0.0.2"},` +
			`{"FilterListID":43,"Text":"||an2.yandex.ru","IP":"127.0.0.3"}],` +
			`"CanonName":"example.com",` +
			`"ServiceName":"example.org",` +
			`"DNSRewriteResult":{"RCode":0,"Response":{"1":["127.0.0.2"]}}},` +
			`"Elapsed":837429}`

		ans, err := base64.StdEncoding.DecodeString(ansStr)
		assert.Nil(t, err)

		want := &logEntry{
			IP:          net.IPv4(127, 0, 0, 1),
			Time:        time.Date(2020, 11, 25, 15, 55, 56, 519796000, time.UTC),
			QHost:       "an.yandex.ru",
			QType:       "A",
			QClass:      "IN",
			ClientID:    "cli42",
			ClientProto: "",
			Answer:      ans,
			Result: dnsfilter.Result{
				IsFiltered:   true,
				Reason:       dnsfilter.FilteredBlockList,
				ReverseHosts: []string{"example.net"},
				IPList:       []net.IP{net.IPv4(127, 0, 0, 2)},
				Rules: []*dnsfilter.ResultRule{{
					FilterListID: 42,
					Text:         "||an.yandex.ru",
					IP:           net.IPv4(127, 0, 0, 2),
				}, {
					FilterListID: 43,
					Text:         "||an2.yandex.ru",
					IP:           net.IPv4(127, 0, 0, 3),
				}},
				CanonName:   "example.com",
				ServiceName: "example.org",
				DNSRewriteResult: &dnsfilter.DNSRewriteResult{
					RCode: dns.RcodeSuccess,
					Response: dnsfilter.DNSRewriteResultResponse{
						dns.TypeA: []rules.RRValue{net.IPv4(127, 0, 0, 2)},
					},
				},
			},
			Elapsed: 837429,
		}

		got := &logEntry{}
		decodeLogEntry(got, data)

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
		want: "decodeResult handler err: strconv.ParseInt: parsing \"1.5\": invalid syntax\n",
	}, {
		name: "bad_is_filtered",
		log:  `{"IP":"127.0.0.1","T":"2020-11-25T18:55:56.519796+03:00","QH":"an.yandex.ru","QT":"A","QC":"IN","CP":"","Answer":"Qz+BgAABAAEAAAAAAmFuBnlhbmRleAJydQAAAQABwAwAAQABAAAACgAEAAAAAA==","Result":{"IsFiltered":trooe,"Reason":3},"Elapsed":837429}`,
		want: "decodeLogEntry err: invalid character 'o' in literal true (expecting 'u')\n",
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
		want: "decodeLogEntry handler err: parsing time \"12/09/1998T15:00:00.000000+05:00\" as \"2006-01-02T15:04:05Z07:00\": cannot parse \"9/1998T15:00:00.000000+05:00\" as \"2006\"\n",
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
		want: "decodeLogEntry handler err: invalid client proto: \"dog\"\n",
	}, {
		name: "bad_answer",
		log:  `{"IP":"127.0.0.1","T":"2020-11-25T18:55:56.519796+03:00","QH":"an.yandex.ru","QT":"A","QC":"IN","CP":"","Answer":0.9,"Result":{"IsFiltered":true,"Reason":3},"Elapsed":837429}`,
		want: "",
	}, {
		name: "very_bad_answer",
		log:  `{"IP":"127.0.0.1","T":"2020-11-25T18:55:56.519796+03:00","QH":"an.yandex.ru","QT":"A","QC":"IN","CP":"","Answer":"Qz+BgAABAAEAAAAAAmuBnlhbmRleAJydQAAAQABwAwAAQABAAAACgAEAAAAAA==","Result":{"IsFiltered":true,"Reason":3},"Elapsed":837429}`,
		want: "decodeLogEntry handler err: illegal base64 data at input byte 61\n",
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
		want: "decodeResultReverseHosts: unexpected delim \"{\"\n",
	}, {
		name: "bad_ip_list",
		log:  `{"IP":"127.0.0.1","T":"2020-11-25T18:55:56.519796+03:00","QH":"an.yandex.ru","QT":"A","QC":"IN","CP":"","Answer":"Qz+BgAABAAEAAAAAAmFuBnlhbmRleAJydQAAAQABwAwAAQABAAAACgAEAAAAAA==","Result":{"IsFiltered":true,"Reason":3,"ReverseHosts":["example.net"],"IPList":[{}]},"Elapsed":837429}`,
		want: "decodeResultIPList: unexpected delim \"{\"\n",
	}}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			l := &logEntry{}
			decodeLogEntry(l, tc.log)

			s := logOutput.String()
			if tc.want == "" {
				assert.Empty(t, s)
			} else {
				assert.True(t, strings.HasSuffix(s, tc.want),
					"got %q", s)
			}

			logOutput.Reset()
		})
	}
}

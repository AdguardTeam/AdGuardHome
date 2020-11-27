package querylog

import (
	"bytes"
	"strings"
	"testing"

	"github.com/AdguardTeam/AdGuardHome/internal/testutil"
	"github.com/AdguardTeam/golibs/log"
	"github.com/stretchr/testify/assert"
)

func TestDecode_decodeQueryLog(t *testing.T) {
	logOutput := &bytes.Buffer{}

	testutil.ReplaceLogWriter(t, logOutput)
	testutil.ReplaceLogLevel(t, log.DEBUG)

	testCases := []struct {
		name string
		log  string
		want string
	}{{
		name: "back_compatibility_all_right",
		log:  `{"Question":"ULgBAAABAAAAAAAAC2FkZ3VhcmR0ZWFtBmdpdGh1YgJpbwAAHAAB","Answer":"ULiBgAABAAAAAQAAC2FkZ3VhcmR0ZWFtBmdpdGh1YgJpbwAAHAABwBgABgABAAADQgBLB25zLTE2MjIJYXdzZG5zLTEwAmNvAnVrABFhd3NkbnMtaG9zdG1hc3RlcgZhbWF6b24DY29tAAAAAAEAABwgAAADhAASdQAAAVGA","Result":{},"Time":"2020-11-13T12:41:25.970861+03:00","Elapsed":244066501,"IP":"127.0.0.1","Upstream":"https://1.1.1.1:443/dns-query"}`,
		want: "default",
	}, {
		name: "back_compatibility_bad_msg",
		log:  `{"Question":"","Answer":"ULiBgAABAAAAAQAAC2FkZ3VhcmR0ZWFtBmdpdGh1YgJpbwAAHAABwBgABgABAAADQgBLB25zLTE2MjIJYXdzZG5zLTEwAmNvAnVrABFhd3NkbnMtaG9zdG1hc3RlcgZhbWF6b24DY29tAAAAAAEAABwgAAADhAASdQAAAVGA","Result":{},"Time":"2020-11-13T12:41:25.970861+03:00","Elapsed":244066501,"IP":"127.0.0.1","Upstream":"https://1.1.1.1:443/dns-query"}`,
		want: "decodeLogEntry err: dns: overflow unpacking uint16\n",
	}, {
		name: "back_compatibility_bad_decoding",
		log:  `{"Question":"LgBAAABAAAAAAAAC2FkZ3VhcmR0ZWFtBmdpdGh1YgJpbwAAHAAB","Answer":"ULiBgAABAAAAAQAAC2FkZ3VhcmR0ZWFtBmdpdGh1YgJpbwAAHAABwBgABgABAAADQgBLB25zLTE2MjIJYXdzZG5zLTEwAmNvAnVrABFhd3NkbnMtaG9zdG1hc3RlcgZhbWF6b24DY29tAAAAAAEAABwgAAADhAASdQAAAVGA","Result":{},"Time":"2020-11-13T12:41:25.970861+03:00","Elapsed":244066501,"IP":"127.0.0.1","Upstream":"https://1.1.1.1:443/dns-query"}`,
		want: "decodeLogEntry err: illegal base64 data at input byte 48\n",
	}, {
		name: "modern_all_right",
		log:  `{"IP":"127.0.0.1","T":"2020-11-25T18:55:56.519796+03:00","QH":"an.yandex.ru","QT":"A","QC":"IN","CP":"","Answer":"Qz+BgAABAAEAAAAAAmFuBnlhbmRleAJydQAAAQABwAwAAQABAAAACgAEAAAAAA==","Result":{"IsFiltered":true,"Reason":3,"Rule":"||an.yandex.","FilterID":1},"Elapsed":837429}`,
		want: "default",
	}, {
		name: "bad_filter_id",
		log:  `{"IP":"127.0.0.1","T":"2020-11-25T18:55:56.519796+03:00","QH":"an.yandex.ru","QT":"A","QC":"IN","CP":"","Answer":"Qz+BgAABAAEAAAAAAmFuBnlhbmRleAJydQAAAQABwAwAAQABAAAACgAEAAAAAA==","Result":{"IsFiltered":true,"Reason":3,"Rule":"||an.yandex.","FilterID":1.5},"Elapsed":837429}`,
		want: "decodeLogEntry err: strconv.ParseInt: parsing \"1.5\": invalid syntax\n",
	}, {
		name: "bad_is_filtered",
		log:  `{"IP":"127.0.0.1","T":"2020-11-25T18:55:56.519796+03:00","QH":"an.yandex.ru","QT":"A","QC":"IN","CP":"","Answer":"Qz+BgAABAAEAAAAAAmFuBnlhbmRleAJydQAAAQABwAwAAQABAAAACgAEAAAAAA==","Result":{"IsFiltered":trooe,"Reason":3,"Rule":"||an.yandex.","FilterID":1},"Elapsed":837429}`,
		want: "default",
	}, {
		name: "bad_elapsed",
		log:  `{"IP":"127.0.0.1","T":"2020-11-25T18:55:56.519796+03:00","QH":"an.yandex.ru","QT":"A","QC":"IN","CP":"","Answer":"Qz+BgAABAAEAAAAAAmFuBnlhbmRleAJydQAAAQABwAwAAQABAAAACgAEAAAAAA==","Result":{"IsFiltered":true,"Reason":3,"Rule":"||an.yandex.","FilterID":1},"Elapsed":-1}`,
		want: "default",
	}, {
		name: "bad_ip",
		log:  `{"IP":127001,"T":"2020-11-25T18:55:56.519796+03:00","QH":"an.yandex.ru","QT":"A","QC":"IN","CP":"","Answer":"Qz+BgAABAAEAAAAAAmFuBnlhbmRleAJydQAAAQABwAwAAQABAAAACgAEAAAAAA==","Result":{"IsFiltered":true,"Reason":3,"Rule":"||an.yandex.","FilterID":1},"Elapsed":837429}`,
		want: "default",
	}, {
		name: "bad_time",
		log:  `{"IP":"127.0.0.1","T":"12/09/1998T15:00:00.000000+05:00","QH":"an.yandex.ru","QT":"A","QC":"IN","CP":"","Answer":"Qz+BgAABAAEAAAAAAmFuBnlhbmRleAJydQAAAQABwAwAAQABAAAACgAEAAAAAA==","Result":{"IsFiltered":true,"Reason":3,"Rule":"||an.yandex.","FilterID":1},"Elapsed":837429}`,
		want: "decodeLogEntry err: parsing time \"12/09/1998T15:00:00.000000+05:00\" as \"2006-01-02T15:04:05Z07:00\": cannot parse \"9/1998T15:00:00.000000+05:00\" as \"2006\"\n",
	}, {
		name: "bad_host",
		log:  `{"IP":"127.0.0.1","T":"2020-11-25T18:55:56.519796+03:00","QH":6,"QT":"A","QC":"IN","CP":"","Answer":"Qz+BgAABAAEAAAAAAmFuBnlhbmRleAJydQAAAQABwAwAAQABAAAACgAEAAAAAA==","Result":{"IsFiltered":true,"Reason":3,"Rule":"||an.yandex.","FilterID":1},"Elapsed":837429}`,
		want: "default",
	}, {
		name: "bad_type",
		log:  `{"IP":"127.0.0.1","T":"2020-11-25T18:55:56.519796+03:00","QH":"an.yandex.ru","QT":true,"QC":"IN","CP":"","Answer":"Qz+BgAABAAEAAAAAAmFuBnlhbmRleAJydQAAAQABwAwAAQABAAAACgAEAAAAAA==","Result":{"IsFiltered":true,"Reason":3,"Rule":"||an.yandex.","FilterID":1},"Elapsed":837429}`,
		want: "default",
	}, {
		name: "bad_class",
		log:  `{"IP":"127.0.0.1","T":"2020-11-25T18:55:56.519796+03:00","QH":"an.yandex.ru","QT":"A","QC":false,"CP":"","Answer":"Qz+BgAABAAEAAAAAAmFuBnlhbmRleAJydQAAAQABwAwAAQABAAAACgAEAAAAAA==","Result":{"IsFiltered":true,"Reason":3,"Rule":"||an.yandex.","FilterID":1},"Elapsed":837429}`,
		want: "default",
	}, {
		name: "bad_client_proto",
		log:  `{"IP":"127.0.0.1","T":"2020-11-25T18:55:56.519796+03:00","QH":"an.yandex.ru","QT":"A","QC":"IN","CP":8,"Answer":"Qz+BgAABAAEAAAAAAmFuBnlhbmRleAJydQAAAQABwAwAAQABAAAACgAEAAAAAA==","Result":{"IsFiltered":true,"Reason":3,"Rule":"||an.yandex.","FilterID":1},"Elapsed":837429}`,
		want: "default",
	}, {
		name: "very_bad_client_proto",
		log:  `{"IP":"127.0.0.1","T":"2020-11-25T18:55:56.519796+03:00","QH":"an.yandex.ru","QT":"A","QC":"IN","CP":"dog","Answer":"Qz+BgAABAAEAAAAAAmFuBnlhbmRleAJydQAAAQABwAwAAQABAAAACgAEAAAAAA==","Result":{"IsFiltered":true,"Reason":3,"Rule":"||an.yandex.","FilterID":1},"Elapsed":837429}`,
		want: "decodeLogEntry err: invalid client proto: \"dog\"\n",
	}, {
		name: "bad_answer",
		log:  `{"IP":"127.0.0.1","T":"2020-11-25T18:55:56.519796+03:00","QH":"an.yandex.ru","QT":"A","QC":"IN","CP":"","Answer":0.9,"Result":{"IsFiltered":true,"Reason":3,"Rule":"||an.yandex.","FilterID":1},"Elapsed":837429}`,
		want: "default",
	}, {
		name: "very_bad_answer",
		log:  `{"IP":"127.0.0.1","T":"2020-11-25T18:55:56.519796+03:00","QH":"an.yandex.ru","QT":"A","QC":"IN","CP":"","Answer":"Qz+BgAABAAEAAAAAAmuBnlhbmRleAJydQAAAQABwAwAAQABAAAACgAEAAAAAA==","Result":{"IsFiltered":true,"Reason":3,"Rule":"||an.yandex.","FilterID":1},"Elapsed":837429}`,
		want: "decodeLogEntry err: illegal base64 data at input byte 61\n",
	}, {
		name: "bad_rule",
		log:  `{"IP":"127.0.0.1","T":"2020-11-25T18:55:56.519796+03:00","QH":"an.yandex.ru","QT":"A","QC":"IN","CP":"","Answer":"Qz+BgAABAAEAAAAAAmFuBnlhbmRleAJydQAAAQABwAwAAQABAAAACgAEAAAAAA==","Result":{"IsFiltered":true,"Reason":3,"Rule":false,"FilterID":1},"Elapsed":837429}`,
		want: "default",
	}, {
		name: "bad_reason",
		log:  `{"IP":"127.0.0.1","T":"2020-11-25T18:55:56.519796+03:00","QH":"an.yandex.ru","QT":"A","QC":"IN","CP":"","Answer":"Qz+BgAABAAEAAAAAAmFuBnlhbmRleAJydQAAAQABwAwAAQABAAAACgAEAAAAAA==","Result":{"IsFiltered":true,"Reason":true,"Rule":"||an.yandex.","FilterID":1},"Elapsed":837429}`,
		want: "default",
	}}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := logOutput.Write([]byte("default"))
			assert.Nil(t, err)

			l := &logEntry{}
			decodeLogEntry(l, tc.log)

			assert.True(t, strings.HasSuffix(logOutput.String(), tc.want), logOutput.String())

			logOutput.Reset()
		})
	}
}

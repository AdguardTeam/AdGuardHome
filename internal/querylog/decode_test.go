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

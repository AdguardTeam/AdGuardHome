package aghtls_test

import (
	"crypto/tls"
	"testing"

	"github.com/AdguardTeam/AdGuardHome/internal/aghtls"
	"github.com/AdguardTeam/golibs/testutil"
	"github.com/stretchr/testify/assert"
)

func TestMain(m *testing.M) {
	testutil.DiscardLogOutput(m)
}

func TestParseCiphers(t *testing.T) {
	testCases := []struct {
		name       string
		wantErrMsg string
		want       []uint16
		in         []string
	}{{
		name:       "nil",
		wantErrMsg: "",
		want:       nil,
		in:         nil,
	}, {
		name:       "empty",
		wantErrMsg: "",
		want:       []uint16{},
		in:         []string{},
	}, {}, {
		name:       "one",
		wantErrMsg: "",
		want:       []uint16{tls.TLS_AES_128_GCM_SHA256},
		in:         []string{"TLS_AES_128_GCM_SHA256"},
	}, {
		name:       "several",
		wantErrMsg: "",
		want:       []uint16{tls.TLS_AES_128_GCM_SHA256, tls.TLS_AES_256_GCM_SHA384},
		in:         []string{"TLS_AES_128_GCM_SHA256", "TLS_AES_256_GCM_SHA384"},
	}, {
		name:       "bad",
		wantErrMsg: `unknown cipher "bad_cipher"`,
		want:       nil,
		in:         []string{"bad_cipher"},
	}}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := aghtls.ParseCiphers(tc.in)
			testutil.AssertErrorMsg(t, tc.wantErrMsg, err)
			assert.Equal(t, tc.want, got)
		})
	}
}

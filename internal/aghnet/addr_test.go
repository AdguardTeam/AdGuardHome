package aghnet

import (
	"net"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestValidateHardwareAddress(t *testing.T) {
	testCases := []struct {
		name       string
		wantErrMsg string
		in         net.HardwareAddr
	}{{
		name:       "success_eui_48",
		wantErrMsg: "",
		in:         net.HardwareAddr{0x00, 0x01, 0x02, 0x03, 0x04, 0x05},
	}, {
		name:       "success_eui_64",
		wantErrMsg: "",
		in:         net.HardwareAddr{0x00, 0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07},
	}, {
		name:       "success_infiniband",
		wantErrMsg: "",
		in: net.HardwareAddr{
			0x00, 0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07,
			0x08, 0x09, 0x0a, 0x0b, 0x0c, 0x0d, 0x0e, 0x0f,
			0x10, 0x11, 0x12, 0x13,
		},
	}, {
		name:       "error_nil",
		wantErrMsg: `validating hardware address "": address is empty`,
		in:         nil,
	}, {
		name:       "error_empty",
		wantErrMsg: `validating hardware address "": address is empty`,
		in:         net.HardwareAddr{},
	}, {
		name:       "error_bad",
		wantErrMsg: `validating hardware address "00:01:02:03": bad len: 4`,
		in:         net.HardwareAddr{0x00, 0x01, 0x02, 0x03},
	}}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			err := ValidateHardwareAddress(tc.in)
			if tc.wantErrMsg == "" {
				assert.NoError(t, err)
			} else {
				require.Error(t, err)

				assert.Equal(t, tc.wantErrMsg, err.Error())
			}
		})
	}
}

func repeatStr(b *strings.Builder, s string, n int) {
	for i := 0; i < n; i++ {
		_, _ = b.WriteString(s)
	}
}

func TestValidateDomainName(t *testing.T) {
	b := &strings.Builder{}
	repeatStr(b, "a", 255)
	longDomainName := b.String()

	b.Reset()
	repeatStr(b, "a", 64)
	longLabel := b.String()

	_, _ = b.WriteString(".com")
	longLabelDomainName := b.String()

	testCases := []struct {
		name       string
		in         string
		wantErrMsg string
	}{{
		name:       "success",
		in:         "example.com",
		wantErrMsg: "",
	}, {
		name:       "success_idna",
		in:         "пример.рф",
		wantErrMsg: "",
	}, {
		name:       "success_one",
		in:         "e",
		wantErrMsg: "",
	}, {
		name:       "empty",
		in:         "",
		wantErrMsg: `validating domain name "": domain name is empty`,
	}, {
		name: "bad_symbol",
		in:   "!!!",
		wantErrMsg: `validating domain name "!!!": invalid domain name label at index 0: ` +
			`validating label "!!!": invalid char '!' at index 0`,
	}, {
		name:       "bad_length",
		in:         longDomainName,
		wantErrMsg: `validating domain name "` + longDomainName + `": too long, max: 253`,
	}, {
		name: "bad_label_length",
		in:   longLabelDomainName,
		wantErrMsg: `validating domain name "` + longLabelDomainName + `": ` +
			`invalid domain name label at index 0: validating label "` + longLabel +
			`": label is too long, max: 63`,
	}, {
		name: "bad_label_empty",
		in:   "example..com",
		wantErrMsg: `validating domain name "example..com": ` +
			`invalid domain name label at index 1: ` +
			`validating label "": label is empty`,
	}, {
		name: "bad_label_first_symbol",
		in:   "example.-aa.com",
		wantErrMsg: `validating domain name "example.-aa.com": ` +
			`invalid domain name label at index 1: ` +
			`validating label "-aa": invalid char '-' at index 0`,
	}, {
		name: "bad_label_last_symbol",
		in:   "example-.aa.com",
		wantErrMsg: `validating domain name "example-.aa.com": ` +
			`invalid domain name label at index 0: ` +
			`validating label "example-": invalid char '-' at index 7`,
	}, {
		name: "bad_label_symbol",
		in:   "example.a!!!.com",
		wantErrMsg: `validating domain name "example.a!!!.com": ` +
			`invalid domain name label at index 1: ` +
			`validating label "a!!!": invalid char '!' at index 1`,
	}}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			err := ValidateDomainName(tc.in)
			if tc.wantErrMsg == "" {
				assert.NoError(t, err)
			} else {
				require.Error(t, err)

				assert.Equal(t, tc.wantErrMsg, err.Error())
			}
		})
	}
}

func TestGenerateHostName(t *testing.T) {
	testCases := []struct {
		name string
		want string
		ip   net.IP
	}{{
		name: "good_ipv4",
		want: "127-0-0-1",
		ip:   net.IP{127, 0, 0, 1},
	}, {
		name: "bad_ipv4",
		want: "",
		ip:   net.IP{127, 0, 0, 1, 0},
	}, {
		name: "good_ipv6",
		want: "fe00-0000-0000-0000-0000-0000-0000-0001",
		ip:   net.ParseIP("fe00::1"),
	}, {
		name: "bad_ipv6",
		want: "",
		ip: net.IP{
			0xff, 0xff, 0xff, 0xff,
			0xff, 0xff, 0xff, 0xff,
			0xff, 0xff, 0xff, 0xff,
			0xff, 0xff, 0xff,
		},
	}, {
		name: "nil",
		want: "",
		ip:   nil,
	}}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			hostname := GenerateHostname(tc.ip)
			assert.Equal(t, tc.want, hostname)
		})
	}
}

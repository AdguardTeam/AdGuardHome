package dhcpd

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDHCPOptionParser(t *testing.T) {
	testCases := []struct {
		name       string
		in         string
		wantErrMsg string
		wantData   []byte
		wantCode   uint8
	}{{
		name:       "hex_success",
		in:         "6 hex c0a80101c0a80102",
		wantErrMsg: "",
		wantData:   []byte{0xC0, 0xA8, 0x01, 0x01, 0xC0, 0xA8, 0x01, 0x02},
		wantCode:   6,
	}, {
		name:       "ip_success",
		in:         "6 ip 1.2.3.4",
		wantErrMsg: "",
		wantData:   []byte{0x01, 0x02, 0x03, 0x04},
		wantCode:   6,
	}, {
		name:       "ip_success_v6",
		in:         "6 ip ::1234",
		wantErrMsg: "",
		wantData: []byte{
			0x00, 0x00, 0x00, 0x00,
			0x00, 0x00, 0x00, 0x00,
			0x00, 0x00, 0x00, 0x00,
			0x00, 0x00, 0x12, 0x34,
		},
		wantCode: 6,
	}, {
		name:       "ips_success",
		in:         "6 ips 192.168.1.1,192.168.1.2",
		wantErrMsg: "",
		wantData:   []byte{0xC0, 0xA8, 0x01, 0x01, 0xC0, 0xA8, 0x01, 0x02},
		wantCode:   6,
	}, {
		name:       "text_success",
		in:         "252 text http://192.168.1.1/",
		wantErrMsg: "",
		wantData:   []byte("http://192.168.1.1/"),
		wantCode:   252,
	}, {
		name:       "bad_parts",
		in:         "6 ip",
		wantErrMsg: `invalid option string "6 ip": need at least three fields`,
		wantCode:   0,
		wantData:   nil,
	}, {
		name: "bad_code",
		in:   "256 ip 1.1.1.1",
		wantErrMsg: `invalid option string "256 ip 1.1.1.1": parsing option code: ` +
			`strconv.ParseUint: parsing "256": value out of range`,
		wantCode: 0,
		wantData: nil,
	}, {
		name:       "bad_type",
		in:         "6 bad 1.1.1.1",
		wantErrMsg: `invalid option string "6 bad 1.1.1.1": unknown option type "bad"`,
		wantCode:   0,
		wantData:   nil,
	}, {
		name: "hex_error",
		in:   "6 hex ZZZ",
		wantErrMsg: `invalid option string "6 hex ZZZ": decoding hex: ` +
			`encoding/hex: invalid byte: U+005A 'Z'`,
		wantData: nil,
		wantCode: 0,
	}, {
		name:       "ip_error",
		in:         "6 ip 1.2.3.x",
		wantErrMsg: `invalid option string "6 ip 1.2.3.x": invalid ip`,
		wantData:   nil,
		wantCode:   0,
	}, {
		name: "ips_error",
		in:   "6 ips 192.168.1.1,192.168.1.x",
		wantErrMsg: `invalid option string "6 ips 192.168.1.1,192.168.1.x": ` +
			`parsing ip at index 1: invalid ip`,
		wantData: nil,
		wantCode: 0,
	}}

	p := newDHCPOptionParser()

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			code, data, err := p.parse(tc.in)
			if tc.wantErrMsg == "" {
				assert.Nil(t, err)
			} else {
				require.NotNil(t, err)
				assert.Equal(t, tc.wantErrMsg, err.Error())
			}

			assert.Equal(t, tc.wantCode, code)
			assert.Equal(t, tc.wantData, data)
		})
	}
}

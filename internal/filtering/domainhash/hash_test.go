package domainhash_test

import (
	"strings"
	"testing"

	"github.com/AdguardTeam/AdGuardHome/internal/filtering/domainhash"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// truncMask is the test-side counterpart of the mask of the lowest n bytes.
func truncMask(n int) (m uint64) {
	return ^uint64(0) >> (64 - 8*n)
}

func TestHashFNV(t *testing.T) {
	testCases := []struct {
		name string
		in   string
		want uint64
	}{{
		name: "empty",
		in:   "",
		want: 0xcbf29ce484222325,
	}, {
		name: "a",
		in:   "a",
		want: 0xaf63dc4c8601ec8c,
	}, {
		name: "foobar",
		in:   "foobar",
		want: 0x85944171f73967e8,
	}}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.want, domainhash.Hash(tc.in, domainhash.MaxHashBytes), "hash")
		})
	}
}

func TestHashTruncation(t *testing.T) {
	const in = "ads.tracker.example.com"

	full := domainhash.Hash(in, domainhash.MaxHashBytes)
	for n := domainhash.MinHashBytes; n <= domainhash.MaxHashBytes; n++ {
		assert.Equal(t, full&truncMask(n), domainhash.Hash(in, n), "hash of %d bytes", n)
	}

	// Zero means the default number of bytes.
	assert.Equal(
		t,
		domainhash.Hash(in, domainhash.DefaultHashBytes),
		domainhash.Hash(in, 0),
		"hash",
	)
}

func TestHashDomain(t *testing.T) {
	// The domain is normalized before hashing.
	want, ok := domainhash.HashDomain("example.com", domainhash.DefaultHashBytes)
	require.True(t, ok, "ok")

	got, ok := domainhash.HashDomain("WWW.Example.COM.", domainhash.DefaultHashBytes)
	require.True(t, ok, "ok")
	assert.Equal(t, want, got, "hash")

	_, ok = domainhash.HashDomain("bad..example.com", domainhash.DefaultHashBytes)
	assert.False(t, ok, "ok")

	_, ok = domainhash.HashDomain("münchen.example", domainhash.DefaultHashBytes)
	assert.False(t, ok, "ok")
}

func TestNormalizeDomain(t *testing.T) {
	testCases := []struct {
		name string
		in   string
		want string
		ok   bool
	}{{
		name: "simple",
		in:   "example.com",
		want: "example.com",
		ok:   true,
	}, {
		name: "uppercase",
		in:   "Example.COM",
		want: "example.com",
		ok:   true,
	}, {
		name: "trailing_dot",
		in:   "example.com.",
		want: "example.com",
		ok:   true,
	}, {
		name: "www_and_dot",
		in:   "WWW.Example.COM.",
		want: "example.com",
		ok:   true,
	}, {
		name: "www_two_labels",
		in:   "www.example.com",
		want: "example.com",
		ok:   true,
	}, {
		name: "www_single_label_left",
		in:   "www.com",
		want: "www.com",
		ok:   true,
	}, {
		name: "spaces",
		in:   "  example.com \n",
		want: "example.com",
		ok:   true,
	}, {
		name: "underscore",
		in:   "_dmarc.example.com",
		want: "_dmarc.example.com",
		ok:   true,
	}, {
		name: "punycode",
		in:   "xn--mnchen-3ya.example",
		want: "xn--mnchen-3ya.example",
		ok:   true,
	}, {
		name: "empty",
		in:   "",
		ok:   false,
	}, {
		name: "dot_only",
		in:   ".",
		ok:   false,
	}, {
		name: "empty_label",
		in:   "example..com",
		ok:   false,
	}, {
		name: "leading_dash",
		in:   "-bad.example",
		ok:   false,
	}, {
		name: "trailing_dash",
		in:   "bad-.example",
		ok:   false,
	}, {
		name: "space_inside",
		in:   "exa mple.com",
		ok:   false,
	}, {
		name: "non_ascii",
		in:   "münchen.example",
		ok:   false,
	}, {
		name: "too_long",
		in:   strings.Repeat("a", 254),
		ok:   false,
	}, {
		name: "label_too_long",
		in:   strings.Repeat("a", 64) + ".example",
		ok:   false,
	}}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			got, ok := domainhash.NormalizeDomain(tc.in)
			assert.Equal(t, tc.ok, ok, "ok")
			if tc.ok {
				assert.Equal(t, tc.want, got, "norm")
			} else {
				assert.Empty(t, got, "norm")
			}
		})
	}
}

func TestPutHashGetHash(t *testing.T) {
	const h = 0x01_02_03_04_05_06_07_08

	var buf [domainhash.MaxHashBytes]byte
	for n := domainhash.MinHashBytes; n <= domainhash.MaxHashBytes; n++ {
		domainhash.PutHash(buf[:], h, n)

		assert.Equal(t, h&truncMask(n), domainhash.GetHash(buf[:n]), "hash of %d bytes", n)

		// Only the low n bytes are written.
		for _, c := range buf[n:] {
			require.Zero(t, c, "byte")
		}
	}

	// The bytes are little-endian.
	domainhash.PutHash(buf[:], h, domainhash.MaxHashBytes)
	assert.Equal(t, []byte{0x08, 0x07, 0x06, 0x05, 0x04, 0x03, 0x02, 0x01}, buf[:], "bytes")

	// Zero is clamped to the default number of bytes.
	domainhash.PutHash(buf[:], h, 0)
	assert.Equal(
		t,
		h&truncMask(domainhash.DefaultHashBytes),
		domainhash.GetHash(buf[:domainhash.DefaultHashBytes]),
		"hash",
	)
}

func TestClampHashBytes(t *testing.T) {
	testCases := []struct {
		in   int
		want int
	}{{
		in:   0,
		want: domainhash.DefaultHashBytes,
	}, {
		in:   -1,
		want: domainhash.DefaultHashBytes,
	}, {
		in:   1,
		want: domainhash.MinHashBytes,
	}, {
		in:   domainhash.MinHashBytes,
		want: domainhash.MinHashBytes,
	}, {
		in:   domainhash.DefaultHashBytes,
		want: domainhash.DefaultHashBytes,
	}, {
		in:   6,
		want: 6,
	}, {
		in:   9,
		want: domainhash.MaxHashBytes,
	}}

	for _, tc := range testCases {
		assert.Equal(t, tc.want, domainhash.ClampHashBytes(tc.in), "for %d", tc.in)
	}
}

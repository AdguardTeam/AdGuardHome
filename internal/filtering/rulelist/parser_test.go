package rulelist_test

import (
	"bufio"
	"bytes"
	"strings"
	"testing"

	"github.com/AdguardTeam/AdGuardHome/internal/filtering/rulelist"
	"github.com/AdguardTeam/golibs/errors"
	"github.com/AdguardTeam/golibs/testutil"
	"github.com/AdguardTeam/golibs/testutil/fakeio"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParser_Parse(t *testing.T) {
	t.Parallel()

	longRule := strings.Repeat("a", rulelist.DefaultRuleBufSize+1) + "\n"
	tooLongRule := strings.Repeat("a", bufio.MaxScanTokenSize+1) + "\n"

	testCases := []struct {
		name         string
		in           string
		wantDst      string
		wantErrMsg   string
		wantTitle    string
		wantRulesNum int
		wantWritten  int
	}{{
		name:         "empty",
		in:           "",
		wantDst:      "",
		wantErrMsg:   "",
		wantTitle:    "",
		wantRulesNum: 0,
		wantWritten:  0,
	}, {
		name:         "html",
		in:           testRuleTextHTML,
		wantErrMsg:   rulelist.ErrHTML.Error(),
		wantTitle:    "",
		wantRulesNum: 0,
		wantWritten:  0,
	}, {
		name: "comments",
		in: "# Comment 1\n" +
			"! Comment 2\n",
		wantErrMsg:   "",
		wantTitle:    "",
		wantRulesNum: 0,
		wantWritten:  0,
	}, {}, {
		name:         "rule",
		in:           testRuleTextBlocked,
		wantDst:      testRuleTextBlocked,
		wantErrMsg:   "",
		wantRulesNum: 1,
		wantTitle:    "",
		wantWritten:  len(testRuleTextBlocked),
	}, {
		name:         "html_in_rule",
		in:           testRuleTextBlocked + testRuleTextHTML,
		wantDst:      testRuleTextBlocked + testRuleTextHTML,
		wantErrMsg:   "",
		wantTitle:    "",
		wantRulesNum: 2,
		wantWritten:  len(testRuleTextBlocked) + len(testRuleTextHTML),
	}, {
		name: "title",
		in: testRuleTextTitle +
			"! Title: Bad, Ignored Title\n" +
			testRuleTextBlocked,
		wantDst:      testRuleTextBlocked,
		wantErrMsg:   "",
		wantTitle:    testTitle,
		wantRulesNum: 1,
		wantWritten:  len(testRuleTextBlocked),
	}, {
		name:         "cosmetic_with_zwnj",
		in:           testRuleTextCosmetic,
		wantDst:      testRuleTextCosmetic,
		wantErrMsg:   "",
		wantTitle:    "",
		wantRulesNum: 1,
		wantWritten:  len(testRuleTextCosmetic),
	}, {
		name: "bad_char",
		in: testRuleTextTitle +
			testRuleTextBlocked +
			">>>\x7F<<<",
		wantDst: testRuleTextBlocked,
		wantErrMsg: "line 3: " +
			"character 4: " +
			"likely binary character '\\x7f'",
		wantTitle:    testTitle,
		wantRulesNum: 1,
		wantWritten:  len(testRuleTextBlocked),
	}, {
		name:         "too_long",
		in:           tooLongRule,
		wantDst:      "",
		wantErrMsg:   "scanning filter contents: bufio.Scanner: token too long",
		wantTitle:    "",
		wantRulesNum: 0,
		wantWritten:  0,
	}, {
		name:         "longer_than_default",
		in:           longRule,
		wantDst:      longRule,
		wantErrMsg:   "",
		wantTitle:    "",
		wantRulesNum: 1,
		wantWritten:  len(longRule),
	}, {
		name:         "bad_tab_and_comment",
		in:           testRuleTextBadTab,
		wantDst:      testRuleTextBadTab,
		wantErrMsg:   "",
		wantTitle:    "",
		wantRulesNum: 1,
		wantWritten:  len(testRuleTextBadTab),
	}, {
		name:         "etc_hosts_tab_and_comment",
		in:           testRuleTextEtcHostsTab,
		wantDst:      testRuleTextEtcHostsTab,
		wantErrMsg:   "",
		wantTitle:    "",
		wantRulesNum: 1,
		wantWritten:  len(testRuleTextEtcHostsTab),
	}}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			dst := &bytes.Buffer{}
			buf := make([]byte, rulelist.DefaultRuleBufSize)

			p := rulelist.NewParser()
			r, err := p.Parse(dst, strings.NewReader(tc.in), buf)
			require.NotNil(t, r)

			testutil.AssertErrorMsg(t, tc.wantErrMsg, err)
			assert.Equal(t, tc.wantDst, dst.String())
			assert.Equal(t, tc.wantTitle, r.Title)
			assert.Equal(t, tc.wantRulesNum, r.RulesCount)
			assert.Equal(t, tc.wantWritten, r.BytesWritten)

			if tc.wantWritten > 0 {
				assert.NotZero(t, r.Checksum)
			}
		})
	}
}

func TestParser_Parse_writeError(t *testing.T) {
	t.Parallel()

	dst := &fakeio.Writer{
		OnWrite: func(b []byte) (n int, err error) {
			return 1, errors.Error("test error")
		},
	}
	buf := make([]byte, rulelist.DefaultRuleBufSize)

	p := rulelist.NewParser()
	r, err := p.Parse(dst, strings.NewReader(testRuleTextBlocked), buf)
	require.NotNil(t, r)

	testutil.AssertErrorMsg(t, "writing rule line: test error", err)
	assert.Equal(t, 1, r.BytesWritten)
}

func TestParser_Parse_checksums(t *testing.T) {
	t.Parallel()

	const (
		withoutComments = testRuleTextBlocked
		withComments    = "! Some comment.\n" +
			"  " + testRuleTextBlocked +
			"# Another comment.\n"
	)

	buf := make([]byte, rulelist.DefaultRuleBufSize)

	p := rulelist.NewParser()
	r, err := p.Parse(&bytes.Buffer{}, strings.NewReader(withoutComments), buf)
	require.NotNil(t, r)
	require.NoError(t, err)

	gotWithoutComments := r.Checksum

	p = rulelist.NewParser()

	r, err = p.Parse(&bytes.Buffer{}, strings.NewReader(withComments), buf)
	require.NotNil(t, r)
	require.NoError(t, err)

	gotWithComments := r.Checksum
	assert.Equal(t, gotWithoutComments, gotWithComments)
}

var (
	resSink *rulelist.ParseResult
	errSink error
)

func BenchmarkParser_Parse(b *testing.B) {
	dst := &bytes.Buffer{}
	src := strings.NewReader(strings.Repeat(testRuleTextBlocked, 1000))
	buf := make([]byte, rulelist.DefaultRuleBufSize)
	p := rulelist.NewParser()

	b.ReportAllocs()
	b.ResetTimer()
	for range b.N {
		resSink, errSink = p.Parse(dst, src, buf)
		dst.Reset()
	}

	require.NoError(b, errSink)
	require.NotNil(b, resSink)

	// Most recent result, on a ThinkPad X13 with a Ryzen Pro 7 CPU:
	//
	//	goos: linux
	//	goarch: amd64
	//	pkg: github.com/AdguardTeam/AdGuardHome/internal/filtering/rulelist
	//	cpu: AMD Ryzen 7 PRO 4750U with Radeon Graphics
	//	BenchmarkParser_Parse-16        100000000              128.0 ns/op            48 B/op          1 allocs/op
}

func FuzzParser_Parse(f *testing.F) {
	const n = 64

	testCases := []string{
		"",
		"# Comment",
		"! Comment",
		"! Title ",
		"! Title XXX",
		testRuleTextBadTab,
		testRuleTextBlocked,
		testRuleTextCosmetic,
		testRuleTextEtcHostsTab,
		testRuleTextHTML,
		"1.2.3.4",
		"1.2.3.4 etc-hosts.example",
		">>>\x00<<<",
		">>>\x7F<<<",
		strings.Repeat("a", rulelist.DefaultRuleBufSize+1),
		strings.Repeat("a", bufio.MaxScanTokenSize+1),
	}

	for _, tc := range testCases {
		f.Add(tc)
	}

	buf := make([]byte, n)

	f.Fuzz(func(t *testing.T, input string) {
		require.Eventually(t, func() (ok bool) {
			dst := &bytes.Buffer{}
			src := strings.NewReader(input)

			p := rulelist.NewParser()
			r, _ := p.Parse(dst, src, buf)
			require.NotNil(t, r)

			return true
		}, testTimeout, testTimeout/100)
	})
}

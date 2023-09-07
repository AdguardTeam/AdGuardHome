package rewrite

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestItem_equal(t *testing.T) {
	const (
		testDomain = "example.org"
		testAnswer = "1.1.1.1"
	)

	testItem := &Item{
		Domain: testDomain,
		Answer: testAnswer,
	}

	testCases := []struct {
		left  *Item
		right *Item
		name  string
		want  bool
	}{{
		name:  "nil_left",
		left:  nil,
		right: testItem,
		want:  false,
	}, {
		name:  "nil_right",
		left:  testItem,
		right: nil,
		want:  false,
	}, {
		name:  "nils",
		left:  nil,
		right: nil,
		want:  true,
	}, {
		name:  "equal",
		left:  testItem,
		right: testItem,
		want:  true,
	}, {
		name: "distinct",
		left: testItem,
		right: &Item{
			Domain: "other",
			Answer: "other",
		},
		want: false,
	}}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			res := tc.left.equal(tc.right)
			assert.Equal(t, tc.want, res)
		})
	}
}

func TestItem_toRule(t *testing.T) {
	const testDomain = "example.org"

	testCases := []struct {
		name string
		item *Item
		want string
	}{{
		name: "nil",
		item: nil,
		want: "",
	}, {
		name: "a_rule",
		item: &Item{
			Domain: testDomain,
			Answer: "1.1.1.1",
		},
		want: "|example.org^$dnsrewrite=NOERROR;A;1.1.1.1",
	}, {
		name: "aaaa_rule",
		item: &Item{
			Domain: testDomain,
			Answer: "1:2:3::4",
		},
		want: "|example.org^$dnsrewrite=NOERROR;AAAA;1:2:3::4",
	}, {
		name: "cname_rule",
		item: &Item{
			Domain: testDomain,
			Answer: "other.org",
		},
		want: "|example.org^$dnsrewrite=NOERROR;CNAME;other.org",
	}, {
		name: "wildcard_rule",
		item: &Item{
			Domain: "*.example.org",
			Answer: "other.org",
		},
		want: "|*.example.org^$dnsrewrite=NOERROR;CNAME;other.org",
	}, {
		name: "aaaa_exception",
		item: &Item{
			Domain: testDomain,
			Answer: "A",
		},
		want: "@@||example.org^$dnstype=A,dnsrewrite",
	}, {
		name: "aaaa_exception",
		item: &Item{
			Domain: testDomain,
			Answer: "AAAA",
		},
		want: "@@||example.org^$dnstype=AAAA,dnsrewrite",
	}}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			res := tc.item.toRule()
			assert.Equal(t, tc.want, res)
		})
	}
}

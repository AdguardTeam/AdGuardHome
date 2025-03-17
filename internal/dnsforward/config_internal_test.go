package dnsforward

import (
	"slices"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestAnyNameMatches(t *testing.T) {
	dnsNames := []string{"host1", "*.host2", "1.2.3.4"}
	slices.Sort(dnsNames)

	testCases := []struct {
		name    string
		dnsName string
		want    bool
	}{{
		name:    "match",
		dnsName: "host1",
		want:    true,
	}, {
		name:    "match",
		dnsName: "a.host2",
		want:    true,
	}, {
		name:    "match",
		dnsName: "b.a.host2",
		want:    true,
	}, {
		name:    "match",
		dnsName: "1.2.3.4",
		want:    true,
	}, {
		name:    "mismatch_bad_ip",
		dnsName: "1.2.3.256",
		want:    false,
	}, {
		name:    "mismatch",
		dnsName: "host2",
		want:    false,
	}, {
		name:    "mismatch",
		dnsName: "",
		want:    false,
	}, {
		name:    "mismatch",
		dnsName: "*.host2",
		want:    false,
	}}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.want, anyNameMatches(dnsNames, tc.dnsName))
		})
	}
}

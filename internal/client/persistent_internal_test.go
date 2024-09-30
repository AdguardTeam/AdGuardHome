package client

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPersistent_EqualIDs(t *testing.T) {
	const (
		ip  = "0.0.0.0"
		ip1 = "1.1.1.1"
		ip2 = "2.2.2.2"

		cidr  = "0.0.0.0/0"
		cidr1 = "1.1.1.1/11"
		cidr2 = "2.2.2.2/22"

		mac  = "00-00-00-00-00-00"
		mac1 = "11-11-11-11-11-11"
		mac2 = "22-22-22-22-22-22"

		cli  = "client0"
		cli1 = "client1"
		cli2 = "client2"
	)

	testCases := []struct {
		want    assert.BoolAssertionFunc
		name    string
		ids     []string
		prevIDs []string
	}{{
		name:    "single_ip",
		ids:     []string{ip1},
		prevIDs: []string{ip1},
		want:    assert.True,
	}, {
		name:    "single_ip_not_equal",
		ids:     []string{ip1},
		prevIDs: []string{ip2},
		want:    assert.False,
	}, {
		name:    "ips_not_equal",
		ids:     []string{ip1, ip2},
		prevIDs: []string{ip1, ip},
		want:    assert.False,
	}, {
		name:    "ips_mixed_equal",
		ids:     []string{ip1, ip2},
		prevIDs: []string{ip2, ip1},
		want:    assert.True,
	}, {
		name:    "single_subnet",
		ids:     []string{cidr1},
		prevIDs: []string{cidr1},
		want:    assert.True,
	}, {
		name:    "subnets_not_equal",
		ids:     []string{ip1, ip2, cidr1, cidr2},
		prevIDs: []string{ip1, ip2, cidr1, cidr},
		want:    assert.False,
	}, {
		name:    "subnets_mixed_equal",
		ids:     []string{ip1, ip2, cidr1, cidr2},
		prevIDs: []string{cidr2, cidr1, ip2, ip1},
		want:    assert.True,
	}, {
		name:    "single_mac",
		ids:     []string{mac1},
		prevIDs: []string{mac1},
		want:    assert.True,
	}, {
		name:    "single_mac_not_equal",
		ids:     []string{mac1},
		prevIDs: []string{mac2},
		want:    assert.False,
	}, {
		name:    "macs_not_equal",
		ids:     []string{ip1, ip2, cidr1, cidr2, mac1, mac2},
		prevIDs: []string{ip1, ip2, cidr1, cidr2, mac1, mac},
		want:    assert.False,
	}, {
		name:    "macs_mixed_equal",
		ids:     []string{ip1, ip2, cidr1, cidr2, mac1, mac2},
		prevIDs: []string{mac2, mac1, cidr2, cidr1, ip2, ip1},
		want:    assert.True,
	}, {
		name:    "single_client_id",
		ids:     []string{cli1},
		prevIDs: []string{cli1},
		want:    assert.True,
	}, {
		name:    "single_client_id_not_equal",
		ids:     []string{cli1},
		prevIDs: []string{cli2},
		want:    assert.False,
	}, {
		name:    "client_ids_not_equal",
		ids:     []string{ip1, ip2, cidr1, cidr2, mac1, mac2, cli1, cli2},
		prevIDs: []string{ip1, ip2, cidr1, cidr2, mac1, mac2, cli1, cli},
		want:    assert.False,
	}, {
		name:    "client_ids_mixed_equal",
		ids:     []string{ip1, ip2, cidr1, cidr2, mac1, mac2, cli1, cli2},
		prevIDs: []string{cli2, cli1, mac2, mac1, cidr2, cidr1, ip2, ip1},
		want:    assert.True,
	}}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			c := &Persistent{}
			err := c.SetIDs(tc.ids)
			require.NoError(t, err)

			prev := &Persistent{}
			err = prev.SetIDs(tc.prevIDs)
			require.NoError(t, err)

			tc.want(t, c.EqualIDs(prev))
		})
	}
}

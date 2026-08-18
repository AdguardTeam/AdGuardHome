package home

import (
	"encoding/json"
	"testing"

	"github.com/AdguardTeam/AdGuardHome/internal/client"
	"github.com/AdguardTeam/AdGuardHome/internal/filtering"
	"github.com/AdguardTeam/AdGuardHome/internal/schedule"
	"github.com/AdguardTeam/golibs/testutil"
	"github.com/AdguardTeam/urlfilter/rules"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	yaml "go.yaml.in/yaml/v4"
)

// TestClientObject_toPersistent_filterLists makes sure that a client stored
// before per-client filter lists were introduced keeps using the global ones,
// since a client with its own but empty lists matches nothing but the user
// rules.
func TestClientObject_toPersistent_filterLists(t *testing.T) {
	testCases := []struct {
		name      string
		conf      string
		wantBlock []rules.ListID
		wantAllow []rules.ListID
		wantOwn   bool
	}{{
		name: "legacy_no_fields",
		conf: `name: legacy
ids: [192.0.2.1]
use_global_settings: true
use_global_blocked_services: true
`,
		wantOwn:   false,
		wantBlock: nil,
		wantAllow: nil,
	}, {
		name: "own_lists",
		conf: `name: own
ids: [192.0.2.2]
use_own_filter_lists: true
filter_list_ids: [1, 3]
allow_filter_list_ids: [5]
`,
		wantOwn:   true,
		wantBlock: []rules.ListID{1, 3},
		wantAllow: []rules.ListID{5},
	}, {
		name: "own_lists_none_selected",
		conf: `name: strict
ids: [192.0.2.3]
use_own_filter_lists: true
`,
		wantOwn:   true,
		wantBlock: nil,
		wantAllow: nil,
	}}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			o := &clientObject{}
			require.NoError(t, yaml.Unmarshal([]byte(tc.conf), o))

			ctx := testutil.ContextWithTimeout(t, testTimeout)
			cli, err := o.toPersistent(ctx, testLogger, 0, 0)
			require.NoError(t, err)

			assert.Equal(t, tc.wantOwn, cli.UseOwnFilterLists)
			assert.Equal(t, tc.wantBlock, cli.FilterListIDs)
			assert.Equal(t, tc.wantAllow, cli.AllowFilterListIDs)
		})
	}
}

// TestClientsContainer_jsonToClient_filterLists makes sure that an add or update
// request that predates per-client filter lists keeps the global ones.
func TestClientsContainer_jsonToClient_filterLists(t *testing.T) {
	clients := newClientsContainer(t)

	testCases := []struct {
		name    string
		body    string
		wantOwn bool
	}{{
		name:    "legacy_no_field",
		body:    `{"name":"legacy","ids":["192.0.2.1"],"use_global_settings":true}`,
		wantOwn: false,
	}, {
		name:    "own_lists",
		body:    `{"name":"own","ids":["192.0.2.2"],"use_own_filter_lists":true,"filter_list_ids":[2]}`,
		wantOwn: true,
	}}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			cj := clientJSON{}
			require.NoError(t, json.Unmarshal([]byte(tc.body), &cj))

			ctx := testutil.ContextWithTimeout(t, testTimeout)
			c, err := clients.jsonToClient(ctx, cj, nil)
			require.NoError(t, err)

			assert.Equal(t, tc.wantOwn, c.UseOwnFilterLists)
		})
	}
}

// TestClientsContainer_jsonToClient_preservesFilterLists makes sure that an
// update request that omits the filter list fields keeps the ones already
// configured, so that editing an unrelated property through an older client
// doesn't silently move the client back onto the global lists.
func TestClientsContainer_jsonToClient_preservesFilterLists(t *testing.T) {
	clients := newClientsContainer(t)

	prev := &client.Persistent{
		Name:               "stored",
		UID:                client.MustNewUID(),
		BlockedServices:    &filtering.BlockedServices{Schedule: schedule.EmptyWeekly()},
		UseOwnFilterLists:  true,
		FilterListIDs:      []rules.ListID{1, 3},
		AllowFilterListIDs: []rules.ListID{10},
	}

	testCases := []struct {
		name      string
		body      string
		wantBlock []rules.ListID
		wantAllow []rules.ListID
		wantOwn   bool
	}{{
		name:      "omitted_keeps_stored",
		body:      `{"name":"stored","ids":["192.0.2.1"],"use_global_settings":true}`,
		wantBlock: []rules.ListID{1, 3},
		wantAllow: []rules.ListID{10},
		wantOwn:   true,
	}, {
		name:      "explicit_false_clears",
		body:      `{"name":"stored","ids":["192.0.2.1"],"use_own_filter_lists":false}`,
		wantBlock: []rules.ListID{1, 3},
		wantAllow: []rules.ListID{10},
		wantOwn:   false,
	}, {
		name:      "empty_list_clears_ids",
		body:      `{"name":"stored","ids":["192.0.2.1"],"use_own_filter_lists":true,"filter_list_ids":[]}`,
		wantBlock: nil,
		wantAllow: []rules.ListID{10},
		wantOwn:   true,
	}, {
		name:      "new_ids_replace_stored",
		body:      `{"name":"stored","ids":["192.0.2.1"],"filter_list_ids":[7]}`,
		wantBlock: []rules.ListID{7},
		wantAllow: []rules.ListID{10},
		wantOwn:   true,
	}}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			cj := clientJSON{}
			require.NoError(t, json.Unmarshal([]byte(tc.body), &cj))

			ctx := testutil.ContextWithTimeout(t, testTimeout)
			c, err := clients.jsonToClient(ctx, cj, prev)
			require.NoError(t, err)

			assert.Equal(t, tc.wantOwn, c.UseOwnFilterLists)
			assert.Equal(t, tc.wantBlock, c.FilterListIDs)
			assert.Equal(t, tc.wantAllow, c.AllowFilterListIDs)
		})
	}
}

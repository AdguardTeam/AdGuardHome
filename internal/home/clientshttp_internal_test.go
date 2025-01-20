package home

import (
	"bytes"
	"cmp"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"net/netip"
	"net/url"
	"slices"
	"testing"
	"time"

	"github.com/AdguardTeam/AdGuardHome/internal/client"
	"github.com/AdguardTeam/AdGuardHome/internal/filtering"
	"github.com/AdguardTeam/AdGuardHome/internal/schedule"
	"github.com/AdguardTeam/AdGuardHome/internal/whois"
	"github.com/AdguardTeam/golibs/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// testTimeout is the common timeout for tests and contexts.
const testTimeout = 1 * time.Second

const (
	testClientIP1 = "1.1.1.1"
	testClientIP2 = "2.2.2.2"
)

// testBlockedClientChecker is a mock implementation of the
// [BlockedClientChecker] interface.
type testBlockedClientChecker struct {
	onIsBlockedClient func(ip netip.Addr, clientiD string) (blocked bool, rule string)
}

// type check
var _ BlockedClientChecker = (*testBlockedClientChecker)(nil)

// IsBlockedClient implements the [BlockedClientChecker] interface for
// *testBlockedClientChecker.
func (c *testBlockedClientChecker) IsBlockedClient(
	ip netip.Addr,
	clientID string,
) (blocked bool, rule string) {
	return c.onIsBlockedClient(ip, clientID)
}

// newPersistentClient is a helper function that returns a persistent client
// with the specified name and newly generated UID.
func newPersistentClient(name string) (c *client.Persistent) {
	return &client.Persistent{
		Name: name,
		UID:  client.MustNewUID(),
		BlockedServices: &filtering.BlockedServices{
			Schedule: schedule.EmptyWeekly(),
		},
	}
}

// newPersistentClientWithIDs is a helper function that returns a persistent
// client with the specified name and ids.
func newPersistentClientWithIDs(tb testing.TB, name string, ids []string) (c *client.Persistent) {
	tb.Helper()

	c = newPersistentClient(name)
	err := c.SetIDs(ids)
	require.NoError(tb, err)

	return c
}

// assertClients is a helper function that compares lists of persistent clients.
func assertClients(tb testing.TB, want, got []*client.Persistent) {
	tb.Helper()

	require.Len(tb, got, len(want))

	sortFunc := func(a, b *client.Persistent) (n int) {
		return cmp.Compare(a.Name, b.Name)
	}

	slices.SortFunc(want, sortFunc)
	slices.SortFunc(got, sortFunc)

	slices.CompareFunc(want, got, func(a, b *client.Persistent) (n int) {
		assert.True(tb, a.EqualIDs(b), "%q doesn't have the same ids as %q", a.Name, b.Name)

		return 0
	})
}

// assertPersistentClients is a helper function that uses HTTP API to check
// whether want persistent clients are the same as the persistent clients stored
// in the clients container.
func assertPersistentClients(tb testing.TB, clients *clientsContainer, want []*client.Persistent) {
	tb.Helper()

	rw := httptest.NewRecorder()
	clients.handleGetClients(rw, &http.Request{})

	body, err := io.ReadAll(rw.Body)
	require.NoError(tb, err)

	clientList := &clientListJSON{}
	err = json.Unmarshal(body, clientList)
	require.NoError(tb, err)

	var got []*client.Persistent
	ctx := testutil.ContextWithTimeout(tb, testTimeout)
	for _, cj := range clientList.Clients {
		var c *client.Persistent
		c, err = clients.jsonToClient(ctx, *cj, nil)
		require.NoError(tb, err)

		got = append(got, c)
	}

	assertClients(tb, want, got)
}

// assertPersistentClientsData is a helper function that checks whether want
// persistent clients are the same as the persistent clients stored in data.
func assertPersistentClientsData(
	tb testing.TB,
	clients *clientsContainer,
	data []map[string]*clientJSON,
	want []*client.Persistent,
) {
	tb.Helper()

	var got []*client.Persistent
	ctx := testutil.ContextWithTimeout(tb, testTimeout)
	for _, cm := range data {
		for _, cj := range cm {
			var c *client.Persistent
			c, err := clients.jsonToClient(ctx, *cj, nil)
			require.NoError(tb, err)

			got = append(got, c)
		}
	}

	assertClients(tb, want, got)
}

func TestClientsContainer_HandleAddClient(t *testing.T) {
	clients := newClientsContainer(t)

	clientOne := newPersistentClientWithIDs(t, "client1", []string{testClientIP1})
	clientTwo := newPersistentClientWithIDs(t, "client2", []string{testClientIP2})

	clientEmptyID := newPersistentClient("empty_client_id")
	clientEmptyID.ClientIDs = []string{""}

	testCases := []struct {
		name       string
		client     *client.Persistent
		wantCode   int
		wantClient []*client.Persistent
	}{{
		name:       "add_one",
		client:     clientOne,
		wantCode:   http.StatusOK,
		wantClient: []*client.Persistent{clientOne},
	}, {
		name:       "add_two",
		client:     clientTwo,
		wantCode:   http.StatusOK,
		wantClient: []*client.Persistent{clientOne, clientTwo},
	}, {
		name:       "duplicate_client",
		client:     clientTwo,
		wantCode:   http.StatusBadRequest,
		wantClient: []*client.Persistent{clientOne, clientTwo},
	}, {
		name:       "empty_client_id",
		client:     clientEmptyID,
		wantCode:   http.StatusBadRequest,
		wantClient: []*client.Persistent{clientOne, clientTwo},
	}}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			cj := clientToJSON(tc.client)

			body, err := json.Marshal(cj)
			require.NoError(t, err)

			r, err := http.NewRequest(http.MethodPost, "", bytes.NewReader(body))
			require.NoError(t, err)

			rw := httptest.NewRecorder()
			clients.handleAddClient(rw, r)
			require.NoError(t, err)
			require.Equal(t, tc.wantCode, rw.Code)

			assertPersistentClients(t, clients, tc.wantClient)
		})
	}
}

func TestClientsContainer_HandleDelClient(t *testing.T) {
	clients := newClientsContainer(t)
	ctx := testutil.ContextWithTimeout(t, testTimeout)

	clientOne := newPersistentClientWithIDs(t, "client1", []string{testClientIP1})
	err := clients.storage.Add(ctx, clientOne)
	require.NoError(t, err)

	clientTwo := newPersistentClientWithIDs(t, "client2", []string{testClientIP2})
	err = clients.storage.Add(ctx, clientTwo)
	require.NoError(t, err)

	assertPersistentClients(t, clients, []*client.Persistent{clientOne, clientTwo})

	testCases := []struct {
		name       string
		client     *client.Persistent
		wantCode   int
		wantClient []*client.Persistent
	}{{
		name:       "remove_one",
		client:     clientOne,
		wantCode:   http.StatusOK,
		wantClient: []*client.Persistent{clientTwo},
	}, {
		name:       "duplicate_client",
		client:     clientOne,
		wantCode:   http.StatusBadRequest,
		wantClient: []*client.Persistent{clientTwo},
	}, {
		name:       "empty_client_name",
		client:     newPersistentClient(""),
		wantCode:   http.StatusBadRequest,
		wantClient: []*client.Persistent{clientTwo},
	}, {
		name:       "remove_two",
		client:     clientTwo,
		wantCode:   http.StatusOK,
		wantClient: []*client.Persistent{},
	}}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			cj := clientToJSON(tc.client)

			var body []byte
			body, err = json.Marshal(cj)
			require.NoError(t, err)

			var r *http.Request
			r, err = http.NewRequest(http.MethodPost, "", bytes.NewReader(body))
			require.NoError(t, err)

			rw := httptest.NewRecorder()
			clients.handleDelClient(rw, r)
			require.NoError(t, err)
			require.Equal(t, tc.wantCode, rw.Code)

			assertPersistentClients(t, clients, tc.wantClient)
		})
	}
}

func TestClientsContainer_HandleUpdateClient(t *testing.T) {
	clients := newClientsContainer(t)
	ctx := testutil.ContextWithTimeout(t, testTimeout)

	clientOne := newPersistentClientWithIDs(t, "client1", []string{testClientIP1})
	err := clients.storage.Add(ctx, clientOne)
	require.NoError(t, err)

	assertPersistentClients(t, clients, []*client.Persistent{clientOne})

	clientModified := newPersistentClientWithIDs(t, "client2", []string{testClientIP2})

	clientEmptyID := newPersistentClient("empty_client_id")
	clientEmptyID.ClientIDs = []string{""}

	testCases := []struct {
		name       string
		clientName string
		modified   *client.Persistent
		wantCode   int
		wantClient []*client.Persistent
	}{{
		name:       "update_one",
		clientName: clientOne.Name,
		modified:   clientModified,
		wantCode:   http.StatusOK,
		wantClient: []*client.Persistent{clientModified},
	}, {
		name:       "empty_name",
		clientName: "",
		modified:   clientOne,
		wantCode:   http.StatusBadRequest,
		wantClient: []*client.Persistent{clientModified},
	}, {
		name:       "client_not_found",
		clientName: "client_not_found",
		modified:   clientOne,
		wantCode:   http.StatusBadRequest,
		wantClient: []*client.Persistent{clientModified},
	}, {
		name:       "empty_client_id",
		clientName: clientModified.Name,
		modified:   clientEmptyID,
		wantCode:   http.StatusBadRequest,
		wantClient: []*client.Persistent{clientModified},
	}, {
		name:       "no_ids",
		clientName: clientModified.Name,
		modified:   newPersistentClient("no_ids"),
		wantCode:   http.StatusBadRequest,
		wantClient: []*client.Persistent{clientModified},
	}}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			uj := updateJSON{
				Name: tc.clientName,
				Data: *clientToJSON(tc.modified),
			}

			var body []byte
			body, err = json.Marshal(uj)
			require.NoError(t, err)

			var r *http.Request
			r, err = http.NewRequest(http.MethodPost, "", bytes.NewReader(body))
			require.NoError(t, err)

			rw := httptest.NewRecorder()
			clients.handleUpdateClient(rw, r)
			require.NoError(t, err)
			require.Equal(t, tc.wantCode, rw.Code)

			assertPersistentClients(t, clients, tc.wantClient)
		})
	}
}

func TestClientsContainer_HandleFindClient(t *testing.T) {
	clients := newClientsContainer(t)
	clients.clientChecker = &testBlockedClientChecker{
		onIsBlockedClient: func(ip netip.Addr, clientID string) (ok bool, rule string) {
			return false, ""
		},
	}

	ctx := testutil.ContextWithTimeout(t, testTimeout)

	clientOne := newPersistentClientWithIDs(t, "client1", []string{testClientIP1})
	err := clients.storage.Add(ctx, clientOne)
	require.NoError(t, err)

	clientTwo := newPersistentClientWithIDs(t, "client2", []string{testClientIP2})
	err = clients.storage.Add(ctx, clientTwo)
	require.NoError(t, err)

	assertPersistentClients(t, clients, []*client.Persistent{clientOne, clientTwo})

	testCases := []struct {
		name       string
		query      url.Values
		wantCode   int
		wantClient []*client.Persistent
	}{{
		name: "single",
		query: url.Values{
			"ip0": []string{testClientIP1},
		},
		wantCode:   http.StatusOK,
		wantClient: []*client.Persistent{clientOne},
	}, {
		name: "multiple",
		query: url.Values{
			"ip0": []string{testClientIP1},
			"ip1": []string{testClientIP2},
		},
		wantCode:   http.StatusOK,
		wantClient: []*client.Persistent{clientOne, clientTwo},
	}}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			var r *http.Request
			r, err = http.NewRequest(http.MethodGet, "", nil)
			require.NoError(t, err)

			r.URL.RawQuery = tc.query.Encode()
			rw := httptest.NewRecorder()
			clients.handleFindClient(rw, r)
			require.NoError(t, err)
			require.Equal(t, tc.wantCode, rw.Code)

			var body []byte
			body, err = io.ReadAll(rw.Body)
			require.NoError(t, err)

			clientData := []map[string]*clientJSON{}
			err = json.Unmarshal(body, &clientData)
			require.NoError(t, err)

			assertPersistentClientsData(t, clients, clientData, tc.wantClient)
		})
	}
}

func TestClientsContainer_HandleSearchClient(t *testing.T) {
	var (
		runtimeCli = "runtime_client1"

		runtimeCliIP     = "3.3.3.3"
		blockedCliIP     = "4.4.4.4"
		nonExistentCliIP = "5.5.5.5"

		allowed     = false
		dissallowed = true

		emptyRule      = ""
		disallowedRule = "disallowed_rule"
	)

	clients := newClientsContainer(t)
	clients.clientChecker = &testBlockedClientChecker{
		onIsBlockedClient: func(ip netip.Addr, _ string) (ok bool, rule string) {
			if ip == netip.MustParseAddr(blockedCliIP) {
				return true, disallowedRule
			}

			return false, emptyRule
		},
	}

	ctx := testutil.ContextWithTimeout(t, testTimeout)

	clientOne := newPersistentClientWithIDs(t, "client1", []string{testClientIP1})
	err := clients.storage.Add(ctx, clientOne)
	require.NoError(t, err)

	clientTwo := newPersistentClientWithIDs(t, "client2", []string{testClientIP2})
	err = clients.storage.Add(ctx, clientTwo)
	require.NoError(t, err)

	assertPersistentClients(t, clients, []*client.Persistent{clientOne, clientTwo})

	clients.UpdateAddress(ctx, netip.MustParseAddr(runtimeCliIP), runtimeCli, nil)

	testCases := []struct {
		name           string
		query          *searchQueryJSON
		wantPersistent []*client.Persistent
		wantRuntime    *clientJSON
	}{{
		name: "single",
		query: &searchQueryJSON{
			Clients: []searchClientJSON{{
				ID: testClientIP1,
			}},
		},
		wantPersistent: []*client.Persistent{clientOne},
	}, {
		name: "multiple",
		query: &searchQueryJSON{
			Clients: []searchClientJSON{{
				ID: testClientIP1,
			}, {
				ID: testClientIP2,
			}},
		},
		wantPersistent: []*client.Persistent{clientOne, clientTwo},
	}, {
		name: "runtime",
		query: &searchQueryJSON{
			Clients: []searchClientJSON{{
				ID: runtimeCliIP,
			}},
		},
		wantRuntime: &clientJSON{
			Name:           runtimeCli,
			IDs:            []string{runtimeCliIP},
			Disallowed:     &allowed,
			DisallowedRule: &emptyRule,
			WHOIS:          &whois.Info{},
		},
	}, {
		name: "blocked_access",
		query: &searchQueryJSON{
			Clients: []searchClientJSON{{
				ID: blockedCliIP,
			}},
		},
		wantRuntime: &clientJSON{
			IDs:            []string{blockedCliIP},
			Disallowed:     &dissallowed,
			DisallowedRule: &disallowedRule,
			WHOIS:          &whois.Info{},
		},
	}, {
		name: "non_existing_client",
		query: &searchQueryJSON{
			Clients: []searchClientJSON{{
				ID: nonExistentCliIP,
			}},
		},
		wantRuntime: &clientJSON{
			IDs:            []string{nonExistentCliIP},
			Disallowed:     &allowed,
			DisallowedRule: &emptyRule,
			WHOIS:          &whois.Info{},
		},
	}}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			var body []byte
			body, err = json.Marshal(tc.query)
			require.NoError(t, err)

			var r *http.Request
			r, err = http.NewRequest(http.MethodPost, "", bytes.NewReader(body))
			require.NoError(t, err)

			rw := httptest.NewRecorder()
			clients.handleSearchClient(rw, r)
			require.NoError(t, err)
			require.Equal(t, http.StatusOK, rw.Code)

			body, err = io.ReadAll(rw.Body)
			require.NoError(t, err)

			clientData := []map[string]*clientJSON{}
			err = json.Unmarshal(body, &clientData)
			require.NoError(t, err)

			if tc.wantPersistent != nil {
				assertPersistentClientsData(t, clients, clientData, tc.wantPersistent)

				return
			}

			require.Len(t, clientData, 1)
			require.Len(t, clientData[0], 1)

			rc := clientData[0][tc.wantRuntime.IDs[0]]
			assert.Equal(t, tc.wantRuntime, rc)
		})
	}
}

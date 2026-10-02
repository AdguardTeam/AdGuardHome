package configmgr_test

import (
	"testing"

	"github.com/AdguardTeam/AdGuardHome/internal/configmgr"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDoHConfig_Validate_routes(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name        string
		wantMessage string
		routes      []string
	}{{
		name:        "valid",
		wantMessage: "",
		routes: []string{
			"GET /dns-query",
			"POST /dns-query",
			"GET /dns-query/{ClientID}",
			"POST /dns-query/{ClientID}",
			"GET /dns-query/",
			"GET /doh/{a}/{b}",
			"/dns-query",
		},
	}, {
		name:        "empty",
		wantMessage: `routes: route "" at index 0: http: invalid pattern`,
		routes:      []string{""},
	}, {
		name: "bad_wildcard",
		wantMessage: `routes: route "GET /dns-query/{ClientID" at index 0: parsing ` +
			`"GET /dns-query/{ClientID"`,
		routes: []string{
			"GET /dns-query/{ClientID",
		},
	}, {
		name:        "duplicate",
		wantMessage: `routes: route "GET /dns-query" at index 1: pattern "GET /dns-query"`,
		routes: []string{
			"GET /dns-query",
			"GET /dns-query",
		},
	}, {
		name: "conflict",
		wantMessage: `routes: route "GET /dns-query/{Other}" at index 1: pattern ` +
			`"GET /dns-query/{Other}"`,
		routes: []string{
			"GET /dns-query/{ClientID}",
			"GET /dns-query/{Other}",
		},
	}}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			conf := &configmgr.DoHConfig{
				Routes: tc.routes,
			}

			err := conf.Validate()
			if tc.wantMessage == "" {
				assert.NoError(t, err)

				return
			}

			require.Error(t, err)
			assert.Contains(t, err.Error(), tc.wantMessage)
		})
	}
}

package filtering

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDNSFilter_handleFilteringSetURL(t *testing.T) {
	filtersDir := t.TempDir()

	var goodRulesEndpoint, anotherGoodRulesEndpoint, badRulesEndpoint string
	for _, rulesSource := range []struct {
		endpoint *string
		content  []byte
	}{{
		endpoint: &goodRulesEndpoint,
		content:  []byte(`||example.org^`),
	}, {
		endpoint: &anotherGoodRulesEndpoint,
		content:  []byte(`||example.com^`),
	}, {
		endpoint: &badRulesEndpoint,
		content:  []byte(`<html></html>`),
	}} {
		*rulesSource.endpoint = serveFiltersLocally(t, rulesSource.content)
	}

	testCases := []struct {
		name     string
		wantBody string
		oldURL   string
		newName  string
		newURL   string
		initial  []FilterYAML
	}{{
		name:     "success",
		wantBody: "",
		oldURL:   goodRulesEndpoint,
		newName:  "default_one",
		newURL:   anotherGoodRulesEndpoint,
		initial: []FilterYAML{{
			Enabled: true,
			URL:     goodRulesEndpoint,
			Name:    "default_one",
			white:   false,
		}},
	}, {
		name:     "non-existing",
		wantBody: "url doesn't exist\n",
		oldURL:   anotherGoodRulesEndpoint,
		newName:  "default_one",
		newURL:   goodRulesEndpoint,
		initial: []FilterYAML{{
			Enabled: true,
			URL:     goodRulesEndpoint,
			Name:    "default_one",
			white:   false,
		}},
	}, {
		name:     "existing",
		wantBody: "url already exists\n",
		oldURL:   goodRulesEndpoint,
		newName:  "default_one",
		newURL:   anotherGoodRulesEndpoint,
		initial: []FilterYAML{{
			Enabled: true,
			URL:     goodRulesEndpoint,
			Name:    "default_one",
			white:   false,
		}, {
			Enabled: true,
			URL:     anotherGoodRulesEndpoint,
			Name:    "another_default_one",
			white:   false,
		}},
	}, {
		name:     "bad_rules",
		wantBody: "data is HTML, not plain text\n",
		oldURL:   goodRulesEndpoint,
		newName:  "default_one",
		newURL:   badRulesEndpoint,
		initial: []FilterYAML{{
			Enabled: true,
			URL:     goodRulesEndpoint,
			Name:    "default_one",
			white:   false,
		}},
	}}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			confModifiedCalled := false
			d, err := New(&Config{
				FilteringEnabled: true,
				Filters:          tc.initial,
				HTTPClient: &http.Client{
					Timeout: 5 * time.Second,
				},
				ConfigModified: func() { confModifiedCalled = true },
				DataDir:        filtersDir,
			}, nil)
			require.NoError(t, err)
			t.Cleanup(d.Close)

			d.Start()

			reqData := &filterURLReq{
				Data: &filterURLReqData{
					// Leave the name of an existing list.
					Name:    tc.newName,
					URL:     tc.newURL,
					Enabled: true,
				},
				URL:       tc.oldURL,
				Whitelist: false,
			}
			data, err := json.Marshal(reqData)
			require.NoError(t, err)

			r := httptest.NewRequest(http.MethodPost, "http://example.org", bytes.NewReader(data))
			w := httptest.NewRecorder()

			d.handleFilteringSetURL(w, r)
			assert.Equal(t, tc.wantBody, w.Body.String())

			// For the moment the non-empty response body only contains occurred
			// error, so the configuration shouldn't be written.
			assert.Equal(t, tc.wantBody == "", confModifiedCalled)
		})
	}
}

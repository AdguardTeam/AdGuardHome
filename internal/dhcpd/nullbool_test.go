package dhcpd

import (
	"encoding/json"
	"testing"

	"github.com/AdguardTeam/golibs/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNullBool_UnmarshalJSON(t *testing.T) {
	testCases := []struct {
		name       string
		wantErrMsg string
		data       []byte
		want       nullBool
	}{{
		name:       "empty",
		wantErrMsg: "",
		data:       []byte{},
		want:       nbNull,
	}, {
		name:       "null",
		wantErrMsg: "",
		data:       []byte("null"),
		want:       nbNull,
	}, {
		name:       "true",
		wantErrMsg: "",
		data:       []byte("true"),
		want:       nbTrue,
	}, {
		name:       "false",
		wantErrMsg: "",
		data:       []byte("false"),
		want:       nbFalse,
	}, {
		name:       "invalid",
		wantErrMsg: `invalid nullBool value "invalid"`,
		data:       []byte("invalid"),
		want:       nbNull,
	}}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			var got nullBool
			err := got.UnmarshalJSON(tc.data)
			testutil.AssertErrorMsg(t, tc.wantErrMsg, err)

			assert.Equal(t, tc.want, got)
		})
	}

	t.Run("json", func(t *testing.T) {
		want := nbTrue
		var got struct {
			A nullBool
		}

		err := json.Unmarshal([]byte(`{"A":true}`), &got)
		require.NoError(t, err)

		assert.Equal(t, want, got.A)
	})
}

package dhcpd

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNullBool_UnmarshalText(t *testing.T) {
	testCases := []struct {
		name       string
		data       []byte
		wantErrMsg string
		want       nullBool
	}{{
		name:       "empty",
		data:       []byte{},
		wantErrMsg: "",
		want:       nbNull,
	}, {
		name:       "null",
		data:       []byte("null"),
		wantErrMsg: "",
		want:       nbNull,
	}, {
		name:       "true",
		data:       []byte("true"),
		wantErrMsg: "",
		want:       nbTrue,
	}, {
		name:       "false",
		data:       []byte("false"),
		wantErrMsg: "",
		want:       nbFalse,
	}, {
		name:       "invalid",
		data:       []byte("flase"),
		wantErrMsg: `invalid nullBool value "flase"`,
		want:       nbNull,
	}}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			var got nullBool
			err := got.UnmarshalJSON(tc.data)
			if tc.wantErrMsg == "" {
				assert.Nil(t, err)
			} else {
				require.NotNil(t, err)
				assert.Equal(t, tc.wantErrMsg, err.Error())
			}

			assert.Equal(t, tc.want, got)
		})
	}

	t.Run("json", func(t *testing.T) {
		want := nbTrue
		var got struct {
			A nullBool
		}

		err := json.Unmarshal([]byte(`{"A":true}`), &got)
		require.Nil(t, err)
		assert.Equal(t, want, got.A)
	})
}

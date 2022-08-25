package aghalg_test

import (
	"encoding/json"
	"testing"

	"github.com/AdguardTeam/AdGuardHome/internal/aghalg"
	"github.com/AdguardTeam/golibs/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNullBool_MarshalJSON(t *testing.T) {
	testCases := []struct {
		name       string
		wantErrMsg string
		want       []byte
		in         aghalg.NullBool
	}{{
		name:       "null",
		wantErrMsg: "",
		want:       []byte("null"),
		in:         aghalg.NBNull,
	}, {
		name:       "true",
		wantErrMsg: "",
		want:       []byte("true"),
		in:         aghalg.NBTrue,
	}, {
		name:       "false",
		wantErrMsg: "",
		want:       []byte("false"),
		in:         aghalg.NBFalse,
	}}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := tc.in.MarshalJSON()
			testutil.AssertErrorMsg(t, tc.wantErrMsg, err)

			assert.Equal(t, tc.want, got)
		})
	}

	t.Run("json", func(t *testing.T) {
		in := &struct {
			A aghalg.NullBool
		}{
			A: aghalg.NBTrue,
		}

		got, err := json.Marshal(in)
		require.NoError(t, err)

		assert.Equal(t, []byte(`{"A":true}`), got)
	})
}

func TestNullBool_UnmarshalJSON(t *testing.T) {
	testCases := []struct {
		name       string
		wantErrMsg string
		data       []byte
		want       aghalg.NullBool
	}{{
		name:       "empty",
		wantErrMsg: "",
		data:       []byte{},
		want:       aghalg.NBNull,
	}, {
		name:       "null",
		wantErrMsg: "",
		data:       []byte("null"),
		want:       aghalg.NBNull,
	}, {
		name:       "true",
		wantErrMsg: "",
		data:       []byte("true"),
		want:       aghalg.NBTrue,
	}, {
		name:       "false",
		wantErrMsg: "",
		data:       []byte("false"),
		want:       aghalg.NBFalse,
	}, {
		name:       "invalid",
		wantErrMsg: `unmarshalling json data into aghalg.NullBool: bad value "invalid"`,
		data:       []byte("invalid"),
		want:       aghalg.NBNull,
	}}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			var got aghalg.NullBool
			err := got.UnmarshalJSON(tc.data)
			testutil.AssertErrorMsg(t, tc.wantErrMsg, err)

			assert.Equal(t, tc.want, got)
		})
	}

	t.Run("json", func(t *testing.T) {
		want := aghalg.NBTrue
		var got struct {
			A aghalg.NullBool
		}

		err := json.Unmarshal([]byte(`{"A":true}`), &got)
		require.NoError(t, err)

		assert.Equal(t, want, got.A)
	})
}

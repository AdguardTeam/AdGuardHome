package websvc

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/AdguardTeam/golibs/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// testJSONTime is the JSON time for tests.
var testJSONTime = jsonTime(time.Unix(1_234_567_890, 123_456_000).UTC())

// testJSONTimeStr is the string with the JSON encoding of testJSONTime.
const testJSONTimeStr = "1234567890123.456"

func TestJSONTime_MarshalJSON(t *testing.T) {
	testCases := []struct {
		name       string
		wantErrMsg string
		in         jsonTime
		want       []byte
	}{{
		name:       "unix_zero",
		wantErrMsg: "",
		in:         jsonTime(time.Unix(0, 0)),
		want:       []byte("0"),
	}, {
		name:       "empty",
		wantErrMsg: "",
		in:         jsonTime{},
		want:       []byte("-6795364578871.345"),
	}, {
		name:       "time",
		wantErrMsg: "",
		in:         testJSONTime,
		want:       []byte(testJSONTimeStr),
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
			A jsonTime
		}{
			A: testJSONTime,
		}

		got, err := json.Marshal(in)
		require.NoError(t, err)

		assert.Equal(t, []byte(`{"A":`+testJSONTimeStr+`}`), got)
	})
}

func TestJSONTime_UnmarshalJSON(t *testing.T) {
	testCases := []struct {
		name       string
		wantErrMsg string
		want       jsonTime
		data       []byte
	}{{
		name:       "time",
		wantErrMsg: "",
		want:       testJSONTime,
		data:       []byte(testJSONTimeStr),
	}, {
		name: "bad",
		wantErrMsg: `parsing json time: strconv.ParseFloat: parsing "{}": ` +
			`invalid syntax`,
		want: jsonTime{},
		data: []byte(`{}`),
	}}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			var got jsonTime
			err := got.UnmarshalJSON(tc.data)
			testutil.AssertErrorMsg(t, tc.wantErrMsg, err)

			assert.Equal(t, tc.want, got)
		})
	}

	t.Run("nil", func(t *testing.T) {
		err := (*jsonTime)(nil).UnmarshalJSON([]byte("0"))
		require.Error(t, err)

		msg := err.Error()
		assert.Equal(t, "json time is nil", msg)
	})

	t.Run("json", func(t *testing.T) {
		want := testJSONTime
		var got struct {
			A jsonTime
		}

		err := json.Unmarshal([]byte(`{"A":`+testJSONTimeStr+`}`), &got)
		require.NoError(t, err)

		assert.Equal(t, want, got.A)
	})
}

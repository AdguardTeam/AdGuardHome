package schedule

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/AdguardTeam/golibs/testutil"
	"github.com/AdguardTeam/golibs/timeutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
)

func TestWeekly_Contains(t *testing.T) {
	baseTime := time.Date(2021, 1, 1, 0, 0, 0, 0, time.UTC)
	otherTime := baseTime.Add(1 * timeutil.Day)

	// NOTE: In the Etc area the sign of the offsets is flipped.  So, Etc/GMT-3
	// is actually UTC+03:00.
	otherTZ := time.FixedZone("Etc/GMT-3", 3*60*60)

	// baseSchedule, 12:00 to 14:00.
	baseSchedule := &Weekly{
		days: [7]dayRange{
			time.Friday: {start: 12 * time.Hour, end: 14 * time.Hour},
		},
		location: time.UTC,
	}

	// allDaySchedule, 00:00 to 24:00.
	allDaySchedule := &Weekly{
		days: [7]dayRange{
			time.Friday: {start: 0, end: 24 * time.Hour},
		},
		location: time.UTC,
	}

	// oneMinSchedule, 00:00 to 00:01.
	oneMinSchedule := &Weekly{
		days: [7]dayRange{
			time.Friday: {start: 0, end: 1 * time.Minute},
		},
		location: time.UTC,
	}

	testCases := []struct {
		schedule *Weekly
		assert   assert.BoolAssertionFunc
		t        time.Time
		name     string
	}{{
		schedule: EmptyWeekly(),
		assert:   assert.False,
		t:        baseTime,
		name:     "empty",
	}, {
		schedule: allDaySchedule,
		assert:   assert.True,
		t:        baseTime,
		name:     "same_day_all_day",
	}, {
		schedule: baseSchedule,
		assert:   assert.True,
		t:        baseTime.Add(13 * time.Hour),
		name:     "same_day_inside",
	}, {
		schedule: baseSchedule,
		assert:   assert.False,
		t:        baseTime.Add(11 * time.Hour),
		name:     "same_day_outside",
	}, {
		schedule: allDaySchedule,
		assert:   assert.True,
		t:        baseTime.Add(24*time.Hour - time.Second),
		name:     "same_day_last_second",
	}, {
		schedule: allDaySchedule,
		assert:   assert.False,
		t:        otherTime,
		name:     "other_day_all_day",
	}, {
		schedule: baseSchedule,
		assert:   assert.False,
		t:        otherTime.Add(13 * time.Hour),
		name:     "other_day_inside",
	}, {
		schedule: baseSchedule,
		assert:   assert.False,
		t:        otherTime.Add(11 * time.Hour),
		name:     "other_day_outside",
	}, {
		schedule: baseSchedule,
		assert:   assert.True,
		t:        baseTime.Add(13 * time.Hour).In(otherTZ),
		name:     "same_day_inside_other_tz",
	}, {
		schedule: baseSchedule,
		assert:   assert.False,
		t:        baseTime.Add(11 * time.Hour).In(otherTZ),
		name:     "same_day_outside_other_tz",
	}, {
		schedule: oneMinSchedule,
		assert:   assert.True,
		t:        baseTime,
		name:     "one_minute_beginning",
	}, {
		schedule: oneMinSchedule,
		assert:   assert.True,
		t:        baseTime.Add(1*time.Minute - 1),
		name:     "one_minute_end",
	}, {
		schedule: oneMinSchedule,
		assert:   assert.False,
		t:        baseTime.Add(1 * time.Minute),
		name:     "one_minute_past_end",
	}}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			tc.assert(t, tc.schedule.Contains(tc.t))
		})
	}
}

const brusselsSundayYAML = `
sun:
    start: 12h
    end: 14h
time_zone: Europe/Brussels
`

func TestWeekly_UnmarshalYAML(t *testing.T) {
	const (
		sameTime = `
sun:
    start: 9h
    end: 9h
`
		negativeStart = `
sun:
    start: -1h
    end: 1h
`
		badTZ = `
time_zone: "bad_timezone"
`
		badYAML = `
yaml: "bad"
yaml: "bad"
`
	)

	brusseltsTZ, err := time.LoadLocation("Europe/Brussels")
	require.NoError(t, err)

	brusselsWeekly := &Weekly{
		days: [7]dayRange{{
			start: time.Hour * 12,
			end:   time.Hour * 14,
		}},
		location: brusseltsTZ,
	}

	testCases := []struct {
		want       *Weekly
		name       string
		wantErrMsg string
		data       []byte
	}{{
		name:       "empty",
		wantErrMsg: "",
		data:       []byte(""),
		want:       &Weekly{},
	}, {
		name:       "null",
		wantErrMsg: "",
		data:       []byte("null"),
		want:       &Weekly{},
	}, {
		name:       "brussels_sunday",
		wantErrMsg: "",
		data:       []byte(brusselsSundayYAML),
		want:       brusselsWeekly,
	}, {
		name:       "start_equal_end",
		wantErrMsg: "weekday Sunday: bad day range: start 9h0m0s is greater or equal to end 9h0m0s",
		data:       []byte(sameTime),
		want:       &Weekly{},
	}, {
		name:       "start_negative",
		wantErrMsg: "weekday Sunday: bad day range: start -1h0m0s is negative",
		data:       []byte(negativeStart),
		want:       &Weekly{},
	}, {
		name:       "bad_time_zone",
		wantErrMsg: "unknown time zone bad_timezone",
		data:       []byte(badTZ),
		want:       &Weekly{},
	}, {
		name:       "bad_yaml",
		wantErrMsg: "yaml: unmarshal errors:\n  line 3: mapping key \"yaml\" already defined at line 2",
		data:       []byte(badYAML),
		want:       &Weekly{},
	}}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			w := &Weekly{}
			err = yaml.Unmarshal(tc.data, w)
			testutil.AssertErrorMsg(t, tc.wantErrMsg, err)

			assert.Equal(t, tc.want, w)
		})
	}
}

func TestWeekly_MarshalYAML(t *testing.T) {
	brusselsTZ, err := time.LoadLocation("Europe/Brussels")
	require.NoError(t, err)

	brusselsWeekly := &Weekly{
		days: [7]dayRange{time.Sunday: {
			start: time.Hour * 12,
			end:   time.Hour * 14,
		}},
		location: brusselsTZ,
	}

	testCases := []struct {
		want *Weekly
		name string
		data []byte
	}{{
		name: "empty",
		data: []byte(""),
		want: &Weekly{},
	}, {
		name: "null",
		data: []byte("null"),
		want: &Weekly{},
	}, {
		name: "brussels_sunday",
		data: []byte(brusselsSundayYAML),
		want: brusselsWeekly,
	}}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			var data []byte
			data, err = yaml.Marshal(brusselsWeekly)
			require.NoError(t, err)

			w := &Weekly{}
			err = yaml.Unmarshal(data, w)
			require.NoError(t, err)

			assert.Equal(t, brusselsWeekly, w)
		})
	}
}

func TestWeekly_Validate(t *testing.T) {
	testCases := []struct {
		name       string
		wantErrMsg string
		in         dayRange
	}{{
		name:       "empty",
		wantErrMsg: "",
		in:         dayRange{},
	}, {
		name:       "start_seconds",
		wantErrMsg: "bad day range: start 1s isn't rounded to minutes",
		in: dayRange{
			start: time.Second,
			end:   time.Hour,
		},
	}, {
		name:       "end_seconds",
		wantErrMsg: "bad day range: end 1s isn't rounded to minutes",
		in: dayRange{
			start: 0,
			end:   time.Second,
		},
	}}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			w := &Weekly{}
			err := w.validate(tc.in)

			testutil.AssertErrorMsg(t, tc.wantErrMsg, err)
		})
	}
}

func TestDayRange_Validate(t *testing.T) {
	testCases := []struct {
		name       string
		wantErrMsg string
		in         dayRange
	}{{
		name:       "empty",
		wantErrMsg: "",
		in:         dayRange{},
	}, {
		name:       "valid",
		wantErrMsg: "",
		in: dayRange{
			start: time.Hour,
			end:   time.Hour * 2,
		},
	}, {
		name:       "valid_end_max",
		wantErrMsg: "",
		in: dayRange{
			start: 0,
			end:   time.Hour * 24,
		},
	}, {
		name:       "start_negative",
		wantErrMsg: "start -1h0m0s is negative",
		in: dayRange{
			start: time.Hour * -1,
			end:   time.Hour * 2,
		},
	}, {
		name:       "end_negative",
		wantErrMsg: "end -1h0m0s is negative",
		in: dayRange{
			start: 0,
			end:   time.Hour * -1,
		},
	}, {
		name:       "start_equal_end",
		wantErrMsg: "start 1h0m0s is greater or equal to end 1h0m0s",
		in: dayRange{
			start: time.Hour,
			end:   time.Hour,
		},
	}, {
		name:       "start_greater_end",
		wantErrMsg: "start 2h0m0s is greater or equal to end 1h0m0s",
		in: dayRange{
			start: time.Hour * 2,
			end:   time.Hour,
		},
	}, {
		name:       "start_equal_max",
		wantErrMsg: "start 24h0m0s is greater or equal to 24h0m0s",
		in: dayRange{
			start: time.Hour * 24,
			end:   time.Hour * 48,
		},
	}, {
		name:       "end_greater_max",
		wantErrMsg: "end 48h0m0s is greater than 24h0m0s",
		in: dayRange{
			start: 0,
			end:   time.Hour * 48,
		},
	}}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			err := tc.in.validate()

			testutil.AssertErrorMsg(t, tc.wantErrMsg, err)
		})
	}
}

const brusselsSundayJSON = `{
  "sun": {
    "end": 50400000,
    "start": 43200000
  },
  "time_zone": "Europe/Brussels"
}`

func TestWeekly_UnmarshalJSON(t *testing.T) {
	const (
		sameTime = `{
  "sun": {
    "end": 32400000,
    "start": 32400000
  }
}`
		negativeStart = `{
  "sun": {
    "end": 3600000,
    "start": -3600000
  }
}`
		badTZ = `{
  "time_zone": "bad_timezone"
}`
		badJSON = `{
  "bad": "json",
}`
	)

	brusseltsTZ, err := time.LoadLocation("Europe/Brussels")
	require.NoError(t, err)

	brusselsWeekly := &Weekly{
		days: [7]dayRange{{
			start: time.Hour * 12,
			end:   time.Hour * 14,
		}},
		location: brusseltsTZ,
	}

	testCases := []struct {
		want       *Weekly
		name       string
		wantErrMsg string
		data       []byte
	}{{
		name:       "empty",
		wantErrMsg: "unexpected end of JSON input",
		data:       []byte(""),
		want:       &Weekly{},
	}, {
		name:       "null",
		wantErrMsg: "",
		data:       []byte("null"),
		want:       &Weekly{location: time.UTC},
	}, {
		name:       "brussels_sunday",
		wantErrMsg: "",
		data:       []byte(brusselsSundayJSON),
		want:       brusselsWeekly,
	}, {
		name:       "start_equal_end",
		wantErrMsg: "weekday Sunday: bad day range: start 9h0m0s is greater or equal to end 9h0m0s",
		data:       []byte(sameTime),
		want:       &Weekly{},
	}, {
		name:       "start_negative",
		wantErrMsg: "weekday Sunday: bad day range: start -1h0m0s is negative",
		data:       []byte(negativeStart),
		want:       &Weekly{},
	}, {
		name:       "bad_time_zone",
		wantErrMsg: "unknown time zone bad_timezone",
		data:       []byte(badTZ),
		want:       &Weekly{},
	}, {
		name:       "bad_json",
		wantErrMsg: "invalid character '}' looking for beginning of object key string",
		data:       []byte(badJSON),
		want:       &Weekly{},
	}}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			w := &Weekly{}
			err = json.Unmarshal(tc.data, w)
			testutil.AssertErrorMsg(t, tc.wantErrMsg, err)

			assert.Equal(t, tc.want, w)
		})
	}
}

func TestWeekly_MarshalJSON(t *testing.T) {
	brusselsTZ, err := time.LoadLocation("Europe/Brussels")
	require.NoError(t, err)

	brusselsWeekly := &Weekly{
		days: [7]dayRange{time.Sunday: {
			start: time.Hour * 12,
			end:   time.Hour * 14,
		}},
		location: brusselsTZ,
	}

	testCases := []struct {
		want *Weekly
		name string
		data []byte
	}{{
		name: "empty",
		data: []byte(""),
		want: &Weekly{},
	}, {
		name: "null",
		data: []byte("null"),
		want: &Weekly{},
	}, {
		name: "brussels_sunday",
		data: []byte(brusselsSundayJSON),
		want: brusselsWeekly,
	}}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			var data []byte
			data, err = json.Marshal(brusselsWeekly)
			require.NoError(t, err)

			w := &Weekly{}
			err = json.Unmarshal(data, w)
			require.NoError(t, err)

			assert.Equal(t, brusselsWeekly, w)
		})
	}
}

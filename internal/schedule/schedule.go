// Package schedule provides types for scheduling.
package schedule

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/AdguardTeam/AdGuardHome/internal/aghhttp"
	"github.com/AdguardTeam/golibs/errors"
	"github.com/AdguardTeam/golibs/timeutil"
	"gopkg.in/yaml.v3"
)

// Weekly is a schedule for one week.  Each day of the week has one range with
// a beginning and an end.
type Weekly struct {
	// location is used to calculate the offsets of the day ranges.
	location *time.Location

	// days are the day ranges of this schedule.  The indexes of this array are
	// the [time.Weekday] values.
	days [7]dayRange
}

// EmptyWeekly creates empty weekly schedule with local time zone.
func EmptyWeekly() (w *Weekly) {
	return &Weekly{
		location: time.Local,
	}
}

// FullWeekly creates full weekly schedule with local time zone.
//
// TODO(s.chzhen):  Consider moving into tests.
func FullWeekly() (w *Weekly) {
	fullDay := dayRange{start: 0, end: maxDayRange}

	return &Weekly{
		location: time.Local,
		days: [7]dayRange{
			time.Sunday:    fullDay,
			time.Monday:    fullDay,
			time.Tuesday:   fullDay,
			time.Wednesday: fullDay,
			time.Thursday:  fullDay,
			time.Friday:    fullDay,
			time.Saturday:  fullDay,
		},
	}
}

// Clone returns a deep copy of a weekly.
func (w *Weekly) Clone() (c *Weekly) {
	if w == nil {
		return nil
	}

	// NOTE:  Do not use time.LoadLocation, because the results will be
	// different on time zone database update.
	return &Weekly{
		location: w.location,
		days:     w.days,
	}
}

// Contains returns true if t is within the corresponding day range of the
// schedule in the schedule's time zone.
func (w *Weekly) Contains(t time.Time) (ok bool) {
	t = t.In(w.location)
	wd := t.Weekday()
	dr := w.days[wd]

	// Calculate the offset of the day range.
	//
	// NOTE: Do not use [time.Truncate] since it requires UTC time zone.
	y, m, d := t.Date()
	day := time.Date(y, m, d, 0, 0, 0, 0, w.location)
	offset := t.Sub(day)

	return dr.contains(offset)
}

// type check
var _ json.Unmarshaler = (*Weekly)(nil)

// UnmarshalJSON implements the [json.Unmarshaler] interface for *Weekly.
func (w *Weekly) UnmarshalJSON(data []byte) (err error) {
	conf := &weeklyConfigJSON{}
	err = json.Unmarshal(data, conf)
	if err != nil {
		return err
	}

	weekly := Weekly{}

	weekly.location, err = time.LoadLocation(conf.TimeZone)
	if err != nil {
		return err
	}

	days := []*dayConfigJSON{
		time.Sunday:    conf.Sunday,
		time.Monday:    conf.Monday,
		time.Tuesday:   conf.Tuesday,
		time.Wednesday: conf.Wednesday,
		time.Thursday:  conf.Thursday,
		time.Friday:    conf.Friday,
		time.Saturday:  conf.Saturday,
	}
	for i, d := range days {
		var r dayRange

		if d != nil {
			r = dayRange{
				start: time.Duration(d.Start),
				end:   time.Duration(d.End),
			}
		}

		err = w.validate(r)
		if err != nil {
			return fmt.Errorf("weekday %s: %w", time.Weekday(i), err)
		}

		weekly.days[i] = r
	}

	*w = weekly

	return nil
}

// type check
var _ yaml.Unmarshaler = (*Weekly)(nil)

// UnmarshalYAML implements the [yaml.Unmarshaler] interface for *Weekly.
func (w *Weekly) UnmarshalYAML(value *yaml.Node) (err error) {
	conf := &weeklyConfigYAML{}

	err = value.Decode(conf)
	if err != nil {
		// Don't wrap the error since it's informative enough as is.
		return err
	}

	weekly := Weekly{}

	weekly.location, err = time.LoadLocation(conf.TimeZone)
	if err != nil {
		// Don't wrap the error since it's informative enough as is.
		return err
	}

	days := []dayConfigYAML{
		time.Sunday:    conf.Sunday,
		time.Monday:    conf.Monday,
		time.Tuesday:   conf.Tuesday,
		time.Wednesday: conf.Wednesday,
		time.Thursday:  conf.Thursday,
		time.Friday:    conf.Friday,
		time.Saturday:  conf.Saturday,
	}
	for i, d := range days {
		r := dayRange{
			start: time.Duration(d.Start),
			end:   time.Duration(d.End),
		}

		err = w.validate(r)
		if err != nil {
			return fmt.Errorf("weekday %s: %w", time.Weekday(i), err)
		}

		weekly.days[i] = r
	}

	*w = weekly

	return nil
}

// weeklyConfigYAML is the YAML configuration structure of Weekly.
type weeklyConfigYAML struct {
	// TimeZone is the local time zone.
	TimeZone string `yaml:"time_zone"`

	// Days of the week.

	Sunday    dayConfigYAML `yaml:"sun,omitempty"`
	Monday    dayConfigYAML `yaml:"mon,omitempty"`
	Tuesday   dayConfigYAML `yaml:"tue,omitempty"`
	Wednesday dayConfigYAML `yaml:"wed,omitempty"`
	Thursday  dayConfigYAML `yaml:"thu,omitempty"`
	Friday    dayConfigYAML `yaml:"fri,omitempty"`
	Saturday  dayConfigYAML `yaml:"sat,omitempty"`
}

// dayConfigYAML is the YAML configuration structure of dayRange.
type dayConfigYAML struct {
	Start timeutil.Duration `yaml:"start"`
	End   timeutil.Duration `yaml:"end"`
}

// maxDayRange is the maximum value for day range end.
const maxDayRange = 24 * time.Hour

// validate returns the day range rounding errors, if any.
func (w *Weekly) validate(r dayRange) (err error) {
	defer func() { err = errors.Annotate(err, "bad day range: %w") }()

	err = r.validate()
	if err != nil {
		// Don't wrap the error since it's informative enough as is.
		return err
	}

	start := r.start.Truncate(time.Minute)
	end := r.end.Truncate(time.Minute)

	switch {
	case start != r.start:
		return fmt.Errorf("start %s isn't rounded to minutes", r.start)
	case end != r.end:
		return fmt.Errorf("end %s isn't rounded to minutes", r.end)
	default:
		return nil
	}
}

// type check
var _ json.Marshaler = (*Weekly)(nil)

// MarshalJSON implements the [json.Marshaler] interface for *Weekly.
func (w *Weekly) MarshalJSON() (data []byte, err error) {
	c := &weeklyConfigJSON{
		TimeZone:  w.location.String(),
		Sunday:    w.days[time.Sunday].toDayConfigJSON(),
		Monday:    w.days[time.Monday].toDayConfigJSON(),
		Tuesday:   w.days[time.Tuesday].toDayConfigJSON(),
		Wednesday: w.days[time.Wednesday].toDayConfigJSON(),
		Thursday:  w.days[time.Thursday].toDayConfigJSON(),
		Friday:    w.days[time.Friday].toDayConfigJSON(),
		Saturday:  w.days[time.Saturday].toDayConfigJSON(),
	}

	return json.Marshal(c)
}

// type check
var _ yaml.Marshaler = (*Weekly)(nil)

// MarshalYAML implements the [yaml.Marshaler] interface for *Weekly.
func (w *Weekly) MarshalYAML() (v any, err error) {
	return weeklyConfigYAML{
		TimeZone: w.location.String(),
		Sunday: dayConfigYAML{
			Start: timeutil.Duration(w.days[time.Sunday].start),
			End:   timeutil.Duration(w.days[time.Sunday].end),
		},
		Monday: dayConfigYAML{
			Start: timeutil.Duration(w.days[time.Monday].start),
			End:   timeutil.Duration(w.days[time.Monday].end),
		},
		Tuesday: dayConfigYAML{
			Start: timeutil.Duration(w.days[time.Tuesday].start),
			End:   timeutil.Duration(w.days[time.Tuesday].end),
		},
		Wednesday: dayConfigYAML{
			Start: timeutil.Duration(w.days[time.Wednesday].start),
			End:   timeutil.Duration(w.days[time.Wednesday].end),
		},
		Thursday: dayConfigYAML{
			Start: timeutil.Duration(w.days[time.Thursday].start),
			End:   timeutil.Duration(w.days[time.Thursday].end),
		},
		Friday: dayConfigYAML{
			Start: timeutil.Duration(w.days[time.Friday].start),
			End:   timeutil.Duration(w.days[time.Friday].end),
		},
		Saturday: dayConfigYAML{
			Start: timeutil.Duration(w.days[time.Saturday].start),
			End:   timeutil.Duration(w.days[time.Saturday].end),
		},
	}, nil
}

// dayRange represents a single interval within a day.  The interval begins at
// start and ends before end.  That is, it contains a time point T if start <=
// T < end.
type dayRange struct {
	// start is an offset from the beginning of the day.  It must be greater
	// than or equal to zero and less than 24h.
	start time.Duration

	// end is an offset from the beginning of the day.  It must be greater than
	// or equal to zero and less than or equal to 24h.
	end time.Duration
}

// validate returns the day range validation errors, if any.
func (r dayRange) validate() (err error) {
	switch {
	case r == dayRange{}:
		return nil
	case r.start < 0:
		return fmt.Errorf("start %s is negative", r.start)
	case r.end < 0:
		return fmt.Errorf("end %s is negative", r.end)
	case r.start >= r.end:
		return fmt.Errorf("start %s is greater or equal to end %s", r.start, r.end)
	case r.start >= maxDayRange:
		return fmt.Errorf("start %s is greater or equal to %s", r.start, maxDayRange)
	case r.end > maxDayRange:
		return fmt.Errorf("end %s is greater than %s", r.end, maxDayRange)
	default:
		return nil
	}
}

// contains returns true if start <= offset < end, where offset is the time
// duration from the beginning of the day.
func (r *dayRange) contains(offset time.Duration) (ok bool) {
	return r.start <= offset && offset < r.end
}

// toDayConfigJSON returns nil if the day range is empty, otherwise returns
// initialized JSON configuration of the day range.
func (r dayRange) toDayConfigJSON() (j *dayConfigJSON) {
	if (r == dayRange{}) {
		return nil
	}

	return &dayConfigJSON{
		Start: aghhttp.JSONDuration(r.start),
		End:   aghhttp.JSONDuration(r.end),
	}
}

// weeklyConfigJSON is the JSON configuration structure of Weekly.
type weeklyConfigJSON struct {
	// Days of the week.

	Sunday    *dayConfigJSON `json:"sun,omitempty"`
	Monday    *dayConfigJSON `json:"mon,omitempty"`
	Tuesday   *dayConfigJSON `json:"tue,omitempty"`
	Wednesday *dayConfigJSON `json:"wed,omitempty"`
	Thursday  *dayConfigJSON `json:"thu,omitempty"`
	Friday    *dayConfigJSON `json:"fri,omitempty"`
	Saturday  *dayConfigJSON `json:"sat,omitempty"`

	// TimeZone is the local time zone.
	TimeZone string `json:"time_zone"`
}

// dayConfigJSON is the JSON configuration structure of dayRange.
type dayConfigJSON struct {
	Start aghhttp.JSONDuration `json:"start"`
	End   aghhttp.JSONDuration `json:"end"`
}

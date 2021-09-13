// Package aghtime defines some types for convenient work with time values.
package aghtime

import (
	"time"

	"github.com/AdguardTeam/golibs/errors"
)

// Duration is a wrapper for time.Duration providing functionality for encoding.
type Duration struct {
	// time.Duration is embedded here to avoid implementing all the methods.
	time.Duration
}

// String implements the fmt.Stringer interface for Duration.  It wraps
// time.Duration.String method and additionally cuts off non-leading zero values
// of minutes and seconds.  Some values which are differ between the
// implementations:
//
//   Duration:   "1m", time.Duration:   "1m0s"
//   Duration:   "1h", time.Duration: "1h0m0s"
//   Duration: "1h1m", time.Duration: "1h1m0s"
//
func (d Duration) String() (str string) {
	str = d.Duration.String()

	const (
		tailMin    = len(`0s`)
		tailMinSec = len(`0m0s`)

		secsInHour = time.Hour / time.Second
		minsInHour = time.Hour / time.Minute
	)

	switch rounded := d.Duration / time.Second; {
	case
		rounded == 0,
		rounded*time.Second != d.Duration,
		rounded%60 != 0:
		// Return the uncut value if it's either equal to zero or has
		// fractions of a second or even whole seconds in it.
		return str

	case (rounded%secsInHour)/minsInHour != 0:
		return str[:len(str)-tailMin]

	default:
		return str[:len(str)-tailMinSec]
	}
}

// MarshalText implements the encoding.TextMarshaler interface for Duration.
func (d Duration) MarshalText() (text []byte, err error) {
	return []byte(d.String()), nil
}

// UnmarshalText implements the encoding.TextUnmarshaler interface for
// *Duration.
//
// TODO(e.burkov): Make it able to parse larger units like days.
func (d *Duration) UnmarshalText(b []byte) (err error) {
	defer func() { err = errors.Annotate(err, "unmarshaling duration: %w") }()

	d.Duration, err = time.ParseDuration(string(b))

	return err
}

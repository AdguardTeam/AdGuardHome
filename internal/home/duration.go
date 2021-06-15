package home

import (
	"time"

	"github.com/AdguardTeam/golibs/errors"
)

// Duration is a wrapper for time.Duration providing functionality for encoding.
type Duration struct {
	// time.Duration is embedded here to avoid implementing all the methods.
	time.Duration
}

// MarshalText implements the encoding.TextMarshaler interface for Duration.
func (d Duration) MarshalText() (text []byte, err error) {
	return []byte(d.String()), nil
}

// UnmarshalText implements the encoding.TextUnmarshaler interface for
// *Duration.
func (d *Duration) UnmarshalText(b []byte) (err error) {
	defer func() { err = errors.Annotate(err, "unmarshalling duration: %w") }()

	d.Duration, err = time.ParseDuration(string(b))

	return err
}

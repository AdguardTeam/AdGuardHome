package websvc

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"time"

	"github.com/AdguardTeam/golibs/log"
)

// JSON Utilities

// jsonTime is a time.Time that can be decoded from JSON and encoded into JSON
// according to our API conventions.
type jsonTime time.Time

// type check
var _ json.Marshaler = jsonTime{}

// nsecPerMsec is the number of nanoseconds in a millisecond.
const nsecPerMsec = float64(time.Millisecond / time.Nanosecond)

// MarshalJSON implements the json.Marshaler interface for jsonTime.  err is
// always nil.
func (t jsonTime) MarshalJSON() (b []byte, err error) {
	msec := float64(time.Time(t).UnixNano()) / nsecPerMsec
	b = strconv.AppendFloat(nil, msec, 'f', 3, 64)

	return b, nil
}

// type check
var _ json.Unmarshaler = (*jsonTime)(nil)

// UnmarshalJSON implements the json.Marshaler interface for *jsonTime.
func (t *jsonTime) UnmarshalJSON(b []byte) (err error) {
	if t == nil {
		return fmt.Errorf("json time is nil")
	}

	msec, err := strconv.ParseFloat(string(b), 64)
	if err != nil {
		return fmt.Errorf("parsing json time: %w", err)
	}

	*t = jsonTime(time.Unix(0, int64(msec*nsecPerMsec)).UTC())

	return nil
}

// writeJSONResponse encodes v into w and logs any errors it encounters.  r is
// used to get additional information from the request.
func writeJSONResponse(w io.Writer, r *http.Request, v any) {
	err := json.NewEncoder(w).Encode(v)
	if err != nil {
		log.Error("websvc: writing resp to %s %s: %s", r.Method, r.URL.Path, err)
	}
}

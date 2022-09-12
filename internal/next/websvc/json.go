package websvc

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"time"

	"github.com/AdguardTeam/AdGuardHome/internal/aghhttp"
	"github.com/AdguardTeam/golibs/log"
)

// JSON Utilities

// nsecPerMsec is the number of nanoseconds in a millisecond.
const nsecPerMsec = float64(time.Millisecond / time.Nanosecond)

// JSONDuration is a time.Duration that can be decoded from JSON and encoded
// into JSON according to our API conventions.
type JSONDuration time.Duration

// type check
var _ json.Marshaler = JSONDuration(0)

// MarshalJSON implements the json.Marshaler interface for JSONDuration.  err is
// always nil.
func (d JSONDuration) MarshalJSON() (b []byte, err error) {
	msec := float64(time.Duration(d)) / nsecPerMsec
	b = strconv.AppendFloat(nil, msec, 'f', -1, 64)

	return b, nil
}

// type check
var _ json.Unmarshaler = (*JSONDuration)(nil)

// UnmarshalJSON implements the json.Marshaler interface for *JSONDuration.
func (d *JSONDuration) UnmarshalJSON(b []byte) (err error) {
	if d == nil {
		return fmt.Errorf("json duration is nil")
	}

	msec, err := strconv.ParseFloat(string(b), 64)
	if err != nil {
		return fmt.Errorf("parsing json time: %w", err)
	}

	*d = JSONDuration(int64(msec * nsecPerMsec))

	return nil
}

// JSONTime is a time.Time that can be decoded from JSON and encoded into JSON
// according to our API conventions.
type JSONTime time.Time

// type check
var _ json.Marshaler = JSONTime{}

// MarshalJSON implements the json.Marshaler interface for JSONTime.  err is
// always nil.
func (t JSONTime) MarshalJSON() (b []byte, err error) {
	msec := float64(time.Time(t).UnixNano()) / nsecPerMsec
	b = strconv.AppendFloat(nil, msec, 'f', -1, 64)

	return b, nil
}

// type check
var _ json.Unmarshaler = (*JSONTime)(nil)

// UnmarshalJSON implements the json.Marshaler interface for *JSONTime.
func (t *JSONTime) UnmarshalJSON(b []byte) (err error) {
	if t == nil {
		return fmt.Errorf("json time is nil")
	}

	msec, err := strconv.ParseFloat(string(b), 64)
	if err != nil {
		return fmt.Errorf("parsing json time: %w", err)
	}

	*t = JSONTime(time.Unix(0, int64(msec*nsecPerMsec)).UTC())

	return nil
}

// writeJSONResponse encodes v into w and logs any errors it encounters.  r is
// used to get additional information from the request.
func writeJSONResponse(w http.ResponseWriter, r *http.Request, v any) {
	// TODO(a.garipov): Put some of these to a middleware.
	h := w.Header()
	h.Set(aghhttp.HdrNameContentType, aghhttp.HdrValApplicationJSON)
	h.Set(aghhttp.HdrNameServer, aghhttp.UserAgent())

	err := json.NewEncoder(w).Encode(v)
	if err != nil {
		log.Error("websvc: writing resp to %s %s: %s", r.Method, r.URL.Path, err)
	}
}

// writeHTTPError is a helper for logging and writing HTTP errors.
//
// TODO(a.garipov): Improve codes, and add JSON error codes.
func writeHTTPError(w http.ResponseWriter, r *http.Request, err error) {
	log.Error("websvc: %s %s: %s", r.Method, r.URL.Path, err)

	w.WriteHeader(http.StatusUnprocessableEntity)
	_, werr := io.WriteString(w, err.Error())
	if werr != nil {
		log.Debug("websvc: writing error resp to %s %s: %s", r.Method, r.URL.Path, werr)
	}
}

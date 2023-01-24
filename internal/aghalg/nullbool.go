package aghalg

import (
	"bytes"
	"encoding/json"
	"fmt"

	"github.com/AdguardTeam/golibs/mathutil"
)

// NullBool is a nullable boolean.  Use these in JSON requests and responses
// instead of pointers to bool.
type NullBool uint8

// NullBool values
const (
	NBNull NullBool = iota
	NBTrue
	NBFalse
)

// String implements the fmt.Stringer interface for NullBool.
func (nb NullBool) String() (s string) {
	switch nb {
	case NBNull:
		return "null"
	case NBTrue:
		return "true"
	case NBFalse:
		return "false"
	}

	return fmt.Sprintf("!invalid NullBool %d", uint8(nb))
}

// BoolToNullBool converts a bool into a NullBool.
func BoolToNullBool(cond bool) (nb NullBool) {
	return NBFalse - mathutil.BoolToNumber[NullBool](cond)
}

// type check
var _ json.Marshaler = NBNull

// MarshalJSON implements the json.Marshaler interface for NullBool.
func (nb NullBool) MarshalJSON() (b []byte, err error) {
	return []byte(nb.String()), nil
}

// type check
var _ json.Unmarshaler = (*NullBool)(nil)

// UnmarshalJSON implements the json.Unmarshaler interface for *NullBool.
func (nb *NullBool) UnmarshalJSON(b []byte) (err error) {
	if len(b) == 0 || bytes.Equal(b, []byte("null")) {
		*nb = NBNull
	} else if bytes.Equal(b, []byte("true")) {
		*nb = NBTrue
	} else if bytes.Equal(b, []byte("false")) {
		*nb = NBFalse
	} else {
		return fmt.Errorf("unmarshalling json data into aghalg.NullBool: bad value %q", b)
	}

	return nil
}

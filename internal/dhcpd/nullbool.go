package dhcpd

import (
	"bytes"
	"fmt"
)

// nullBool is a nullable boolean.  Use these in JSON requests and responses
// instead of pointers to bool.
//
// TODO(a.garipov): Inspect uses of *bool, move this type into some new package
// if we need it somewhere else.
type nullBool uint8

// nullBool values
const (
	nbNull nullBool = iota
	nbTrue
	nbFalse
)

// String implements the fmt.Stringer interface for nullBool.
func (nb nullBool) String() (s string) {
	switch nb {
	case nbNull:
		return "null"
	case nbTrue:
		return "true"
	case nbFalse:
		return "false"
	}

	return fmt.Sprintf("!invalid nullBool %d", uint8(nb))
}

// boolToNullBool converts a bool into a nullBool.
func boolToNullBool(cond bool) (nb nullBool) {
	if cond {
		return nbTrue
	}

	return nbFalse
}

// UnmarshalJSON implements the json.Unmarshaler interface for *nullBool.
func (nb *nullBool) UnmarshalJSON(b []byte) (err error) {
	if len(b) == 0 || bytes.Equal(b, []byte("null")) {
		*nb = nbNull
	} else if bytes.Equal(b, []byte("true")) {
		*nb = nbTrue
	} else if bytes.Equal(b, []byte("false")) {
		*nb = nbFalse
	} else {
		return fmt.Errorf("invalid nullBool value %q", b)
	}

	return nil
}

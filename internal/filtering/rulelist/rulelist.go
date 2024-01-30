// Package rulelist contains the implementation of the standard rule-list
// filter that wraps an urlfilter filtering-engine.
//
// TODO(a.garipov): Add a new update worker.
package rulelist

import (
	"fmt"

	"github.com/c2h5oh/datasize"
	"github.com/google/uuid"
)

// DefaultRuleBufSize is the default length of a buffer used to read a line with
// a filtering rule, in bytes.
//
// TODO(a.garipov): Consider using [datasize.ByteSize].  It is currently only
// used as an int.
const DefaultRuleBufSize = 1024

// DefaultMaxRuleListSize is the default maximum filtering-rule list size.
const DefaultMaxRuleListSize = 64 * datasize.MB

// URLFilterID is a semantic type-alias for IDs used for working with package
// urlfilter.
type URLFilterID = int

// UID is the type for the unique IDs of filtering-rule lists.
type UID uuid.UUID

// NewUID returns a new filtering-rule list UID.  Any error returned is an error
// from the cryptographic randomness reader.
func NewUID() (uid UID, err error) {
	uuidv7, err := uuid.NewV7()

	return UID(uuidv7), err
}

// MustNewUID is a wrapper around [NewUID] that panics if there is an error.
func MustNewUID() (uid UID) {
	uid, err := NewUID()
	if err != nil {
		panic(fmt.Errorf("unexpected uuidv7 error: %w", err))
	}

	return uid
}

// type check
var _ fmt.Stringer = UID{}

// String implements the [fmt.Stringer] interface for UID.
func (id UID) String() (s string) {
	return uuid.UUID(id).String()
}

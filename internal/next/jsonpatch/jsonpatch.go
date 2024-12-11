// Package jsonpatch contains utilities for JSON Merge Patch APIs.
//
// See https://www.rfc-editor.org/rfc/rfc7396.
package jsonpatch

import (
	"bytes"
	"encoding/json"

	"github.com/AdguardTeam/golibs/errors"
)

// NonRemovable is a type that prevents JSON null from being used to try and
// remove a value.
type NonRemovable[T any] struct {
	Value T
	IsSet bool
}

// type check
var _ json.Unmarshaler = (*NonRemovable[struct{}])(nil)

// UnmarshalJSON implements the [json.Unmarshaler] interface for *NonRemovable.
func (v *NonRemovable[T]) UnmarshalJSON(b []byte) (err error) {
	if v == nil {
		return errors.Error("jsonpatch.NonRemovable is nil")
	}

	if bytes.Equal(b, []byte("null")) {
		return errors.Error("property cannot be removed")
	}

	v.IsSet = true

	return json.Unmarshal(b, &v.Value)
}

// Set sets ptr if the value has been provided.
func (v NonRemovable[T]) Set(ptr *T) {
	if v.IsSet {
		*ptr = v.Value
	}
}

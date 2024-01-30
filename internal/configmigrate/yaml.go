package configmigrate

import (
	"fmt"
)

type (
	// yarr is the convenience alias for YAML array.
	yarr = []any

	// yobj is the convenience alias for YAML key-value object.
	yobj = map[string]any
)

// fieldVal returns the value of type T for key from obj.  Use [any] if the
// field's type doesn't matter.
func fieldVal[T any](obj yobj, key string) (v T, ok bool, err error) {
	val, ok := obj[key]
	if !ok {
		return v, false, nil
	}

	if val == nil {
		return v, true, nil
	}

	v, ok = val.(T)
	if !ok {
		return v, false, fmt.Errorf("unexpected type of %q: %T", key, val)
	}

	return v, true, nil
}

// moveVal copies the value for srcKey from src into dst for dstKey and deletes
// it from src.
func moveVal[T any](src, dst yobj, srcKey, dstKey string) (err error) {
	newVal, ok, err := fieldVal[T](src, srcKey)
	if !ok {
		return err
	}

	dst[dstKey] = newVal
	delete(src, srcKey)

	return nil
}

// moveSameVal moves the value for key from src into dst.
func moveSameVal[T any](src, dst yobj, key string) (err error) {
	return moveVal[T](src, dst, key, key)
}

// coalesceError returns the first non-nil error.  It is named after function
// COALESCE in SQL.  If all errors are nil, it returns nil.
//
// TODO(e.burkov): Replace with [errors.Join].
//
// TODO(a.garipov): Think of ways to merge with [aghalg.Coalesce].
func coalesceError(errors ...error) (res error) {
	for _, err := range errors {
		if err != nil {
			return err
		}
	}

	return nil
}

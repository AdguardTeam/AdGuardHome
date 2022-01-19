// Package aghalg contains common generic algorithms and data structures.
//
// TODO(a.garipov): Update to use type parameters in Go 1.18.
package aghalg

import (
	"fmt"
	"sort"
)

// comparable is an alias for interface{}.  Values passed as arguments of this
// type alias must be comparable.
//
// TODO(a.garipov): Remove in Go 1.18.
type comparable = interface{}

// UniqChecker allows validating uniqueness of comparable items.
type UniqChecker map[comparable]int64

// Add adds a value to the validator.  v must not be nil.
func (uc UniqChecker) Add(elems ...comparable) {
	for _, e := range elems {
		uc[e]++
	}
}

// Merge returns a validator containing data from both v and other.
func (uc UniqChecker) Merge(other UniqChecker) (merged UniqChecker) {
	merged = make(UniqChecker, len(uc)+len(other))
	for elem, num := range uc {
		merged[elem] += num
	}

	for elem, num := range other {
		merged[elem] += num
	}

	return merged
}

// Validate returns an error enumerating all elements that aren't unique.
// isBefore is an optional sorting function to make the error message
// deterministic.
func (uc UniqChecker) Validate(isBefore func(a, b comparable) (less bool)) (err error) {
	var dup []comparable
	for elem, num := range uc {
		if num > 1 {
			dup = append(dup, elem)
		}
	}

	if len(dup) == 0 {
		return nil
	}

	if isBefore != nil {
		sort.Slice(dup, func(i, j int) (less bool) {
			return isBefore(dup[i], dup[j])
		})
	}

	return fmt.Errorf("duplicated values: %v", dup)
}

// IntIsBefore is a helper sort function for UniqChecker.Validate.
// a and b must be of type int.
func IntIsBefore(a, b comparable) (less bool) {
	return a.(int) < b.(int)
}

// StringIsBefore is a helper sort function for UniqChecker.Validate.
// a and b must be of type string.
func StringIsBefore(a, b comparable) (less bool) {
	return a.(string) < b.(string)
}

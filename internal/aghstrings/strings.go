// Package aghstrings contains utilities dealing with strings.
package aghstrings

import (
	"strings"
)

// CloneSliceOrEmpty returns the copy of a or empty strings slice if a is nil.
func CloneSliceOrEmpty(a []string) (b []string) {
	return append([]string{}, a...)
}

// CloneSlice returns the exact copy of a.
func CloneSlice(a []string) (b []string) {
	if a == nil {
		return nil
	}

	return CloneSliceOrEmpty(a)
}

// Coalesce returns the first non-empty string.  It is named after the function
// COALESCE in SQL except that since strings in Go are non-nullable, it uses an
// empty string as a NULL value.  If strs or all it's elements are empty, it
// returns an empty string.
func Coalesce(strs ...string) (res string) {
	for _, s := range strs {
		if s != "" {
			return s
		}
	}

	return ""
}

// FilterOut returns a copy of strs with all strings for which f returned true
// removed.
func FilterOut(strs []string, f func(s string) (ok bool)) (filtered []string) {
	for _, s := range strs {
		if !f(s) {
			filtered = append(filtered, s)
		}
	}

	return filtered
}

// InSlice checks if string is in the slice of strings.
func InSlice(strs []string, str string) (ok bool) {
	for _, s := range strs {
		if s == str {
			return true
		}
	}

	return false
}

// IsCommentOrEmpty returns true of the string starts with a "#" character or is
// an empty string.
func IsCommentOrEmpty(s string) (ok bool) {
	return len(s) == 0 || s[0] == '#'
}

// SplitNext splits string by a byte and returns the first chunk skipping empty
// ones.  Whitespaces are trimmed.
func SplitNext(s *string, sep rune) (chunk string) {
	if s == nil {
		return chunk
	}

	i := strings.IndexByte(*s, byte(sep))
	if i == -1 {
		chunk = *s
		*s = ""

		return strings.TrimSpace(chunk)
	}

	chunk = (*s)[:i]
	*s = (*s)[i+1:]
	var j int
	var r rune
	for j, r = range *s {
		if r != sep {
			break
		}
	}

	*s = (*s)[j:]

	return strings.TrimSpace(chunk)
}

// WriteToBuilder is a convenient wrapper for strings.(*Builder).WriteString
// that deals with multiple strings and ignores errors that are guaranteed to be
// nil.
func WriteToBuilder(b *strings.Builder, strs ...string) {
	// TODO(e.burkov): Recover from panic?
	for _, s := range strs {
		_, _ = b.WriteString(s)
	}
}

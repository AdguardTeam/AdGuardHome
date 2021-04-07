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

// InSlice checks if string is in the slice of strings.
func InSlice(strs []string, str string) (ok bool) {
	for _, s := range strs {
		if s == str {
			return true
		}
	}

	return false
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

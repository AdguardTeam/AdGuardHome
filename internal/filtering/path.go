package filtering

import (
	"fmt"
	"path/filepath"
)

// pathMatchesAny returns true if filePath matches one of globs.  globs must be
// valid.  filePath must be absolute and clean.  If globs are empty,
// pathMatchesAny returns false.
//
// TODO(a.garipov): Move to golibs?
func pathMatchesAny(globs []string, filePath string) (ok bool) {
	if len(globs) == 0 {
		return false
	}

	clean, err := filepath.Abs(filePath)
	if err != nil {
		panic(fmt.Errorf("pathMatchesAny: %w", err))
	} else if clean != filePath {
		panic(fmt.Errorf("pathMatchesAny: filepath %q is not absolute", filePath))
	}

	for _, g := range globs {
		ok, err = filepath.Match(g, filePath)
		if err != nil {
			panic(fmt.Errorf("pathMatchesAny: bad pattern: %w", err))
		}

		if ok {
			return true
		}
	}

	return false
}

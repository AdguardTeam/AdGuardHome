package aghos

import (
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/AdguardTeam/AdGuardHome/internal/aghio"
	"github.com/AdguardTeam/golibs/errors"
	"github.com/AdguardTeam/golibs/stringutil"
)

// FileWalker is the signature of a function called for files in the file tree.
// As opposed to filepath.Walk it only walk the files (not directories) matching
// the provided pattern and those returned by function itself.  All patterns
// should be valid for filepath.Glob.  If cont is false, the walking terminates.
// Each opened file is also limited for reading to MaxWalkedFileSize.
//
// TODO(e.burkov):  Consider moving to the separate package like pathutil.
//
// TODO(e.burkov):  Think about passing filename or any additional data.
type FileWalker func(r io.Reader) (patterns []string, cont bool, err error)

// MaxWalkedFileSize is the maximum length of the file that FileWalker can
// check.
const MaxWalkedFileSize = 1024 * 1024

// checkFile tries to open and process a single file located on sourcePath.
func checkFile(c FileWalker, sourcePath string) (patterns []string, cont bool, err error) {
	var f *os.File
	f, err = os.Open(sourcePath)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			// Ignore non-existing files since this may only happen
			// when the file was removed after filepath.Glob matched
			// it.
			return nil, true, nil
		}

		return nil, false, err
	}
	defer func() { err = errors.WithDeferred(err, f.Close()) }()

	var r io.Reader
	// Ignore the error since LimitReader function returns error only if
	// passed limit value is less than zero, but the constant used.
	//
	// TODO(e.burkov):  Make variable.
	r, _ = aghio.LimitReader(f, MaxWalkedFileSize)

	return c(r)
}

// handlePatterns parses the patterns and ignores duplicates using srcSet.
// srcSet must be non-nil.
func handlePatterns(srcSet *stringutil.Set, patterns ...string) (sub []string, err error) {
	sub = make([]string, 0, len(patterns))
	for _, p := range patterns {
		var matches []string
		matches, err = filepath.Glob(p)
		if err != nil {
			// Enrich error with the pattern because filepath.Glob
			// doesn't do it.
			return nil, fmt.Errorf("invalid pattern %q: %w", p, err)
		}

		for _, m := range matches {
			if srcSet.Has(m) {
				continue
			}

			srcSet.Add(m)
			sub = append(sub, m)
		}
	}

	return sub, nil
}

// Walk starts walking the files defined by initPattern.  It only returns true
// if c signed to stop walking.
func (c FileWalker) Walk(initPattern string) (ok bool, err error) {
	// The slice of sources keeps the order in which the files are walked
	// since srcSet.Values() returns strings in undefined order.
	srcSet := stringutil.NewSet()
	var src []string
	src, err = handlePatterns(srcSet, initPattern)
	if err != nil {
		return false, err
	}

	var filename string
	defer func() { err = errors.Annotate(err, "checking %q: %w", filename) }()

	for i := 0; i < len(src); i++ {
		var patterns []string
		var cont bool
		filename = src[i]
		patterns, cont, err = checkFile(c, src[i])
		if err != nil {
			return false, err
		}

		if !cont {
			return true, nil
		}

		var subsrc []string
		subsrc, err = handlePatterns(srcSet, patterns...)
		if err != nil {
			return false, err
		}

		src = append(src, subsrc...)
	}

	return false, nil
}

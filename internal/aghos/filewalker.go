package aghos

import (
	"fmt"
	"io"
	"io/fs"

	"github.com/AdguardTeam/golibs/container"
	"github.com/AdguardTeam/golibs/errors"
)

// FileWalker is the signature of a function called for files in the file tree.
// As opposed to filepath.Walk it only walk the files (not directories) matching
// the provided pattern and those returned by function itself.  All patterns
// should be valid for fs.Glob.  If FileWalker returns false for cont then
// walking terminates.  Prefer using bufio.Scanner to read the r since the input
// is not limited.
//
// TODO(e.burkov, a.garipov):  Move into another package like aghfs.
//
// TODO(e.burkov):  Think about passing filename or any additional data.
type FileWalker func(r io.Reader) (patterns []string, cont bool, err error)

// checkFile tries to open and process a single file located on sourcePath in
// the specified fsys.  The path is skipped if it's a directory.
func checkFile(
	fsys fs.FS,
	c FileWalker,
	sourcePath string,
) (patterns []string, cont bool, err error) {
	var f fs.File
	f, err = fsys.Open(sourcePath)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			// Ignore non-existing files since this may only happen when the
			// file was removed after filepath.Glob matched it.
			return nil, true, nil
		}

		return nil, false, err
	}
	defer func() { err = errors.WithDeferred(err, f.Close()) }()

	var fi fs.FileInfo
	if fi, err = f.Stat(); err != nil {
		return nil, true, err
	} else if fi.IsDir() {
		// Skip the directories.
		return nil, true, nil
	}

	return c(f)
}

// handlePatterns parses the patterns in fsys and ignores duplicates using
// srcSet.  srcSet must be non-nil.
func handlePatterns(
	fsys fs.FS,
	srcSet *container.MapSet[string],
	patterns ...string,
) (sub []string, err error) {
	sub = make([]string, 0, len(patterns))
	for _, p := range patterns {
		var matches []string
		matches, err = fs.Glob(fsys, p)
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

// Walk starts walking the files in fsys defined by patterns from initial.
// It only returns true if fw signed to stop walking.
func (fw FileWalker) Walk(fsys fs.FS, initial ...string) (ok bool, err error) {
	// The slice of sources keeps the order in which the files are walked since
	// srcSet.Values() returns strings in undefined order.
	srcSet := container.NewMapSet[string]()
	var src []string
	src, err = handlePatterns(fsys, srcSet, initial...)
	if err != nil {
		return false, err
	}

	var filename string
	defer func() { err = errors.Annotate(err, "checking %q: %w", filename) }()

	// TODO(e.burkov):  Redo this loop, as it modifies the very same slice it
	// iterates over.
	for i := 0; i < len(src); i++ {
		var patterns []string
		var cont bool
		filename = src[i]
		patterns, cont, err = checkFile(fsys, fw, src[i])
		if err != nil {
			return false, err
		}

		if !cont {
			return true, nil
		}

		var subsrc []string
		subsrc, err = handlePatterns(fsys, srcSet, patterns...)
		if err != nil {
			return false, err
		}

		src = append(src, subsrc...)
	}

	return false, nil
}

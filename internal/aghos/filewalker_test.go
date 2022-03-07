package aghos

import (
	"bufio"
	"io"
	"io/fs"
	"path"
	"testing"
	"testing/fstest"

	"github.com/AdguardTeam/golibs/errors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFileWalker_Walk(t *testing.T) {
	const attribute = `000`

	makeFileWalker := func(_ string) (fw FileWalker) {
		return func(r io.Reader) (patterns []string, cont bool, err error) {
			s := bufio.NewScanner(r)
			for s.Scan() {
				line := s.Text()
				if line == attribute {
					return nil, false, nil
				}

				if len(line) != 0 {
					patterns = append(patterns, path.Join(".", line))
				}
			}

			return patterns, true, s.Err()
		}
	}

	const nl = "\n"

	testCases := []struct {
		testFS      fstest.MapFS
		want        assert.BoolAssertionFunc
		initPattern string
		name        string
	}{{
		name: "simple",
		testFS: fstest.MapFS{
			"simple_0001.txt": &fstest.MapFile{Data: []byte(attribute + nl)},
		},
		initPattern: "simple_0001.txt",
		want:        assert.True,
	}, {
		name: "chain",
		testFS: fstest.MapFS{
			"chain_0001.txt": &fstest.MapFile{Data: []byte(`chain_0002.txt` + nl)},
			"chain_0002.txt": &fstest.MapFile{Data: []byte(`chain_0003.txt` + nl)},
			"chain_0003.txt": &fstest.MapFile{Data: []byte(attribute + nl)},
		},
		initPattern: "chain_0001.txt",
		want:        assert.True,
	}, {
		name: "several",
		testFS: fstest.MapFS{
			"several_0001.txt": &fstest.MapFile{Data: []byte(`several_*` + nl)},
			"several_0002.txt": &fstest.MapFile{Data: []byte(`several_0001.txt` + nl)},
			"several_0003.txt": &fstest.MapFile{Data: []byte(attribute + nl)},
		},
		initPattern: "several_0001.txt",
		want:        assert.True,
	}, {
		name: "no",
		testFS: fstest.MapFS{
			"no_0001.txt": &fstest.MapFile{Data: []byte(nl)},
			"no_0002.txt": &fstest.MapFile{Data: []byte(nl)},
			"no_0003.txt": &fstest.MapFile{Data: []byte(nl)},
		},
		initPattern: "no_*",
		want:        assert.False,
	}, {
		name: "subdirectory",
		testFS: fstest.MapFS{
			path.Join("dir", "subdir_0002.txt"): &fstest.MapFile{
				Data: []byte(attribute + nl),
			},
			"subdir_0001.txt": &fstest.MapFile{Data: []byte(`dir/*`)},
		},
		initPattern: "subdir_0001.txt",
		want:        assert.True,
	}}

	for _, tc := range testCases {
		fw := makeFileWalker("")

		t.Run(tc.name, func(t *testing.T) {
			ok, err := fw.Walk(tc.testFS, tc.initPattern)
			require.NoError(t, err)

			tc.want(t, ok)
		})
	}

	t.Run("pattern_malformed", func(t *testing.T) {
		f := fstest.MapFS{}
		ok, err := makeFileWalker("").Walk(f, "[]")
		require.Error(t, err)

		assert.False(t, ok)
		assert.ErrorIs(t, err, path.ErrBadPattern)
	})

	t.Run("bad_filename", func(t *testing.T) {
		const filename = "bad_filename.txt"

		f := fstest.MapFS{
			filename: &fstest.MapFile{Data: []byte("[]")},
		}
		ok, err := FileWalker(func(r io.Reader) (patterns []string, cont bool, err error) {
			s := bufio.NewScanner(r)
			for s.Scan() {
				patterns = append(patterns, s.Text())
			}

			return patterns, true, s.Err()
		}).Walk(f, filename)
		require.Error(t, err)

		assert.False(t, ok)
		assert.ErrorIs(t, err, path.ErrBadPattern)
	})

	t.Run("itself_error", func(t *testing.T) {
		const rerr errors.Error = "returned error"

		f := fstest.MapFS{
			"mockfile.txt": &fstest.MapFile{Data: []byte(`mockdata`)},
		}

		ok, err := FileWalker(func(r io.Reader) (patterns []string, ok bool, err error) {
			return nil, true, rerr
		}).Walk(f, "*")
		require.ErrorIs(t, err, rerr)

		assert.False(t, ok)
	})
}

type errFS struct {
	fs.GlobFS
}

const errErrFSOpen errors.Error = "this error is always returned"

func (efs *errFS) Open(name string) (fs.File, error) {
	return nil, errErrFSOpen
}

func TestWalkerFunc_CheckFile(t *testing.T) {
	emptyFS := fstest.MapFS{}

	t.Run("non-existing", func(t *testing.T) {
		_, ok, err := checkFile(emptyFS, nil, "lol")
		require.NoError(t, err)

		assert.True(t, ok)
	})

	t.Run("invalid_argument", func(t *testing.T) {
		_, ok, err := checkFile(&errFS{}, nil, "")
		require.ErrorIs(t, err, errErrFSOpen)

		assert.False(t, ok)
	})

	t.Run("ignore_dirs", func(t *testing.T) {
		const dirName = "dir"

		testFS := fstest.MapFS{
			path.Join(dirName, "file"): &fstest.MapFile{Data: []byte{}},
		}

		patterns, ok, err := checkFile(testFS, nil, dirName)
		require.NoError(t, err)

		assert.Empty(t, patterns)
		assert.True(t, ok)
	})
}

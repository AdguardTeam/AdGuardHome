package aghos

import (
	"bufio"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"testing"

	"github.com/AdguardTeam/golibs/errors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// testFSDir maps entries' names to entries which should either be a testFSDir
// or byte slice.
type testFSDir map[string]interface{}

// testFSGen is used to generate a temporary filesystem consisting of
// directories and plain text files from itself.
type testFSGen testFSDir

// gen returns the name of top directory of the generated filesystem.
func (g testFSGen) gen(t *testing.T) (dirName string) {
	t.Helper()

	dirName = t.TempDir()
	g.rangeThrough(t, dirName)

	return dirName
}

func (g testFSGen) rangeThrough(t *testing.T, dirName string) {
	const perm fs.FileMode = 0o777

	for k, e := range g {
		switch e := e.(type) {
		case []byte:
			require.NoError(t, os.WriteFile(filepath.Join(dirName, k), e, perm))

		case testFSDir:
			newDir := filepath.Join(dirName, k)
			require.NoError(t, os.Mkdir(newDir, perm))

			testFSGen(e).rangeThrough(t, newDir)
		default:
			t.Fatalf("unexpected entry type %T", e)
		}
	}
}

func TestFileWalker_Walk(t *testing.T) {
	const attribute = `000`

	makeFileWalker := func(dirName string) (fw FileWalker) {
		return func(r io.Reader) (patterns []string, cont bool, err error) {
			s := bufio.NewScanner(r)
			for s.Scan() {
				line := s.Text()
				if line == attribute {
					return nil, false, nil
				}

				if len(line) != 0 {
					patterns = append(patterns, filepath.Join(dirName, line))
				}
			}

			return patterns, true, s.Err()
		}
	}

	const nl = "\n"

	testCases := []struct {
		name        string
		testFS      testFSGen
		initPattern string
		want        bool
	}{{
		name: "simple",
		testFS: testFSGen{
			"simple_0001.txt": []byte(attribute + nl),
		},
		initPattern: "simple_0001.txt",
		want:        true,
	}, {
		name: "chain",
		testFS: testFSGen{
			"chain_0001.txt": []byte(`chain_0002.txt` + nl),
			"chain_0002.txt": []byte(`chain_0003.txt` + nl),
			"chain_0003.txt": []byte(attribute + nl),
		},
		initPattern: "chain_0001.txt",
		want:        true,
	}, {
		name: "several",
		testFS: testFSGen{
			"several_0001.txt": []byte(`several_*` + nl),
			"several_0002.txt": []byte(`several_0001.txt` + nl),
			"several_0003.txt": []byte(attribute + nl),
		},
		initPattern: "several_0001.txt",
		want:        true,
	}, {
		name: "no",
		testFS: testFSGen{
			"no_0001.txt": []byte(nl),
			"no_0002.txt": []byte(nl),
			"no_0003.txt": []byte(nl),
		},
		initPattern: "no_*",
		want:        false,
	}, {
		name: "subdirectory",
		testFS: testFSGen{
			"dir": testFSDir{
				"subdir_0002.txt": []byte(attribute + nl),
			},
			"subdir_0001.txt": []byte(`dir/*`),
		},
		initPattern: "subdir_0001.txt",
		want:        true,
	}}

	for _, tc := range testCases {
		testDir := tc.testFS.gen(t)
		fw := makeFileWalker(testDir)

		t.Run(tc.name, func(t *testing.T) {
			ok, err := fw.Walk(filepath.Join(testDir, tc.initPattern))
			require.NoError(t, err)

			assert.Equal(t, tc.want, ok)
		})
	}

	t.Run("pattern_malformed", func(t *testing.T) {
		ok, err := makeFileWalker("").Walk("[]")
		require.Error(t, err)

		assert.False(t, ok)
		assert.ErrorIs(t, err, filepath.ErrBadPattern)
	})

	t.Run("bad_filename", func(t *testing.T) {
		dir := testFSGen{
			"bad_filename.txt": []byte("[]"),
		}.gen(t)
		fw := FileWalker(func(r io.Reader) (patterns []string, cont bool, err error) {
			s := bufio.NewScanner(r)
			for s.Scan() {
				patterns = append(patterns, s.Text())
			}

			return patterns, true, s.Err()
		})

		ok, err := fw.Walk(filepath.Join(dir, "bad_filename.txt"))
		require.Error(t, err)

		assert.False(t, ok)
		assert.ErrorIs(t, err, filepath.ErrBadPattern)
	})

	t.Run("itself_error", func(t *testing.T) {
		const rerr errors.Error = "returned error"

		dir := testFSGen{
			"mockfile.txt": []byte(`mockdata`),
		}.gen(t)

		ok, err := FileWalker(func(r io.Reader) (patterns []string, ok bool, err error) {
			return nil, true, rerr
		}).Walk(filepath.Join(dir, "*"))
		require.Error(t, err)
		require.False(t, ok)

		assert.ErrorIs(t, err, rerr)
	})
}

func TestWalkerFunc_CheckFile(t *testing.T) {
	t.Run("non-existing", func(t *testing.T) {
		_, ok, err := checkFile(nil, "lol")
		require.NoError(t, err)

		assert.True(t, ok)
	})

	t.Run("invalid_argument", func(t *testing.T) {
		const badPath = "\x00"

		_, ok, err := checkFile(nil, badPath)
		require.Error(t, err)

		assert.False(t, ok)
		// TODO(e.burkov):  Use assert.ErrorsIs within the error from
		// less platform-dependent package instead of syscall.EINVAL.
		//
		// See https://github.com/golang/go/issues/46849 and
		// https://github.com/golang/go/issues/30322.
		pathErr := &os.PathError{}
		require.ErrorAs(t, err, &pathErr)
		assert.Equal(t, "open", pathErr.Op)
		assert.Equal(t, badPath, pathErr.Path)
	})
}

package aghos

import (
	"io/fs"
	"path"
	"testing"
	"testing/fstest"

	"github.com/AdguardTeam/golibs/errors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// errFS is an fs.FS implementation, method Open of which always returns
// errFSOpen.
type errFS struct{}

// errFSOpen is returned from errFS.Open.
const errFSOpen errors.Error = "test open error"

// Open implements the fs.FS interface for *errFS.  fsys is always nil and err
// is always errFSOpen.
func (efs *errFS) Open(name string) (fsys fs.File, err error) {
	return nil, errFSOpen
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
		require.ErrorIs(t, err, errFSOpen)

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

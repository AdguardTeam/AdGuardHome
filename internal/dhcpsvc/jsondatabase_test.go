package dhcpsvc_test

import (
	"io/fs"
	"os"
	"path/filepath"
	"testing"

	"github.com/AdguardTeam/AdGuardHome/internal/dhcpsvc"
	"github.com/AdguardTeam/golibs/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// dbFilePath returns the path to the database file for the given test.
func dbFilePath(tb testing.TB) (p string) {
	return filepath.Join("testdata", tb.Name()+".json")
}

func TestJSONDatabase_Load(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name       string
		wantErrMsg string
		want       []*dhcpsvc.Lease
	}{{
		name:       "success",
		wantErrMsg: "",
		want:       testLeases,
	}, {
		name:       "no_file",
		wantErrMsg: "",
		want:       nil,
	}, {
		name:       "empty",
		wantErrMsg: "",
		want:       nil,
	}, {
		want: nil,
		name: "bad_format",
		wantErrMsg: "loading db: decoding db: json: cannot unmarshal array " +
			"into Go value of type dhcpsvc.jsonLeasesData",
	}}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			db := dhcpsvc.NewJSONDatabase(&dhcpsvc.JSONDatabaseConfig{
				Logger:   testLogger,
				FilePath: dbFilePath(t),
			})

			ctx := testutil.ContextWithTimeout(t, testTimeout)

			leases, err := db.Load(ctx)
			testutil.AssertErrorMsg(t, tc.wantErrMsg, err)

			// Use unordered comparison since the order of leases is undefined.
			assert.ElementsMatch(t, tc.want, leases)
		})
	}
}

func TestJSONDatabase_Store(t *testing.T) {
	t.Parallel()

	const dbFileName = "leases.json"

	dirToRemove := t.TempDir()
	require.NoError(t, os.RemoveAll(dirToRemove))

	testCases := []struct {
		wantErr error
		name    string
		path    string
		in      []*dhcpsvc.Lease
	}{{
		wantErr: nil,
		name:    "success",
		path:    filepath.Join(t.TempDir(), dbFileName),
		in:      testLeases,
	}, {
		wantErr: nil,
		name:    "nil",
		path:    filepath.Join(t.TempDir(), dbFileName),
		in:      nil,
	}, {
		wantErr: nil,
		name:    "empty",
		path:    filepath.Join(t.TempDir(), dbFileName),
		in:      []*dhcpsvc.Lease{},
	}, {
		wantErr: fs.ErrNotExist,
		name:    "bad_dir",
		path:    filepath.Join(dirToRemove, dbFileName),
		in:      testLeases,
	}}

	for _, tc := range testCases {
		db := dhcpsvc.NewJSONDatabase(&dhcpsvc.JSONDatabaseConfig{
			Logger:   testLogger,
			FilePath: tc.path,
		})

		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			ctx := testutil.ContextWithTimeout(t, testTimeout)

			err := db.Store(ctx, tc.in)

			assert.ErrorIs(t, err, tc.wantErr)
			if tc.wantErr != nil {
				return
			}

			require.FileExists(t, tc.path)

			content, err := os.ReadFile(tc.path)
			require.NoError(t, err)

			wantContent, err := os.ReadFile(dbFilePath(t))
			require.NoError(t, err)

			assert.Equal(t, wantContent, content)
		})
	}
}

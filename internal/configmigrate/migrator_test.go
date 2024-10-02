package configmigrate_test

import (
	"bytes"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/AdguardTeam/AdGuardHome/internal/configmigrate"
	"github.com/AdguardTeam/golibs/testutil"
	"github.com/stretchr/testify/require"
	"golang.org/x/crypto/bcrypt"
	yaml "gopkg.in/yaml.v3"
)

func TestMain(m *testing.M) {
	testutil.DiscardLogOutput(m)
}

// testdata is a virtual filesystem containing test data.
var testdata = os.DirFS("testdata")

// getField returns the value located at the given indexes in the given object.
// It fails the test if the value is not found or of the expected type.  The
// indexes can be either strings or integers, and are interpreted as map keys or
// array indexes, respectively.
func getField[T any](t require.TestingT, obj any, indexes ...any) (val T) {
	for _, index := range indexes {
		switch index := index.(type) {
		case string:
			require.IsType(t, map[string]any(nil), obj)
			typedObj := obj.(map[string]any)

			require.Contains(t, typedObj, index)
			obj = typedObj[index]
		case int:
			require.IsType(t, []any(nil), obj)
			typedObj := obj.([]any)

			require.Less(t, index, len(typedObj))
			obj = typedObj[index]
		default:
			t.Errorf("unexpected index type: %T", index)
			t.FailNow()
		}
	}

	require.IsType(t, val, obj)

	return obj.(T)
}

func TestMigrateConfig_Migrate(t *testing.T) {
	const (
		inputFileName  = "input.yml"
		outputFileName = "output.yml"
	)

	testCases := []struct {
		yamlEqFunc    func(t require.TestingT, expected, actual string, msgAndArgs ...any)
		name          string
		targetVersion uint
	}{{
		yamlEqFunc:    require.YAMLEq,
		name:          "v1",
		targetVersion: 1,
	}, {
		yamlEqFunc:    require.YAMLEq,
		name:          "v2",
		targetVersion: 2,
	}, {
		yamlEqFunc:    require.YAMLEq,
		name:          "v3",
		targetVersion: 3,
	}, {
		yamlEqFunc:    require.YAMLEq,
		name:          "v4",
		targetVersion: 4,
	}, {
		// Compare passwords separately because bcrypt hashes those with a
		// different salt every time.
		yamlEqFunc: func(t require.TestingT, expected, actual string, msgAndArgs ...any) {
			if h, ok := t.(interface{ Helper() }); ok {
				h.Helper()
			}

			var want, got map[string]any
			err := yaml.Unmarshal([]byte(expected), &want)
			require.NoError(t, err)

			err = yaml.Unmarshal([]byte(actual), &got)
			require.NoError(t, err)

			gotPass := getField[string](t, got, "users", 0, "password")
			wantPass := getField[string](t, want, "users", 0, "password")
			require.NoError(t, bcrypt.CompareHashAndPassword([]byte(gotPass), []byte(wantPass)))

			delete(getField[map[string]any](t, got, "users", 0), "password")
			delete(getField[map[string]any](t, want, "users", 0), "password")

			require.Equal(t, want, got, msgAndArgs...)
		},
		name:          "v5",
		targetVersion: 5,
	}, {
		yamlEqFunc:    require.YAMLEq,
		name:          "v6",
		targetVersion: 6,
	}, {
		yamlEqFunc:    require.YAMLEq,
		name:          "v7",
		targetVersion: 7,
	}, {
		yamlEqFunc:    require.YAMLEq,
		name:          "v8",
		targetVersion: 8,
	}, {
		yamlEqFunc:    require.YAMLEq,
		name:          "v9",
		targetVersion: 9,
	}, {
		yamlEqFunc:    require.YAMLEq,
		name:          "v10",
		targetVersion: 10,
	}, {
		yamlEqFunc:    require.YAMLEq,
		name:          "v11",
		targetVersion: 11,
	}, {
		yamlEqFunc:    require.YAMLEq,
		name:          "v12",
		targetVersion: 12,
	}, {
		yamlEqFunc:    require.YAMLEq,
		name:          "v13",
		targetVersion: 13,
	}, {
		yamlEqFunc:    require.YAMLEq,
		name:          "v14",
		targetVersion: 14,
	}, {
		yamlEqFunc:    require.YAMLEq,
		name:          "v15",
		targetVersion: 15,
	}, {
		yamlEqFunc:    require.YAMLEq,
		name:          "v16",
		targetVersion: 16,
	}, {
		yamlEqFunc:    require.YAMLEq,
		name:          "v17",
		targetVersion: 17,
	}, {
		yamlEqFunc:    require.YAMLEq,
		name:          "v18",
		targetVersion: 18,
	}, {
		yamlEqFunc:    require.YAMLEq,
		name:          "v19",
		targetVersion: 19,
	}, {
		yamlEqFunc:    require.YAMLEq,
		name:          "v20",
		targetVersion: 20,
	}, {
		yamlEqFunc:    require.YAMLEq,
		name:          "v21",
		targetVersion: 21,
	}, {
		yamlEqFunc:    require.YAMLEq,
		name:          "v22",
		targetVersion: 22,
	}, {
		yamlEqFunc:    require.YAMLEq,
		name:          "v23",
		targetVersion: 23,
	}, {
		yamlEqFunc:    require.YAMLEq,
		name:          "v24",
		targetVersion: 24,
	}, {
		yamlEqFunc:    require.YAMLEq,
		name:          "v25",
		targetVersion: 25,
	}, {
		yamlEqFunc:    require.YAMLEq,
		name:          "v26",
		targetVersion: 26,
	}, {
		yamlEqFunc:    require.YAMLEq,
		name:          "v27",
		targetVersion: 27,
	}}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			body, err := fs.ReadFile(testdata, path.Join(t.Name(), inputFileName))
			require.NoError(t, err)

			wantBody, err := fs.ReadFile(testdata, path.Join(t.Name(), outputFileName))
			require.NoError(t, err)

			migrator := configmigrate.New(&configmigrate.Config{
				WorkingDir: t.Name(),
				DataDir:    filepath.Join(t.Name(), "data"),
			})
			newBody, upgraded, err := migrator.Migrate(body, tc.targetVersion)
			require.NoError(t, err)
			require.True(t, upgraded)

			tc.yamlEqFunc(t, string(wantBody), string(newBody))
		})
	}
}

// TODO(a.garipov):  Consider ways of merging into the previous one.
func TestMigrateConfig_Migrate_v29(t *testing.T) {
	const (
		pathUnix       = `/path/to/file.txt`
		userDirPatUnix = `TestMigrateConfig_Migrate/v29/data/userfilters/*`

		pathWindows       = `C:\path\to\file.txt`
		userDirPatWindows = `TestMigrateConfig_Migrate\v29\data\userfilters\*`
	)

	pathToReplace := pathUnix
	patternToReplace := userDirPatUnix
	if runtime.GOOS == "windows" {
		pathToReplace = pathWindows
		patternToReplace = userDirPatWindows
	}

	body, err := fs.ReadFile(testdata, "TestMigrateConfig_Migrate/v29/input.yml")
	require.NoError(t, err)

	body = bytes.ReplaceAll(body, []byte("FILEPATH"), []byte(pathToReplace))

	wantBody, err := fs.ReadFile(testdata, "TestMigrateConfig_Migrate/v29/output.yml")
	require.NoError(t, err)

	wantBody = bytes.ReplaceAll(wantBody, []byte("FILEPATH"), []byte(pathToReplace))
	wantBody = bytes.ReplaceAll(wantBody, []byte("USERFILTERSPATH"), []byte(patternToReplace))

	migrator := configmigrate.New(&configmigrate.Config{
		WorkingDir: t.Name(),
		DataDir:    "TestMigrateConfig_Migrate/v29/data",
	})

	newBody, upgraded, err := migrator.Migrate(body, 29)
	require.NoError(t, err)
	require.True(t, upgraded)

	require.YAMLEq(t, string(wantBody), string(newBody))
}

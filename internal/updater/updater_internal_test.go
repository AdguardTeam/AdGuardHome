package updater

import (
	"context"
	"io/fs"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"syscall"
	"testing"
	"time"

	"github.com/AdguardTeam/AdGuardHome/internal/aghos"
	"github.com/AdguardTeam/AdGuardHome/internal/aghtest"
	"github.com/AdguardTeam/golibs/errors"
	"github.com/AdguardTeam/golibs/logutil/slogutil"
	"github.com/AdguardTeam/golibs/osutil/executil"
	"github.com/AdguardTeam/golibs/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestUpdater_internal(t *testing.T) {
	wd := t.TempDir()

	exePathUnix := filepath.Join(wd, "AdGuardHome.exe")
	exePathWindows := filepath.Join(wd, "AdGuardHome")
	yamlPath := filepath.Join(wd, "AdGuardHome.yaml")
	readmePath := filepath.Join(wd, "README.md")
	licensePath := filepath.Join(wd, "LICENSE.txt")

	require.NoError(t, os.WriteFile(exePathUnix, []byte("AdGuardHome.exe"), 0o755))
	require.NoError(t, os.WriteFile(exePathWindows, []byte("AdGuardHome"), 0o755))
	require.NoError(t, os.WriteFile(yamlPath, []byte("AdGuardHome.yaml"), 0o644))
	require.NoError(t, os.WriteFile(readmePath, []byte("README.md"), 0o644))
	require.NoError(t, os.WriteFile(licensePath, []byte("LICENSE.txt"), 0o644))

	testCases := []struct {
		name        string
		exeName     string
		os          string
		archiveName string
	}{{
		name:        "unix",
		os:          "linux",
		exeName:     "AdGuardHome",
		archiveName: "AdGuardHome.tar.gz",
	}, {
		name:        "windows",
		os:          "windows",
		exeName:     "AdGuardHome.exe",
		archiveName: "AdGuardHome.zip",
	}}

	for _, tc := range testCases {
		exePath := filepath.Join(wd, tc.exeName)

		// Start server for returning package file.
		pkgData, err := os.ReadFile(filepath.Join("testdata", tc.archiveName))
		require.NoError(t, err)

		handler := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			pt := testutil.NewPanicT(t)

			_, werr := w.Write(pkgData)
			require.NoError(pt, werr)
		})

		fakeClient, fakeURL := aghtest.StartHTTPServer(t, handler)
		fakeURL = fakeURL.JoinPath(tc.archiveName)

		u := NewUpdater(&Config{
			Client:             fakeClient,
			Logger:             slogutil.NewDiscardLogger(),
			CommandConstructor: executil.EmptyCommandConstructor{},
			GOOS:               tc.os,
			Version:            "v0.103.0",
			ExecPath:           exePath,
			WorkDir:            wd,
			ConfName:           yamlPath,
			// TODO(e.burkov):  Rewrite the test to use a fake version check
			// URL with a fake URLs for the package files.
			VersionCheckURL: &url.URL{},
		})

		u.newVersion = "v0.103.1"
		u.packageURL = fakeURL.String()

		require.NoError(t, u.prepare(newCtx(t)))
		require.NoError(t, u.downloadPackageFile(newCtx(t)))
		require.NoError(t, u.unpack(newCtx(t)))
		require.NoError(t, u.backup(newCtx(t), false))
		require.NoError(t, u.replace(newCtx(t)))

		u.clean(newCtx(t))

		require.True(t, t.Run("backup", func(t *testing.T) {
			var d []byte
			d, err = os.ReadFile(filepath.Join(wd, "agh-backup", "AdGuardHome.yaml"))
			require.NoError(t, err)

			assert.Equal(t, "AdGuardHome.yaml", string(d))

			d, err = os.ReadFile(filepath.Join(wd, "agh-backup", tc.exeName))
			require.NoError(t, err)

			assert.Equal(t, tc.exeName, string(d))
		}))

		require.True(t, t.Run("updated", func(t *testing.T) {
			var d []byte
			d, err = os.ReadFile(exePath)
			require.NoError(t, err)

			assert.Equal(t, "1", string(d))

			d, err = os.ReadFile(readmePath)
			require.NoError(t, err)

			assert.Equal(t, "2", string(d))

			d, err = os.ReadFile(licensePath)
			require.NoError(t, err)

			assert.Equal(t, "3", string(d))

			d, err = os.ReadFile(yamlPath)
			require.NoError(t, err)

			assert.Equal(t, "AdGuardHome.yaml", string(d))
		}))
	}
}

func TestUpdater_replace(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		u, paths := newReplaceTestUpdater(t, "linux")
		realRename := u.renameFile
		u.renameFile = func(oldPath, newPath string) (err error) {
			assert.Equal(t, paths.installDir, filepath.Dir(oldPath))
			assert.Equal(t, paths.installDir, filepath.Dir(newPath))

			return realRename(oldPath, newPath)
		}

		require.NoError(t, u.backup(newCtx(t), true))
		require.NoError(t, u.replace(newCtx(t)))

		assertFileContents(t, paths.currentExe, "new executable")
		assertFileContents(t, paths.readme, "new readme")
		assertFileContents(t, paths.license, "new license")
		assertFileContents(t, u.backupExeName, "old executable")
		assertFileContents(t, filepath.Join(u.backupDir, "README.md"), "old readme")
		assertFileContents(t, filepath.Join(u.backupDir, "LICENSE.txt"), "old license")
		assertNoStagedExecutable(t, paths.installDir)
	})

	t.Run("rename_exdev", func(t *testing.T) {
		u, paths := newReplaceTestUpdater(t, "linux")
		require.NoError(t, u.backup(newCtx(t), true))

		u.renameFile = func(oldPath, newPath string) (err error) {
			return &os.LinkError{
				Op:  "rename",
				Old: oldPath,
				New: newPath,
				Err: syscall.EXDEV,
			}
		}

		err := u.replace(newCtx(t))
		require.ErrorIs(t, err, syscall.EXDEV)

		assertFileContents(t, paths.currentExe, "old executable")
		assertFileContents(t, paths.readme, "old readme")
		assertFileContents(t, paths.license, "old license")
		assertNoStagedExecutable(t, paths.installDir)
	})

	t.Run("new_supporting_file_rollback", func(t *testing.T) {
		u, paths := newReplaceTestUpdater(t, "linux")
		require.NoError(t, os.Remove(paths.readme))
		require.NoError(t, os.Mkdir(u.backupDir, aghos.DefaultPermDir))
		require.NoError(t, os.WriteFile(
			filepath.Join(u.backupDir, "README.md"),
			[]byte("stale backup"),
			aghos.DefaultPermFile,
		))
		require.NoError(t, u.backup(newCtx(t), true))

		u.renameFile = func(oldPath, newPath string) (err error) {
			return &os.LinkError{
				Op:  "rename",
				Old: oldPath,
				New: newPath,
				Err: syscall.EXDEV,
			}
		}

		err := u.replace(newCtx(t))
		require.ErrorIs(t, err, syscall.EXDEV)

		_, err = os.Stat(paths.readme)
		require.ErrorIs(t, err, os.ErrNotExist)
		assertFileContents(t, paths.currentExe, "old executable")
		assertFileContents(t, paths.license, "old license")
		assertNoStagedExecutable(t, paths.installDir)
	})

	t.Run("staging_failure", func(t *testing.T) {
		u, paths := newReplaceTestUpdater(t, "linux")
		require.NoError(t, u.backup(newCtx(t), true))

		const wantErr errors.Error = "test staging failure"

		realCopy := u.copyFile
		u.copyFile = func(src, dst string, perm fs.FileMode) (err error) {
			if src == u.updateExeName {
				return wantErr
			}

			return realCopy(src, dst, perm)
		}

		err := u.replace(newCtx(t))
		require.ErrorIs(t, err, wantErr)

		assertFileContents(t, paths.currentExe, "old executable")
		assertFileContents(t, paths.readme, "old readme")
		assertFileContents(t, paths.license, "old license")
		assertNoStagedExecutable(t, paths.installDir)
	})

	t.Run("supporting_file_failure", func(t *testing.T) {
		u, paths := newReplaceTestUpdater(t, "linux")
		require.NoError(t, u.backup(newCtx(t), true))

		const wantErr errors.Error = "test supporting-file failure"

		realCopy := u.copyFile
		u.copyFile = func(src, dst string, perm fs.FileMode) (err error) {
			if src == filepath.Join(u.updateDir, "LICENSE.txt") {
				return wantErr
			}

			return realCopy(src, dst, perm)
		}

		err := u.replace(newCtx(t))
		require.ErrorIs(t, err, wantErr)

		assertFileContents(t, paths.currentExe, "old executable")
		assertFileContents(t, paths.readme, "old readme")
		assertFileContents(t, paths.license, "old license")
		assertNoStagedExecutable(t, paths.installDir)
	})

	t.Run("windows_install_failure", func(t *testing.T) {
		u, paths := newReplaceTestUpdater(t, "windows")
		require.NoError(t, u.backup(newCtx(t), true))

		const wantErr errors.Error = "test executable-install failure"

		realCopy := u.copyFile
		failed := false
		u.copyFile = func(src, dst string, perm fs.FileMode) (err error) {
			if !failed && src != u.updateExeName && dst == u.currentExeName {
				failed = true

				return wantErr
			}

			return realCopy(src, dst, perm)
		}

		err := u.replace(newCtx(t))
		require.ErrorIs(t, err, wantErr)

		assertFileContents(t, paths.currentExe, "old executable")
		assertFileContents(t, paths.readme, "old readme")
		assertFileContents(t, paths.license, "old license")
		assertNoStagedExecutable(t, paths.installDir)
	})
}

type replaceTestPaths struct {
	installDir string
	currentExe string
	readme     string
	license    string
}

func newReplaceTestUpdater(t *testing.T, goos string) (u *Updater, paths *replaceTestPaths) {
	t.Helper()

	testDir := t.TempDir()
	workDir := filepath.Join(testDir, "work")
	installDir := filepath.Join(testDir, "install")
	updateDir := filepath.Join(workDir, "update")
	backupDir := filepath.Join(workDir, "backup")
	require.NoError(t, os.Mkdir(workDir, aghos.DefaultPermDir))
	require.NoError(t, os.Mkdir(installDir, aghos.DefaultPermDir))
	require.NoError(t, os.Mkdir(updateDir, aghos.DefaultPermDir))

	exeName := "AdGuardHome"
	if goos == "windows" {
		exeName = "AdGuardHome.exe"
	}

	paths = &replaceTestPaths{
		installDir: installDir,
		currentExe: filepath.Join(installDir, exeName),
		readme:     filepath.Join(installDir, "README.md"),
		license:    filepath.Join(installDir, "LICENSE.txt"),
	}

	updateExe := filepath.Join(updateDir, exeName)
	require.NoError(t, os.WriteFile(paths.currentExe, []byte("old executable"), aghos.DefaultPermExe))
	require.NoError(t, os.WriteFile(paths.readme, []byte("old readme"), aghos.DefaultPermFile))
	require.NoError(t, os.WriteFile(paths.license, []byte("old license"), aghos.DefaultPermFile))
	require.NoError(t, os.WriteFile(updateExe, []byte("new executable"), aghos.DefaultPermExe))
	require.NoError(t, os.WriteFile(
		filepath.Join(updateDir, "README.md"),
		[]byte("new readme"),
		aghos.DefaultPermFile,
	))
	require.NoError(t, os.WriteFile(
		filepath.Join(updateDir, "LICENSE.txt"),
		[]byte("new license"),
		aghos.DefaultPermFile,
	))

	u = NewUpdater(&Config{
		Logger:             slogutil.NewDiscardLogger(),
		CommandConstructor: executil.EmptyCommandConstructor{},
		GOOS:               goos,
		WorkDir:            workDir,
		ExecPath:           paths.currentExe,
		VersionCheckURL:    &url.URL{},
	})
	u.currentExeName = paths.currentExe
	u.updateDir = updateDir
	u.backupDir = backupDir
	u.backupExeName = filepath.Join(backupDir, exeName)
	u.updateExeName = updateExe
	u.unpackedFiles = []string{exeName, "README.md", "LICENSE.txt"}

	return u, paths
}

func assertFileContents(t *testing.T, path, want string) {
	t.Helper()

	b, err := os.ReadFile(path)
	require.NoError(t, err)
	assert.Equal(t, want, string(b))
}

func assertNoStagedExecutable(t *testing.T, installDir string) {
	t.Helper()

	matches, err := filepath.Glob(filepath.Join(installDir, ".AdGuardHome-update-*"))
	require.NoError(t, err)
	assert.Empty(t, matches)
}

// newCtx is a helper that returns a new context with a timeout.
func newCtx(tb testing.TB) (ctx context.Context) {
	return testutil.ContextWithTimeout(tb, 1*time.Second)
}

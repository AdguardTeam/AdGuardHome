// Package updater provides an updater for AdGuardHome.
package updater

import (
	"archive/tar"
	"archive/zip"
	"bytes"
	"compress/gzip"
	"context"
	"fmt"
	"io"
	"io/fs"
	"log/slog"
	"net/http"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/AdguardTeam/AdGuardHome/internal/aghos"
	"github.com/AdguardTeam/AdGuardHome/internal/version"
	"github.com/AdguardTeam/golibs/errors"
	"github.com/AdguardTeam/golibs/ioutil"
	"github.com/AdguardTeam/golibs/logutil/slogutil"
	"github.com/AdguardTeam/golibs/netutil/urlutil"
	"github.com/AdguardTeam/golibs/osutil/executil"
)

// Updater is the AdGuard Home updater.
type Updater struct {
	client *http.Client
	logger *slog.Logger

	cmdCons executil.CommandConstructor

	version string
	channel string
	goarch  string
	goos    string
	goarm   string
	gomips  string

	workDir         string
	confName        string
	execPath        string
	versionCheckURL string

	// mu protects all fields below.
	mu *sync.RWMutex

	// TODO(a.garipov): See if all of these fields actually have to be in
	// this struct.
	currentExeName string // current binary executable
	updateDir      string // "workDir/agh-update-v0.103.0"
	packageName    string // "workDir/agh-update-v0.103.0/pkg_name.tar.gz"
	backupDir      string // "workDir/agh-backup"
	backupExeName  string // "workDir/agh-backup/AdGuardHome[.exe]"
	updateExeName  string // "workDir/agh-update-v0.103.0/AdGuardHome[.exe]"
	unpackedFiles  []string

	newVersion string
	packageURL string

	// Cached fields to prevent too many API requests.
	prevCheckError  error
	prevCheckTime   time.Time
	prevCheckResult VersionInfo
}

// DefaultVersionURL returns the default URL for the version announcement.
func DefaultVersionURL() *url.URL {
	return &url.URL{
		Scheme: urlutil.SchemeHTTPS,
		Host:   "static.adtidy.org",
		Path:   path.Join("adguardhome", version.Channel(), "version.json"),
	}
}

// Config is the AdGuard Home updater configuration.
type Config struct {
	// Client is used to perform HTTP requests.  It must not be nil.
	Client *http.Client

	// Logger is used for logging the update process.  It must not be nil.
	Logger *slog.Logger

	// VersionCheckURL is URL to the latest version announcement.  It must not
	// be nil, see [DefaultVersionURL].
	VersionCheckURL *url.URL

	// CommandConstructor is used to run external commands.  It must not be nil.
	CommandConstructor executil.CommandConstructor

	// Version is the current AdGuard Home version.  It must not be empty.
	Version string

	// Channel is the current AdGuard Home update channel.  It must be a valid
	// channel, see [version.ChannelBeta] and the related constants.
	Channel string

	// GOARCH is the current CPU architecture.  It must not be empty and must be
	// one of the supported architectures.
	GOARCH string

	// GOOS is the current operating system.  It must not be empty and must be
	// one of the supported OSs.
	GOOS string

	// GOARM is the current ARM variant, if any.  It must either be empty or be
	// a valid and supported GOARM value.
	GOARM string

	// GOMIPS is the current MIPS variant, if any.  It must either be empty or
	// be a valid and supported GOMIPS value.
	GOMIPS string

	// ConfName is the name of the current configuration file.  It must not be
	// empty.
	ConfName string

	// WorkDir is the working directory that is used for temporary files.  It
	// must not be empty.
	WorkDir string

	// ExecPath is path to the executable file.  It must not be empty.
	ExecPath string
}

// NewUpdater creates a new Updater.  conf must not be nil.
func NewUpdater(conf *Config) *Updater {
	return &Updater{
		client: conf.Client,
		logger: conf.Logger,

		cmdCons: conf.CommandConstructor,

		version: conf.Version,
		channel: conf.Channel,
		goarch:  conf.GOARCH,
		goos:    conf.GOOS,
		goarm:   conf.GOARM,
		gomips:  conf.GOMIPS,

		confName:        conf.ConfName,
		workDir:         conf.WorkDir,
		execPath:        conf.ExecPath,
		versionCheckURL: conf.VersionCheckURL.String(),

		mu: &sync.RWMutex{},
	}
}

// Update performs the auto-update.  It returns an error if the update fails.
// If firstRun is true, it assumes the configuration file doesn't exist.
func (u *Updater) Update(ctx context.Context, firstRun bool) (err error) {
	u.mu.Lock()
	defer u.mu.Unlock()

	u.logger.InfoContext(ctx, "starting update", "first_run", firstRun)
	defer func() {
		u.logUpdateResult(ctx, err)
	}()

	err = u.prepare(ctx)
	if err != nil {
		return fmt.Errorf("preparing: %w", err)
	}

	defer u.clean(ctx)

	err = u.downloadPackageFile(ctx)
	if err != nil {
		return fmt.Errorf("downloading package file: %w", err)
	}

	err = u.unpack(ctx)
	if err != nil {
		return fmt.Errorf("unpacking: %w", err)
	}

	if !firstRun {
		err = u.check(ctx)
		if err != nil {
			return fmt.Errorf("checking config: %w", err)
		}
	}

	err = u.backup(ctx, firstRun)
	if err != nil {
		return fmt.Errorf("making backup: %w", err)
	}

	err = u.replace(ctx)
	if err != nil {
		return fmt.Errorf("replacing: %w", err)
	}

	return nil
}

// logUpdateResult logs the result of the update operation.
func (u *Updater) logUpdateResult(ctx context.Context, err error) {
	if err != nil {
		u.logger.ErrorContext(ctx, "update failed", slogutil.KeyError, err)

		return
	}

	u.logger.InfoContext(ctx, "update finished")
}

// NewVersion returns the available new version.
func (u *Updater) NewVersion() (nv string) {
	u.mu.RLock()
	defer u.mu.RUnlock()

	return u.newVersion
}

// prepare fills all necessary fields in Updater object.
func (u *Updater) prepare(ctx context.Context) (err error) {
	u.updateDir = filepath.Join(u.workDir, fmt.Sprintf("agh-update-%s", u.newVersion))

	_, pkgNameOnly := filepath.Split(u.packageURL)
	if pkgNameOnly == "" {
		return fmt.Errorf("invalid PackageURL: %q", u.packageURL)
	}

	u.packageName = filepath.Join(u.updateDir, pkgNameOnly)
	u.backupDir = filepath.Join(u.workDir, "agh-backup")

	updateExeName := "AdGuardHome"
	if u.goos == "windows" {
		updateExeName = "AdGuardHome.exe"
	}

	u.backupExeName = filepath.Join(u.backupDir, filepath.Base(u.execPath))
	u.updateExeName = filepath.Join(u.updateDir, updateExeName)

	u.logger.InfoContext(
		ctx,
		"updating",
		"from", version.Version(),
		"to", u.newVersion,
		"package_url", u.packageURL,
	)

	u.currentExeName = u.execPath
	_, err = os.Stat(u.currentExeName)
	if err != nil {
		return fmt.Errorf("checking %q: %w", u.currentExeName, err)
	}

	return nil
}

// unpack extracts the files from the downloaded archive.
func (u *Updater) unpack(ctx context.Context) (err error) {
	_, pkgNameOnly := filepath.Split(u.packageURL)

	u.logger.InfoContext(ctx, "unpacking package", "package_name", pkgNameOnly)
	if strings.HasSuffix(pkgNameOnly, ".zip") {
		u.unpackedFiles, err = u.unpackZip(ctx, u.packageName, u.updateDir)
		if err != nil {
			return fmt.Errorf(".zip unpack failed: %w", err)
		}
	} else if strings.HasSuffix(pkgNameOnly, ".tar.gz") {
		u.unpackedFiles, err = u.unpackTarGz(ctx, u.packageName, u.updateDir)
		if err != nil {
			return fmt.Errorf(".tar.gz unpack failed: %w", err)
		}
	} else {
		return fmt.Errorf("unknown package extension")
	}

	return nil
}

// check returns an error if the configuration file couldn't be used with the
// version of AdGuard Home just downloaded.
func (u *Updater) check(ctx context.Context) (err error) {
	u.logger.InfoContext(ctx, "checking configuration")

	err = copyFile(u.confName, filepath.Join(u.updateDir, "AdGuardHome.yaml"), aghos.DefaultPermFile)
	if err != nil {
		return fmt.Errorf("copyFile() failed: %w", err)
	}

	const format = "executing configuration check command: %w %d:\n" +
		"below is the output of configuration check:\n" +
		"%s" +
		"end of the output"

	var (
		args = []string{"--check-config"}
		buf  bytes.Buffer
	)

	u.logger.DebugContext(ctx, "executing", "cmd", u.updateExeName, "args", args)

	err = executil.Run(
		ctx,
		u.cmdCons,
		&executil.CommandConfig{
			Path:   u.updateExeName,
			Args:   args,
			Stdout: &buf,
			Stderr: &buf,
		},
	)
	if err != nil {
		code, _ := executil.ExitCodeFromError(err)

		return fmt.Errorf(format, err, code, buf.Bytes())
	}

	return nil
}

// backup makes a backup of the current configuration and supporting files.  It
// ignores the configuration file if firstRun is true.
func (u *Updater) backup(ctx context.Context, firstRun bool) (err error) {
	u.logger.InfoContext(ctx, "backing up current configuration")

	_ = os.Mkdir(u.backupDir, aghos.DefaultPermDir)
	if !firstRun {
		err = copyFile(u.confName, filepath.Join(u.backupDir, "AdGuardHome.yaml"), aghos.DefaultPermFile)
		if err != nil {
			return fmt.Errorf("copyFile() failed: %w", err)
		}
	}

	wd := u.workDir
	err = u.copySupportingFiles(ctx, u.unpackedFiles, wd, u.backupDir)
	if err != nil {
		return fmt.Errorf("copySupportingFiles(%s, %s) failed: %w", wd, u.backupDir, err)
	}

	return nil
}

// replace moves the current executable with the updated one and also copies the
// supporting files.
func (u *Updater) replace(ctx context.Context) (err error) {
	err = u.copySupportingFiles(ctx, u.unpackedFiles, u.updateDir, u.workDir)
	if err != nil {
		return fmt.Errorf("copySupportingFiles(%s, %s) failed: %w", u.updateDir, u.workDir, err)
	}

	u.logger.InfoContext(
		ctx,
		"backing up current executable",
		"from", u.currentExeName,
		"to", u.backupExeName,
	)
	err = os.Rename(u.currentExeName, u.backupExeName)
	if err != nil {
		return err
	}

	if u.goos == "windows" {
		// Use copy, since renaming fails with "File in use" error.
		err = copyFile(u.updateExeName, u.currentExeName, aghos.DefaultPermExe)
	} else {
		err = os.Rename(u.updateExeName, u.currentExeName)
	}
	if err != nil {
		return err
	}

	u.logger.InfoContext(
		ctx,
		"replacing current executable",
		"from", u.updateExeName,
		"to", u.currentExeName,
	)

	return nil
}

// clean removes the temporary directory itself and all it's contents.
func (u *Updater) clean(ctx context.Context) {
	err := os.RemoveAll(u.updateDir)
	if err != nil {
		u.logger.WarnContext(ctx, "removing update dir", slogutil.KeyError, err)
	}
}

// MaxPackageFileSize is a maximum package file length in bytes.  The largest
// package whose size is limited by this constant currently has the size of
// approximately 9 MiB.
const MaxPackageFileSize = 32 * 1024 * 1024

// Download package file and save it to disk
func (u *Updater) downloadPackageFile(ctx context.Context) (err error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.packageURL, nil)
	if err != nil {
		return fmt.Errorf("constructing package request: %w", err)
	}

	resp, err := u.client.Do(req)
	if err != nil {
		return fmt.Errorf("requesting package: %w", err)
	}
	defer func() { err = errors.WithDeferred(err, resp.Body.Close()) }()

	r := ioutil.LimitReader(resp.Body, MaxPackageFileSize)

	u.logger.InfoContext(ctx, "reading http body")

	// This use of ReadAll is now safe, because we limited body's Reader.
	body, err := io.ReadAll(r)
	if err != nil {
		return fmt.Errorf("io.ReadAll() failed: %w", err)
	}

	err = os.Mkdir(u.updateDir, aghos.DefaultPermDir)
	if err != nil {
		// TODO(a.garipov):  Consider returning this error.
		u.logger.WarnContext(ctx, "creating update dir", slogutil.KeyError, err)
	}

	u.logger.InfoContext(ctx, "saving package", "to", u.packageName)

	err = os.WriteFile(u.packageName, body, aghos.DefaultPermFile)
	if err != nil {
		return fmt.Errorf("writing package file: %w", err)
	}

	return nil
}

// unpackTarGzFile unpacks one file from a .tar.gz archive into outDir.  All
// arguments must not be empty.
func (u *Updater) unpackTarGzFile(
	ctx context.Context,
	outDir string,
	tr *tar.Reader,
	hdr *tar.Header,
) (name string, err error) {
	name = filepath.Base(hdr.Name)
	if name == "" {
		return "", nil
	}

	outName := filepath.Join(outDir, name)

	if hdr.Typeflag == tar.TypeDir {
		if name == "AdGuardHome" {
			// Top-level AdGuardHome/.  Skip it.
			//
			// TODO(a.garipov): This whole package needs to be rewritten and
			// covered in more integration tests.  It has weird assumptions and
			// file mode issues.
			return "", nil
		}

		err = os.Mkdir(outName, os.FileMode(hdr.Mode&0o755))
		if err != nil && !errors.Is(err, os.ErrExist) {
			return "", fmt.Errorf("creating directory %q: %w", outName, err)
		}

		u.logger.InfoContext(ctx, "created directory", "name", outName)

		return "", nil
	}

	if hdr.Typeflag != tar.TypeReg {
		u.logger.WarnContext(
			ctx,
			"unknown file type; skipping",
			"file_name", name,
			"type", hdr.Typeflag,
		)

		return "", nil
	}

	var wc io.WriteCloser
	wc, err = os.OpenFile(outName, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, os.FileMode(hdr.Mode)&0o755)
	if err != nil {
		return "", fmt.Errorf("os.OpenFile(%s): %w", outName, err)
	}
	defer func() { err = errors.WithDeferred(err, wc.Close()) }()

	_, err = io.Copy(wc, tr)
	if err != nil {
		return "", fmt.Errorf("io.Copy(): %w", err)
	}

	u.logger.InfoContext(ctx, "created file", "name", outName)

	return name, nil
}

// unpackTarGz unpack all files from a .tar.gz archive to outDir.  Existing
// files are overwritten.  All files are created inside outDir.  files are the
// list of created files.
func (u *Updater) unpackTarGz(
	ctx context.Context,
	tarfile string,
	outDir string,
) (files []string, err error) {
	f, err := os.Open(tarfile)
	if err != nil {
		return nil, fmt.Errorf("os.Open(): %w", err)
	}
	defer func() { err = errors.WithDeferred(err, f.Close()) }()

	gzReader, err := gzip.NewReader(f)
	if err != nil {
		return nil, fmt.Errorf("gzip.NewReader(): %w", err)
	}
	defer func() { err = errors.WithDeferred(err, gzReader.Close()) }()

	tarReader := tar.NewReader(gzReader)
	for {
		var hdr *tar.Header
		hdr, err = tarReader.Next()
		if errors.Is(err, io.EOF) {
			err = nil

			break
		} else if err != nil {
			err = fmt.Errorf("tarReader.Next(): %w", err)

			break
		}

		var name string
		name, err = u.unpackTarGzFile(ctx, outDir, tarReader, hdr)

		if name != "" {
			files = append(files, name)
		}
	}

	return files, err
}

// unpackZipFile unpacks one file from a .zip archive into outDir.  All
// arguments must not be empty.
func (u *Updater) unpackZipFile(
	ctx context.Context,
	outDir string,
	zf *zip.File,
) (name string, err error) {
	var rc io.ReadCloser
	rc, err = zf.Open()
	if err != nil {
		return "", fmt.Errorf("zip file Open(): %w", err)
	}
	defer func() { err = errors.WithDeferred(err, rc.Close()) }()

	fi := zf.FileInfo()
	name = fi.Name()
	if name == "" {
		return "", nil
	}

	outputName := filepath.Join(outDir, name)
	if fi.IsDir() {
		if name == "AdGuardHome" {
			// Top-level AdGuardHome/.  Skip it.
			//
			// TODO(a.garipov): See the similar TODO in
			// [Updater.unpackTarGzFile].
			return "", nil
		}

		err = os.Mkdir(outputName, fi.Mode())
		if err != nil && !errors.Is(err, os.ErrExist) {
			return "", fmt.Errorf("creating directory %q: %w", outputName, err)
		}

		u.logger.InfoContext(ctx, "created directory", "name", outputName)

		return "", nil
	}

	var wc io.WriteCloser
	wc, err = os.OpenFile(outputName, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, fi.Mode())
	if err != nil {
		return "", fmt.Errorf("os.OpenFile(): %w", err)
	}
	defer func() { err = errors.WithDeferred(err, wc.Close()) }()

	_, err = io.Copy(wc, rc)
	if err != nil {
		return "", fmt.Errorf("io.Copy(): %w", err)
	}

	u.logger.InfoContext(ctx, "created file", "name", outputName)

	return name, nil
}

// unpackZip unpack all files from a .zip archive to outDir.  Existing files are
// overwritten.  All files are created inside outDir.  files are the list of
// created files.
func (u *Updater) unpackZip(
	ctx context.Context,
	zipfile string,
	outDir string,
) (files []string, err error) {
	zrc, err := zip.OpenReader(zipfile)
	if err != nil {
		return nil, fmt.Errorf("zip.OpenReader(): %w", err)
	}
	defer func() { err = errors.WithDeferred(err, zrc.Close()) }()

	for _, zf := range zrc.File {
		var name string
		name, err = u.unpackZipFile(ctx, outDir, zf)
		if err != nil {
			break
		}

		if name != "" {
			files = append(files, name)
		}
	}

	return files, err
}

// copyFile copies a file from src to dst with the specified permissions.
func copyFile(src, dst string, perm fs.FileMode) (err error) {
	d, err := os.ReadFile(src)
	if err != nil {
		// Don't wrap the error, since it's informative enough as is.
		return err
	}

	err = os.WriteFile(dst, d, perm)
	if err != nil {
		// Don't wrap the error, since it's informative enough as is.
		return err
	}

	return nil
}

// copySupportingFiles copies each file specified in files from srcdir to
// dstdir.  If a file specified as a path, only the name of the file is used.
// It skips AdGuardHome, AdGuardHome.exe, and AdGuardHome.yaml.
func (u *Updater) copySupportingFiles(
	ctx context.Context,
	files []string,
	srcdir string,
	dstdir string,
) (err error) {
	for _, f := range files {
		_, name := filepath.Split(f)
		if name == "AdGuardHome" || name == "AdGuardHome.exe" || name == "AdGuardHome.yaml" {
			continue
		}

		src := filepath.Join(srcdir, name)
		dst := filepath.Join(dstdir, name)

		err = copyFile(src, dst, aghos.DefaultPermFile)
		if err != nil && !errors.Is(err, os.ErrNotExist) {
			return err
		}

		u.logger.InfoContext(ctx, "copied", "from", src, "to", dst)
	}

	return nil
}

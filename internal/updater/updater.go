// Package updater provides an updater for AdGuardHome.
package updater

import (
	"archive/tar"
	"archive/zip"
	"compress/gzip"
	"fmt"
	"io"
	"io/fs"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/AdguardTeam/AdGuardHome/internal/aghos"
	"github.com/AdguardTeam/AdGuardHome/internal/version"
	"github.com/AdguardTeam/golibs/errors"
	"github.com/AdguardTeam/golibs/ioutil"
	"github.com/AdguardTeam/golibs/log"
	"github.com/AdguardTeam/golibs/netutil/urlutil"
)

// Updater is the AdGuard Home updater.
type Updater struct {
	client *http.Client

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
	Client *http.Client

	// VersionCheckURL is URL to the latest version announcement.  It must not
	// be nil, see [DefaultVersionURL].
	VersionCheckURL *url.URL

	Version string
	Channel string
	GOARCH  string
	GOOS    string
	GOARM   string
	GOMIPS  string

	// ConfName is the name of the current configuration file.  Typically,
	// "AdGuardHome.yaml".
	ConfName string

	// WorkDir is the working directory that is used for temporary files.
	WorkDir string

	// ExecPath is path to the executable file.
	ExecPath string
}

// NewUpdater creates a new Updater.  conf must not be nil.
func NewUpdater(conf *Config) *Updater {
	return &Updater{
		client: conf.Client,

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

// Update performs the auto-update.  It returns an error if the update failed.
// If firstRun is true, it assumes the configuration file doesn't exist.
func (u *Updater) Update(firstRun bool) (err error) {
	u.mu.Lock()
	defer u.mu.Unlock()

	log.Info("updater: updating")
	defer func() {
		if err != nil {
			log.Info("updater: failed")
		} else {
			log.Info("updater: finished successfully")
		}
	}()

	err = u.prepare()
	if err != nil {
		return fmt.Errorf("preparing: %w", err)
	}

	defer u.clean()

	err = u.downloadPackageFile()
	if err != nil {
		return fmt.Errorf("downloading package file: %w", err)
	}

	err = u.unpack()
	if err != nil {
		return fmt.Errorf("unpacking: %w", err)
	}

	if !firstRun {
		err = u.check()
		if err != nil {
			return fmt.Errorf("checking config: %w", err)
		}
	}

	err = u.backup(firstRun)
	if err != nil {
		return fmt.Errorf("making backup: %w", err)
	}

	err = u.replace()
	if err != nil {
		return fmt.Errorf("replacing: %w", err)
	}

	return nil
}

// NewVersion returns the available new version.
func (u *Updater) NewVersion() (nv string) {
	u.mu.RLock()
	defer u.mu.RUnlock()

	return u.newVersion
}

// prepare fills all necessary fields in Updater object.
func (u *Updater) prepare() (err error) {
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

	log.Debug(
		"updater: updating from %s to %s using url: %s",
		version.Version(),
		u.newVersion,
		u.packageURL,
	)

	u.currentExeName = u.execPath
	_, err = os.Stat(u.currentExeName)
	if err != nil {
		return fmt.Errorf("checking %q: %w", u.currentExeName, err)
	}

	return nil
}

// unpack extracts the files from the downloaded archive.
func (u *Updater) unpack() error {
	var err error
	_, pkgNameOnly := filepath.Split(u.packageURL)

	log.Debug("updater: unpacking package")
	if strings.HasSuffix(pkgNameOnly, ".zip") {
		u.unpackedFiles, err = zipFileUnpack(u.packageName, u.updateDir)
		if err != nil {
			return fmt.Errorf(".zip unpack failed: %w", err)
		}

	} else if strings.HasSuffix(pkgNameOnly, ".tar.gz") {
		u.unpackedFiles, err = tarGzFileUnpack(u.packageName, u.updateDir)
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
func (u *Updater) check() (err error) {
	log.Debug("updater: checking configuration")

	err = copyFile(u.confName, filepath.Join(u.updateDir, "AdGuardHome.yaml"), aghos.DefaultPermFile)
	if err != nil {
		return fmt.Errorf("copyFile() failed: %w", err)
	}

	const format = "executing configuration check command: %w %d:\n" +
		"below is the output of configuration check:\n" +
		"%s" +
		"end of the output"

	cmd := exec.Command(u.updateExeName, "--check-config")
	out, err := cmd.CombinedOutput()
	code := cmd.ProcessState.ExitCode()
	if err != nil || code != 0 {
		return fmt.Errorf(format, err, code, out)
	}

	return nil
}

// backup makes a backup of the current configuration and supporting files.  It
// ignores the configuration file if firstRun is true.
func (u *Updater) backup(firstRun bool) (err error) {
	log.Debug("updater: backing up current configuration")
	_ = os.Mkdir(u.backupDir, aghos.DefaultPermDir)
	if !firstRun {
		err = copyFile(u.confName, filepath.Join(u.backupDir, "AdGuardHome.yaml"), aghos.DefaultPermFile)
		if err != nil {
			return fmt.Errorf("copyFile() failed: %w", err)
		}
	}

	wd := u.workDir
	err = copySupportingFiles(u.unpackedFiles, wd, u.backupDir)
	if err != nil {
		return fmt.Errorf("copySupportingFiles(%s, %s) failed: %w", wd, u.backupDir, err)
	}

	return nil
}

// replace moves the current executable with the updated one and also copies the
// supporting files.
func (u *Updater) replace() error {
	err := copySupportingFiles(u.unpackedFiles, u.updateDir, u.workDir)
	if err != nil {
		return fmt.Errorf("copySupportingFiles(%s, %s) failed: %w", u.updateDir, u.workDir, err)
	}

	log.Debug("updater: renaming: %s to %s", u.currentExeName, u.backupExeName)
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

	log.Debug("updater: renamed: %s to %s", u.updateExeName, u.currentExeName)

	return nil
}

// clean removes the temporary directory itself and all it's contents.
func (u *Updater) clean() {
	_ = os.RemoveAll(u.updateDir)
}

// MaxPackageFileSize is a maximum package file length in bytes.  The largest
// package whose size is limited by this constant currently has the size of
// approximately 9 MiB.
const MaxPackageFileSize = 32 * 1024 * 1024

// Download package file and save it to disk
func (u *Updater) downloadPackageFile() (err error) {
	var resp *http.Response
	resp, err = u.client.Get(u.packageURL)
	if err != nil {
		return fmt.Errorf("http request failed: %w", err)
	}
	defer func() { err = errors.WithDeferred(err, resp.Body.Close()) }()

	r := ioutil.LimitReader(resp.Body, MaxPackageFileSize)

	log.Debug("updater: reading http body")
	// This use of ReadAll is now safe, because we limited body's Reader.
	body, err := io.ReadAll(r)
	if err != nil {
		return fmt.Errorf("io.ReadAll() failed: %w", err)
	}

	_ = os.Mkdir(u.updateDir, aghos.DefaultPermDir)

	log.Debug("updater: saving package to file")
	err = os.WriteFile(u.packageName, body, aghos.DefaultPermFile)
	if err != nil {
		return fmt.Errorf("writing package file: %w", err)
	}
	return nil
}

func tarGzFileUnpackOne(outDir string, tr *tar.Reader, hdr *tar.Header) (name string, err error) {
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

		log.Debug("updater: created directory %q", outName)

		return "", nil
	}

	if hdr.Typeflag != tar.TypeReg {
		log.Info("updater: %s: unknown file type %d, skipping", name, hdr.Typeflag)

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

	log.Debug("updater: created file %q", outName)

	return name, nil
}

// Unpack all files from .tar.gz file to the specified directory
// Existing files are overwritten
// All files are created inside outDir, subdirectories are not created
// Return the list of files (not directories) written
func tarGzFileUnpack(tarfile, outDir string) (files []string, err error) {
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
		name, err = tarGzFileUnpackOne(outDir, tarReader, hdr)

		if name != "" {
			files = append(files, name)
		}
	}

	return files, err
}

func zipFileUnpackOne(outDir string, zf *zip.File) (name string, err error) {
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
			// TODO(a.garipov): See the similar todo in tarGzFileUnpack.
			return "", nil
		}

		err = os.Mkdir(outputName, fi.Mode())
		if err != nil && !errors.Is(err, os.ErrExist) {
			return "", fmt.Errorf("creating directory %q: %w", outputName, err)
		}

		log.Debug("updater: created directory %q", outputName)

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

	log.Debug("updater: created file %q", outputName)

	return name, nil
}

// Unpack all files from .zip file to the specified directory
// Existing files are overwritten
// All files are created inside 'outDir', subdirectories are not created
// Return the list of files (not directories) written
func zipFileUnpack(zipfile, outDir string) (files []string, err error) {
	zrc, err := zip.OpenReader(zipfile)
	if err != nil {
		return nil, fmt.Errorf("zip.OpenReader(): %w", err)
	}
	defer func() { err = errors.WithDeferred(err, zrc.Close()) }()

	for _, zf := range zrc.File {
		var name string
		name, err = zipFileUnpackOne(outDir, zf)
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
func copySupportingFiles(files []string, srcdir, dstdir string) error {
	for _, f := range files {
		_, name := filepath.Split(f)
		if name == "AdGuardHome" || name == "AdGuardHome.exe" || name == "AdGuardHome.yaml" {
			continue
		}

		src := filepath.Join(srcdir, name)
		dst := filepath.Join(dstdir, name)

		err := copyFile(src, dst, aghos.DefaultPermFile)
		if err != nil && !errors.Is(err, os.ErrNotExist) {
			return err
		}

		log.Debug("updater: copied: %q to %q", src, dst)
	}

	return nil
}

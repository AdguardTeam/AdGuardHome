// Package updater provides an updater for AdGuardHome.
package updater

import (
	"archive/tar"
	"archive/zip"
	"compress/gzip"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/AdguardTeam/AdGuardHome/internal/aghio"
	"github.com/AdguardTeam/AdGuardHome/internal/version"
	"github.com/AdguardTeam/golibs/errors"
	"github.com/AdguardTeam/golibs/log"
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

// Config is the AdGuard Home updater configuration.
type Config struct {
	Client *http.Client

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
}

// NewUpdater creates a new Updater.
func NewUpdater(conf *Config) *Updater {
	u := &url.URL{
		Scheme: "https",
		Host:   "static.adguard.com",
		Path:   path.Join("adguardhome", conf.Channel, "version.json"),
	}
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
		versionCheckURL: u.String(),

		mu: &sync.RWMutex{},
	}
}

// Update performs the auto-update.
func (u *Updater) Update() error {
	u.mu.Lock()
	defer u.mu.Unlock()

	err := u.prepare()
	if err != nil {
		return err
	}

	defer u.clean()

	err = u.downloadPackageFile(u.packageURL, u.packageName)
	if err != nil {
		return err
	}

	err = u.unpack()
	if err != nil {
		return err
	}

	err = u.check()
	if err != nil {
		return err
	}

	err = u.backup()
	if err != nil {
		return err
	}

	err = u.replace()
	if err != nil {
		return err
	}

	return nil
}

// NewVersion returns the available new version.
func (u *Updater) NewVersion() (nv string) {
	u.mu.RLock()
	defer u.mu.RUnlock()

	return u.newVersion
}

// VersionCheckURL returns the version check URL.
func (u *Updater) VersionCheckURL() (vcu string) {
	u.mu.RLock()
	defer u.mu.RUnlock()

	return u.versionCheckURL
}

func (u *Updater) prepare() (err error) {
	u.updateDir = filepath.Join(u.workDir, fmt.Sprintf("agh-update-%s", u.newVersion))

	_, pkgNameOnly := filepath.Split(u.packageURL)
	if pkgNameOnly == "" {
		return fmt.Errorf("invalid PackageURL")
	}

	u.packageName = filepath.Join(u.updateDir, pkgNameOnly)
	u.backupDir = filepath.Join(u.workDir, "agh-backup")

	exeName := "AdGuardHome"
	if u.goos == "windows" {
		exeName = "AdGuardHome.exe"
	}

	u.backupExeName = filepath.Join(u.backupDir, exeName)
	u.updateExeName = filepath.Join(u.updateDir, exeName)

	log.Info("Updating from %s to %s.  URL:%s", version.Version(), u.newVersion, u.packageURL)

	// TODO(a.garipov): Use os.Args[0] instead?
	u.currentExeName = filepath.Join(u.workDir, exeName)
	_, err = os.Stat(u.currentExeName)
	if err != nil {
		return fmt.Errorf("checking %q: %w", u.currentExeName, err)
	}

	return nil
}

func (u *Updater) unpack() error {
	var err error
	_, pkgNameOnly := filepath.Split(u.packageURL)

	log.Debug("updater: unpacking the package")
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

func (u *Updater) check() error {
	log.Debug("updater: checking configuration")
	err := copyFile(u.confName, filepath.Join(u.updateDir, "AdGuardHome.yaml"))
	if err != nil {
		return fmt.Errorf("copyFile() failed: %w", err)
	}
	cmd := exec.Command(u.updateExeName, "--check-config")
	err = cmd.Run()
	if err != nil || cmd.ProcessState.ExitCode() != 0 {
		return fmt.Errorf("exec.Command(): %s %d", err, cmd.ProcessState.ExitCode())
	}
	return nil
}

func (u *Updater) backup() error {
	log.Debug("updater: backing up the current configuration")
	_ = os.Mkdir(u.backupDir, 0o755)
	err := copyFile(u.confName, filepath.Join(u.backupDir, "AdGuardHome.yaml"))
	if err != nil {
		return fmt.Errorf("copyFile() failed: %w", err)
	}

	wd := u.workDir
	err = copySupportingFiles(u.unpackedFiles, wd, u.backupDir)
	if err != nil {
		return fmt.Errorf("copySupportingFiles(%s, %s) failed: %s",
			wd, u.backupDir, err)
	}

	return nil
}

func (u *Updater) replace() error {
	err := copySupportingFiles(u.unpackedFiles, u.updateDir, u.workDir)
	if err != nil {
		return fmt.Errorf("copySupportingFiles(%s, %s) failed: %s", u.updateDir, u.workDir, err)
	}

	log.Debug("updater: renaming: %s -> %s", u.currentExeName, u.backupExeName)
	err = os.Rename(u.currentExeName, u.backupExeName)
	if err != nil {
		return err
	}

	if u.goos == "windows" {
		// rename fails with "File in use" error
		err = copyFile(u.updateExeName, u.currentExeName)
	} else {
		err = os.Rename(u.updateExeName, u.currentExeName)
	}
	if err != nil {
		return err
	}

	log.Debug("updater: renamed: %s -> %s", u.updateExeName, u.currentExeName)

	return nil
}

func (u *Updater) clean() {
	_ = os.RemoveAll(u.updateDir)
}

// MaxPackageFileSize is a maximum package file length in bytes. The largest
// package whose size is limited by this constant currently has the size of
// approximately 9 MiB.
const MaxPackageFileSize = 32 * 1024 * 1024

// Download package file and save it to disk
func (u *Updater) downloadPackageFile(url, filename string) (err error) {
	var resp *http.Response
	resp, err = u.client.Get(url)
	if err != nil {
		return fmt.Errorf("http request failed: %w", err)
	}
	defer func() { err = errors.WithDeferred(err, resp.Body.Close()) }()

	var r io.Reader
	r, err = aghio.LimitReader(resp.Body, MaxPackageFileSize)
	if err != nil {
		return fmt.Errorf("http request failed: %w", err)
	}

	log.Debug("updater: reading HTTP body")
	// This use of ReadAll is now safe, because we limited body's Reader.
	body, err := io.ReadAll(r)
	if err != nil {
		return fmt.Errorf("io.ReadAll() failed: %w", err)
	}

	_ = os.Mkdir(u.updateDir, 0o755)

	log.Debug("updater: saving package to file")
	err = os.WriteFile(filename, body, 0o644)
	if err != nil {
		return fmt.Errorf("os.WriteFile() failed: %w", err)
	}
	return nil
}

func tarGzFileUnpackOne(outDir string, tr *tar.Reader, hdr *tar.Header) (name string, err error) {
	name = filepath.Base(hdr.Name)
	if name == "" {
		return "", nil
	}

	outputName := filepath.Join(outDir, name)

	if hdr.Typeflag == tar.TypeDir {
		if name == "AdGuardHome" {
			// Top-level AdGuardHome/.  Skip it.
			//
			// TODO(a.garipov): This whole package needs to be
			// rewritten and covered in more integration tests.  It
			// has weird assumptions and file mode issues.
			return "", nil
		}

		err = os.Mkdir(outputName, os.FileMode(hdr.Mode&0o755))
		if err != nil && !errors.Is(err, os.ErrExist) {
			return "", fmt.Errorf("os.Mkdir(%q): %w", outputName, err)
		}

		log.Debug("updater: created directory %q", outputName)

		return "", nil
	}

	if hdr.Typeflag != tar.TypeReg {
		log.Debug("updater: %s: unknown file type %d, skipping", name, hdr.Typeflag)

		return "", nil
	}

	var wc io.WriteCloser
	wc, err = os.OpenFile(
		outputName,
		os.O_WRONLY|os.O_CREATE|os.O_TRUNC,
		os.FileMode(hdr.Mode&0o755),
	)
	if err != nil {
		return "", fmt.Errorf("os.OpenFile(%s): %w", outputName, err)
	}
	defer func() { err = errors.WithDeferred(err, wc.Close()) }()

	_, err = io.Copy(wc, tr)
	if err != nil {
		return "", fmt.Errorf("io.Copy(): %w", err)
	}

	log.Tracef("updater: created file %s", outputName)

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
			// TODO(a.garipov): See the similar todo in
			// tarGzFileUnpack.
			return "", nil
		}

		err = os.Mkdir(outputName, fi.Mode())
		if err != nil && !errors.Is(err, os.ErrExist) {
			return "", fmt.Errorf("os.Mkdir(%q): %w", outputName, err)
		}

		log.Tracef("created directory %q", outputName)

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

	log.Tracef("created file %s", outputName)

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

// Copy file on disk
func copyFile(src, dst string) error {
	d, e := os.ReadFile(src)
	if e != nil {
		return e
	}
	e = os.WriteFile(dst, d, 0o644)
	if e != nil {
		return e
	}
	return nil
}

func copySupportingFiles(files []string, srcdir, dstdir string) error {
	for _, f := range files {
		_, name := filepath.Split(f)
		if name == "AdGuardHome" || name == "AdGuardHome.exe" || name == "AdGuardHome.yaml" {
			continue
		}

		src := filepath.Join(srcdir, name)
		dst := filepath.Join(dstdir, name)

		err := copyFile(src, dst)
		if err != nil && !errors.Is(err, os.ErrNotExist) {
			return err
		}

		log.Debug("updater: copied: %q -> %q", src, dst)
	}

	return nil
}

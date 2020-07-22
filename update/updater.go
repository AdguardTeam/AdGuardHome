package update

import (
	"archive/tar"
	"archive/zip"
	"compress/gzip"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/AdguardTeam/AdGuardHome/util"
	"github.com/AdguardTeam/golibs/log"
)

// Updater - Updater
type Updater struct {
	Config // Updater configuration

	currentExeName string // current binary executable
	updateDir      string // "work_dir/agh-update-v0.103.0"
	packageName    string // "work_dir/agh-update-v0.103.0/pkg_name.tar.gz"
	backupDir      string // "work_dir/agh-backup"
	backupExeName  string // "work_dir/agh-backup/AdGuardHome[.exe]"
	updateExeName  string // "work_dir/agh-update-v0.103.0/AdGuardHome[.exe]"
	unpackedFiles  []string

	// cached version.json to avoid hammering github.io for each page reload
	versionJSON          []byte
	versionCheckLastTime time.Time
}

// Config - updater config
type Config struct {
	Client *http.Client

	VersionURL    string // version.json URL
	VersionString string
	OS            string // GOOS
	Arch          string // GOARCH
	ARMVersion    string // ARM version, e.g. "6"
	NewVersion    string // VersionInfo.NewVersion
	PackageURL    string // VersionInfo.PackageURL
	ConfigName    string // current config file ".../AdGuardHome.yaml"
	WorkDir       string // updater work dir (where backup/upd dirs will be created)
}

// NewUpdater - creates a new instance of the Updater
func NewUpdater(cfg Config) *Updater {
	return &Updater{
		Config: cfg,
	}
}

// DoUpdate - conducts the auto-update
// 1. Downloads the update file
// 2. Unpacks it and checks the contents
// 3. Backups the current version and configuration
// 4. Replaces the old files
func (u *Updater) DoUpdate() error {
	err := u.prepare()
	if err != nil {
		return err
	}

	defer u.clean()

	err = u.downloadPackageFile(u.PackageURL, u.packageName)
	if err != nil {
		return err
	}

	err = u.unpack()
	if err != nil {
		return err
	}

	err = u.check()
	if err != nil {
		u.clean()
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

func (u *Updater) prepare() error {
	u.updateDir = filepath.Join(u.WorkDir, fmt.Sprintf("agh-update-%s", u.NewVersion))

	_, pkgNameOnly := filepath.Split(u.PackageURL)
	if len(pkgNameOnly) == 0 {
		return fmt.Errorf("invalid PackageURL")
	}
	u.packageName = filepath.Join(u.updateDir, pkgNameOnly)
	u.backupDir = filepath.Join(u.WorkDir, "agh-backup")

	exeName := "AdGuardHome"
	if u.OS == "windows" {
		exeName = "AdGuardHome.exe"
	}

	u.backupExeName = filepath.Join(u.backupDir, exeName)
	u.updateExeName = filepath.Join(u.updateDir, exeName)

	log.Info("Updating from %s to %s.  URL:%s",
		u.VersionString, u.NewVersion, u.PackageURL)

	// If the binary file isn't found in working directory, we won't be able to auto-update
	// Getting the full path to the current binary file on UNIX and checking write permissions
	//  is more difficult.
	u.currentExeName = filepath.Join(u.WorkDir, exeName)
	if !util.FileExists(u.currentExeName) {
		return fmt.Errorf("executable file %s doesn't exist", u.currentExeName)
	}
	return nil
}

func (u *Updater) unpack() error {
	var err error
	_, pkgNameOnly := filepath.Split(u.PackageURL)

	log.Debug("updater: unpacking the package")
	if strings.HasSuffix(pkgNameOnly, ".zip") {
		u.unpackedFiles, err = zipFileUnpack(u.packageName, u.updateDir)
		if err != nil {
			return fmt.Errorf(".zip unpack failed: %s", err)
		}

	} else if strings.HasSuffix(pkgNameOnly, ".tar.gz") {
		u.unpackedFiles, err = tarGzFileUnpack(u.packageName, u.updateDir)
		if err != nil {
			return fmt.Errorf(".tar.gz unpack failed: %s", err)
		}

	} else {
		return fmt.Errorf("unknown package extension")
	}

	return nil
}

func (u *Updater) check() error {
	log.Debug("updater: checking configuration")
	err := copyFile(u.ConfigName, filepath.Join(u.updateDir, "AdGuardHome.yaml"))
	if err != nil {
		return fmt.Errorf("copyFile() failed: %s", err)
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
	_ = os.Mkdir(u.backupDir, 0755)
	err := copyFile(u.ConfigName, filepath.Join(u.backupDir, "AdGuardHome.yaml"))
	if err != nil {
		return fmt.Errorf("copyFile() failed: %s", err)
	}

	// workdir/README.md -> backup/README.md
	err = copySupportingFiles(u.unpackedFiles, u.WorkDir, u.backupDir)
	if err != nil {
		return fmt.Errorf("copySupportingFiles(%s, %s) failed: %s",
			u.WorkDir, u.backupDir, err)
	}

	return nil
}

func (u *Updater) replace() error {
	// update/README.md -> workdir/README.md
	err := copySupportingFiles(u.unpackedFiles, u.updateDir, u.WorkDir)
	if err != nil {
		return fmt.Errorf("copySupportingFiles(%s, %s) failed: %s",
			u.updateDir, u.WorkDir, err)
	}

	log.Debug("updater: renaming: %s -> %s", u.currentExeName, u.backupExeName)
	err = os.Rename(u.currentExeName, u.backupExeName)
	if err != nil {
		return err
	}

	if u.OS == "windows" {
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

// Download package file and save it to disk
func (u *Updater) downloadPackageFile(url string, filename string) error {
	resp, err := u.Client.Get(url)
	if err != nil {
		return fmt.Errorf("HTTP request failed: %s", err)
	}
	if resp != nil && resp.Body != nil {
		defer resp.Body.Close()
	}

	log.Debug("updater: reading HTTP body")
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("ioutil.ReadAll() failed: %s", err)
	}

	_ = os.Mkdir(u.updateDir, 0755)

	log.Debug("updater: saving package to file")
	err = ioutil.WriteFile(filename, body, 0644)
	if err != nil {
		return fmt.Errorf("ioutil.WriteFile() failed: %s", err)
	}
	return nil
}

// Unpack all files from .tar.gz file to the specified directory
// Existing files are overwritten
// All files are created inside 'outdir', subdirectories are not created
// Return the list of files (not directories) written
func tarGzFileUnpack(tarfile, outdir string) ([]string, error) {
	f, err := os.Open(tarfile)
	if err != nil {
		return nil, fmt.Errorf("os.Open(): %s", err)
	}
	defer func() {
		_ = f.Close()
	}()

	gzReader, err := gzip.NewReader(f)
	if err != nil {
		return nil, fmt.Errorf("gzip.NewReader(): %s", err)
	}

	var files []string
	var err2 error
	tarReader := tar.NewReader(gzReader)
	for {
		header, err := tarReader.Next()
		if err == io.EOF {
			err2 = nil
			break
		}
		if err != nil {
			err2 = fmt.Errorf("tarReader.Next(): %s", err)
			break
		}

		_, inputNameOnly := filepath.Split(header.Name)
		if len(inputNameOnly) == 0 {
			continue
		}

		outputName := filepath.Join(outdir, inputNameOnly)

		if header.Typeflag == tar.TypeDir {
			err = os.Mkdir(outputName, os.FileMode(header.Mode&0777))
			if err != nil && !os.IsExist(err) {
				err2 = fmt.Errorf("os.Mkdir(%s): %s", outputName, err)
				break
			}
			log.Debug("updater: created directory %s", outputName)
			continue
		} else if header.Typeflag != tar.TypeReg {
			log.Debug("updater: %s: unknown file type %d, skipping", inputNameOnly, header.Typeflag)
			continue
		}

		f, err := os.OpenFile(outputName, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, os.FileMode(header.Mode&0777))
		if err != nil {
			err2 = fmt.Errorf("os.OpenFile(%s): %s", outputName, err)
			break
		}
		_, err = io.Copy(f, tarReader)
		if err != nil {
			_ = f.Close()
			err2 = fmt.Errorf("io.Copy(): %s", err)
			break
		}
		err = f.Close()
		if err != nil {
			err2 = fmt.Errorf("f.Close(): %s", err)
			break
		}

		log.Debug("updater: created file %s", outputName)
		files = append(files, header.Name)
	}

	_ = gzReader.Close()
	return files, err2
}

// Unpack all files from .zip file to the specified directory
// Existing files are overwritten
// All files are created inside 'outdir', subdirectories are not created
// Return the list of files (not directories) written
func zipFileUnpack(zipfile, outdir string) ([]string, error) {
	r, err := zip.OpenReader(zipfile)
	if err != nil {
		return nil, fmt.Errorf("zip.OpenReader(): %s", err)
	}
	defer r.Close()

	var files []string
	var err2 error
	var zr io.ReadCloser
	for _, zf := range r.File {
		zr, err = zf.Open()
		if err != nil {
			err2 = fmt.Errorf("zip file Open(): %s", err)
			break
		}

		fi := zf.FileInfo()
		inputNameOnly := fi.Name()
		if len(inputNameOnly) == 0 {
			continue
		}

		outputName := filepath.Join(outdir, inputNameOnly)

		if fi.IsDir() {
			err = os.Mkdir(outputName, fi.Mode())
			if err != nil && !os.IsExist(err) {
				err2 = fmt.Errorf("os.Mkdir(): %s", err)
				break
			}
			log.Tracef("created directory %s", outputName)
			continue
		}

		f, err := os.OpenFile(outputName, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, fi.Mode())
		if err != nil {
			err2 = fmt.Errorf("os.OpenFile(): %s", err)
			break
		}
		_, err = io.Copy(f, zr)
		if err != nil {
			_ = f.Close()
			err2 = fmt.Errorf("io.Copy(): %s", err)
			break
		}
		err = f.Close()
		if err != nil {
			err2 = fmt.Errorf("f.Close(): %s", err)
			break
		}

		log.Tracef("created file %s", outputName)
		files = append(files, inputNameOnly)
	}

	_ = zr.Close()
	return files, err2
}

// Copy file on disk
func copyFile(src, dst string) error {
	d, e := ioutil.ReadFile(src)
	if e != nil {
		return e
	}
	e = ioutil.WriteFile(dst, d, 0644)
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
		if err != nil && !os.IsNotExist(err) {
			return err
		}

		log.Debug("updater: copied: %s -> %s", src, dst)
	}
	return nil
}

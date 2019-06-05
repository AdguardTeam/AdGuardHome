package main

import (
	"archive/tar"
	"archive/zip"
	"compress/gzip"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"syscall"
	"time"

	"github.com/AdguardTeam/golibs/log"
)

// Convert version.json data to our JSON response
func getVersionResp(data []byte) []byte {
	versionJSON := make(map[string]interface{})
	err := json.Unmarshal(data, &versionJSON)
	if err != nil {
		log.Error("version.json: %s", err)
		return []byte{}
	}

	ret := make(map[string]interface{})
	ret["can_autoupdate"] = false

	var ok1, ok2, ok3 bool
	ret["new_version"], ok1 = versionJSON["version"].(string)
	ret["announcement"], ok2 = versionJSON["announcement"].(string)
	ret["announcement_url"], ok3 = versionJSON["announcement_url"].(string)
	selfUpdateMinVersion, ok4 := versionJSON["selfupdate_min_version"].(string)
	if !ok1 || !ok2 || !ok3 || !ok4 {
		log.Error("version.json: invalid data")
		return []byte{}
	}

	_, ok := versionJSON[fmt.Sprintf("download_%s_%s", runtime.GOOS, runtime.GOARCH)]
	if ok && ret["new_version"] != VersionString && VersionString >= selfUpdateMinVersion {
		ret["can_autoupdate"] = true
	}

	d, _ := json.Marshal(ret)
	return d
}

// Get the latest available version from the Internet
func handleGetVersionJSON(w http.ResponseWriter, r *http.Request) {
	log.Tracef("%s %v", r.Method, r.URL)

	if config.disableUpdate {
		log.Tracef("New app version check is disabled by user")
		return
	}

	now := time.Now()
	controlLock.Lock()
	cached := now.Sub(versionCheckLastTime) <= versionCheckPeriod && len(versionCheckJSON) != 0
	data := versionCheckJSON
	controlLock.Unlock()

	if cached {
		// return cached copy
		w.Header().Set("Content-Type", "application/json")
		w.Write(getVersionResp(data))
		return
	}

	resp, err := client.Get(versionCheckURL)
	if err != nil {
		httpError(w, http.StatusBadGateway, "Couldn't get version check json from %s: %T %s\n", versionCheckURL, err, err)
		return
	}
	if resp != nil && resp.Body != nil {
		defer resp.Body.Close()
	}

	// read the body entirely
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		httpError(w, http.StatusBadGateway, "Couldn't read response body from %s: %s", versionCheckURL, err)
		return
	}

	controlLock.Lock()
	versionCheckLastTime = now
	versionCheckJSON = body
	controlLock.Unlock()

	w.Header().Set("Content-Type", "application/json")
	_, err = w.Write(getVersionResp(body))
	if err != nil {
		httpError(w, http.StatusInternalServerError, "Couldn't write body: %s", err)
	}
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

type updateInfo struct {
	pkgURL           string // URL for the new package
	pkgName          string // Full path to package file
	newVer           string // New version string
	updateDir        string // Full path to the directory containing unpacked files from the new package
	backupDir        string // Full path to backup directory
	configName       string // Full path to the current configuration file
	updateConfigName string // Full path to the configuration file to check by the new binary
	curBinName       string // Full path to the current executable file
	bkpBinName       string // Full path to the current executable file in backup directory
	newBinName       string // Full path to the new executable file
}

// Fill in updateInfo object
func getUpdateInfo(jsonData []byte) (*updateInfo, error) {
	var u updateInfo

	workDir := config.ourWorkingDir

	versionJSON := make(map[string]interface{})
	err := json.Unmarshal(jsonData, &versionJSON)
	if err != nil {
		return nil, fmt.Errorf("JSON parse: %s", err)
	}

	u.pkgURL = versionJSON[fmt.Sprintf("download_%s_%s", runtime.GOOS, runtime.GOARCH)].(string)
	u.newVer = versionJSON["version"].(string)
	if len(u.pkgURL) == 0 || len(u.newVer) == 0 {
		return nil, fmt.Errorf("Invalid JSON")
	}

	if u.newVer == VersionString {
		return nil, fmt.Errorf("No need to update")
	}

	_, pkgFileName := filepath.Split(u.pkgURL)
	if len(pkgFileName) == 0 {
		return nil, fmt.Errorf("Invalid JSON")
	}
	u.pkgName = filepath.Join(workDir, pkgFileName)

	u.updateDir = filepath.Join(workDir, fmt.Sprintf("update-%s", u.newVer))
	u.backupDir = filepath.Join(workDir, fmt.Sprintf("backup-%s", VersionString))
	u.configName = config.getConfigFilename()
	u.updateConfigName = filepath.Join(u.updateDir, "AdGuardHome", "AdGuardHome.yaml")
	if strings.HasSuffix(pkgFileName, ".zip") {
		u.updateConfigName = filepath.Join(u.updateDir, "AdGuardHome.yaml")
	}

	binName := "AdGuardHome"
	if runtime.GOOS == "windows" {
		binName = "AdGuardHome.exe"
	}
	u.curBinName = filepath.Join(workDir, binName)
	u.bkpBinName = filepath.Join(u.backupDir, binName)
	u.newBinName = filepath.Join(u.updateDir, "AdGuardHome", binName)
	if strings.HasSuffix(pkgFileName, ".zip") {
		u.newBinName = filepath.Join(u.updateDir, binName)
	}

	return &u, nil
}

// Unpack all files from .zip file to the specified directory
// Existing files are overwritten
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
		if len(fi.Name()) == 0 {
			continue
		}

		fn := filepath.Join(outdir, fi.Name())

		if fi.IsDir() {
			err = os.Mkdir(fn, fi.Mode())
			if err != nil && !os.IsExist(err) {
				err2 = fmt.Errorf("os.Mkdir(): %s", err)
				break
			}
			log.Tracef("created directory %s", fn)
			continue
		}

		f, err := os.OpenFile(fn, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, fi.Mode())
		if err != nil {
			err2 = fmt.Errorf("os.OpenFile(): %s", err)
			break
		}
		_, err = io.Copy(f, zr)
		if err != nil {
			f.Close()
			err2 = fmt.Errorf("io.Copy(): %s", err)
			break
		}
		f.Close()

		log.Tracef("created file %s", fn)
		files = append(files, fi.Name())
	}

	zr.Close()
	return files, err2
}

// Unpack all files from .tar.gz file to the specified directory
// Existing files are overwritten
// Return the list of files (not directories) written
func targzFileUnpack(tarfile, outdir string) ([]string, error) {

	f, err := os.Open(tarfile)
	if err != nil {
		return nil, fmt.Errorf("os.Open(): %s", err)
	}
	defer f.Close()

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
		if len(header.Name) == 0 {
			continue
		}

		fn := filepath.Join(outdir, header.Name)

		if header.Typeflag == tar.TypeDir {
			err = os.Mkdir(fn, os.FileMode(header.Mode&0777))
			if err != nil && !os.IsExist(err) {
				err2 = fmt.Errorf("os.Mkdir(%s): %s", fn, err)
				break
			}
			log.Tracef("created directory %s", fn)
			continue
		} else if header.Typeflag != tar.TypeReg {
			log.Tracef("%s: unknown file type %d, skipping", header.Name, header.Typeflag)
			continue
		}

		f, err := os.OpenFile(fn, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, os.FileMode(header.Mode&0777))
		if err != nil {
			err2 = fmt.Errorf("os.OpenFile(%s): %s", fn, err)
			break
		}
		_, err = io.Copy(f, tarReader)
		if err != nil {
			f.Close()
			err2 = fmt.Errorf("io.Copy(): %s", err)
			break
		}
		f.Close()

		log.Tracef("created file %s", fn)
		files = append(files, header.Name)
	}

	gzReader.Close()
	return files, err2
}

// Perform an update procedure
func doUpdate(u *updateInfo) error {
	log.Info("Updating from %s to %s.  URL:%s  Package:%s",
		VersionString, u.newVer, u.pkgURL, u.pkgName)

	resp, err := client.Get(u.pkgURL)
	if err != nil {
		return fmt.Errorf("HTTP request failed: %s", err)
	}
	if resp != nil && resp.Body != nil {
		defer resp.Body.Close()
	}

	log.Tracef("Reading HTTP body")
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("ioutil.ReadAll() failed: %s", err)
	}

	log.Tracef("Saving package to file")
	err = ioutil.WriteFile(u.pkgName, body, 0644)
	if err != nil {
		return fmt.Errorf("ioutil.WriteFile() failed: %s", err)
	}

	log.Tracef("Unpacking the package")
	_ = os.Mkdir(u.updateDir, 0755)
	_, file := filepath.Split(u.pkgName)
	if strings.HasSuffix(file, ".zip") {
		_, err = zipFileUnpack(u.pkgName, u.updateDir)
		if err != nil {
			return fmt.Errorf("zipFileUnpack() failed: %s", err)
		}
	} else if strings.HasSuffix(file, ".tar.gz") {
		_, err = targzFileUnpack(u.pkgName, u.updateDir)
		if err != nil {
			return fmt.Errorf("zipFileUnpack() failed: %s", err)
		}
	} else {
		return fmt.Errorf("Unknown package extension")
	}

	log.Tracef("Checking configuration")
	err = copyFile(u.configName, u.updateConfigName)
	if err != nil {
		return fmt.Errorf("copyFile() failed: %s", err)
	}
	cmd := exec.Command(u.newBinName, "--check-config")
	err = cmd.Run()
	if err != nil || cmd.ProcessState.ExitCode() != 0 {
		return fmt.Errorf("exec.Command(): %s %d", err, cmd.ProcessState.ExitCode())
	}

	log.Tracef("Backing up the current configuration")
	_ = os.Mkdir(u.backupDir, 0755)
	err = copyFile(u.configName, filepath.Join(u.backupDir, "AdGuardHome.yaml"))
	if err != nil {
		return fmt.Errorf("copyFile() failed: %s", err)
	}

	log.Tracef("Renaming: %s -> %s", u.curBinName, u.bkpBinName)
	err = os.Rename(u.curBinName, u.bkpBinName)
	if err != nil {
		return err
	}
	if runtime.GOOS == "windows" {
		// rename fails with "File in use" error
		err = copyFile(u.newBinName, u.curBinName)
	} else {
		err = os.Rename(u.newBinName, u.curBinName)
	}
	if err != nil {
		return err
	}
	log.Tracef("Renamed: %s -> %s", u.newBinName, u.curBinName)

	_ = os.Remove(u.pkgName)
	// _ = os.RemoveAll(u.updateDir)
	return nil
}

// Complete an update procedure
func finishUpdate(u *updateInfo) {
	log.Info("Stopping all tasks")
	cleanup()
	stopHTTPServer()
	cleanupAlways()

	if runtime.GOOS == "windows" {

		if config.runningAsService {
			// Note:
			// we can't restart the service via "kardianos/service" package - it kills the process first
			// we can't start a new instance - Windows doesn't allow it
			cmd := exec.Command("cmd", "/c", "net stop AdGuardHome & net start AdGuardHome")
			err := cmd.Start()
			if err != nil {
				log.Fatalf("exec.Command() failed: %s", err)
			}
			os.Exit(0)
		}

		cmd := exec.Command(u.curBinName, os.Args[1:]...)
		log.Info("Restarting: %v", cmd.Args)
		cmd.Stdin = os.Stdin
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		err := cmd.Start()
		if err != nil {
			log.Fatalf("exec.Command() failed: %s", err)
		}
		os.Exit(0)

	} else {

		log.Info("Restarting: %v", os.Args)
		err := syscall.Exec(u.curBinName, os.Args, os.Environ())
		if err != nil {
			log.Fatalf("syscall.Exec() failed: %s", err)
		}
		// Unreachable code
	}
}

// Perform an update procedure to the latest available version
func handleUpdate(w http.ResponseWriter, r *http.Request) {
	log.Tracef("%s %v", r.Method, r.URL)

	if len(versionCheckJSON) == 0 {
		httpError(w, http.StatusBadRequest, "/update request isn't allowed now")
		return
	}

	u, err := getUpdateInfo(versionCheckJSON)
	if err != nil {
		httpError(w, http.StatusInternalServerError, "%s", err)
		return
	}

	err = doUpdate(u)
	if err != nil {
		httpError(w, http.StatusInternalServerError, "%s", err)
		return
	}

	returnOK(w)

	time.Sleep(time.Second) // wait (hopefully) until response is sent (not sure whether it's really necessary)
	go finishUpdate(u)
}

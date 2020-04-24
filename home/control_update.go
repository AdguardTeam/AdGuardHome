package home

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

	"github.com/AdguardTeam/AdGuardHome/util"

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

	// the key is download_linux_arm or download_linux_arm64 for regular ARM versions
	dloadName := fmt.Sprintf("download_%s_%s", runtime.GOOS, runtime.GOARCH)
	if runtime.GOARCH == "arm" && ARMVersion == "5" {
		// the key is download_linux_armv5 for ARMv5
		dloadName = fmt.Sprintf("download_%s_%sv%s", runtime.GOOS, runtime.GOARCH, ARMVersion)
	}
	_, ok := versionJSON[dloadName]
	if ok && ret["new_version"] != versionString && versionString >= selfUpdateMinVersion {
		canUpdate := true

		tlsConf := tlsConfigSettings{}
		Context.tls.WriteDiskConfig(&tlsConf)

		if runtime.GOOS != "windows" &&
			((tlsConf.Enabled && (tlsConf.PortHTTPS < 1024 || tlsConf.PortDNSOverTLS < 1024)) ||
				config.BindPort < 1024 ||
				config.DNS.Port < 1024) {
			// On UNIX, if we're running under a regular user,
			//  but with CAP_NET_BIND_SERVICE set on a binary file,
			//  and we're listening on ports <1024,
			//  we won't be able to restart after we replace the binary file,
			//  because we'll lose CAP_NET_BIND_SERVICE capability.
			canUpdate, _ = util.HaveAdminRights()
		}
		ret["can_autoupdate"] = canUpdate
	}

	d, _ := json.Marshal(ret)
	return d
}

type getVersionJSONRequest struct {
	RecheckNow bool `json:"recheck_now"`
}

// Get the latest available version from the Internet
func handleGetVersionJSON(w http.ResponseWriter, r *http.Request) {

	if Context.disableUpdate {
		return
	}

	req := getVersionJSONRequest{}
	err := json.NewDecoder(r.Body).Decode(&req)
	if err != nil {
		httpError(w, http.StatusBadRequest, "JSON parse: %s", err)
		return
	}

	now := time.Now()
	if !req.RecheckNow {
		Context.controlLock.Lock()
		cached := now.Sub(config.versionCheckLastTime) <= versionCheckPeriod && len(config.versionCheckJSON) != 0
		data := config.versionCheckJSON
		Context.controlLock.Unlock()

		if cached {
			log.Tracef("Returning cached data")
			w.Header().Set("Content-Type", "application/json")
			w.Write(getVersionResp(data))
			return
		}
	}

	var resp *http.Response
	for i := 0; i != 3; i++ {
		log.Tracef("Downloading data from %s", versionCheckURL)
		resp, err = Context.client.Get(versionCheckURL)
		if resp != nil && resp.Body != nil {
			defer resp.Body.Close()
		}
		if err != nil && strings.HasSuffix(err.Error(), "i/o timeout") {
			// This case may happen while we're restarting DNS server
			// https://github.com/AdguardTeam/AdGuardHome/issues/934
			continue
		}
		break
	}
	if err != nil {
		httpError(w, http.StatusBadGateway, "Couldn't get version check json from %s: %T %s\n", versionCheckURL, err, err)
		return
	}

	// read the body entirely
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		httpError(w, http.StatusBadGateway, "Couldn't read response body from %s: %s", versionCheckURL, err)
		return
	}

	Context.controlLock.Lock()
	config.versionCheckLastTime = now
	config.versionCheckJSON = body
	Context.controlLock.Unlock()

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

	workDir := Context.workDir

	versionJSON := make(map[string]interface{})
	err := json.Unmarshal(jsonData, &versionJSON)
	if err != nil {
		return nil, fmt.Errorf("JSON parse: %s", err)
	}

	u.pkgURL = versionJSON[fmt.Sprintf("download_%s_%s", runtime.GOOS, runtime.GOARCH)].(string)
	u.newVer = versionJSON["version"].(string)
	if len(u.pkgURL) == 0 || len(u.newVer) == 0 {
		return nil, fmt.Errorf("invalid JSON")
	}

	if u.newVer == versionString {
		return nil, fmt.Errorf("no need to update")
	}

	u.updateDir = filepath.Join(workDir, fmt.Sprintf("agh-update-%s", u.newVer))
	u.backupDir = filepath.Join(workDir, "agh-backup")

	_, pkgFileName := filepath.Split(u.pkgURL)
	if len(pkgFileName) == 0 {
		return nil, fmt.Errorf("invalid JSON")
	}
	u.pkgName = filepath.Join(u.updateDir, pkgFileName)

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
	if !util.FileExists(u.curBinName) {
		return nil, fmt.Errorf("executable file %s doesn't exist", u.curBinName)
	}
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

func copySupportingFiles(files []string, srcdir, dstdir string, useSrcNameOnly, useDstNameOnly bool) error {
	for _, f := range files {
		_, name := filepath.Split(f)
		if name == "AdGuardHome" || name == "AdGuardHome.exe" || name == "AdGuardHome.yaml" {
			continue
		}

		src := filepath.Join(srcdir, f)
		if useSrcNameOnly {
			src = filepath.Join(srcdir, name)
		}

		dst := filepath.Join(dstdir, f)
		if useDstNameOnly {
			dst = filepath.Join(dstdir, name)
		}

		err := copyFile(src, dst)
		if err != nil && !os.IsNotExist(err) {
			return err
		}

		log.Tracef("Copied: %s -> %s", src, dst)
	}
	return nil
}

// Download package file and save it to disk
func getPackageFile(u *updateInfo) error {
	resp, err := Context.client.Get(u.pkgURL)
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
	return nil
}

// Perform an update procedure
func doUpdate(u *updateInfo) error {
	log.Info("Updating from %s to %s.  URL:%s  Package:%s",
		versionString, u.newVer, u.pkgURL, u.pkgName)

	_ = os.Mkdir(u.updateDir, 0755)

	var err error
	err = getPackageFile(u)
	if err != nil {
		return err
	}

	log.Tracef("Unpacking the package")
	_, file := filepath.Split(u.pkgName)
	var files []string
	if strings.HasSuffix(file, ".zip") {
		files, err = zipFileUnpack(u.pkgName, u.updateDir)
		if err != nil {
			return fmt.Errorf("zipFileUnpack() failed: %s", err)
		}
	} else if strings.HasSuffix(file, ".tar.gz") {
		files, err = targzFileUnpack(u.pkgName, u.updateDir)
		if err != nil {
			return fmt.Errorf("targzFileUnpack() failed: %s", err)
		}
	} else {
		return fmt.Errorf("unknown package extension")
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

	// ./README.md -> backup/README.md
	err = copySupportingFiles(files, Context.workDir, u.backupDir, true, true)
	if err != nil {
		return fmt.Errorf("copySupportingFiles(%s, %s) failed: %s",
			Context.workDir, u.backupDir, err)
	}

	// update/[AdGuardHome/]README.md -> ./README.md
	err = copySupportingFiles(files, u.updateDir, Context.workDir, false, true)
	if err != nil {
		return fmt.Errorf("copySupportingFiles(%s, %s) failed: %s",
			u.updateDir, Context.workDir, err)
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
	_ = os.RemoveAll(u.updateDir)
	return nil
}

// Complete an update procedure
func finishUpdate(u *updateInfo) {
	log.Info("Stopping all tasks")
	cleanup()
	cleanupAlways()

	if runtime.GOOS == "windows" {
		if Context.runningAsService {
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

	if len(config.versionCheckJSON) == 0 {
		httpError(w, http.StatusBadRequest, "/update request isn't allowed now")
		return
	}

	u, err := getUpdateInfo(config.versionCheckJSON)
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
	if f, ok := w.(http.Flusher); ok {
		f.Flush()
	}

	go finishUpdate(u)
}

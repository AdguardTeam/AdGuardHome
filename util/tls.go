package util

import (
	"crypto/x509"
	"io/ioutil"
	"os"
	"runtime"

	"github.com/AdguardTeam/golibs/log"
)

// LoadSystemRootCAs - load root CAs from the system
// Return the x509 certificate pool object
// Return nil if nothing has been found.
//  This means that Go.crypto will use its default algorithm to find system root CA list.
// https://github.com/AdguardTeam/AdGuardHome/issues/1311
func LoadSystemRootCAs() *x509.CertPool {
	if runtime.GOOS != "linux" {
		return nil
	}

	// Directories with the system root certificates, that aren't supported by Go.crypto
	dirs := []string{
		"/opt/etc/ssl/certs", // Entware
	}
	roots := x509.NewCertPool()
	for _, dir := range dirs {
		fis, err := ioutil.ReadDir(dir)
		if err != nil {
			if !os.IsNotExist(err) {
				log.Error("Opening directory: %s: %s", dir, err)
			}
			continue
		}
		rootsAdded := false
		for _, fi := range fis {
			data, err := ioutil.ReadFile(dir + "/" + fi.Name())
			if err == nil && roots.AppendCertsFromPEM(data) {
				rootsAdded = true
			}
		}
		if rootsAdded {
			return roots
		}
	}
	return nil
}

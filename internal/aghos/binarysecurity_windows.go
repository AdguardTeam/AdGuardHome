//go:build windows

package aghos

import (
	"fmt"
	"os"
	"strings"
)

// securePrefixDirectories is a list of directories where a service binary
// has the appropriate permissions to mitigate a binary planting attack
var securePrefixDirectories = []string{
	"C:\\Program Files",
	"C:\\Program Files (x86)",

	// Some Windows users place binaries within /Windows/System32 to add it to %PATH%
	"C:\\Windows",
}

// SecureBinary is used before service.Install(). This function protects AdGuardHome from
// privilege escalation vulnerabilities caused by writable files
func SecureBinary() error {
	// Get current file path
	binary, err := os.Executable()
	if err != nil {
		return fmt.Errorf("os.Executable(): %w", err)
	}

	for i := 0; i < len(securePrefixDirectories); i++ {
		// Check if binary is within a secure folder write protected folder
		if strings.HasPrefix(binary, securePrefixDirectories[i]) {
			// The binary is within a secure directory already
			return nil
		}
	}

	// No secure directories matched
	return fmt.Errorf("insecure binary location for service instalation: %q. Please view: https://adguard-dns.io/kb/adguard-home/running-securely/", binary)
}

// CurrentDirAvaliable returns true if it is okay to use this directory to store application
// data.
func CurrentDirAvaliable() (bool, error) {
	// We do not mind what directory is used on Windows
	return true, nil
}

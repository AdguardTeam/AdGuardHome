//go:build !windows

package aghos

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/AdguardTeam/golibs/errors"
	"github.com/AdguardTeam/golibs/log"
)

// protectedDirectories are directories which contain other application binaries,
// as such AdGuard Home should never attempt store application data here, at risk of
// overwriting other files. Moreover, these directories are innapproriate for storage of
// config files or session storage.
var protectedDirectories = []string{
	"/usr/bin"
	"/usr/sbin"
	"/user/bin"
}

// serviceInstallDir is a executable path in a directory which secure permissions
// which prevent the manipulation of the binary. 
const serviceInstallDir = "/usr/bin/AdGuardHome"

// SecureBinary is used before service.Install(). This function protects AdGuardHome from
// privilege escalation vulnerabilities caused by writable files
func SecureBinary() error {
	// Installalation can only be completed with root privileges, so check and handle if not
	if os.Getuid() != 0 {
		return errors.Error("permission denied. Root privileges required")
	}

	// Get current file path
	binary, err := os.Executable()
	if err != nil {
		return fmt.Errorf("os.Executable(): %w", err)
	}

	// Change owner to root:root
	err = os.Chown(binary, 0, 0)
	if err != nil {
		return fmt.Errorf("os.Chown() %q: %w", binary, err)
	}

	// Set permissions to root(read,write,exec), group(read,exec), public(read)
	// This combined with changing the owner make the file undeletable without root privlages
	// UNLESS THE PARENT FOLDER IS WRITABLE!
	if err := os.Chmod(binary, 0755); err != nil {
		return fmt.Errorf("os.Chmod() %q: %w", binary, err)
	}


	// Move binary to the PATH in a folder which is read-only to non root users
	// If already moved, this is a no-op
	if err := os.Rename(binary, serviceInstallDir); err != nil {
		return fmt.Errorf("os.Rename() %q to %q: %w", binary, installDir, err)
	}

	return nil
}

// CurrentDirAvaliable returns true if it is okay to use this directory to store application
// data.
func CurrentDirAvaliable() (bool, error) {
	binary, err := os.Executable()
	if err != nil {
		return fmt.Errorf("os.Executable(): %w", err)
	}

	for i := 0; i < len(protectedDirectories); i++ {
		// Check if binary is within a protected directory
		if strings.HasPrefix(binary, protectedDirectories[i]) {
			// The binary is within a protected directory
			return false, nil
		}
	}

	// The binary is outside of all checked protected directories
	return true, nil
}
//go:build !linux

package home

// Notifies the service manager that the program is ready to serve
func notifyReady() error {
	return nil
}

// Notifies the service manager that the program is beginning to reload its
// configuration
func notifyReload() error {
	return nil
}

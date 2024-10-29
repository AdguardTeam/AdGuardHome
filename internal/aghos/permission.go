package aghos

import (
	"io/fs"
	"os"
)

// TODO(e.burkov):  Add platform-independent tests.

// Chmod is an extension for [os.Chmod] that properly handles Windows access
// rights.
func Chmod(name string, perm fs.FileMode) (err error) {
	return chmod(name, perm)
}

// Mkdir is an extension for [os.Mkdir] that properly handles Windows access
// rights.
func Mkdir(name string, perm fs.FileMode) (err error) {
	return mkdir(name, perm)
}

// MkdirAll is an extension for [os.MkdirAll] that properly handles Windows
// access rights.
func MkdirAll(path string, perm fs.FileMode) (err error) {
	return mkdirAll(path, perm)
}

// WriteFile is an extension for [os.WriteFile] that properly handles Windows
// access rights.
func WriteFile(filename string, data []byte, perm fs.FileMode) (err error) {
	return writeFile(filename, data, perm)
}

// OpenFile is an extension for [os.OpenFile] that properly handles Windows
// access rights.
func OpenFile(name string, flag int, perm fs.FileMode) (file *os.File, err error) {
	return openFile(name, flag, perm)
}

// Stat is an extension for [os.Stat] that properly handles Windows access
// rights.
//
// Note that on Windows the "other" permission bits combines the access rights
// of any trustee that is neither the owner nor the owning group for the file.
//
// TODO(e.burkov):  Inspect the behavior for the World (everyone) well-known
// SID and, perhaps, use it.
func Stat(name string) (fi fs.FileInfo, err error) {
	return stat(name)
}

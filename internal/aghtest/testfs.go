package aghtest

import "io/fs"

// type check
var _ fs.FS = &FS{}

// FS is a mock fs.FS implementation to use in tests.
type FS struct {
	OnOpen func(name string) (fs.File, error)
}

// Open implements the fs.FS interface for *FS.
func (fsys *FS) Open(name string) (fs.File, error) {
	return fsys.OnOpen(name)
}

// type check
var _ fs.StatFS = &StatFS{}

// StatFS is a mock fs.StatFS implementation to use in tests.
type StatFS struct {
	// FS is embedded here to avoid implementing all it's methods.
	FS
	OnStat func(name string) (fs.FileInfo, error)
}

// Stat implements the fs.StatFS interface for *StatFS.
func (fsys *StatFS) Stat(name string) (fs.FileInfo, error) {
	return fsys.OnStat(name)
}

// type check
var _ fs.GlobFS = &GlobFS{}

// GlobFS is a mock fs.GlobFS implementation to use in tests.
type GlobFS struct {
	// FS is embedded here to avoid implementing all it's methods.
	FS
	OnGlob func(pattern string) ([]string, error)
}

// Glob implements the fs.GlobFS interface for *GlobFS.
func (fsys *GlobFS) Glob(pattern string) ([]string, error) {
	return fsys.OnGlob(pattern)
}

//go:build !(openbsd || linux)

package home

// chooseSystem checks the current system detected and substitutes it with local
// implementation if needed.
func chooseSystem() {}

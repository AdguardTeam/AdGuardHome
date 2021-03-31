// Package version contains AdGuard Home version information.
package version

import (
	"fmt"
	"runtime"
	"runtime/debug"
	"strconv"
	"strings"
)

// Channel constants.
const (
	ChannelDevelopment = "development"
	ChannelEdge        = "edge"
	ChannelBeta        = "beta"
	ChannelRelease     = "release"
)

// These are set by the linker.  Unfortunately we cannot set constants during
// linking, and Go doesn't have a concept of immutable variables, so to be
// thorough we have to only export them through getters.
//
// TODO(a.garipov): Find out if we can get GOARM and GOMIPS values the same way
// we can GOARCH and GOOS.
var (
	channel   string = ChannelDevelopment
	goarm     string
	gomips    string
	version   string
	buildtime string
)

// Channel returns the current AdGuard Home release channel.
func Channel() (v string) {
	return channel
}

// vFmtFull defines the format of full version output.
const vFmtFull = "AdGuard Home, version %s"

// Full returns the full current version of AdGuard Home.
func Full() (v string) {
	return fmt.Sprintf(vFmtFull, version)
}

// GOARM returns the GOARM value used to build the current AdGuard Home release.
func GOARM() (v string) {
	return goarm
}

// GOMIPS returns the GOMIPS value used to build the current AdGuard Home
// release.
func GOMIPS() (v string) {
	return gomips
}

// Version returns the AdGuard Home build version.
func Version() (v string) {
	return version
}

// Common formatting constants.
const (
	sp   = " "
	nl   = "\n"
	tb   = "\t"
	nltb = nl + tb
)

// writeStrings is a convenient wrapper for strings.(*Builder).WriteString that
// deals with multiple strings and ignores errors that are guaranteed to be nil.
func writeStrings(b *strings.Builder, strs ...string) {
	for _, s := range strs {
		_, _ = b.WriteString(s)
	}
}

// Constants defining the format of module information string.
const (
	modInfoAtSep    = "@"
	modInfoDevSep   = sp
	modInfoSumLeft  = " (sum: "
	modInfoSumRight = ")"
)

// fmtModule returns formatted information about module.  The result looks like:
//
//   github.com/Username/module@v1.2.3 (sum: someHASHSUM=)
//
func fmtModule(m *debug.Module) (formatted string) {
	if m == nil {
		return ""
	}

	if repl := m.Replace; repl != nil {
		return fmtModule(repl)
	}

	b := &strings.Builder{}

	writeStrings(b, m.Path)
	if ver := m.Version; ver != "" {
		sep := modInfoAtSep
		if ver == "(devel)" {
			sep = modInfoDevSep
		}
		writeStrings(b, sep, ver)
	}
	if sum := m.Sum; sum != "" {
		writeStrings(b, modInfoSumLeft, sum, modInfoSumRight)
	}

	return b.String()
}

// Constants defining the headers of build information message.
const (
	vFmtAGHHdr    = "AdGuard Home"
	vFmtVerHdr    = "Version: "
	vFmtChanHdr   = "Channel: "
	vFmtGoHdr     = "Go version: "
	vFmtTimeHdr   = "Build time: "
	vFmtRaceHdr   = "Race: "
	vFmtGOOSHdr   = "GOOS: " + runtime.GOOS
	vFmtGOARCHHdr = "GOARCH: " + runtime.GOARCH
	vFmtGOARMHdr  = "GOARM: "
	vFmtGOMIPSHdr = "GOMIPS: "
	vFmtMainHdr   = "Main module:"
	vFmtDepsHdr   = "Dependencies:"
)

// Verbose returns formatted build information.  Output example:
//
//   AdGuard Home
//   Version: v0.105.3
//   Channel: development
//   Go version: go1.15.3
//   Build time: 2021-03-30T16:26:08Z+0300
//   GOOS: darwin
//   GOARCH: amd64
//   Race: false
//   Main module:
//           ...
//   Dependencies:
//           ...
//
// TODO(e.burkov): Make it write into passed io.Writer.
func Verbose() (v string) {
	b := &strings.Builder{}

	writeStrings(
		b,
		vFmtAGHHdr,
		nl,
		vFmtVerHdr,
		version,
		nl,
		vFmtChanHdr,
		channel,
		nl,
		vFmtGoHdr,
		runtime.Version(),
	)
	if buildtime != "" {
		writeStrings(b, nl, vFmtTimeHdr, buildtime)
	}
	writeStrings(b, nl, vFmtGOOSHdr, nl, vFmtGOARCHHdr)
	if goarm != "" {
		writeStrings(b, nl, vFmtGOARMHdr, "v", goarm)
	} else if gomips != "" {
		writeStrings(b, nl, vFmtGOMIPSHdr, gomips)
	}
	writeStrings(b, nl, vFmtRaceHdr, strconv.FormatBool(isRace))

	info, ok := debug.ReadBuildInfo()
	if !ok {
		return b.String()
	}

	writeStrings(b, nl, vFmtMainHdr, nltb, fmtModule(&info.Main))

	if len(info.Deps) == 0 {
		return b.String()
	}

	writeStrings(b, nl, vFmtDepsHdr)
	for _, dep := range info.Deps {
		if depStr := fmtModule(dep); depStr != "" {
			writeStrings(b, nltb, depStr)
		}
	}

	return b.String()
}

// Package version contains AdGuard Home version information.
package version

import (
	"fmt"
	"runtime"
	"runtime/debug"
	"strconv"
	"strings"
	"time"

	"github.com/AdguardTeam/golibs/stringutil"
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
	channel    string = ChannelDevelopment
	goarm      string
	gomips     string
	version    string
	committime string
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

// fmtModule returns formatted information about module.  The result looks like:
//
//	github.com/Username/module@v1.2.3 (sum: someHASHSUM=)
func fmtModule(m *debug.Module) (formatted string) {
	if m == nil {
		return ""
	}

	if repl := m.Replace; repl != nil {
		return fmtModule(repl)
	}

	b := &strings.Builder{}

	stringutil.WriteToBuilder(b, m.Path)
	if ver := m.Version; ver != "" {
		sep := "@"
		if ver == "(devel)" {
			sep = " "
		}

		stringutil.WriteToBuilder(b, sep, ver)
	}

	if sum := m.Sum; sum != "" {
		stringutil.WriteToBuilder(b, "(sum: ", sum, ")")
	}

	return b.String()
}

// Constants defining the headers of build information message.
const (
	vFmtAGHHdr    = "AdGuard Home"
	vFmtVerHdr    = "Version: "
	vFmtChanHdr   = "Channel: "
	vFmtGoHdr     = "Go version: "
	vFmtTimeHdr   = "Commit time: "
	vFmtRaceHdr   = "Race: "
	vFmtGOOSHdr   = "GOOS: " + runtime.GOOS
	vFmtGOARCHHdr = "GOARCH: " + runtime.GOARCH
	vFmtGOARMHdr  = "GOARM: "
	vFmtGOMIPSHdr = "GOMIPS: "
	vFmtDepsHdr   = "Dependencies:"
)

// Verbose returns formatted build information.  Output example:
//
//	AdGuard Home
//	Version: v0.105.3
//	Channel: development
//	Go version: go1.15.3
//	Build time: 2021-03-30T16:26:08Z+0300
//	GOOS: darwin
//	GOARCH: amd64
//	Race: false
//	Main module:
//	        ...
//	Dependencies:
//	        ...
//
// TODO(e.burkov): Make it write into passed io.Writer.
func Verbose() (v string) {
	b := &strings.Builder{}

	const nl = "\n"
	stringutil.WriteToBuilder(
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

	if committime != "" {
		commitTimeUnix, err := strconv.ParseInt(committime, 10, 64)
		if err != nil {
			stringutil.WriteToBuilder(b, nl, vFmtTimeHdr, fmt.Sprintf("parse error: %s", err))
		} else {
			stringutil.WriteToBuilder(b, nl, vFmtTimeHdr, time.Unix(commitTimeUnix, 0).String())
		}
	}

	stringutil.WriteToBuilder(b, nl, vFmtGOOSHdr, nl, vFmtGOARCHHdr)
	if goarm != "" {
		stringutil.WriteToBuilder(b, nl, vFmtGOARMHdr, "v", goarm)
	} else if gomips != "" {
		stringutil.WriteToBuilder(b, nl, vFmtGOMIPSHdr, gomips)
	}

	stringutil.WriteToBuilder(b, nl, vFmtRaceHdr, strconv.FormatBool(isRace))

	info, ok := debug.ReadBuildInfo()
	if !ok {
		return b.String()
	}

	if len(info.Deps) == 0 {
		return b.String()
	}

	stringutil.WriteToBuilder(b, nl, vFmtDepsHdr)
	for _, dep := range info.Deps {
		if depStr := fmtModule(dep); depStr != "" {
			stringutil.WriteToBuilder(b, "\n\t", depStr)
		}
	}

	return b.String()
}

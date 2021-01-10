//go:generate packr clean
//go:generate packr -z
package main

import (
	"github.com/AdguardTeam/AdGuardHome/internal/home"
)

// version is the release version.  It is set by the linker.
var version = "undefined"

// channel is the release channel.  It is set by the linker.
var channel = "release"

// goarm is the GOARM value.  It is set by the linker.
var goarm = ""

// gomips is the GOMIPS value.  It is set by the linker.
var gomips = ""

func main() {
	home.Main(version, channel, goarm, gomips)
}

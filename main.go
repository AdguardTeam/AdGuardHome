//go:generate go install -v github.com/gobuffalo/packr/packr
//go:generate packr clean
//go:generate packr -z
package main

import (
	"github.com/AdguardTeam/AdGuardHome/home"
)

// version will be set through ldflags, contains current version
var version = "undefined"

// channel can be set via ldflags
var channel = "release"

// GOARM value - set via ldflags
var goarm = ""

func main() {
	home.Main(version, channel, goarm)
}

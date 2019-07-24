package main

import (
	"runtime/debug"

	"github.com/AdguardTeam/AdGuardHome/home"
)

// version will be set through ldflags, contains current version
var version = "undefined"

// channel can be set via ldflags
var channel = "release"

func main() {
	debug.SetGCPercent(10)
	home.Main(version, channel)
}

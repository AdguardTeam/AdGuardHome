package main

import (
	"os"
	"runtime/debug"
	"time"

	"github.com/AdguardTeam/AdGuardHome/home"
)

// version will be set through ldflags, contains current version
var version = "undefined"

// channel can be set via ldflags
var channel = "release"

// GOARM value - set via ldflags
var goarm = ""

func main() {
	memoryUsage()

	home.Main(version, channel, goarm)
}

// memoryUsage implements a couple of not really beautiful hacks which purpose is to
// make OS reclaim the memory freed by AdGuard Home as soon as possible.
func memoryUsage() {
	debug.SetGCPercent(10)

	// madvdontneed: setting madvdontneed=1 will use MADV_DONTNEED
	// instead of MADV_FREE on Linux when returning memory to the
	// kernel. This is less efficient, but causes RSS numbers to drop
	// more quickly.
	_ = os.Setenv("GODEBUG", "madvdontneed=1")

	// periodically call "debug.FreeOSMemory" so
	// that the OS could reclaim the free memory
	go func() {
		ticker := time.NewTicker(15 * time.Second)
		for {
			select {
			case t := <-ticker.C:
				t.Second()
				debug.FreeOSMemory()
			}
		}
	}()
}

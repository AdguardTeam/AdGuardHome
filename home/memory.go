package home

import (
	"os"
	"runtime/debug"
	"time"

	"github.com/AdguardTeam/golibs/log"
)

// memoryUsage implements a couple of not really beautiful hacks which purpose is to
// make OS reclaim the memory freed by AdGuard Home as soon as possible.
// See this for the details on the performance hits & gains:
// https://github.com/AdguardTeam/AdGuardHome/issues/2044#issuecomment-687042211
func memoryUsage(args options) {
	if args.disableMemoryOptimization {
		log.Info("Memory optimization is disabled")
		return
	}

	// Makes Go allocate heap at a slower pace
	// By default we keep it at 50%
	debug.SetGCPercent(50)

	// madvdontneed: setting madvdontneed=1 will use MADV_DONTNEED
	// instead of MADV_FREE on Linux when returning memory to the
	// kernel. This is less efficient, but causes RSS numbers to drop
	// more quickly.
	_ = os.Setenv("GODEBUG", "madvdontneed=1")

	// periodically call "debug.FreeOSMemory" so
	// that the OS could reclaim the free memory
	go func() {
		ticker := time.NewTicker(5 * time.Minute)
		for {
			select {
			case t := <-ticker.C:
				t.Second()
				log.Debug("Free OS memory")
				debug.FreeOSMemory()
			}
		}
	}()
}

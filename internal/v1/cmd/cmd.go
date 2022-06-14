// Package cmd is the AdGuard Home entry point.  It contains the on-disk
// configuration file utilities, signal processing logic, and so on.
//
// TODO(a.garipov): Move to the upper-level internal/.
package cmd

import (
	"context"
	"io/fs"
	"math/rand"
	"net"
	"time"

	"github.com/AdguardTeam/AdGuardHome/internal/v1/websvc"
	"github.com/AdguardTeam/golibs/log"
	"github.com/AdguardTeam/golibs/netutil"
)

// Main is the entry point of application.
func Main(clientBuildFS fs.FS) {
	// # Initial Configuration

	start := time.Now()
	rand.Seed(start.UnixNano())

	// TODO(a.garipov): Set up logging.

	// # Web Service

	// TODO(a.garipov): Use in the Web service.
	_ = clientBuildFS

	// TODO(a.garipov): Make configurable.
	web := websvc.New(&websvc.Config{
		Addresses: []*netutil.IPPort{{
			IP:   net.IP{127, 0, 0, 1},
			Port: 3001,
		}},
		Start:   start,
		Timeout: 60 * time.Second,
	})

	err := web.Start()
	fatalOnError(err)

	sigHdlr := newSignalHandler(
		web,
	)

	go sigHdlr.handle()

	select {}
}

// defaultTimeout is the timeout used for some operations where another timeout
// hasn't been defined yet.
const defaultTimeout = 15 * time.Second

// ctxWithDefaultTimeout is a helper function that returns a context with
// timeout set to defaultTimeout.
func ctxWithDefaultTimeout() (ctx context.Context, cancel context.CancelFunc) {
	return context.WithTimeout(context.Background(), defaultTimeout)
}

// fatalOnError is a helper that exits the program with an error code if err is
// not nil.  It must only be used within Main.
func fatalOnError(err error) {
	if err != nil {
		log.Fatal(err)
	}
}

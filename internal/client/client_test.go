package client_test

import (
	"net/netip"
	"testing"
	"time"

	"github.com/AdguardTeam/golibs/testutil"
)

func TestMain(m *testing.M) {
	testutil.DiscardLogOutput(m)
}

// testHost is the common hostname for tests.
const testHost = "client.example"

// testTimeout is the common timeout for tests.
const testTimeout = 1 * time.Second

// testWHOISCity is the common city for tests.
const testWHOISCity = "Brussels"

// testIP is the common IP address for tests.
var testIP = netip.MustParseAddr("1.2.3.4")

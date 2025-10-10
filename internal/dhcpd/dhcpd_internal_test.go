package dhcpd

import (
	"time"

	"github.com/AdguardTeam/golibs/logutil/slogutil"
)

// testTimeout is a common timeout for tests.
const testTimeout = 1 * time.Second

// testLogger is a logger used in tests.
var testLogger = slogutil.NewDiscardLogger()

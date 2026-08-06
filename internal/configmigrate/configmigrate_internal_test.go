package configmigrate

import (
	"time"

	"github.com/AdguardTeam/golibs/logutil/slogutil"
)

// testTimeout is the common timeout for tests.
const testTimeout = 1 * time.Second

// testLogger is a logger used in tests.
var testLogger = slogutil.NewDiscardLogger()

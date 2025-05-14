package home

import (
	"testing"

	"github.com/AdguardTeam/golibs/logutil/slogutil"
	"github.com/AdguardTeam/golibs/testutil"
)

// TODO(s.chzhen): !! Use everywhere.
var testLogger = slogutil.NewDiscardLogger()

func TestMain(m *testing.M) {
	initCmdLineOpts()
	testutil.DiscardLogOutput(m)
}

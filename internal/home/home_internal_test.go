package home

import (
	"testing"

	"github.com/AdguardTeam/golibs/logutil/slogutil"
	"github.com/AdguardTeam/golibs/testutil"
)

var testLogger = slogutil.NewDiscardLogger()

func TestMain(m *testing.M) {
	initCmdLineOpts()
	testutil.DiscardLogOutput(m)
}

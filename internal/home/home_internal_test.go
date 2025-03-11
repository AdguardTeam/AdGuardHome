package home

import (
	"testing"

	"github.com/AdguardTeam/golibs/testutil"
)

func TestMain(m *testing.M) {
	initCmdLineOpts()
	testutil.DiscardLogOutput(m)
}

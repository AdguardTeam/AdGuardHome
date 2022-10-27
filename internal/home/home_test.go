package home

import (
	"testing"

	"github.com/AdguardTeam/golibs/testutil"
)

func TestMain(m *testing.M) {
	testutil.DiscardLogOutput(m)
	initCmdLineOpts()
}

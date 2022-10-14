package home

import (
	"testing"

	"github.com/AdguardTeam/AdGuardHome/internal/aghtest"
)

func TestMain(m *testing.M) {
	aghtest.DiscardLogOutput(m)
	initCmdLineOpts()
}

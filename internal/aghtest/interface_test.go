package aghtest_test

import (
	"github.com/AdguardTeam/AdGuardHome/internal/aghos"
	"github.com/AdguardTeam/AdGuardHome/internal/aghtest"
)

// type check
var _ aghos.FSWatcher = (*aghtest.FSWatcher)(nil)

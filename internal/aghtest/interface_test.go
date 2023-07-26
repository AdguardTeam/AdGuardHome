package aghtest_test

import (
	"github.com/AdguardTeam/AdGuardHome/internal/aghtest"
	"github.com/AdguardTeam/AdGuardHome/internal/filtering"
)

// Put interface checks that cause import cycles here.

// type check
var _ filtering.Resolver = (*aghtest.Resolver)(nil)

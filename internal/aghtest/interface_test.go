package aghtest_test

import (
	"github.com/AdguardTeam/AdGuardHome/internal/aghtest"
	"github.com/AdguardTeam/AdGuardHome/internal/next/websvc"
)

// type check
var _ websvc.ServiceWithConfig[struct{}] = (*aghtest.ServiceWithConfig[struct{}])(nil)

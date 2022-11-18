//go:build linux

package aghnet

import (
	"github.com/AdguardTeam/AdGuardHome/internal/aghos"
)

func defaultHostsPaths() (paths []string) {
	paths = []string{"etc/hosts"}

	if aghos.IsOpenWrt() {
		paths = append(paths, "tmp/hosts")
	}

	return paths
}

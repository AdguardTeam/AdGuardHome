//go:generate packr clean
//go:generate packr -z
package main

import (
	"github.com/AdguardTeam/AdGuardHome/internal/home"
)

func main() {
	home.Main()
}

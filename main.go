//go:generate rm -f ./internal/home/a_home-packr.go
//go:generate packr -i ./internal/home -z
package main

import (
	"github.com/AdguardTeam/AdGuardHome/internal/home"
)

func main() {
	home.Main()
}

package main

import (
	"embed"

	"github.com/AdguardTeam/AdGuardHome/internal/home"
)

//go:embed build build2
var clientBuildFS embed.FS

func main() {
	home.Main(clientBuildFS)
}

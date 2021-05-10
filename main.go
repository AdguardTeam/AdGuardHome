package main

import (
	"embed"

	"github.com/AdguardTeam/AdGuardHome/internal/home"
	"github.com/AdguardTeam/AdGuardHome/internal/webembed"
)

//go:embed build/static build2/static
var webEmbed embed.FS

func main() {
	webembed.Embed(webEmbed)
	home.Main()
}

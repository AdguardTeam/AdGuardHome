//go:build !next

package main

import (
	"embed"
	// Embed tzdata in binary.
	//
	// See https://github.com/AdguardTeam/AdGuardHome/issues/6758
	_ "time/tzdata"

	"log"
	"net/http"

	"github.com/AdguardTeam/AdGuardHome/internal/home"
	"github.com/tukimoto/AdGuardHome/internal/handlers"
)

// Embed the prebuilt client here since we strive to keep .go files inside the
// internal directory and the embed package is unable to embed files located
// outside of the same or underlying directory.

//go:embed build
var clientBuildFS embed.FS

func main() {
	home.Main(clientBuildFS)

	http.HandleFunc("/api/license", handlers.GetLicenseInfo)
	log.Fatal(http.ListenAndServe(":8080", nil))
}

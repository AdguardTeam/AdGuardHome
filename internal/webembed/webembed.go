// Package webembed contains AdGuard Home Web-UI resources.
package webembed

import (
	"embed"
	"io/fs"
	"net/http"
)

var webEmbed embed.FS

// Embed - create embed
func Embed(e embed.FS) {
	webEmbed = e
}

// MakeFS - returns a embed http.FileSystem
func MakeFS(prefix string) http.FileSystem {
	subFS, err := fs.Sub(webEmbed, prefix)
	if err != nil {
		panic(err)
	}
	return http.FS(subFS)
}
